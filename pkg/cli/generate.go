package cli

import (
	"fmt"
	"os"

	"github.com/jpeach/envoy-bootstrap/pkg/bootstrap"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
)

func NewGenerateCommand() *cobra.Command{
	generate := cobra.Command{
		Use:   "generate [FLAGS ...]",
		Short: "Generate and Envoy bootstrap configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			b := bootstrap.NewBootstrap()

			data, err := protojson.MarshalOptions{
				Multiline: true,
				Indent: "  ",
			}.Marshal(proto.MessageV2(b))

			if err != nil {
				return err
			}

			os.Stdout.Write(data)
			fmt.Fprintln(os.Stdout)

			return nil
		},
	}

	return &generate
}
