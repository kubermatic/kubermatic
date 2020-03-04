// Code generated by protoc-gen-go. DO NOT EDIT.
// source: envoy/api/v2/core/address.proto

package envoy_api_v2_core

import (
	fmt "fmt"
	_ "github.com/cncf/udpa/go/udpa/annotations"
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	proto "github.com/golang/protobuf/proto"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type SocketAddress_Protocol int32

const (
	SocketAddress_TCP SocketAddress_Protocol = 0
	SocketAddress_UDP SocketAddress_Protocol = 1
)

var SocketAddress_Protocol_name = map[int32]string{
	0: "TCP",
	1: "UDP",
}

var SocketAddress_Protocol_value = map[string]int32{
	"TCP": 0,
	"UDP": 1,
}

func (x SocketAddress_Protocol) String() string {
	return proto.EnumName(SocketAddress_Protocol_name, int32(x))
}

func (SocketAddress_Protocol) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_6906417f87bcce55, []int{1, 0}
}

type Pipe struct {
	Path                 string   `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	Mode                 uint32   `protobuf:"varint,2,opt,name=mode,proto3" json:"mode,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Pipe) Reset()         { *m = Pipe{} }
func (m *Pipe) String() string { return proto.CompactTextString(m) }
func (*Pipe) ProtoMessage()    {}
func (*Pipe) Descriptor() ([]byte, []int) {
	return fileDescriptor_6906417f87bcce55, []int{0}
}

func (m *Pipe) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Pipe.Unmarshal(m, b)
}
func (m *Pipe) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Pipe.Marshal(b, m, deterministic)
}
func (m *Pipe) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Pipe.Merge(m, src)
}
func (m *Pipe) XXX_Size() int {
	return xxx_messageInfo_Pipe.Size(m)
}
func (m *Pipe) XXX_DiscardUnknown() {
	xxx_messageInfo_Pipe.DiscardUnknown(m)
}

var xxx_messageInfo_Pipe proto.InternalMessageInfo

func (m *Pipe) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

func (m *Pipe) GetMode() uint32 {
	if m != nil {
		return m.Mode
	}
	return 0
}

