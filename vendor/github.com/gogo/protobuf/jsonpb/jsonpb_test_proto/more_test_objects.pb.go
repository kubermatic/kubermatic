// Code generated by protoc-gen-gogo.
// source: more_test_objects.proto
// DO NOT EDIT!

/*
Package jsonpb is a generated protocol buffer package.

It is generated from these files:
	more_test_objects.proto
	test_objects.proto

It has these top-level messages:
	Simple3
	Mappy
	Simple
	Repeats
	Widget
	Maps
	MsgWithOneof
	Real
	Complex
*/
package jsonpb

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.GoGoProtoPackageIsVersion1

type Simple3 struct {
	Dub float64 `protobuf:"fixed64,1,opt,name=dub,proto3" json:"dub,omitempty"`
}

func (m *Simple3) Reset()                    { *m = Simple3{} }
func (m *Simple3) String() string            { return proto.CompactTextString(m) }
func (*Simple3) ProtoMessage()               {}
func (*Simple3) Descriptor() ([]byte, []int) { return fileDescriptorMoreTestObjects, []int{0} }

type Mappy struct {
	Nummy map[int64]int32    `protobuf:"bytes,1,rep,name=nummy" json:"nummy,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	Strry map[string]string  `protobuf:"bytes,2,rep,name=strry" json:"strry,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Objjy map[int32]*Simple3 `protobuf:"bytes,3,rep,name=objjy" json:"objjy,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value"`
	Buggy map[int64]string   `protobuf:"bytes,4,rep,name=buggy" json:"buggy,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Booly map[bool]bool      `protobuf:"bytes,5,rep,name=booly" json:"booly,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

func (m *Mappy) Reset()                    { *m = Mappy{} }
func (m *Mappy) String() string            { return proto.CompactTextString(m) }
func (*Mappy) ProtoMessage()               {}
func (*Mappy) Descriptor() ([]byte, []int) { return fileDescriptorMoreTestObjects, []int{1} }

func (m *Mappy) GetNummy() map[int64]int32 {
	if m != nil {
		return m.Nummy
	}
	return nil
}

func (m *Mappy) GetStrry() map[string]string {
	if m != nil {
		return m.Strry
	}
	return nil
}

func (m *Mappy) GetObjjy() map[int32]*Simple3 {
	if m != nil {
		return m.Objjy
	}
	return nil
}

func (m *Mappy) GetBuggy() map[int64]string {
	if m != nil {
		return m.Buggy
	}
	return nil
}

func (m *Mappy) GetBooly() map[bool]bool {
	if m != nil {
		return m.Booly
	}
	return nil
}

func init() {
	proto.RegisterType((*Simple3)(nil), "jsonpb.Simple3")
	proto.RegisterType((*Mappy)(nil), "jsonpb.Mappy")
}

var fileDescriptorMoreTestObjects = []byte{
	// 288 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x84, 0x92, 0x41, 0x4b, 0xc3, 0x30,
	0x14, 0xc7, 0xe9, 0xba, 0xcc, 0xf5, 0xed, 0xa0, 0x14, 0xc1, 0xa0, 0x17, 0x19, 0x08, 0x3d, 0xe5,
	0xb0, 0x5d, 0x86, 0x47, 0xc1, 0x83, 0x07, 0x15, 0xba, 0x0f, 0x30, 0x16, 0x0d, 0xc3, 0xda, 0x36,
	0x21, 0x4d, 0x85, 0x7c, 0x25, 0x3f, 0xa5, 0x7d, 0x69, 0x67, 0xc3, 0x08, 0xec, 0xf6, 0xca, 0xff,
	0xf7, 0x83, 0xf7, 0x7f, 0x0d, 0xdc, 0x54, 0x52, 0x8b, 0x9d, 0x11, 0x8d, 0xd9, 0x49, 0x5e, 0x88,
	0x0f, 0xd3, 0x30, 0xa5, 0xa5, 0x91, 0xe9, 0xac, 0x68, 0x64, 0xad, 0xf8, 0xf2, 0x0e, 0x2e, 0xb6,
	0x5f, 0x95, 0x2a, 0xc5, 0x3a, 0xbd, 0x82, 0xf8, 0xb3, 0xe5, 0x34, 0xba, 0x8f, 0xb2, 0x28, 0xc7,
	0x71, 0xf9, 0x3b, 0x05, 0xf2, 0xba, 0x57, 0xca, 0xa6, 0x0c, 0x48, 0xdd, 0x56, 0x95, 0xed, 0xd2,
	0x38, 0x5b, 0xac, 0x28, 0xeb, 0x75, 0xe6, 0x52, 0xf6, 0x86, 0xd1, 0x73, 0x6d, 0xb4, 0xcd, 0x7b,
	0x0c, 0xf9, 0xc6, 0x68, 0x6d, 0xe9, 0x24, 0xc4, 0x6f, 0x31, 0x1a, 0x78, 0x87, 0x21, 0xdf, 0xed,
	0x57, 0x58, 0x1a, 0x87, 0xf8, 0x77, 0x8c, 0x06, 0xde, 0x61, 0xc8, 0xf3, 0xf6, 0x70, 0xb0, 0x74,
	0x1a, 0xe2, 0x9f, 0x30, 0x1a, 0x78, 0x87, 0x39, 0x5e, 0xca, 0xd2, 0x52, 0x12, 0xe4, 0x31, 0x3a,
	0xf2, 0x38, 0xdf, 0x6e, 0x00, 0xc6, 0x52, 0x78, 0x99, 0x6f, 0x61, 0xdd, 0x65, 0xe2, 0x1c, 0xc7,
	0xf4, 0x1a, 0xc8, 0xcf, 0xbe, 0x6c, 0x45, 0xd7, 0x2f, 0xca, 0x48, 0xde, 0x7f, 0x3c, 0x4e, 0x36,
	0x11, 0x9a, 0x63, 0x3d, 0xdf, 0x4c, 0x02, 0x66, 0xe2, 0x9b, 0x2f, 0x00, 0x63, 0x51, 0xdf, 0x24,
	0xbd, 0xf9, 0xe0, 0x9b, 0x8b, 0xd5, 0xe5, 0xb1, 0xc3, 0xf0, 0xff, 0x4e, 0x96, 0x18, 0x6f, 0x70,
	0x6e, 0xfd, 0xe4, 0xd4, 0xfc, 0xbf, 0x86, 0x6f, 0xce, 0x03, 0xe6, 0xdc, 0x33, 0xf9, 0xcc, 0x3d,
	0xac, 0xf5, 0x5f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x39, 0x61, 0xe7, 0x9b, 0x73, 0x02, 0x00, 0x00,
}
