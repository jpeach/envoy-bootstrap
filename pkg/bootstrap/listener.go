package bootstrap

import (
	"fmt"
	"net"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_extensions_filters_network_http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"google.golang.org/protobuf/proto"
)

type Listener = envoy_config_listener_v3.Listener
type ListenerFilter = envoy_config_listener_v3.ListenerFilter
type Filter = envoy_config_listener_v3.Filter
type FilterChain = envoy_config_listener_v3.FilterChain
type FilterChainMatch = envoy_config_listener_v3.FilterChainMatch

type HTTPFilter = envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter

type CidrRange = envoy_config_core_v3.CidrRange

const TCP = envoy_config_core_v3.SocketAddress_TCP
const UDP = envoy_config_core_v3.SocketAddress_UDP

const INBOUND = envoy_config_core_v3.TrafficDirection_INBOUND
const OUTBOUND = envoy_config_core_v3.TrafficDirection_OUTBOUND

func NewFilter(name string, config proto.Message) *Filter {
	type TypedConfig = envoy_config_listener_v3.Filter_TypedConfig

	any, err := MarshalAny(config)
	if err != nil {
		panic(fmt.Errorf("failed to marshall %q type to Any: %s",
			config.ProtoReflect().Descriptor().FullName(), err))
	}

	return &Filter{
		Name: name,
		ConfigType: &TypedConfig{
			TypedConfig: any,
		},
	}
}

func NewHTTPFilter(name string, config proto.Message) *HTTPFilter {
	type TypedConfig = envoy_extensions_filters_network_http_connection_manager_v3.HttpFilter_TypedConfig

	any, err := MarshalAny(config)
	if err != nil {
		panic(fmt.Errorf("failed to marshall %q type to Any: %s",
			config.ProtoReflect().Descriptor().FullName(), err))
	}

	return &HTTPFilter{
		Name: name,
		ConfigType: &TypedConfig{
			TypedConfig: any,
		},
	}
}

// NewCidrForIP returns a /32 for IPv6 and a /128 for IPv6.
func NewCidrForIP(ip net.IP) *CidrRange {
	if ip != nil {
		nbytes := uint32(16)
		if ip.To4() != nil {
			nbytes = 4
		}

		return &CidrRange{
			AddressPrefix: ip.String(),
			PrefixLen:     UInt32(nbytes * 8),
		}
	}

	return nil
}
