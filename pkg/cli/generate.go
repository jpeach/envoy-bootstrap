package cli

import (
	"fmt"
	"os"

	"github.com/jpeach/envoy-bootstrap/pkg/bootstrap"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func NewGenerateCommand() *cobra.Command {
	generate := cobra.Command{
		Use:   "generate [FLAGS ...]",
		Short: "Generate and Envoy bootstrap configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			b := bootstrap.NewBootstrap()

			b.StaticResources.Listeners = []*bootstrap.Listener{
				&bootstrap.Listener{
					Name: "default",
					Address: bootstrap.NewSocketAddress(&bootstrap.SocketAddress{
						Address:       "127.0.0.1",
						PortSpecifier: bootstrap.NewPortValue(443),
					}),
					FilterChains: []*bootstrap.FilterChain{
						&bootstrap.FilterChain{
							FilterChainMatch: &bootstrap.FilterChainMatch{
								ServerNames: []string{"target.example.com"},
							},
							Filters:       nil,
							UseProxyProto: &wrapperspb.BoolValue{Value: true},
							TransportSocket: &bootstrap.TransportSocket{
								Name:       "tls",
								ConfigType: nil,
							},
						},
					},
				},
			}

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