type SocketAddress struct {
	Protocol SocketAddress_Protocol `protobuf:"varint,1,opt,name=protocol,proto3,enum=envoy.api.v2.core.SocketAddress_Protocol" json:"protocol,omitempty"`
	Address  string                 `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`
	// Types that are valid to be assigned to PortSpecifier:
	//	*SocketAddress_PortValue
	//	*SocketAddress_NamedPort
	PortSpecifier        isSocketAddress_PortSpecifier `protobuf_oneof:"port_specifier"`
	ResolverName         string                        `protobuf:"bytes,5,opt,name=resolver_name,json=resolverName,proto3" json:"resolver_name,omitempty"`
	Ipv4Compat           bool                          `protobuf:"varint,6,opt,name=ipv4_compat,json=ipv4Compat,proto3" json:"ipv4_compat,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                      `json:"-"`
	XXX_unrecognized     []byte                        `json:"-"`
	XXX_sizecache        int32                         `json:"-"`
}

func (m *SocketAddress) Reset()         { *m = SocketAddress{} }
func (m *SocketAddress) String() string { return proto.CompactTextString(m) }
func (*SocketAddress) ProtoMessage()    {}
func (*SocketAddress) Descriptor() ([]byte, []int) {
	return fileDescriptor_6906417f87bcce55, []int{1}
}

func (m *SocketAddress) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SocketAddress.Unmarshal(m, b)
}
func (m *SocketAddress) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SocketAddress.Marshal(b, m, deterministic)
}
func (m *SocketAddress) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SocketAddress.Merge(m, src)
}
func (m *SocketAddress) XXX_Size() int {
	return xxx_messageInfo_SocketAddress.Size(m)
}
func (m *SocketAddress) XXX_DiscardUnknown() {
	xxx_messageInfo_SocketAddress.DiscardUnknown(m)
}

var xxx_messageInfo_SocketAddress proto.InternalMessageInfo

func (m *SocketAddress) GetProtocol() SocketAddress_Protocol {
	if m != nil {
		return m.Protocol
	}
	return SocketAddress_TCP
}

func (m *SocketAddress) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

type isSocketAddress_PortSpecifier interface {
	isSocketAddress_PortSpecifier()
}

type SocketAddress_PortValue struct {
	PortValue uint32 `protobuf:"varint,3,opt,name=port_value,json=portValue,proto3,oneof"`
}

type SocketAddress_NamedPort struct {
	NamedPort string `protobuf:"bytes,4,opt,name=named_port,json=namedPort,proto3,oneof"`
}

func (*SocketAddress_PortValue) isSocketAddress_PortSpecifier() {}

func (*SocketAddress_NamedPort) isSocketAddress_PortSpecifier() {}

func (m *SocketAddress) GetPortSpecifier() isSocketAddress_PortSpecifier {
	if m != nil {
		return m.PortSpecifier
	}
	return nil
}

func (m *SocketAddress) GetPortValue() uint32 {
	if x, ok := m.GetPortSpecifier().(*SocketAddress_PortValue); ok {
		return x.PortValue
	}
	return 0
}

func (m *SocketAddress) GetNamedPort() string {
	if x, ok := m.GetPortSpecifier().(*SocketAddress_NamedPort); ok {
		return x.NamedPort
	}
	return ""
}

func (m *SocketAddress) GetResolverName() string {
	if m != nil {
		return m.ResolverName
	}
	return ""
}

func (m *SocketAddress) GetIpv4Compat() bool {
	if m != nil {
		return m.Ipv4Compat
	}
	return false
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*SocketAddress) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*SocketAddress_PortValue)(nil),
		(*SocketAddress_NamedPort)(nil),
	}
}

type TcpKeepalive struct {
	KeepaliveProbes      *wrappers.UInt32Value `protobuf:"bytes,1,opt,name=keepalive_probes,json=keepaliveProbes,proto3" json:"keepalive_probes,omitempty"`
	KeepaliveTime        *wrappers.UInt32Value `protobuf:"bytes,2,opt,name=keepalive_time,json=keepaliveTime,proto3" json:"keepalive_time,omitempty"`
	KeepaliveInterval    *wrappers.UInt32Value `protobuf:"bytes,3,opt,name=keepalive_interval,json=keepaliveInterval,proto3" json:"keepalive_interval,omitempty"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *TcpKeepalive) Reset()         { *m = TcpKeepalive{} }
func (m *TcpKeepalive) String() string { return proto.CompactTextString(m) }
func (*TcpKeepalive) ProtoMessage()    {}
func (*TcpKeepalive) Descriptor() ([]byte, []int) {
	return fileDescriptor_6906417f87bcce55, []int{2}
}

func (m *TcpKeepalive) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TcpKeepalive.Unmarshal(m, b)
}
func (m *TcpKeepalive) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TcpKeepalive.Marshal(b, m, deterministic)
}
func (m *TcpKeepalive) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TcpKeepalive.Merge(m, src)
}
func (m *TcpKeepalive) XXX_Size() int {
	return xxx_messageInfo_TcpKeepalive.Size(m)
}
func (m *TcpKeepalive) XXX_DiscardUnknown() {
	xxx_messageInfo_TcpKeepalive.DiscardUnknown(m)
}

var xxx_messageInfo_TcpKeepalive proto.InternalMessageInfo

func (m *TcpKeepalive) GetKeepaliveProbes() *wrappers.UInt32Value {
	if m != nil {
		return m.KeepaliveProbes
	}
	return nil
}

func (m *TcpKeepalive) GetKeepaliveTime() *wrappers.UInt32Value {
	if m != nil {
		return m.KeepaliveTime
	}
	return nil
}

func (m *TcpKeepalive) GetKeepaliveInterval() *wrappers.UInt32Value {
	if m != nil {
		return m.KeepaliveInterval
	}
	return nil
}

