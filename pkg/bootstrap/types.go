package bootstrap

import (
	"fmt"
	"io"

	"github.com/jpeach/envoy-bootstrap/pkg/must"

	envoy_config_bootstrap_v3 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type Bootstrap = envoy_config_bootstrap_v3.Bootstrap
type Admin = envoy_config_bootstrap_v3.Admin

type Node = envoy_config_core_v3.Node

type Address = envoy_config_core_v3.Address
type SocketAddress = envoy_config_core_v3.SocketAddress
type PipeAddress = envoy_config_core_v3.Pipe

type PortValue = envoy_config_core_v3.SocketAddress_PortValue
type NamedPort = envoy_config_core_v3.SocketAddress_NamedPort

type TransportSocket = envoy_config_core_v3.TransportSocket

type Listener = envoy_config_listener_v3.Listener
type FilterChain = envoy_config_listener_v3.FilterChain
type FilterChainMatch = envoy_config_listener_v3.FilterChainMatch

func NewSocketAddress(addr *SocketAddress) *Address {
	return &Address{Address: &envoy_config_core_v3.Address_SocketAddress{SocketAddress: addr}}
}

func NewPipeAddress(addr *PipeAddress) *Address {
	return &Address{Address: &envoy_config_core_v3.Address_Pipe{Pipe: addr}}
}

func NewPortValue(val uint32) *PortValue {
	return &PortValue{PortValue: val}
}

func NewNamedPort(name string) *NamedPort {
	return &NamedPort{NamedPort: name}
}

func NewMessage(typeName string) (proto.Message, error) {
	mtype, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(typeName))
	if err != nil {
		return nil, fmt.Errorf("message type %q: %s", typeName, err)
	}

	return mtype.New().Interface(), nil
}

func FormatMessage(out io.Writer, m proto.Message, marshal *protojson.MarshalOptions) error {
	if marshal == nil {
		marshal = &protojson.MarshalOptions{
			Multiline: true,
			Indent:    "  ",
		}
	}

	_, err := out.Write(must.Bytes(marshal.Marshal(m)))
	return err
}
