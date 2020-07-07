package cli

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/jpeach/envoy-bootstrap/pkg/bootstrap"
	"github.com/jpeach/envoy-bootstrap/pkg/must"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// NewTypeCommand returns a "type" subcommand.
func NewTypeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "type",
		Short: "Inspect Envoy protobuf API types",
	}

	cmd.AddCommand(
		Defaults(NewTypeListCommand()),
		Defaults(NewTypeShowCommand()),
		Defaults(NewTypeURLCommand()),
		Defaults(NewTypeContainsCommand()),
		Defaults(NewCRDGenCommand()),
	)

	return cmd
}

// NewTypeListCommand ...
func NewTypeListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Envoy API types",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			names := []string{}

			// Just find all the names that are registered. We
			// don't filter for `envoy` package names because
			// the Envoy API pulls in other protobuf definitions
			// (e.g Prometheus and OpenCensus) and we want to see
			// those too.
			protoregistry.GlobalTypes.RangeMessages(
				func(m protoreflect.MessageType) bool {
					names = append(names, string(m.Descriptor().FullName()))
					return true
				})

			sort.Strings(names)
			for _, n := range names {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", n)
			}
		},
	}
}

// NewTypeShowCommand ...
func NewTypeShowCommand() *cobra.Command {
	formatters := map[string]func(io.Writer, proto.Message){}

	formatters["json"] = func(out io.Writer, m proto.Message) {
		bootstrap.FormatMessage(out, m,
			&protojson.MarshalOptions{
				Indent:          "  ",
				Multiline:       true,
				EmitUnpopulated: true,
			})

		fmt.Fprintln(out)
	}

	formatters["yaml"] = func(out io.Writer, m proto.Message) {
		jsonBytes := bytes.Buffer{}

		bootstrap.FormatMessage(&jsonBytes, m,
			&protojson.MarshalOptions{
				Indent:          "  ",
				Multiline:       true,
				EmitUnpopulated: true,
			})

		yamlBytes, _ := yaml.JSONToYAML(jsonBytes.Bytes())

		out.Write(yamlBytes)
		fmt.Fprintln(out)
	}

	formatters["fields"] = func(out io.Writer, m proto.Message) {
		w := tabwriter.NewWriter(out, 8, 8, 2, ' ', 0)

		fields := m.ProtoReflect().Descriptor().Fields()
		for i := 0; i < fields.Len(); i++ {
			f := fields.Get(i)
			cardinality := ""

			if f.Cardinality() == protoreflect.Repeated {
				cardinality = " (repeated)"
			}

			switch f.Kind() {
			case protoreflect.MessageKind:
				fmt.Fprintf(w, "%s\t%s%s\n", f.Name(), f.Message().FullName(), cardinality)
			case protoreflect.EnumKind:
				fmt.Fprintf(w, "%s\t%s%s\n", f.Name(), f.Enum().FullName(), cardinality)
				fmt.Fprintf(w, "\t  %s\n", f.Enum().Values())
			default:
				fmt.Fprintf(w, "%s\t%s%s\n", f.Name(), f.Kind(), cardinality)
			}
		}

		w.Flush()
	}

	cmd := &cobra.Command{
		Use:   "show [OPTIONS] TYPENAME",
		Short: "Show the specified Envoy API types",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			message, err := bootstrap.NewMessage(args[0])
			if err != nil {
				return err
			}

			format := must.String(cmd.Flags().GetString("format"))
			f, ok := formatters[format]
			if !ok {
				return fmt.Errorf("invalid format %q", format)
			}

			f(cmd.OutOrStdout(), message)
			return nil
		},
	}

	cmd.Flags().StringP("format", "f", "fields", `Output format ("fields", "yaml" or "json")`)

	return cmd
}

// NewTypeContainsCommand ...
func NewTypeContainsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "contains TYPENAME",
		Short: "List the Envoy API messages that contain the given type",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			names := []string{}
			wanted := args[0]

			// For each registered type, collect its name if it contains
			// a field whose type is the one we want.
			protoregistry.GlobalTypes.RangeMessages(
				func(m protoreflect.MessageType) bool {
					fields := m.Descriptor().Fields()

					for i := 0; i < fields.Len(); i++ {
						f := fields.Get(i)

						switch f.Kind() {
						case protoreflect.MessageKind:
							if string(f.Message().FullName()) == wanted {
								names = append(names, string(m.Descriptor().FullName()))
								return true
							}
						case protoreflect.EnumKind:
							if string(f.Enum().FullName()) == wanted {
								names = append(names, string(m.Descriptor().FullName()))
								return true
							}
						}

					}

					return true
				})

			sort.Strings(names)
			for _, n := range names {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", n)
			}

			return nil
		},
	}
}

// NewTypeURLCommand ...
func NewTypeURLCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "url TYPENAME",
		Short: "Show the gRPC type URL for the given Envoy API types",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, typeName := range args {
				message, err := bootstrap.NewMessage(typeName)
				if err != nil {
					return err
				}

				any, err := bootstrap.MarshalAny(message)
				if err != nil {
					return err
				}

				fmt.Printf("%s\n", any.TypeUrl)
			}
			return nil
		},
	}
}