type BindConfig struct {
	SourceAddress        *SocketAddress      `protobuf:"bytes,1,opt,name=source_address,json=sourceAddress,proto3" json:"source_address,omitempty"`
	Freebind             *wrappers.BoolValue `protobuf:"bytes,2,opt,name=freebind,proto3" json:"freebind,omitempty"`
	SocketOptions        []*SocketOption     `protobuf:"bytes,3,rep,name=socket_options,json=socketOptions,proto3" json:"socket_options,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *BindConfig) Reset()         { *m = BindConfig{} }
func (m *BindConfig) String() string { return proto.CompactTextString(m) }
func (*BindConfig) ProtoMessage()    {}
func (*BindConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_6906417f87bcce55, []int{3}
}

func (m *BindConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BindConfig.Unmarshal(m, b)
}
func (m *BindConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BindConfig.Marshal(b, m, deterministic)
}
func (m *BindConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BindConfig.Merge(m, src)
}
func (m *BindConfig) XXX_Size() int {
	return xxx_messageInfo_BindConfig.Size(m)
}
func (m *BindConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_BindConfig.DiscardUnknown(m)
}

var xxx_messageInfo_BindConfig proto.InternalMessageInfo

func (m *BindConfig) GetSourceAddress() *SocketAddress {
	if m != nil {
		return m.SourceAddress
	}
	return nil
}

func (m *BindConfig) GetFreebind() *wrappers.BoolValue {
	if m != nil {
		return m.Freebind
	}
	return nil
}

func (m *BindConfig) GetSocketOptions() []*SocketOption {
	if m != nil {
		return m.SocketOptions
	}
	return nil
}

type Address struct {
	// Types that are valid to be assigned to Address:
	//	*Address_SocketAddress
	//	*Address_Pipe
	Address              isAddress_Address `protobuf_oneof:"address"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *Address) Reset()         { *m = Address{} }
func (m *Address) String() string { return proto.CompactTextString(m) }
func (*Address) ProtoMessage()    {}
func (*Address) Descriptor() ([]byte, []int) {
	return fileDescriptor_6906417f87bcce55, []int{4}
}

func (m *Address) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Address.Unmarshal(m, b)
}
func (m *Address) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Address.Marshal(b, m, deterministic)
}
func (m *Address) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Address.Merge(m, src)
}
func (m *Address) XXX_Size() int {
	return xxx_messageInfo_Address.Size(m)
}
func (m *Address) XXX_DiscardUnknown() {
	xxx_messageInfo_Address.DiscardUnknown(m)
}

var xxx_messageInfo_Address proto.InternalMessageInfo

type isAddress_Address interface {
	isAddress_Address()
}

type Address_SocketAddress struct {
	SocketAddress *SocketAddress `protobuf:"bytes,1,opt,name=socket_address,json=socketAddress,proto3,oneof"`
}

type Address_Pipe struct {
	Pipe *Pipe `protobuf:"bytes,2,opt,name=pipe,proto3,oneof"`
}

func (*Address_SocketAddress) isAddress_Address() {}

func (*Address_Pipe) isAddress_Address() {}

func (m *Address) GetAddress() isAddress_Address {
	if m != nil {
		return m.Address
	}
	return nil
}

func (m *Address) GetSocketAddress() *SocketAddress {
	if x, ok := m.GetAddress().(*Address_SocketAddress); ok {
		return x.SocketAddress
	}
	return nil
}

func (m *Address) GetPipe() *Pipe {
	if x, ok := m.GetAddress().(*Address_Pipe); ok {
		return x.Pipe
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*Address) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*Address_SocketAddress)(nil),
		(*Address_Pipe)(nil),
	}
}

