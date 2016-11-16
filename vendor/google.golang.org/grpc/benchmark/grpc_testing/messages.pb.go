// Code generated by protoc-gen-go.
// source: messages.proto
// DO NOT EDIT!

package grpc_testing

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// The type of payload that should be returned.
type PayloadType int32

const (
	// Compressable text format.
	PayloadType_COMPRESSABLE PayloadType = 0
	// Uncompressable binary format.
	PayloadType_UNCOMPRESSABLE PayloadType = 1
	// Randomly chosen from all other formats defined in this enum.
	PayloadType_RANDOM PayloadType = 2
)

var PayloadType_name = map[int32]string{
	0: "COMPRESSABLE",
	1: "UNCOMPRESSABLE",
	2: "RANDOM",
}
var PayloadType_value = map[string]int32{
	"COMPRESSABLE":   0,
	"UNCOMPRESSABLE": 1,
	"RANDOM":         2,
}

func (x PayloadType) String() string {
	return proto.EnumName(PayloadType_name, int32(x))
}
func (PayloadType) EnumDescriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

// Compression algorithms
type CompressionType int32

const (
	// No compression
	CompressionType_NONE    CompressionType = 0
	CompressionType_GZIP    CompressionType = 1
	CompressionType_DEFLATE CompressionType = 2
)

var CompressionType_name = map[int32]string{
	0: "NONE",
	1: "GZIP",
	2: "DEFLATE",
}
var CompressionType_value = map[string]int32{
	"NONE":    0,
	"GZIP":    1,
	"DEFLATE": 2,
}

func (x CompressionType) String() string {
	return proto.EnumName(CompressionType_name, int32(x))
}
func (CompressionType) EnumDescriptor() ([]byte, []int) { return fileDescriptor1, []int{1} }

// A block of data, to simply increase gRPC message size.
type Payload struct {
	// The type of data in body.
	Type PayloadType `protobuf:"varint,1,opt,name=type,enum=grpc.testing.PayloadType" json:"type,omitempty"`
	// Primary contents of payload.
	Body []byte `protobuf:"bytes,2,opt,name=body,proto3" json:"body,omitempty"`
}

func (m *Payload) Reset()                    { *m = Payload{} }
func (m *Payload) String() string            { return proto.CompactTextString(m) }
func (*Payload) ProtoMessage()               {}
func (*Payload) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

// A protobuf representation for grpc status. This is used by test
// clients to specify a status that the server should attempt to return.
type EchoStatus struct {
	Code    int32  `protobuf:"varint,1,opt,name=code" json:"code,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message" json:"message,omitempty"`
}

func (m *EchoStatus) Reset()                    { *m = EchoStatus{} }
func (m *EchoStatus) String() string            { return proto.CompactTextString(m) }
func (*EchoStatus) ProtoMessage()               {}
func (*EchoStatus) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{1} }

// Unary request.
type SimpleRequest struct {
	// Desired payload type in the response from the server.
	// If response_type is RANDOM, server randomly chooses one from other formats.
	ResponseType PayloadType `protobuf:"varint,1,opt,name=response_type,json=responseType,enum=grpc.testing.PayloadType" json:"response_type,omitempty"`
	// Desired payload size in the response from the server.
	// If response_type is COMPRESSABLE, this denotes the size before compression.
	ResponseSize int32 `protobuf:"varint,2,opt,name=response_size,json=responseSize" json:"response_size,omitempty"`
	// Optional input payload sent along with the request.
	Payload *Payload `protobuf:"bytes,3,opt,name=payload" json:"payload,omitempty"`
	// Whether SimpleResponse should include username.
	FillUsername bool `protobuf:"varint,4,opt,name=fill_username,json=fillUsername" json:"fill_username,omitempty"`
	// Whether SimpleResponse should include OAuth scope.
	FillOauthScope bool `protobuf:"varint,5,opt,name=fill_oauth_scope,json=fillOauthScope" json:"fill_oauth_scope,omitempty"`
	// Compression algorithm to be used by the server for the response (stream)
	ResponseCompression CompressionType `protobuf:"varint,6,opt,name=response_compression,json=responseCompression,enum=grpc.testing.CompressionType" json:"response_compression,omitempty"`
	// Whether server should return a given status
	ResponseStatus *EchoStatus `protobuf:"bytes,7,opt,name=response_status,json=responseStatus" json:"response_status,omitempty"`
}

