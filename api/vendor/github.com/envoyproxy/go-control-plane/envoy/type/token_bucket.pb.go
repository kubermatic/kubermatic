// Code generated by protoc-gen-go. DO NOT EDIT.
// source: envoy/type/token_bucket.proto

package envoy_type

import (
	fmt "fmt"
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	proto "github.com/golang/protobuf/proto"
	duration "github.com/golang/protobuf/ptypes/duration"
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

type TokenBucket struct {
	MaxTokens            uint32                `protobuf:"varint,1,opt,name=max_tokens,json=maxTokens,proto3" json:"max_tokens,omitempty"`
	TokensPerFill        *wrappers.UInt32Value `protobuf:"bytes,2,opt,name=tokens_per_fill,json=tokensPerFill,proto3" json:"tokens_per_fill,omitempty"`
	FillInterval         *duration.Duration    `protobuf:"bytes,3,opt,name=fill_interval,json=fillInterval,proto3" json:"fill_interval,omitempty"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *TokenBucket) Reset()         { *m = TokenBucket{} }
func (m *TokenBucket) String() string { return proto.CompactTextString(m) }
func (*TokenBucket) ProtoMessage()    {}
func (*TokenBucket) Descriptor() ([]byte, []int) {
	return fileDescriptor_04e870b436501a80, []int{0}
}

func (m *TokenBucket) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TokenBucket.Unmarshal(m, b)
}
func (m *TokenBucket) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TokenBucket.Marshal(b, m, deterministic)
}
func (m *TokenBucket) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TokenBucket.Merge(m, src)
}
func (m *TokenBucket) XXX_Size() int {
	return xxx_messageInfo_TokenBucket.Size(m)
}
func (m *TokenBucket) XXX_DiscardUnknown() {
	xxx_messageInfo_TokenBucket.DiscardUnknown(m)
}

var xxx_messageInfo_TokenBucket proto.InternalMessageInfo

func (m *TokenBucket) GetMaxTokens() uint32 {
	if m != nil {
		return m.MaxTokens
	}
	return 0
}

func (m *TokenBucket) GetTokensPerFill() *wrappers.UInt32Value {
	if m != nil {
		return m.TokensPerFill
	}
	return nil
}

func (m *TokenBucket) GetFillInterval() *duration.Duration {
	if m != nil {
		return m.FillInterval
	}
	return nil
}

func init() {
	proto.RegisterType((*TokenBucket)(nil), "envoy.type.TokenBucket")
}

func init() { proto.RegisterFile("envoy/type/token_bucket.proto", fileDescriptor_04e870b436501a80) }

var fileDescriptor_04e870b436501a80 = []byte{
	// 282 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x90, 0xd1, 0x4a, 0xc3, 0x30,
	0x14, 0x86, 0x97, 0x39, 0x9c, 0x66, 0x16, 0xa5, 0x37, 0x56, 0x51, 0x29, 0x5e, 0xc8, 0xd8, 0x45,
	0x0a, 0xdb, 0x1b, 0x04, 0x11, 0x26, 0x08, 0x65, 0xa8, 0xb7, 0x25, 0x75, 0x67, 0x23, 0x2c, 0x6b,
	0x42, 0x9a, 0xd6, 0xf6, 0x95, 0x7c, 0x1a, 0x9f, 0x67, 0x57, 0x92, 0xa4, 0x63, 0xc2, 0xee, 0x02,
	0xdf, 0xf9, 0xbf, 0xfc, 0xe7, 0xe0, 0x7b, 0x28, 0x6a, 0xd9, 0x26, 0xa6, 0x55, 0x90, 0x18, 0xb9,
	0x81, 0x22, 0xcb, 0xab, 0xaf, 0x0d, 0x18, 0xa2, 0xb4, 0x34, 0x32, 0xc4, 0x0e, 0x13, 0x8b, 0x6f,
	0x1f, 0xd6, 0x52, 0xae, 0x05, 0x24, 0x8e, 0xe4, 0xd5, 0x2a, 0x59, 0x56, 0x9a, 0x19, 0x2e, 0x0b,
	0x3f, 0x7b, 0xcc, 0xbf, 0x35, 0x53, 0x0a, 0x74, 0xd9, 0xf1, 0xeb, 0x9a, 0x09, 0xbe, 0x64, 0x06,
	0x92, 0xfd, 0xc3, 0x83, 0xc7, 0x5f, 0x84, 0x47, 0xef, 0xf6, 0x6f, 0xea, 0xbe, 0x0e, 0x9f, 0x30,
	0xde, 0xb2, 0x26, 0x73, 0x75, 0xca, 0x08, 0xc5, 0x68, 0x1c, 0xd0, 0xe1, 0x8e, 0x0e, 0x26, 0xfd,
	0xb8, 0xb7, 0x38, 0xdf, 0xb2, 0xc6, 0x0d, 0x97, 0xe1, 0x1b, 0xbe, 0xf4, 0x33, 0x99, 0x02, 0x9d,
	0xad, 0xb8, 0x10, 0x51, 0x3f, 0x46, 0xe3, 0xd1, 0xf4, 0x8e, 0xf8, 0x2a, 0x64, 0x5f, 0x85, 0x7c,
	0xcc, 0x0b, 0x33, 0x9b, 0x7e, 0x32, 0x51, 0xc1, 0x41, 0x15, 0xf8, 0x74, 0x0a, 0xfa, 0x85, 0x0b,
	0x11, 0xbe, 0xe2, 0xc0, 0x3a, 0x32, 0x5e, 0x18, 0xd0, 0x35, 0x13, 0xd1, 0x89, 0x93, 0xdd, 0x1c,
	0xc9, 0x9e, 0xbb, 0xbd, 0x29, 0xde, 0xd1, 0xe1, 0x0f, 0x1a, 0x9c, 0xa1, 0x49, 0x6f, 0x71, 0x61,
	0xb3, 0xf3, 0x2e, 0x4a, 0x09, 0x8e, 0xb8, 0x24, 0xee, 0x78, 0x4a, 0xcb, 0xa6, 0x25, 0x87, 0x3b,
	0xd2, 0xab, 0x7f, 0xbb, 0xa6, 0xd6, 0x99, 0xa2, 0xfc, 0xd4, 0xc9, 0x67, 0x7f, 0x01, 0x00, 0x00,
	0xff, 0xff, 0x36, 0x93, 0x6c, 0x84, 0x8f, 0x01, 0x00, 0x00,
}
