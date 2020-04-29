package cli

import (
	"fmt"
	"os"

	"github.com/jpeach/envoy-bootstrap/pkg/bootstrap"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
)

func NewGenerateCommand() *cobra.Command {
	generate := cobra.Command{
		Use:   "generate [FLAGS ...]",
		Short: "Generate and Envoy bootstrap configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			b := bootstrap.NewBootstrap()

			bootstrap.FormatMessage(os.Stdout, proto.MessageV2(b), nil)
			fmt.Fprintln(os.Stdout)

			for _, a := range args {
				m, err := bootstrap.NewMessage(a)
				if err != nil {
					return err
				}

				bootstrap.FormatMessage(os.Stdout, m,
					&protojson.MarshalOptions{
						Multiline:       true,
						Indent:          "  ",
						EmitUnpopulated: true,
					})
				fmt.Fprintln(os.Stdout)
			}

			return nil
		},
	}

	return &generate
}
