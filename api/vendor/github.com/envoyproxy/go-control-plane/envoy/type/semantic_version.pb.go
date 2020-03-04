// Code generated by protoc-gen-go. DO NOT EDIT.
// source: envoy/type/semantic_version.proto

package envoy_type

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
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

type SemanticVersion struct {
	MajorNumber          uint32   `protobuf:"varint,1,opt,name=major_number,json=majorNumber,proto3" json:"major_number,omitempty"`
	MinorNumber          uint32   `protobuf:"varint,2,opt,name=minor_number,json=minorNumber,proto3" json:"minor_number,omitempty"`
	Patch                uint32   `protobuf:"varint,3,opt,name=patch,proto3" json:"patch,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SemanticVersion) Reset()         { *m = SemanticVersion{} }
func (m *SemanticVersion) String() string { return proto.CompactTextString(m) }
func (*SemanticVersion) ProtoMessage()    {}
func (*SemanticVersion) Descriptor() ([]byte, []int) {
	return fileDescriptor_9d201ef6b9829a8e, []int{0}
}

func (m *SemanticVersion) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SemanticVersion.Unmarshal(m, b)
}
func (m *SemanticVersion) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SemanticVersion.Marshal(b, m, deterministic)
}
func (m *SemanticVersion) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SemanticVersion.Merge(m, src)
}
func (m *SemanticVersion) XXX_Size() int {
	return xxx_messageInfo_SemanticVersion.Size(m)
}
func (m *SemanticVersion) XXX_DiscardUnknown() {
	xxx_messageInfo_SemanticVersion.DiscardUnknown(m)
}

var xxx_messageInfo_SemanticVersion proto.InternalMessageInfo

func (m *SemanticVersion) GetMajorNumber() uint32 {
	if m != nil {
		return m.MajorNumber
	}
	return 0
}

func (m *SemanticVersion) GetMinorNumber() uint32 {
	if m != nil {
		return m.MinorNumber
	}
	return 0
}

func (m *SemanticVersion) GetPatch() uint32 {
	if m != nil {
		return m.Patch
	}
	return 0
}

func init() {
	proto.RegisterType((*SemanticVersion)(nil), "envoy.type.SemanticVersion")
}

func init() { proto.RegisterFile("envoy/type/semantic_version.proto", fileDescriptor_9d201ef6b9829a8e) }

var fileDescriptor_9d201ef6b9829a8e = []byte{
	// 164 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0x4c, 0xcd, 0x2b, 0xcb,
	0xaf, 0xd4, 0x2f, 0xa9, 0x2c, 0x48, 0xd5, 0x2f, 0x4e, 0xcd, 0x4d, 0xcc, 0x2b, 0xc9, 0x4c, 0x8e,
	0x2f, 0x4b, 0x2d, 0x2a, 0xce, 0xcc, 0xcf, 0xd3, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x02,
	0x2b, 0xd1, 0x03, 0x29, 0x51, 0xca, 0xe5, 0xe2, 0x0f, 0x86, 0xaa, 0x0a, 0x83, 0x28, 0x12, 0x52,
	0xe4, 0xe2, 0xc9, 0x4d, 0xcc, 0xca, 0x2f, 0x8a, 0xcf, 0x2b, 0xcd, 0x4d, 0x4a, 0x2d, 0x92, 0x60,
	0x54, 0x60, 0xd4, 0xe0, 0x0d, 0xe2, 0x06, 0x8b, 0xf9, 0x81, 0x85, 0xc0, 0x4a, 0x32, 0xf3, 0x10,
	0x4a, 0x98, 0xa0, 0x4a, 0x40, 0x62, 0x50, 0x25, 0x22, 0x5c, 0xac, 0x05, 0x89, 0x25, 0xc9, 0x19,
	0x12, 0xcc, 0x60, 0x39, 0x08, 0xc7, 0xc9, 0x88, 0x4b, 0x22, 0x33, 0x5f, 0x0f, 0x6c, 0x7f, 0x41,
	0x51, 0x7e, 0x45, 0xa5, 0x1e, 0xc2, 0x29, 0x4e, 0x22, 0x68, 0x0e, 0x09, 0x00, 0x39, 0x36, 0x80,
	0x31, 0x89, 0x0d, 0xec, 0x6a, 0x63, 0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0x4b, 0x89, 0x07, 0x6b,
	0xda, 0x00, 0x00, 0x00,
}
