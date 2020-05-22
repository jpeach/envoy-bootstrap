package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/jpeach/envoy-bootstrap/pkg/bootstrap"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// NewTypeCommand returns a "type" subcommand.
func NewTypeCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "type",
		Short: "Inspect Envoy protobuf API types",
	}

	cmd.AddCommand(
		Defaults(&cobra.Command{
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
					fmt.Printf("%s\n", n)
				}
			},
		}),

		Defaults(&cobra.Command{
			Use:   "show",
			Short: "Show the specified Envoy API types",
			RunE: func(cmd *cobra.Command, args []string) error {
				for _, a := range args {
					m, err := bootstrap.NewMessage(a)
					if err != nil {
						return err
					}

					bootstrap.FormatMessage(os.Stdout, m,
						&protojson.MarshalOptions{
							Indent:          "  ",
							Multiline:       true,
							EmitUnpopulated: true,
						})

					fmt.Fprintln(os.Stdout)
				}

				return nil
			},
		}),

		Defaults(&cobra.Command{
			Use:   "contains TYPENAME",
			Short: "List the Envoy API messages that contain the given type",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				names := []string{}

				m, err := bootstrap.NewMessage(args[0])
				if err != nil {
					return err
				}

				wanted := m.ProtoReflect().Descriptor().FullName()

				// For each registered type, collect its name if it contains
				// a field whose type is the one we want.
				protoregistry.GlobalTypes.RangeMessages(
					func(m protoreflect.MessageType) bool {
						fields := m.Descriptor().Fields()

						for i := 0; i < fields.Len(); i++ {
							f := fields.Get(i)

							if f.Kind() != protoreflect.MessageKind {
								continue
							}

							if f.Message().FullName() == wanted {
								names = append(names, string(m.Descriptor().FullName()))
								return true
							}
						}

						return true
					})

				sort.Strings(names)
				for _, n := range names {
					fmt.Printf("%s\n", n)
				}

				return nil
			},
		}),
	)

	return Defaults(&cmd)
}
