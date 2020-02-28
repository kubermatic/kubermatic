// Code generated by protoc-gen-go. DO NOT EDIT.
// source: envoy/config/listener/v2/api_listener.proto

package envoy_config_listener_v2

import (
	fmt "fmt"
	_ "github.com/cncf/udpa/go/udpa/annotations"
	proto "github.com/golang/protobuf/proto"
	any "github.com/golang/protobuf/ptypes/any"
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

type ApiListener struct {
	ApiListener          *any.Any `protobuf:"bytes,1,opt,name=api_listener,json=apiListener,proto3" json:"api_listener,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ApiListener) Reset()         { *m = ApiListener{} }
func (m *ApiListener) String() string { return proto.CompactTextString(m) }
func (*ApiListener) ProtoMessage()    {}
func (*ApiListener) Descriptor() ([]byte, []int) {
	return fileDescriptor_c474ffebb332e36e, []int{0}
}

func (m *ApiListener) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ApiListener.Unmarshal(m, b)
}
func (m *ApiListener) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ApiListener.Marshal(b, m, deterministic)
}
func (m *ApiListener) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ApiListener.Merge(m, src)
}
func (m *ApiListener) XXX_Size() int {
	return xxx_messageInfo_ApiListener.Size(m)
}
func (m *ApiListener) XXX_DiscardUnknown() {
	xxx_messageInfo_ApiListener.DiscardUnknown(m)
}

var xxx_messageInfo_ApiListener proto.InternalMessageInfo

func (m *ApiListener) GetApiListener() *any.Any {
	if m != nil {
		return m.ApiListener
	}
	return nil
}

func init() {
	proto.RegisterType((*ApiListener)(nil), "envoy.config.listener.v2.ApiListener")
}

func init() {
	proto.RegisterFile("envoy/config/listener/v2/api_listener.proto", fileDescriptor_c474ffebb332e36e)
}

var fileDescriptor_c474ffebb332e36e = []byte{
	// 204 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xd2, 0x4e, 0xcd, 0x2b, 0xcb,
	0xaf, 0xd4, 0x4f, 0xce, 0xcf, 0x4b, 0xcb, 0x4c, 0xd7, 0xcf, 0xc9, 0x2c, 0x2e, 0x49, 0xcd, 0x4b,
	0x2d, 0xd2, 0x2f, 0x33, 0xd2, 0x4f, 0x2c, 0xc8, 0x8c, 0x87, 0xf1, 0xf5, 0x0a, 0x8a, 0xf2, 0x4b,
	0xf2, 0x85, 0x24, 0xc0, 0x8a, 0xf5, 0x20, 0x8a, 0xf5, 0xe0, 0x92, 0x65, 0x46, 0x52, 0x92, 0xe9,
	0xf9, 0xf9, 0xe9, 0x39, 0xa9, 0xfa, 0x60, 0x75, 0x49, 0xa5, 0x69, 0xfa, 0x89, 0x79, 0x95, 0x10,
	0x4d, 0x52, 0x72, 0xa5, 0x29, 0x05, 0x89, 0xfa, 0x89, 0x79, 0x79, 0xf9, 0x25, 0x89, 0x25, 0x99,
	0xf9, 0x79, 0xc5, 0xfa, 0xb9, 0x99, 0xe9, 0x45, 0x89, 0x25, 0xa9, 0x10, 0x79, 0x25, 0x37, 0x2e,
	0x6e, 0xc7, 0x82, 0x4c, 0x1f, 0xa8, 0x61, 0x42, 0xe6, 0x5c, 0x3c, 0xc8, 0x36, 0x4b, 0x30, 0x2a,
	0x30, 0x6a, 0x70, 0x1b, 0x89, 0xe8, 0x41, 0x2c, 0xd0, 0x83, 0x59, 0xa0, 0xe7, 0x98, 0x57, 0x19,
	0xc4, 0x9d, 0x88, 0xd0, 0xe8, 0x14, 0xf3, 0x69, 0xc6, 0xbf, 0x7e, 0x56, 0x29, 0x9c, 0x8e, 0x34,
	0xe6, 0x52, 0xcb, 0xcc, 0xd7, 0x03, 0x4b, 0x16, 0x14, 0xe5, 0x57, 0x54, 0xea, 0xe1, 0xf2, 0x8c,
	0x93, 0x00, 0x92, 0x7b, 0x02, 0x40, 0xf6, 0x05, 0x30, 0x26, 0xb1, 0x81, 0x2d, 0x36, 0x06, 0x04,
	0x00, 0x00, 0xff, 0xff, 0x2a, 0x58, 0xbe, 0x26, 0x30, 0x01, 0x00, 0x00,
}
