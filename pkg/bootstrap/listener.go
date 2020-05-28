package bootstrap

import (
	"fmt"
	"net"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"google.golang.org/protobuf/proto"
)

type Listener = envoy_config_listener_v3.Listener
type Filter = envoy_config_listener_v3.Filter
type FilterChain = envoy_config_listener_v3.FilterChain
type FilterChainMatch = envoy_config_listener_v3.FilterChainMatch

type CidrRange = envoy_config_core_v3.CidrRange

const TCP = envoy_config_core_v3.SocketAddress_TCP
const UDP = envoy_config_core_v3.SocketAddress_UDP

const INBOUND = envoy_config_core_v3.TrafficDirection_INBOUND
const OUTBOUND = envoy_config_core_v3.TrafficDirection_OUTBOUND

func NewFilter(name string, config proto.Message) *Filter {
	any, err := MarshalAny(config)
	if err != nil {
		panic(fmt.Errorf("failed to marshall %q type to Any: %s",
			config.ProtoReflect().Descriptor().FullName(), err))
	}

	return &Filter{
		Name: name,
		ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
			TypedConfig: any,
		},
	}
}

// NewCidrForIP ...
func NewCidrForIP(ip net.IP) *CidrRange {
	var cidr *CidrRange

	if ip != nil {
		cidr = &CidrRange{
			AddressPrefix: ip.String(),
			PrefixLen:     UInt32(uint32(len(ip) * 8)),
		}
	}

	return cidr
}
