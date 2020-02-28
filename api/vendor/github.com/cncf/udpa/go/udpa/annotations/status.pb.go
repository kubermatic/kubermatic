// Code generated by protoc-gen-go. DO NOT EDIT.
// source: udpa/annotations/status.proto

package udpa_annotations

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	descriptor "github.com/golang/protobuf/protoc-gen-go/descriptor"
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

type StatusAnnotation struct {
	WorkInProgress       bool     `protobuf:"varint,1,opt,name=work_in_progress,json=workInProgress,proto3" json:"work_in_progress,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StatusAnnotation) Reset()         { *m = StatusAnnotation{} }
func (m *StatusAnnotation) String() string { return proto.CompactTextString(m) }
func (*StatusAnnotation) ProtoMessage()    {}
func (*StatusAnnotation) Descriptor() ([]byte, []int) {
	return fileDescriptor_011cc2e7e491b0ff, []int{0}
}

func (m *StatusAnnotation) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StatusAnnotation.Unmarshal(m, b)
}
func (m *StatusAnnotation) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StatusAnnotation.Marshal(b, m, deterministic)
}
func (m *StatusAnnotation) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StatusAnnotation.Merge(m, src)
}
func (m *StatusAnnotation) XXX_Size() int {
	return xxx_messageInfo_StatusAnnotation.Size(m)
}
func (m *StatusAnnotation) XXX_DiscardUnknown() {
	xxx_messageInfo_StatusAnnotation.DiscardUnknown(m)
}

var xxx_messageInfo_StatusAnnotation proto.InternalMessageInfo

func (m *StatusAnnotation) GetWorkInProgress() bool {
	if m != nil {
		return m.WorkInProgress
	}
	return false
}

var E_FileStatus = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FileOptions)(nil),
	ExtensionType: (*StatusAnnotation)(nil),
	Field:         222707719,
	Name:          "udpa.annotations.file_status",
	Tag:           "bytes,222707719,opt,name=file_status",
	Filename:      "udpa/annotations/status.proto",
}

func init() {
	proto.RegisterType((*StatusAnnotation)(nil), "udpa.annotations.StatusAnnotation")
	proto.RegisterExtension(E_FileStatus)
}

func init() { proto.RegisterFile("udpa/annotations/status.proto", fileDescriptor_011cc2e7e491b0ff) }

var fileDescriptor_011cc2e7e491b0ff = []byte{
	// 191 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x2d, 0x4d, 0x29, 0x48,
	0xd4, 0x4f, 0xcc, 0xcb, 0xcb, 0x2f, 0x49, 0x2c, 0xc9, 0xcc, 0xcf, 0x2b, 0xd6, 0x2f, 0x2e, 0x49,
	0x2c, 0x29, 0x2d, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12, 0x00, 0x49, 0xeb, 0x21, 0x49,
	0x4b, 0x29, 0xa4, 0xe7, 0xe7, 0xa7, 0xe7, 0xa4, 0xea, 0x83, 0xe5, 0x93, 0x4a, 0xd3, 0xf4, 0x53,
	0x52, 0x8b, 0x93, 0x8b, 0x32, 0x0b, 0x4a, 0xf2, 0x8b, 0x20, 0x7a, 0x94, 0x6c, 0xb8, 0x04, 0x82,
	0xc1, 0x66, 0x38, 0xc2, 0xb5, 0x09, 0x69, 0x70, 0x09, 0x94, 0xe7, 0x17, 0x65, 0xc7, 0x67, 0xe6,
	0xc5, 0x17, 0x14, 0xe5, 0xa7, 0x17, 0xa5, 0x16, 0x17, 0x4b, 0x30, 0x2a, 0x30, 0x6a, 0x70, 0x04,
	0xf1, 0x81, 0xc4, 0x3d, 0xf3, 0x02, 0xa0, 0xa2, 0x56, 0x29, 0x5c, 0xdc, 0x69, 0x99, 0x39, 0xa9,
	0xf1, 0x10, 0x67, 0x08, 0xc9, 0xe8, 0x41, 0xec, 0xd3, 0x83, 0xd9, 0xa7, 0xe7, 0x96, 0x99, 0x93,
	0xea, 0x5f, 0x00, 0x76, 0x8c, 0x44, 0x7b, 0xc3, 0xcc, 0x2c, 0x05, 0x46, 0x0d, 0x6e, 0x23, 0x25,
	0x3d, 0x74, 0x87, 0xea, 0xa1, 0xbb, 0x21, 0x88, 0x0b, 0x64, 0x2e, 0x44, 0x34, 0x89, 0x0d, 0x6c,
	0x9c, 0x31, 0x20, 0x00, 0x00, 0xff, 0xff, 0xda, 0x3c, 0x97, 0x43, 0xff, 0x00, 0x00, 0x00,
}
