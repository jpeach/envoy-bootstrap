package hacks

import (
	"fmt"
	"net"
	"time"

	envoy_extensions_filters_network_tcp_proxy_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/jpeach/envoy-bootstrap/pkg/bootstrap"
)

// NewTCPListener ...
func NewTCPListener(name string, port uint32) *bootstrap.Listener {
	anyAddr := bootstrap.NewSocketAddress(
		&bootstrap.SocketAddress{
			Protocol:      bootstrap.TCP,
			Address:       "0.0.0.0",
			PortSpecifier: bootstrap.NewPortValue(port),
			Ipv4Compat:    false,
		})

	chains := &bootstrap.FilterChain{
		FilterChainMatch: &bootstrap.FilterChainMatch{
			PrefixRanges: []*bootstrap.CidrRange{
				bootstrap.NewCidrForIP(net.ParseIP("127.0.0.8")),
			},
			SourceType:           0,
			SourcePrefixRanges:   nil,
			SourcePorts:          nil,
			ServerNames:          nil,
			TransportProtocol:    "",
			ApplicationProtocols: nil,
		},
		Filters: []*bootstrap.Filter{
			bootstrap.NewFilter("envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy",
				bootstrap.ProtoV2(&envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy{
					StatPrefix: fmt.Sprintf("%s:%s", name, port),
					ClusterSpecifier: &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_WeightedClusters{
						WeightedClusters: &envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_WeightedCluster{
							Clusters: []*envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_WeightedCluster_ClusterWeight{
								&envoy_extensions_filters_network_tcp_proxy_v3.TcpProxy_WeightedCluster_ClusterWeight{
									Name:   fmt.Sprintf("cluster/%s/%d", name, port),
									Weight: 100,
								},
							},
						},
					},
					IdleTimeout:        ptypes.DurationProto(time.Hour), // Default.
					AccessLog:          nil,
					MaxConnectAttempts: bootstrap.UInt32(5),
				}),
			),
		},
		UseProxyProto: bootstrap.False(),
	}

	return &bootstrap.Listener{
		Name:                             name,
		Address:                          anyAddr,
		FilterChains:                     []*bootstrap.FilterChain{chains},
		ListenerFilters:                  nil,
		ListenerFiltersTimeout:           ptypes.DurationProto(time.Second * 15), // Default.
		ContinueOnListenerFiltersTimeout: false,
		Transparent:                      bootstrap.False(),
		// XXX(jpeach) Setting this makes Envoy crash unless it is running on Linux. Need to filter it, depending on the node.
		// Freebind:         bootstrap.True(),
		SocketOptions:    nil,
		TrafficDirection: bootstrap.INBOUND,
		ReusePort:        true,
		AccessLog:        nil,
	}
}
