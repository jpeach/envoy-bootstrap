package xds

import (
	"context"
	"log"
	"os"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimeservice "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretservice "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	logger "github.com/envoyproxy/go-control-plane/pkg/log"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"google.golang.org/grpc"
)

type Server = server.Server
type Callbacks = server.Callbacks

type Snapshot = cache.Snapshot
type SnapshotCache = cache.SnapshotCache
type Cache = cache.Cache
type NodeHash = cache.NodeHash
type IDHash = cache.IDHash

type Logger = logger.Logger

// CallbackFuncs is a convenience type for generating Server callbacks.
type CallbackFuncs struct {
	// OnStreamOpen is called once an xDS stream is open with a stream ID and the type URL (or "" for ADS).
	// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
	StreamOpenFunc func(context.Context, int64, string) error
	// OnStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
	StreamClosedFunc func(int64)
	// OnStreamRequest is called once a request is received on a stream.
	// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
	StreamRequestFunc func(int64, *discovery.DiscoveryRequest) error
	// OnStreamResponse is called immediately prior to sending a response on a stream.
	StreamResponseFunc func(int64, *discovery.DiscoveryRequest, *discovery.DiscoveryResponse)
	// OnFetchRequest is called for each Fetch request. Returning an error will end processing of the
	// request and respond with an error.
	FetchRequestFunc func(context.Context, *discovery.DiscoveryRequest) error
	// OnFetchResponse is called immediately prior to sending a response.
	FetchResponseFunc func(*discovery.DiscoveryRequest, *discovery.DiscoveryResponse)
}

var _ Callbacks = CallbackFuncs{}

// OnStreamOpen is called once an xDS stream is open with a stream ID and the type URL (or "" for ADS).
// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
func (c CallbackFuncs) OnStreamOpen(ctx context.Context, streamID int64, typeURL string) error {
	if c.StreamOpenFunc != nil {
		return c.StreamOpenFunc(ctx, streamID, typeURL)
	}

	return nil
}

// OnStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
func (c CallbackFuncs) OnStreamClosed(streamID int64) {
	if c.StreamClosedFunc != nil {
		c.StreamClosedFunc(streamID)
	}
}

// OnStreamRequest is called once a request is received on a stream.
// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
func (c CallbackFuncs) OnStreamRequest(streamID int64, req *discovery.DiscoveryRequest) error {
	if c.StreamRequestFunc != nil {
		return c.StreamRequestFunc(streamID, req)
	}

	return nil
}

// OnStreamResponse is called immediately prior to sending a response on a stream.
func (c CallbackFuncs) OnStreamResponse(streamID int64, req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	if c.StreamResponseFunc != nil {
		c.StreamResponseFunc(streamID, req, resp)
	}
}

// OnFetchRequest is called for each Fetch request. Returning an error will end processing of the
// request and respond with an error.
func (c CallbackFuncs) OnFetchRequest(ctx context.Context, req *discovery.DiscoveryRequest) error {
	if c.FetchRequestFunc != nil {
		return c.FetchRequestFunc(ctx, req)
	}

	return nil
}

// OnFetchResponse is called immediately prior to sending a response.
func (c CallbackFuncs) OnFetchResponse(req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	if c.FetchResponseFunc != nil {
		c.FetchResponseFunc(req, resp)
	}
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
