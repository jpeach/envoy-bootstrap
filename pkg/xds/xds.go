package xds

import (
	"context"
	"log"
	"os"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimeservice "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretservice "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	logger "github.com/envoyproxy/go-control-plane/pkg/log"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)

type Server = server.Server
type Callbacks = server.Callbacks
type CallbackFuncs = server.CallbackFuncs

type Snapshot = cache.Snapshot
type SnapshotCache = cache.SnapshotCache
type Cache = cache.Cache
type Resources = cache.Resources

type NodeHash = cache.NodeHash
type IDHash = cache.IDHash

type Logger = logger.Logger

type ResponseType = types.ResponseType

const (
	EndpointType = types.Endpoint
	ClusterType  = types.Cluster
	RouteType    = types.Route
	ListenerType = types.Listener
	SecretType   = types.Secret
	RuntimeType  = types.Runtime
	UnknownType  = types.UnknownType
)

type ConstantHash string

var _ cache.NodeHash = ConstantHash("")

func (c ConstantHash) ID(*envoy_config_core_v3.Node) string {
	return string(c)
}

// StandardLogger implements Logger using the Go log package.
type StandardLogger struct {
	log *log.Logger
}

var _ Logger = &StandardLogger{}

func (s *StandardLogger) logger() *log.Logger {
	// TODO(jpeach): should make this threadsafe :-(
	if s.log == nil {
		s.log = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	}

	return s.log
}

// Debugf logs a formatted debugging message.
func (s *StandardLogger) Debugf(format string, args ...interface{}) {
	s.logger().Printf(format, args...)
}

// Infof logs a formatted informational message.
func (s *StandardLogger) Infof(format string, args ...interface{}) {
	s.logger().Printf(format, args...)
}

// Warnf logs a formatted warning message.
func (s *StandardLogger) Warnf(format string, args ...interface{}) {
	s.logger().Printf(format, args...)
}

// Errorf logs a formatted error message.
func (s *StandardLogger) Errorf(format string, args ...interface{}) {
	s.logger().Printf(format, args...)
}

func RegisterServer(g *grpc.Server, x Server) {
	discovery.RegisterAggregatedDiscoveryServiceServer(g, x)
	endpointservice.RegisterEndpointDiscoveryServiceServer(g, x)
	clusterservice.RegisterClusterDiscoveryServiceServer(g, x)
	routeservice.RegisterRouteDiscoveryServiceServer(g, x)
	listenerservice.RegisterListenerDiscoveryServiceServer(g, x)
	secretservice.RegisterSecretDiscoveryServiceServer(g, x)
	runtimeservice.RegisterRuntimeDiscoveryServiceServer(g, x)
}

func NewServer(ctx context.Context, configCache Cache, cb Callbacks) Server {
	return server.NewServer(ctx, configCache, cb)
}

func NewSnapshotCache(hash cache.NodeHash, l Logger) SnapshotCache {
	return cache.NewSnapshotCache(true /* ads */, hash, l)
}

func NewResources(version string, items ...proto.Message) Resources {
	resources := make([]types.Resource, len(items), len(items))
	for n, i := range items {
		resources[n] = i
	}
	return cache.NewResources(version, resources)
}
