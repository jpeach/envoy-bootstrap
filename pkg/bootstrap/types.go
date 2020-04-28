package bootstrap

import (
	envoy_config_bootstrap_v3 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
)

type Bootstrap = envoy_config_bootstrap_v3.Bootstrap
type Admin = envoy_config_bootstrap_v3.Admin

type Node = envoy_config_core_v3.Node

type Address = envoy_config_core_v3.Address
type SocketAddress = envoy_config_core_v3.SocketAddress
type PipeAddress = envoy_config_core_v3.Pipe

func NewSocketAddress(addr *SocketAddress) *Address {
	return &Address{Address: &envoy_config_core_v3.Address_SocketAddress{SocketAddress: addr}}
}

func NewPipeAddress(addr *PipeAddress) *Address {
	return &Address{Address: &envoy_config_core_v3.Address_Pipe{Pipe: addr}}
}
