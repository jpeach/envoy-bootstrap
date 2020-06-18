package hacks

import (
	"fmt"
	"strings"
	"time"

	"github.com/jpeach/envoy-bootstrap/pkg/must"
	"github.com/jpeach/envoy-bootstrap/pkg/xds"

	envoy_extensions_filters_network_tcp_proxy_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/jpeach/envoy-bootstrap/pkg/bootstrap"
)

// HackTCPProxy ...
func HackTCPProxy(spec Spec) xds.Snapshot {
	addr := must.IP(spec.Parameters["address"].IP())
	name := must.String(spec.Parameters["name"].AsString())
	port := must.Int64(spec.Parameters["port"].AsInt64())
	cluster := must.String(spec.Parameters["cluster"].AsString())

	osname := must.String(spec.Parameters["os"].AsString())

	if cluster == "" {
		cluster = fmt.Sprintf("tcpproxy/cluster/%s/%d", name, port)
	}

	anyAddr := bootstrap.NewSocketAddress(
		&bootstrap.SocketAddress{
			Protocol:      bootstrap.TCP,
			Address:       addr.String(),
			PortSpecifier: bootstrap.NewPortValue(uint32(port)),
			Ipv4Compat:    false,
		})

	chains := &bootstrap.FilterChain{
		FilterChainMatch: &bootstrap.FilterChainMatch{
			PrefixRanges: []*bootstrap.CidrRange{
				bootstrap.NewCidrForIP(addr),
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
									Name:   cluster,
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

	listener := &bootstrap.Listener{
		Name:                             name,
		Address:                          anyAddr,
		FilterChains:                     []*bootstrap.FilterChain{chains},
		ListenerFilters:                  nil,
		ListenerFiltersTimeout:           ptypes.DurationProto(time.Second * 15), // Default.
		ContinueOnListenerFiltersTimeout: false,
		Transparent:                      bootstrap.False(),
		SocketOptions:                    nil,
		TrafficDirection:                 bootstrap.INBOUND,
		ReusePort:                        true,
		AccessLog:                        nil,
	}

	// NOTE(jpeach) Setting this makes Envoy crash unless it
	// is running on Linux. Need to filter it, depending on the
	// node.
	//
	// https://github.com/envoyproxy/envoy/issues/11340
	if strings.ToLower(osname) == "linux" {
		listener.Freebind = bootstrap.True()
	}

	snap := xds.Snapshot{}
	snap.Resources[xds.ListenerType] = xds.NewResources(NewVersion(), listener)

	return snap
}
