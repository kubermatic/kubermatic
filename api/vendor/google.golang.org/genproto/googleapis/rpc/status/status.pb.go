// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/rpc/status.proto

package status

import (
	fmt "fmt"
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
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// The `Status` type defines a logical error model that is suitable for different
// programming environments, including REST APIs and RPC APIs. It is used by
// [gRPC](https://github.com/grpc). The error model is designed to be:
//
// - Simple to use and understand for most users
// - Flexible enough to meet unexpected needs
//
// # Overview
//
// The `Status` message contains three pieces of data: error code, error message,
// and error details. The error code should be an enum value of
// [google.rpc.Code][google.rpc.Code], but it may accept additional error codes if needed.  The
// error message should be a developer-facing English message that helps
// developers *understand* and *resolve* the error. If a localized user-facing
// error message is needed, put the localized message in the error details or
// localize it in the client. The optional error details may contain arbitrary
// information about the error. There is a predefined set of error detail types
// in the package `google.rpc` that can be used for common error conditions.
//
// # Language mapping
//
// The `Status` message is the logical representation of the error model, but it
// is not necessarily the actual wire format. When the `Status` message is
// exposed in different client libraries and different wire protocols, it can be
// mapped differently. For example, it will likely be mapped to some exceptions
// in Java, but more likely mapped to some error codes in C.
//
// # Other uses
//
// The error model and the `Status` message can be used in a variety of
// environments, either with or without APIs, to provide a
// consistent developer experience across different environments.
//
// Example uses of this error model include:
//
// - Partial errors. If a service needs to return partial errors to the client,
//     it may embed the `Status` in the normal response to indicate the partial
//     errors.
//
// - Workflow errors. A typical workflow has multiple steps. Each step may
//     have a `Status` message for error reporting.
//
// - Batch operations. If a client uses batch request and batch response, the
//     `Status` message should be used directly inside batch response, one for
//     each error sub-response.
//
// - Asynchronous operations. If an API call embeds asynchronous operation
//     results in its response, the status of those operations should be
//     represented directly using the `Status` message.
//
// - Logging. If some API errors are stored in logs, the message `Status` could
//     be used directly after any stripping needed for security/privacy reasons.
type Status struct {
	// The status code, which should be an enum value of [google.rpc.Code][google.rpc.Code].
	Code int32 `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	// A developer-facing error message, which should be in English. Any
	// user-facing error message should be localized and sent in the
	// [google.rpc.Status.details][google.rpc.Status.details] field, or localized by the client.
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	// A list of messages that carry the error details.  There is a common set of
	// message types for APIs to use.
	Details              []*any.Any `protobuf:"bytes,3,rep,name=details,proto3" json:"details,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *Status) Reset()         { *m = Status{} }
func (m *Status) String() string { return proto.CompactTextString(m) }
func (*Status) ProtoMessage()    {}
func (*Status) Descriptor() ([]byte, []int) {
	return fileDescriptor_24d244abaf643bfe, []int{0}
}

func (m *Status) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Status.Unmarshal(m, b)
}
func (m *Status) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Status.Marshal(b, m, deterministic)
}
func (m *Status) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Status.Merge(m, src)
}
func (m *Status) XXX_Size() int {
	return xxx_messageInfo_Status.Size(m)
}
func (m *Status) XXX_DiscardUnknown() {
	xxx_messageInfo_Status.DiscardUnknown(m)
}

var xxx_messageInfo_Status proto.InternalMessageInfo

func (m *Status) GetCode() int32 {
	if m != nil {
		return m.Code
	}
	return 0
}

func (m *Status) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *Status) GetDetails() []*any.Any {
	if m != nil {
		return m.Details
	}
	return nil
}

func init() {
	proto.RegisterType((*Status)(nil), "google.rpc.Status")
}

func init() { proto.RegisterFile("google/rpc/status.proto", fileDescriptor_24d244abaf643bfe) }

var fileDescriptor_24d244abaf643bfe = []byte{
	// 209 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x4f, 0xcf, 0xcf, 0x4f,
	0xcf, 0x49, 0xd5, 0x2f, 0x2a, 0x48, 0xd6, 0x2f, 0x2e, 0x49, 0x2c, 0x29, 0x2d, 0xd6, 0x2b, 0x28,
	0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x82, 0x48, 0xe8, 0x15, 0x15, 0x24, 0x4b, 0x49, 0x42, 0x15, 0x81,
	0x65, 0x92, 0x4a, 0xd3, 0xf4, 0x13, 0xf3, 0x2a, 0x21, 0xca, 0x94, 0xd2, 0xb8, 0xd8, 0x82, 0xc1,
	0xda, 0x84, 0x84, 0xb8, 0x58, 0x92, 0xf3, 0x53, 0x52, 0x25, 0x18, 0x15, 0x18, 0x35, 0x58, 0x83,
	0xc0, 0x6c, 0x21, 0x09, 0x2e, 0xf6, 0xdc, 0xd4, 0xe2, 0xe2, 0xc4, 0xf4, 0x54, 0x09, 0x26, 0x05,
	0x46, 0x0d, 0xce, 0x20, 0x18, 0x57, 0x48, 0x8f, 0x8b, 0x3d, 0x25, 0xb5, 0x24, 0x31, 0x33, 0xa7,
	0x58, 0x82, 0x59, 0x81, 0x59, 0x83, 0xdb, 0x48, 0x44, 0x0f, 0x6a, 0x21, 0xcc, 0x12, 0x3d, 0xc7,
	0xbc, 0xca, 0x20, 0x98, 0x22, 0xa7, 0x38, 0x2e, 0xbe, 0xe4, 0xfc, 0x5c, 0x3d, 0x84, 0xa3, 0x9c,
	0xb8, 0x21, 0xf6, 0x06, 0x80, 0x94, 0x07, 0x30, 0x46, 0x99, 0x43, 0xa5, 0xd2, 0xf3, 0x73, 0x12,
	0xf3, 0xd2, 0xf5, 0xf2, 0x8b, 0xd2, 0xf5, 0xd3, 0x53, 0xf3, 0xc0, 0x86, 0xe9, 0x43, 0xa4, 0x12,
	0x0b, 0x32, 0x8b, 0x91, 0xfc, 0x69, 0x0d, 0xa1, 0x16, 0x31, 0x31, 0x07, 0x05, 0x38, 0x27, 0xb1,
	0x81, 0x55, 0x1a, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0xa4, 0x53, 0xf0, 0x7c, 0x10, 0x01, 0x00,
	0x00,
}
