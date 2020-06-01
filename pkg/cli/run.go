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

	"github.com/jpeach/envoy-bootstrap/pkg/bootstrap"
	"github.com/jpeach/envoy-bootstrap/pkg/hacks"
	"github.com/jpeach/envoy-bootstrap/pkg/must"
	"github.com/jpeach/envoy-bootstrap/pkg/xds"

	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_service_discovery_v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// NewRunCommand ...
func NewRunCommand() *cobra.Command {
	run := cobra.Command{
		Use:   "run ENVOY [ENVOY ARGS...]",
		Short: "Bootstrap and run Envoy",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runEnvoy,
	}

	run.Flags().StringArray("hack", []string{}, "Hack workload specification")

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
			log.Printf("[%d] opened stream for %q", streamID, typeURL)
			return nil
		},
		StreamRequestFunc: func(streamID int64, request *envoy_service_discovery_v3.DiscoveryRequest) error {
			log.Printf("[%d] requesting %s", streamID, request.GetTypeUrl())
			log.Printf("[%d] wanted resources %s", streamID, request.GetResourceNames())
			return nil
		},
		StreamResponseFunc: func(streamID int64, request *envoy_service_discovery_v3.DiscoveryRequest, response *envoy_service_discovery_v3.DiscoveryResponse) {
			log.Printf("[%d] %s", streamID, response)
		},
	}

	options := []grpc.ServerOption{}
	run.grpcServer = grpc.NewServer(options...)

	// NOTE(jpeach): we use ConstantHash so that we server all nodes the same resources.
	run.snapshots = xds.NewSnapshotCache(xds.ConstantHash("*"), &xds.StandardLogger{})
	run.xdsServer = xds.NewServer(context.Background(), run.snapshots, callbacks)

	xds.RegisterServer(run.grpcServer, run.xdsServer)

	return &run
}

func writeProtobuf(path string, message proto.Message) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	if err := bootstrap.FormatMessage(file, message, nil); err != nil {
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
		ResourceApiVersion:    envoy_config_core_v3.ApiVersion_V3,
	}

	envoyBootstrap.DynamicResources.LdsConfig = &bootstrap.ConfigSource{
		ConfigSourceSpecifier: bootstrap.NewAdsConfigSource(),
		ResourceApiVersion:    envoy_config_core_v3.ApiVersion_V3,
	}

	envoyBootstrap.DynamicResources.AdsConfig = bootstrap.NewApiConfigSource("xds").ApiConfigSource
	envoyBootstrap.DynamicResources.AdsConfig.TransportApiVersion = envoy_config_core_v3.ApiVersion_V3

	if err := writeProtobuf(bootstrapPath, bootstrap.ProtoV2(envoyBootstrap)); err != nil {
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
		Args: func() []string {
			args := []string{envoyPath}
			args = append(args, envoyArgs...)
			args = append(args, "--config-path", bootstrapPath)
			return args
		}(),
		Stdin:  os.Stdin,
		Stdout: cmd.OutOrStdout(),
		Stderr: cmd.ErrOrStderr(),
	}

	if err := envoyCmd.Start(); err != nil {
		log.Fatalf("%s", err)
	}

	for _, h := range must.StringSlice(cmd.Flags().GetStringArray("hack")) {
		spec, err := hacks.ParseSpec(h)
		if err != nil {
			return fmt.Errorf("invalid hack spec %q: %w", h, err)
		}

		var snap xds.Snapshot

		switch spec.Hack {
		case "tcpproxy":
			snap = hacks.HackTCPProxy(spec)
		default:
			return fmt.Errorf("invalid hack spec %q: not found", h)
		}

		// NOTE(jpeach): The NodeID we pass here matches the ConstantHash value.
		if err := run.snapshots.SetSnapshot("*", snap); err != nil {
			log.Printf("ERROR: %s", err)
		}
	}

	if err := envoyCmd.Wait(); err != nil {
		log.Fatalf("envoy exited: %s", err)
	}

	return nil
}
