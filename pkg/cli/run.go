package cli

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"time"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/jpeach/envoy-bootstrap/pkg/bootstrap"
	"github.com/jpeach/envoy-bootstrap/pkg/xds"

	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
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

type runState struct {
	grpcServer *grpc.Server
	xdsServer  xds.Server
	snapshots  xds.SnapshotCache
}

func newServer() *runState {
	run := runState{}
	callbacks := xds.CallbackFuncs{
		StreamOpenFunc: func(ctx context.Context, streamID int64, typeURL string) error {
			log.Printf("opened stream %d for %q", streamID, typeURL)
			return nil
		},
	}

	options := []grpc.ServerOption{}
	run.grpcServer = grpc.NewServer(options...)

	run.snapshots = xds.NewSnapshotCache(xds.IDHash{}, &xds.StandardLogger{})
	run.xdsServer = xds.NewServer(context.Background(), run.snapshots, callbacks)

	xds.RegisterServer(run.grpcServer, run.xdsServer)

	return &run
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

func newManagementCluster(name string, addr *bootstrap.Address) *envoy_config_cluster_v3.Cluster {
	return &envoy_config_cluster_v3.Cluster{
		Name:                 "xds",
		ConnectTimeout:       ptypes.DurationProto(time.Second * 10),
		Http2ProtocolOptions: &envoy_config_core_v3.Http2ProtocolOptions{},
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
									Address: addr,
								},
							},
						}},
				}},
		},
	}
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
		newManagementCluster("xds",
			bootstrap.NewPipeAddress(&bootstrap.PipeAddress{
				Path: xdsSocketPath,
			}),
		),
	}

	envoyBootstrap.DynamicResources.CdsConfig = &bootstrap.ConfigSource{
		ConfigSourceSpecifier: bootstrap.NewAdsConfigSource(),
	}
	envoyBootstrap.DynamicResources.LdsConfig = &bootstrap.ConfigSource{
		ConfigSourceSpecifier: bootstrap.NewAdsConfigSource(),
	}
	envoyBootstrap.DynamicResources.AdsConfig = bootstrap.NewApiConfigSource("xds").ApiConfigSource

	if err := writeProtobuf(bootstrapPath, envoyBootstrap); err != nil {
		return err
	}

	// Need to listen before starting envoy, since it will fail to start if the socket isn't there.
	listener, err := net.Listen("unix", xdsSocketPath)
	if err != nil {
		return err
	}

	run := newServer()

	go func() {
		log.Printf("serving xDS on %s", xdsSocketPath)
		if err := run.grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC server failed: %s", err)
		}
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

	if err := envoyCmd.Start(); err != nil {
		log.Fatalf("%s", err)
	}

	if err := envoyCmd.Wait(); err != nil {
		log.Fatalf("%s", err)
	}

	return nil
}
