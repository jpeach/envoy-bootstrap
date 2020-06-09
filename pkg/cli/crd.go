package cli

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"reflect"
	"strings"
	"text/template"

	"github.com/jpeach/envoy-bootstrap/pkg/protogen"

	protov1 "github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func NewCRDGenCommand() *cobra.Command {
	run := cobra.Command{
		Use:   "crdgen TYPENAME",
		Short: "Generate a Kubernetes CRD type for the given Envoy type.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			typeName := args[0]

			msg, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(typeName))
			if err != nil {
				return fmt.Errorf("failed to find type %q: %s", typeName, err)
			}

			var buf bytes.Buffer

			if err := generateCustomResource(msg, &buf); err != nil {
				return fmt.Errorf("failed to generate CRD: %s", err)
			}

			data, err := format.Source(buf.Bytes())
			if err != nil {
				return fmt.Errorf("failed to format CRD source: %s", err)
			}

			fmt.Printf("%s\n", string(data))
			return nil
		},
	}

	return &run
}

type ResourceTypeInfo struct {
	// Name of the root resource.
	Name string

	// Name of the CRD package.
	Package string

	// Package import path for the protobuf type.
	ImportPath string

	// Import alias name to use for the protobuf type package.
	ImportName string

	// Root protobuf message.
	MessageType protoreflect.MessageType
}

type NestedTypeGenerators struct {
	Map map[protoreflect.FullName]func() error
}

func (n *NestedTypeGenerators) Push(k protoreflect.FullName, v func() error) {
	n.Map[k] = v
}

func StringOrDefault(which bool, val string, def string) string {
	if which {
		return val
	}

	return def
}

func generateSkeleton(info *ResourceTypeInfo, out io.Writer) error {
	skel := `
package {{ .Package }}

import (
        metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
        {{ .ImportPath }} "{{ .ImportName }}"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type {{ .Name }} struct {
        metav1.TypeMeta {{ json ",inline" }}
        metav1.ObjectMeta {{ json "metadata,omitempty" }}

        Spec {{ .Name }}Spec {{ json "spec,omitempty" }}
        Status {{ .Name }}Status {{ json "status,omitempty" }}
}

// +kubebuilder:object:root=true
type {{ .Name }}List struct {
        metav1.TypeMeta {{ json ",inline" }}
        metav1.ListMeta {{ json "metadata,omitempty" }}
        Items           []{{ .Name }} {{ json "items" }}
}

type {{ .Name }}Status struct {
	Conditions []{{ .Name }}Condition {{ json "conditions" }}
}

// See https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/1623-standardize-conditions.
type {{ .Name }}Condition struct {
	// Type of condition in CamelCase or in foo.example.com/CamelCase.
	// Many .condition.type values are consistent across resources like Available, but because arbitrary conditions can be
	// useful (see .node.status.conditions), the ability to deconflict is important.
	// +required
	Type string {{ json "type" }}

	// Status of the condition, one of True, False, Unknown.
	// +required
	Status metav1.ConditionStatus {{ json "status"  }}

	// If set, this represents the .metadata.generation that the condition was set based upon.
	// For instance, if .metadata.generation is currently 12, but the .status.condition[x].observedGeneration is 9, the condition is out of date
	// with respect to the current state of the instance.
	// +optional
	ObservedGeneration int64 {{ json "observedGeneration,omitempty"  }}

	// Last time the condition transitioned from one status to another.
	// This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.
	// +required
	LastTransitionTime metav1.Time {{ json "lastTransitionTime"  }}

	// The reason for the condition's last transition in CamelCase.
	// The specific API may choose whether or not this field is considered a guaranteed API.
	// This field may not be empty.
	// +required
	Reason string {{ json "reason"  }}

	// A human readable message indicating details about the transition.
	// This field may be empty.
	// +required
	Message string {{ json "message"  }}
}
`

	t, err := template.New("skeleton").
		Funcs(template.FuncMap{
			"json": func(params string) string {
				return fmt.Sprintf("`json:\"%s\"`", params)
			},
		}).
		Parse(skel)
	if err != nil {
		return err
	}

	return t.Execute(out, info)
}

func generateSpec(info *ResourceTypeInfo, out io.Writer) error {
	nested := NestedTypeGenerators{
		Map: map[protoreflect.FullName]func() error{},
	}

	if err := generateMessage(info.MessageType.Descriptor(), fmt.Sprintf("%sSpec", info.Name), &nested, out); err != nil {
		return nil
	}

	fmt.Fprintf(out, "\n")

	visited := map[protoreflect.FullName]struct{}{}

	// Each time we generate a message, more nested messages
	// might need generating. Keep track of types we have already
	// generated, then keep going until we have generated everything.
	for {
		done := true
		for typeName, gen := range nested.Map {
			if _, ok := visited[typeName]; !ok {
				if err := gen(); err != nil {
					panic(err.Error())
					return err
				}

				visited[typeName] = struct{}{}
				done = false
			}
		}

		if done {
			break
		}
	}

	return nil
}