func (m *SimpleRequest) Reset()                    { *m = SimpleRequest{} }
func (m *SimpleRequest) String() string            { return proto.CompactTextString(m) }
func (*SimpleRequest) ProtoMessage()               {}
func (*SimpleRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{2} }

func (m *SimpleRequest) GetPayload() *Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *SimpleRequest) GetResponseStatus() *EchoStatus {
	if m != nil {
		return m.ResponseStatus
	}
	return nil
}

// Unary response, as configured by the request.
type SimpleResponse struct {
	// Payload to increase message size.
	Payload *Payload `protobuf:"bytes,1,opt,name=payload" json:"payload,omitempty"`
	// The user the request came from, for verifying authentication was
	// successful when the client expected it.
	Username string `protobuf:"bytes,2,opt,name=username" json:"username,omitempty"`
	// OAuth scope.
	OauthScope string `protobuf:"bytes,3,opt,name=oauth_scope,json=oauthScope" json:"oauth_scope,omitempty"`
}

func (m *SimpleResponse) Reset()                    { *m = SimpleResponse{} }
func (m *SimpleResponse) String() string            { return proto.CompactTextString(m) }
func (*SimpleResponse) ProtoMessage()               {}
func (*SimpleResponse) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{3} }

func (m *SimpleResponse) GetPayload() *Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

// Client-streaming request.
type StreamingInputCallRequest struct {
	// Optional input payload sent along with the request.
	Payload *Payload `protobuf:"bytes,1,opt,name=payload" json:"payload,omitempty"`
}

func (m *StreamingInputCallRequest) Reset()                    { *m = StreamingInputCallRequest{} }
func (m *StreamingInputCallRequest) String() string            { return proto.CompactTextString(m) }
func (*StreamingInputCallRequest) ProtoMessage()               {}
func (*StreamingInputCallRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{4} }

func (m *StreamingInputCallRequest) GetPayload() *Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

// Client-streaming response.
type StreamingInputCallResponse struct {
	// Aggregated size of payloads received from the client.
	AggregatedPayloadSize int32 `protobuf:"varint,1,opt,name=aggregated_payload_size,json=aggregatedPayloadSize" json:"aggregated_payload_size,omitempty"`
}

func (m *StreamingInputCallResponse) Reset()                    { *m = StreamingInputCallResponse{} }
func (m *StreamingInputCallResponse) String() string            { return proto.CompactTextString(m) }
func (*StreamingInputCallResponse) ProtoMessage()               {}
func (*StreamingInputCallResponse) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{5} }

// Configuration for a particular response.
type ResponseParameters struct {
	// Desired payload sizes in responses from the server.
	// If response_type is COMPRESSABLE, this denotes the size before compression.
	Size int32 `protobuf:"varint,1,opt,name=size" json:"size,omitempty"`
	// Desired interval between consecutive responses in the response stream in
	// microseconds.
	IntervalUs int32 `protobuf:"varint,2,opt,name=interval_us,json=intervalUs" json:"interval_us,omitempty"`
}

func (m *ResponseParameters) Reset()                    { *m = ResponseParameters{} }
func (m *ResponseParameters) String() string            { return proto.CompactTextString(m) }
func (*ResponseParameters) ProtoMessage()               {}
func (*ResponseParameters) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{6} }

// Server-streaming request.
type StreamingOutputCallRequest struct {
	// Desired payload type in the response from the server.
	// If response_type is RANDOM, the payload from each response in the stream
	// might be of different types. This is to simulate a mixed type of payload
	// stream.
	ResponseType PayloadType `protobuf:"varint,1,opt,name=response_type,json=responseType,enum=grpc.testing.PayloadType" json:"response_type,omitempty"`
	// Configuration for each expected response message.
	ResponseParameters []*ResponseParameters `protobuf:"bytes,2,rep,name=response_parameters,json=responseParameters" json:"response_parameters,omitempty"`
	// Optional input payload sent along with the request.
	Payload *Payload `protobuf:"bytes,3,opt,name=payload" json:"payload,omitempty"`
	// Compression algorithm to be used by the server for the response (stream)
	ResponseCompression CompressionType `protobuf:"varint,6,opt,name=response_compression,json=responseCompression,enum=grpc.testing.CompressionType" json:"response_compression,omitempty"`
	// Whether server should return a given status
	ResponseStatus *EchoStatus `protobuf:"bytes,7,opt,name=response_status,json=responseStatus" json:"response_status,omitempty"`
}

