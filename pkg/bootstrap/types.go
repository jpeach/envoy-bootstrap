package bootstrap

import (
	"fmt"
	"io"

	"github.com/golang/protobuf/ptypes/any"
	"github.com/jpeach/envoy-bootstrap/pkg/must"

	envoy_config_bootstrap_v3 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	protov1 "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/wrappers"
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

func True() *wrappers.BoolValue {
	return &wrappers.BoolValue{Value: true}
}

func False() *wrappers.BoolValue {
	return &wrappers.BoolValue{Value: false}
}

func UInt32(i uint32) *wrappers.UInt32Value {
	return &wrappers.UInt32Value{Value: i}
}

func Int32(i int32) *wrappers.Int32Value {
	return &wrappers.Int32Value{Value: i}
}

func MarshalAny(message proto.Message) (*any.Any, error) {
	return ptypes.MarshalAny(ProtoV1(message))
}

// ProtoV1 converts a V2 message to V1.
func ProtoV1(message proto.Message) protov1.Message {
	return protov1.MessageV1(message)
}

// ProtoV2 converts a V1 message to V2.
func ProtoV2(message protov1.Message) proto.Message {
	return protov1.MessageV2(message)
}