type CidrRange struct {
	AddressPrefix        string                `protobuf:"bytes,1,opt,name=address_prefix,json=addressPrefix,proto3" json:"address_prefix,omitempty"`
	PrefixLen            *wrappers.UInt32Value `protobuf:"bytes,2,opt,name=prefix_len,json=prefixLen,proto3" json:"prefix_len,omitempty"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *CidrRange) Reset()         { *m = CidrRange{} }
func (m *CidrRange) String() string { return proto.CompactTextString(m) }
func (*CidrRange) ProtoMessage()    {}
func (*CidrRange) Descriptor() ([]byte, []int) {
	return fileDescriptor_6906417f87bcce55, []int{5}
}

func (m *CidrRange) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CidrRange.Unmarshal(m, b)
}
func (m *CidrRange) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CidrRange.Marshal(b, m, deterministic)
}
func (m *CidrRange) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CidrRange.Merge(m, src)
}
func (m *CidrRange) XXX_Size() int {
	return xxx_messageInfo_CidrRange.Size(m)
}
func (m *CidrRange) XXX_DiscardUnknown() {
	xxx_messageInfo_CidrRange.DiscardUnknown(m)
}

var xxx_messageInfo_CidrRange proto.InternalMessageInfo

func (m *CidrRange) GetAddressPrefix() string {
	if m != nil {
		return m.AddressPrefix
	}
	return ""
}

func (m *CidrRange) GetPrefixLen() *wrappers.UInt32Value {
	if m != nil {
		return m.PrefixLen
	}
	return nil
}

func init() {
	proto.RegisterEnum("envoy.api.v2.core.SocketAddress_Protocol", SocketAddress_Protocol_name, SocketAddress_Protocol_value)
	proto.RegisterType((*Pipe)(nil), "envoy.api.v2.core.Pipe")
	proto.RegisterType((*SocketAddress)(nil), "envoy.api.v2.core.SocketAddress")
	proto.RegisterType((*TcpKeepalive)(nil), "envoy.api.v2.core.TcpKeepalive")
	proto.RegisterType((*BindConfig)(nil), "envoy.api.v2.core.BindConfig")
	proto.RegisterType((*Address)(nil), "envoy.api.v2.core.Address")
	proto.RegisterType((*CidrRange)(nil), "envoy.api.v2.core.CidrRange")
}

func init() { proto.RegisterFile("envoy/api/v2/core/address.proto", fileDescriptor_6906417f87bcce55) }

var fileDescriptor_6906417f87bcce55 = []byte{
	// 711 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x54, 0x4d, 0x6f, 0xd3, 0x4a,
	0x14, 0x8d, 0xe3, 0xb4, 0x49, 0x6e, 0x9b, 0xbc, 0x74, 0xf4, 0xde, 0xab, 0xd5, 0x17, 0xbd, 0x84,
	0xb0, 0x09, 0x95, 0xb0, 0xa5, 0x14, 0xb1, 0xaf, 0xc3, 0x47, 0xab, 0x02, 0x35, 0xa6, 0x65, 0x6b,
	0x4d, 0xec, 0x49, 0x18, 0xd5, 0xf1, 0x8c, 0xc6, 0x8e, 0x69, 0x77, 0xa8, 0x4b, 0x36, 0x2c, 0xd8,
	0xf0, 0x9f, 0xf8, 0x11, 0xac, 0xd9, 0xb2, 0x42, 0x5d, 0x50, 0x34, 0xe3, 0x4c, 0x0a, 0x04, 0x54,
	0xd8, 0x79, 0xee, 0xbd, 0xe7, 0xcc, 0x39, 0x33, 0x67, 0x0c, 0x1d, 0x92, 0xe4, 0xec, 0xcc, 0xc1,
	0x9c, 0x3a, 0xf9, 0xc0, 0x09, 0x99, 0x20, 0x0e, 0x8e, 0x22, 0x41, 0xd2, 0xd4, 0xe6, 0x82, 0x65,
	0x0c, 0x6d, 0xa8, 0x01, 0x1b, 0x73, 0x6a, 0xe7, 0x03, 0x5b, 0x0e, 0x6c, 0xb5, 0x97, 0x31, 0x23,
	0x9c, 0x92, 0x02, 0xb0, 0xf5, 0xff, 0x84, 0xb1, 0x49, 0x4c, 0x1c, 0xb5, 0x1a, 0xcd, 0xc6, 0xce,
	0x4b, 0x81, 0x39, 0x27, 0x22, 0xd5, 0xfd, 0x59, 0xc4, 0xb1, 0x83, 0x93, 0x84, 0x65, 0x38, 0xa3,
	0x2c, 0x49, 0x9d, 0x29, 0x9d, 0x08, 0x9c, 0x69, 0xfc, 0x66, 0x8e, 0x63, 0x1a, 0xe1, 0x8c, 0x38,
	0xfa, 0xa3, 0x68, 0xf4, 0x76, 0xa1, 0xe2, 0x51, 0x4e, 0xd0, 0x7f, 0x50, 0xe1, 0x38, 0x7b, 0x61,
	0x19, 0x5d, 0xa3, 0x5f, 0x77, 0xab, 0x17, 0x6e, 0x45, 0x94, 0xbb, 0x86, 0xaf, 0x8a, 0xa8, 0x0d,
	0x95, 0x29, 0x8b, 0x88, 0x55, 0xee, 0x1a, 0xfd, 0x86, 0x5b, 0xbb, 0x70, 0x57, 0xb6, 0x4d, 0xeb,
	0xd2, 0xf4, 0x55, 0xb5, 0xf7, 0xbe, 0x0c, 0x8d, 0x67, 0x2c, 0x3c, 0x21, 0xd9, 0x6e, 0x61, 0x12,
	0x1d, 0x42, 0x4d, 0xb1, 0x87, 0x2c, 0x56, 0x84, 0xcd, 0xc1, 0x2d, 0x7b, 0xc9, 0xb1, 0xfd, 0x1d,
	0xc6, 0xf6, 0xe6, 0x00, 0x45, 0x7f, 0x6e, 0x94, 0x5b, 0x86, 0xbf, 0x20, 0x41, 0x37, 0xa0, 0x3a,
	0x3f, 0x40, 0xa5, 0xe1, 0x1b, 0x81, 0xba, 0x8e, 0xb6, 0x01, 0x38, 0x13, 0x59, 0x90, 0xe3, 0x78,
	0x46, 0x2c, 0x53, 0x29, 0xad, 0x5f, 0xb8, 0xab, 0xdb, 0x15, 0xeb, 0xf2, 0xd2, 0xdc, 0x2b, 0xf9,
	0x75, 0xd9, 0x7e, 0x2e, 0xbb, 0xa8, 0x03, 0x90, 0xe0, 0x29, 0x89, 0x02, 0x59, 0xb2, 0x2a, 0x92,
	0x51, 0x0e, 0xa8, 0x9a, 0xc7, 0x44, 0x86, 0x6e, 0x42, 0x43, 0x90, 0x94, 0xc5, 0x39, 0x11, 0x81,
	0xac, 0x5a, 0x2b, 0x72, 0xc6, 0x5f, 0xd7, 0xc5, 0x27, 0x78, 0x2a, 0x59, 0xd6, 0x28, 0xcf, 0xef,
	0x04, 0x21, 0x9b, 0x72, 0x9c, 0x59, 0xab, 0x5d, 0xa3, 0x5f, 0xf3, 0x41, 0x96, 0x86, 0xaa, 0xd2,
	0x6b, 0x43, 0x4d, 0xbb, 0x42, 0x55, 0x30, 0x8f, 0x86, 0x5e, 0xab, 0x24, 0x3f, 0x8e, 0xef, 0x79,
	0x2d, 0xc3, 0xfd, 0x07, 0x9a, 0x4a, 0x70, 0xca, 0x49, 0x48, 0xc7, 0x94, 0x08, 0x64, 0x7e, 0x76,
	0x8d, 0xde, 0x47, 0x03, 0xd6, 0x8f, 0x42, 0x7e, 0x40, 0x08, 0xc7, 0x31, 0xcd, 0x09, 0x7a, 0x08,
	0xad, 0x13, 0xbd, 0x08, 0xb8, 0x60, 0x23, 0x92, 0xaa, 0x43, 0x5d, 0x1b, 0xb4, 0xed, 0x22, 0x15,
	0xb6, 0x4e, 0x85, 0x7d, 0xbc, 0x9f, 0x64, 0x3b, 0x03, 0x65, 0xd2, 0xff, 0x6b, 0x81, 0xf2, 0x14,
	0x08, 0x0d, 0xa1, 0x79, 0x45, 0x94, 0xd1, 0x69, 0x71, 0x9f, 0xd7, 0xd1, 0x34, 0x16, 0x98, 0x23,
	0x3a, 0x25, 0xe8, 0x00, 0xd0, 0x15, 0x09, 0x4d, 0x32, 0x22, 0x72, 0x1c, 0xab, 0xe3, 0xbe, 0x8e,
	0x68, 0x63, 0x81, 0xdb, 0x9f, 0xc3, 0x7a, 0x1f, 0x0c, 0x00, 0x97, 0x26, 0xd1, 0x90, 0x25, 0x63,
	0x3a, 0x41, 0x4f, 0xa1, 0x99, 0xb2, 0x99, 0x08, 0x49, 0xa0, 0x2f, 0xbb, 0xf0, 0xd9, 0xbd, 0x2e,
	0x3c, 0x2a, 0x33, 0xaf, 0x55, 0x66, 0x1a, 0x05, 0x83, 0x4e, 0xe2, 0x5d, 0xa8, 0x8d, 0x05, 0x21,
	0x23, 0x9a, 0x44, 0x73, 0xb7, 0x5b, 0x4b, 0x22, 0x5d, 0xc6, 0xe2, 0x42, 0xe2, 0x62, 0x16, 0x3d,
	0x90, 0x52, 0xe4, 0x0e, 0x01, 0xe3, 0xea, 0x3d, 0x59, 0x66, 0xd7, 0xec, 0xaf, 0x0d, 0x3a, 0xbf,
	0x94, 0x72, 0xa8, 0xe6, 0xe4, 0xfe, 0x57, 0xab, 0xb4, 0xf7, 0xd6, 0x80, 0xaa, 0xd6, 0xb2, 0xbf,
	0xe0, 0xfc, 0x43, 0x7b, 0x7b, 0x25, 0x4d, 0xab, 0xa9, 0x6e, 0x43, 0x85, 0x53, 0xae, 0x2f, 0x70,
	0xf3, 0x27, 0x04, 0xf2, 0x51, 0xef, 0x95, 0x7c, 0x35, 0xe6, 0x36, 0x17, 0xcf, 0xa7, 0xc8, 0xd8,
	0xb9, 0x01, 0xf5, 0x21, 0x8d, 0x84, 0x8f, 0x93, 0x09, 0x41, 0x36, 0x34, 0xe7, 0xdd, 0x80, 0x0b,
	0x32, 0xa6, 0xa7, 0x3f, 0xfe, 0x04, 0x1a, 0xf3, 0xb6, 0xa7, 0xba, 0xe8, 0x3e, 0x40, 0x31, 0x17,
	0xc4, 0x24, 0xf9, 0x9d, 0x0c, 0xe9, 0x3f, 0xc6, 0x2b, 0xc3, 0xaf, 0x17, 0xc8, 0x47, 0x24, 0x71,
	0x1f, 0x7f, 0x7a, 0xf7, 0xe5, 0xcd, 0xca, 0xbf, 0xe8, 0xef, 0x42, 0x7c, 0xa8, 0x32, 0x50, 0x88,
	0xcf, 0x77, 0xa0, 0x43, 0x59, 0xe1, 0x8a, 0x0b, 0x76, 0x7a, 0xb6, 0x6c, 0xd0, 0x5d, 0xdf, 0xd5,
	0xa2, 0x58, 0xc6, 0x3c, 0x63, 0xb4, 0xaa, 0x76, 0xde, 0xf9, 0x1a, 0x00, 0x00, 0xff, 0xff, 0x69,
	0xf0, 0x91, 0x75, 0x7c, 0x05, 0x00, 0x00,
}