func (m *StreamingOutputCallRequest) Reset()                    { *m = StreamingOutputCallRequest{} }
func (m *StreamingOutputCallRequest) String() string            { return proto.CompactTextString(m) }
func (*StreamingOutputCallRequest) ProtoMessage()               {}
func (*StreamingOutputCallRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{7} }

func (m *StreamingOutputCallRequest) GetResponseParameters() []*ResponseParameters {
	if m != nil {
		return m.ResponseParameters
	}
	return nil
}

func (m *StreamingOutputCallRequest) GetPayload() *Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

func (m *StreamingOutputCallRequest) GetResponseStatus() *EchoStatus {
	if m != nil {
		return m.ResponseStatus
	}
	return nil
}

// Server-streaming response, as configured by the request and parameters.
type StreamingOutputCallResponse struct {
	// Payload to increase response size.
	Payload *Payload `protobuf:"bytes,1,opt,name=payload" json:"payload,omitempty"`
}

func (m *StreamingOutputCallResponse) Reset()                    { *m = StreamingOutputCallResponse{} }
func (m *StreamingOutputCallResponse) String() string            { return proto.CompactTextString(m) }
func (*StreamingOutputCallResponse) ProtoMessage()               {}
func (*StreamingOutputCallResponse) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{8} }

func (m *StreamingOutputCallResponse) GetPayload() *Payload {
	if m != nil {
		return m.Payload
	}
	return nil
}

// For reconnect interop test only.
// Client tells server what reconnection parameters it used.
type ReconnectParams struct {
	MaxReconnectBackoffMs int32 `protobuf:"varint,1,opt,name=max_reconnect_backoff_ms,json=maxReconnectBackoffMs" json:"max_reconnect_backoff_ms,omitempty"`
}

func (m *ReconnectParams) Reset()                    { *m = ReconnectParams{} }
func (m *ReconnectParams) String() string            { return proto.CompactTextString(m) }
func (*ReconnectParams) ProtoMessage()               {}
func (*ReconnectParams) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{9} }

// For reconnect interop test only.
// Server tells client whether its reconnects are following the spec and the
// reconnect backoffs it saw.
type ReconnectInfo struct {
	Passed    bool    `protobuf:"varint,1,opt,name=passed" json:"passed,omitempty"`
	BackoffMs []int32 `protobuf:"varint,2,rep,name=backoff_ms,json=backoffMs" json:"backoff_ms,omitempty"`
}

func (m *ReconnectInfo) Reset()                    { *m = ReconnectInfo{} }
func (m *ReconnectInfo) String() string            { return proto.CompactTextString(m) }
func (*ReconnectInfo) ProtoMessage()               {}
func (*ReconnectInfo) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{10} }

func init() {
	proto.RegisterType((*Payload)(nil), "grpc.testing.Payload")
	proto.RegisterType((*EchoStatus)(nil), "grpc.testing.EchoStatus")
	proto.RegisterType((*SimpleRequest)(nil), "grpc.testing.SimpleRequest")
	proto.RegisterType((*SimpleResponse)(nil), "grpc.testing.SimpleResponse")
	proto.RegisterType((*StreamingInputCallRequest)(nil), "grpc.testing.StreamingInputCallRequest")
	proto.RegisterType((*StreamingInputCallResponse)(nil), "grpc.testing.StreamingInputCallResponse")
	proto.RegisterType((*ResponseParameters)(nil), "grpc.testing.ResponseParameters")
	proto.RegisterType((*StreamingOutputCallRequest)(nil), "grpc.testing.StreamingOutputCallRequest")
	proto.RegisterType((*StreamingOutputCallResponse)(nil), "grpc.testing.StreamingOutputCallResponse")
	proto.RegisterType((*ReconnectParams)(nil), "grpc.testing.ReconnectParams")
	proto.RegisterType((*ReconnectInfo)(nil), "grpc.testing.ReconnectInfo")
	proto.RegisterEnum("grpc.testing.PayloadType", PayloadType_name, PayloadType_value)
	proto.RegisterEnum("grpc.testing.CompressionType", CompressionType_name, CompressionType_value)
}

func init() { proto.RegisterFile("messages.proto", fileDescriptor1) }