func enumTypeName(enum protoreflect.EnumDescriptor) string {
	typeName := protogen.GoCamelCase(string(enum.Name()))

	if !strings.HasSuffix(typeName, "Type") {
		return typeName + "Type"
	}

	return typeName
}

func enumValueName(enum protoreflect.EnumDescriptor, value protoreflect.EnumValueDescriptor) string {
	typeName := enumTypeName(enum)
	valueName := protogen.GoCamelCase(strings.ToLower(string(value.Name()))) + typeName

	return protogen.GoCamelCase(valueName)
}

func generateMessage(msg protoreflect.MessageDescriptor, structName string, nested *NestedTypeGenerators, out io.Writer) error {
	fmt.Fprintf(out, "\n// %s\n", msg.FullName())
	fmt.Fprintf(out, "type %s struct {\n", structName)

	fields := msg.Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)

		if i > 0 {
			fmt.Fprintf(out, "\n")
		}

		fmt.Fprintf(out, "// %s\n", f.FullName())

		_, isPointer := protogen.FieldGoType(f)
		if isPointer {
			fmt.Fprintf(out, "// +optional\n")
		}

		switch f.Kind() {
		case protoreflect.MessageKind:
			fmt.Fprintf(out, "%s %s%s%s `json:\"%s\"`\n",
				protogen.GoCamelCase(string(f.Name())),
				StringOrDefault(f.IsList(), "[]", ""),
				StringOrDefault(isPointer, "*", ""),
				protogen.GoCamelCase(string(f.Message().Name())),
				protogen.JSONCamelCase(string(f.Name())),
			)

			nested.Push(
				f.Message().FullName(),
				func() error {
					return generateMessage(f.Message(),
						protogen.GoCamelCase(string(f.Message().Name())),
						nested,
						out,
					)
				})

		case protoreflect.GroupKind:
			fmt.Fprintf(out, "// TODO: Group field %s not implemented\n", f.FullName())
			fmt.Fprintf(out, "%s %s%s `json:\"%s\"`\n",
				protogen.GoCamelCase(string(f.Name())),
				StringOrDefault(f.IsList(), "[]", ""),
				protogen.GoCamelCase(string(f.Message().Name())),
				protogen.JSONCamelCase(string(f.Name())),
			)

		case protoreflect.EnumKind:
			fmt.Fprintf(out, "%s %s `json:\"%s\"`\n",
				protogen.GoCamelCase(string(f.Name())),
				enumTypeName(f.Enum()),
				protogen.JSONCamelCase(string(f.Name())),
			)

			nested.Push(
				f.Enum().FullName(),
				func() error {
					return generateEnum(f.Enum(), out)
				})

		case protoreflect.FloatKind, protoreflect.DoubleKind:
			// CRD generation doesn't support floating point.
			// https://github.com/kubernetes-sigs/controller-tools/issues/245
			fmt.Fprintf(out, "// TODO: Floating point field %s not implemented\n", f.FullName())

		default:
			// Deal with scalar fields.

			typeName, _ := protogen.FieldGoType(f)

			fmt.Fprintf(out, "%s %s%s `json:\"%s\"`\n",
				protogen.GoCamelCase(string(f.Name())),
				StringOrDefault(isPointer, "*", ""),
				typeName,
				protogen.JSONCamelCase(string(f.Name())),
			)
		}
	}

	fmt.Fprintf(out, "\n}\n")

	return nil
}

func generateEnum(enum protoreflect.EnumDescriptor, out io.Writer) error {
	typeName := enumTypeName(enum)

	fmt.Fprintf(out, "// %s\n", enum.FullName())
	fmt.Fprintf(out, "type %s string\n", typeName)

	values := enum.Values()
	for i := 0; i < values.Len(); i++ {
		v := values.Get(i)
		fmt.Fprintf(out, "// %s\n", v.FullName())
		fmt.Fprintf(out, "const %s %s = \"%s\"\n",
			enumValueName(enum, v),
			typeName,
			v.Name())
	}

	return nil
}

func generateCustomResource(msg protoreflect.MessageType, out io.Writer) error {
	// The protov1 message gives us *Type, but to get the package
	// path from reflection, we need to use Elem() and reflection
	// on the dereferenced Type.
	msgType := reflect.TypeOf(protov1.MessageV1(msg.Zero().Interface())).Elem()

	info := ResourceTypeInfo{
		MessageType: msg,
		Name:        string(msg.Descriptor().Name()),
		Package:     string(msg.Descriptor().Parent().Name()),
		ImportName:  msgType.PkgPath(),
		ImportPath:  strings.Replace(string(msg.Descriptor().Parent().FullName()), ".", "_", -1),
	}

	if err := generateSkeleton(&info, out); err != nil {
		return err
	}

	if err := generateSpec(&info, out); err != nil {
		return err
	}

	return nil
}
