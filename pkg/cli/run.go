package cli

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"

	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"

	"github.com/jpeach/envoy-bootstrap/pkg/bootstrap"
)

// NewRunCommand ...
func NewRunCommand() *cobra.Command {
	run := cobra.Command{
		Use:   "run ENVOY [ENVOY ARGS...]",
		Short: "Bootstrap and run Envoy",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runEnvoy,
	}

	return Defaults(&run)
}

func writeProtobuf(path string, p proto.Message) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	if err := bootstrap.FormatMessage(file, proto.MessageV2(p), nil); err != nil {
		file.Close()
		return err
	}

	return file.Close()
}

func runEnvoy(cmd *cobra.Command, args []string) error {
	envoyPath := args[0]
	envoyArgs := args[1:]

	if err := unix.Access(envoyPath, unix.R_OK|unix.X_OK); err != nil {
		return fmt.Errorf("%s: %w", envoyPath, err)
	}

	tmpDir := path.Join(os.TempDir(), fmt.Sprintf("bootstrap.%d", os.Getpid()))
	if err := os.MkdirAll(tmpDir, 0750); err != nil {
		return err
	}

	bootstrapPath := path.Join(tmpDir, "bootstrap.conf")
	xdsSocketPath := path.Join(tmpDir, "xds.sock")

	// TODO(jpeach): Move this into core code so that the `bootstrap` and `run` commands generate the same thing.
	envoyBootstrap := bootstrap.NewBootstrap()

	envoyBootstrap.Admin = &bootstrap.Admin{
		AccessLogPath: "/dev/null",
		Address: bootstrap.NewPipeAddress(&bootstrap.PipeAddress{
			Path: path.Join(tmpDir, "admin.sock"),
			Mode: 0644,
		}),
	}

	// Configure a GRPC bootstrap cluster for the xDS socket. This has the minimum
	// number of required fields.
	envoyBootstrap.StaticResources.Clusters = []*envoy_config_cluster_v3.Cluster{
		&envoy_config_cluster_v3.Cluster{
			Name:           "xds",
			ConnectTimeout: ptypes.DurationProto(time.Second * 10),
			ClusterDiscoveryType: &envoy_config_cluster_v3.Cluster_Type{
				Type: envoy_config_cluster_v3.Cluster_STATIC,
			},
			LoadAssignment: &envoy_config_endpoint_v3.ClusterLoadAssignment{
				ClusterName: "xds",
				Endpoints: []*envoy_config_endpoint_v3.LocalityLbEndpoints{
					&envoy_config_endpoint_v3.LocalityLbEndpoints{
						LbEndpoints: []*envoy_config_endpoint_v3.LbEndpoint{
							&envoy_config_endpoint_v3.LbEndpoint{
								HostIdentifier: &envoy_config_endpoint_v3.LbEndpoint_Endpoint{
									Endpoint: &envoy_config_endpoint_v3.Endpoint{
										Address: bootstrap.NewPipeAddress(
											&bootstrap.PipeAddress{
												Path: xdsSocketPath,
											},
										),
									},
								},
							}},
					}},
			},
		},
	}

	envoyBootstrap.DynamicResources.CdsConfig = &bootstrap.ConfigSource{
		ConfigSourceSpecifier: bootstrap.NewApiConfigSource("xds"),
	}
	envoyBootstrap.DynamicResources.LdsConfig = &bootstrap.ConfigSource{
		ConfigSourceSpecifier: bootstrap.NewApiConfigSource("xds"),
	}

	if err := writeProtobuf(bootstrapPath, envoyBootstrap); err != nil {
		return err
	}

	// TODO(jpeach): This is trash, we need to start both
	// background tasks, then exit if either of them stop.
	// In the // foreground we would wait on a signal notification.
	wg := sync.WaitGroup{}

	wg.Add(2)

	// Need to listen before starting envoy, since it will fail to start if the socket isn't there.
	listener, err := net.Listen("unix", xdsSocketPath)
	if err != nil {
		return err
	}

	go func() {
		rpcServer := grpc.NewServer()

		if err := rpcServer.Serve(listener); err != nil {
			log.Printf("%s", err)
		}

		wg.Done()
	}()

	envoyCmd := exec.Cmd{
		Path: envoyPath,
		Args: append(
			[]string{envoyPath, "--config-path", bootstrapPath},
			envoyArgs...),
		Stdin:  nil,
		Stdout: cmd.OutOrStdout(),
		Stderr: cmd.ErrOrStderr(),
	}

	go func() {
		defer wg.Done()

		if err := envoyCmd.Start(); err != nil {
			log.Printf("%s", err)
			return
		}

		if err := envoyCmd.Wait(); err != nil {
			log.Printf("%s", err)
		}
	}()

	wg.Wait()
	return nil
}