var fileDescriptor1 = []byte{
	// 645 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xcc, 0x55, 0x4d, 0x6f, 0xd3, 0x40,
	0x10, 0x25, 0xdf, 0xe9, 0x24, 0x4d, 0xa3, 0x85, 0x82, 0x5b, 0x54, 0x51, 0x99, 0x4b, 0x55, 0x89,
	0x20, 0x15, 0x09, 0x24, 0x0e, 0xa0, 0xb4, 0x4d, 0x51, 0x50, 0x9b, 0x84, 0x75, 0x7b, 0xe1, 0x62,
	0x6d, 0x9c, 0x4d, 0x1a, 0x11, 0x7b, 0x8d, 0x77, 0x8d, 0x28, 0x07, 0xee, 0xfc, 0x60, 0xee, 0xec,
	0xae, 0xbd, 0x8e, 0xd3, 0xf6, 0xd0, 0xc2, 0x85, 0xdb, 0xce, 0xcc, 0x9b, 0x97, 0x79, 0x33, 0xcf,
	0x0a, 0xb4, 0x7c, 0xca, 0x39, 0x99, 0x51, 0xde, 0x09, 0x23, 0x26, 0x18, 0x6a, 0xce, 0xa2, 0xd0,
	0xeb, 0x08, 0xca, 0xc5, 0x3c, 0x98, 0xd9, 0xa7, 0x50, 0x1b, 0x91, 0xab, 0x05, 0x23, 0x13, 0xf4,
	0x02, 0xca, 0xe2, 0x2a, 0xa4, 0x56, 0x61, 0xb7, 0xb0, 0xd7, 0x3a, 0xd8, 0xea, 0xe4, 0x71, 0x9d,
	0x14, 0x74, 0x2e, 0x01, 0x58, 0xc3, 0x10, 0x82, 0xf2, 0x98, 0x4d, 0xae, 0xac, 0xa2, 0x84, 0x37,
	0xb1, 0x7e, 0xdb, 0x6f, 0x01, 0x7a, 0xde, 0x25, 0x73, 0x04, 0x11, 0x31, 0x57, 0x08, 0x8f, 0x4d,
	0x12, 0xc2, 0x0a, 0xd6, 0x6f, 0x64, 0x41, 0x2d, 0x9d, 0x47, 0x37, 0xae, 0x61, 0x13, 0xda, 0xbf,
	0x4a, 0xb0, 0xee, 0xcc, 0xfd, 0x70, 0x41, 0x31, 0xfd, 0x1a, 0xcb, 0x9f, 0x45, 0xef, 0x60, 0x3d,
	0xa2, 0x3c, 0x64, 0x01, 0xa7, 0xee, 0xdd, 0x26, 0x6b, 0x1a, 0xbc, 0x8a, 0xd0, 0xf3, 0x5c, 0x3f,
	0x9f, 0xff, 0x48, 0x7e, 0xb1, 0xb2, 0x04, 0x39, 0x32, 0x87, 0x5e, 0x42, 0x2d, 0x4c, 0x18, 0xac,
	0x92, 0x2c, 0x37, 0x0e, 0x36, 0x6f, 0xa5, 0xc7, 0x06, 0xa5, 0x58, 0xa7, 0xf3, 0xc5, 0xc2, 0x8d,
	0x39, 0x8d, 0x02, 0xe2, 0x53, 0xab, 0x2c, 0xdb, 0xea, 0xb8, 0xa9, 0x92, 0x17, 0x69, 0x0e, 0xed,
	0x41, 0x5b, 0x83, 0x18, 0x89, 0xc5, 0xa5, 0xcb, 0x3d, 0x26, 0xa7, 0xaf, 0x68, 0x5c, 0x4b, 0xe5,
	0x87, 0x2a, 0xed, 0xa8, 0x2c, 0x1a, 0xc1, 0xa3, 0x6c, 0x48, 0x8f, 0xf9, 0xa1, 0x0c, 0xf8, 0x9c,
	0x05, 0x56, 0x55, 0x6b, 0xdd, 0x59, 0x1d, 0xe6, 0x68, 0x09, 0xd0, 0x7a, 0x1f, 0x9a, 0xd6, 0x5c,
	0x01, 0x75, 0x61, 0x63, 0x29, 0x5b, 0x5f, 0xc2, 0xaa, 0x69, 0x65, 0xd6, 0x2a, 0xd9, 0xf2, 0x52,
	0xb8, 0x95, 0xad, 0x44, 0xc7, 0xf6, 0x4f, 0x68, 0x99, 0x53, 0x24, 0xf9, 0xfc, 0x9a, 0x0a, 0x77,
	0x5a, 0xd3, 0x36, 0xd4, 0xb3, 0x0d, 0x25, 0x97, 0xce, 0x62, 0xf4, 0x0c, 0x1a, 0xf9, 0xc5, 0x94,
	0x74, 0x19, 0x58, 0xb6, 0x14, 0xe9, 0xca, 0x2d, 0x47, 0x44, 0x94, 0xf8, 0x92, 0xba, 0x1f, 0x84,
	0xb1, 0x38, 0x22, 0x8b, 0x85, 0xb1, 0xc5, 0x7d, 0x47, 0xb1, 0xcf, 0x61, 0xfb, 0x36, 0xb6, 0x54,
	0xd9, 0x6b, 0x78, 0x42, 0x66, 0xb3, 0x88, 0xce, 0x88, 0xa0, 0x13, 0x37, 0xed, 0x49, 0xfc, 0x92,
	0x18, 0x77, 0x73, 0x59, 0x4e, 0xa9, 0x95, 0x71, 0xec, 0x3e, 0x20, 0xc3, 0x31, 0x22, 0x91, 0x94,
	0x25, 0x68, 0xa4, 0x3d, 0x9f, 0x6b, 0xd5, 0x6f, 0x25, 0x77, 0x1e, 0xc8, 0xea, 0x37, 0xa2, 0x5c,
	0x93, 0xba, 0x10, 0x4c, 0xea, 0x82, 0xdb, 0xbf, 0x8b, 0xb9, 0x09, 0x87, 0xb1, 0xb8, 0x26, 0xf8,
	0x5f, 0xbf, 0x83, 0x4f, 0x90, 0xf9, 0x44, 0xea, 0x33, 0xa3, 0xca, 0x39, 0x4a, 0x72, 0x79, 0xbb,
	0xab, 0x2c, 0x37, 0x25, 0x61, 0x14, 0xdd, 0x94, 0x79, 0xef, 0xaf, 0xe6, 0xbf, 0xb4, 0xf9, 0x00,
	0x9e, 0xde, 0xba, 0xf6, 0xbf, 0xf4, 0xbc, 0xfd, 0x11, 0x36, 0x30, 0xf5, 0x58, 0x10, 0x50, 0x4f,
	0xe8, 0x65, 0x71, 0xf4, 0x06, 0x2c, 0x9f, 0x7c, 0x77, 0x23, 0x93, 0x76, 0xc7, 0xc4, 0xfb, 0xc2,
	0xa6, 0x53, 0xd7, 0xe7, 0xc6, 0x5e, 0xb2, 0x9e, 0x75, 0x1d, 0x26, 0xd5, 0x33, 0x6e, 0x9f, 0xc0,
	0x7a, 0x96, 0xed, 0x07, 0x53, 0x86, 0x1e, 0x43, 0x35, 0x24, 0x9c, 0xd3, 0x64, 0x98, 0x3a, 0x4e,
	0x23, 0xb4, 0x03, 0x90, 0xe3, 0x54, 0x47, 0xad, 0xe0, 0xb5, 0xb1, 0xe1, 0xd9, 0x7f, 0x0f, 0x8d,
	0x9c, 0x33, 0x50, 0x1b, 0x9a, 0x47, 0xc3, 0xb3, 0x11, 0xee, 0x39, 0x4e, 0xf7, 0xf0, 0xb4, 0xd7,
	0x7e, 0x20, 0x1d, 0xdb, 0xba, 0x18, 0xac, 0xe4, 0x0a, 0x08, 0xa0, 0x8a, 0xbb, 0x83, 0xe3, 0xe1,
	0x59, 0xbb, 0xb8, 0x7f, 0x00, 0x1b, 0xd7, 0xee, 0x81, 0xea, 0x50, 0x1e, 0x0c, 0x07, 0xaa, 0x59,
	0xbe, 0x3e, 0x7c, 0xee, 0x8f, 0x64, 0x4b, 0x03, 0x6a, 0xc7, 0xbd, 0x93, 0xd3, 0xee, 0x79, 0xaf,
	0x5d, 0x1c, 0x57, 0xf5, 0x5f, 0xcd, 0xab, 0x3f, 0x01, 0x00, 0x00, 0xff, 0xff, 0xc2, 0x6a, 0xce,
	0x1e, 0x7c, 0x06, 0x00, 0x00,
}
