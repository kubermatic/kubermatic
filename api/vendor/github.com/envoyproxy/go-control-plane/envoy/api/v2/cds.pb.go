// Code generated by protoc-gen-go. DO NOT EDIT.
// source: envoy/api/v2/cds.proto

package envoy_api_v2

import (
	context "context"
	fmt "fmt"
	_ "github.com/cncf/udpa/go/udpa/annotations"
	_ "github.com/envoyproxy/go-control-plane/envoy/annotations"
	proto "github.com/golang/protobuf/proto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

type CdsDummy struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CdsDummy) Reset()         { *m = CdsDummy{} }
func (m *CdsDummy) String() string { return proto.CompactTextString(m) }
func (*CdsDummy) ProtoMessage()    {}
func (*CdsDummy) Descriptor() ([]byte, []int) {
	return fileDescriptor_e73f50fbb1daa302, []int{0}
}

func (m *CdsDummy) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CdsDummy.Unmarshal(m, b)
}
func (m *CdsDummy) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CdsDummy.Marshal(b, m, deterministic)
}
func (m *CdsDummy) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CdsDummy.Merge(m, src)
}
func (m *CdsDummy) XXX_Size() int {
	return xxx_messageInfo_CdsDummy.Size(m)
}
func (m *CdsDummy) XXX_DiscardUnknown() {
	xxx_messageInfo_CdsDummy.DiscardUnknown(m)
}

var xxx_messageInfo_CdsDummy proto.InternalMessageInfo

func init() {
	proto.RegisterType((*CdsDummy)(nil), "envoy.api.v2.CdsDummy")
}

func init() { proto.RegisterFile("envoy/api/v2/cds.proto", fileDescriptor_e73f50fbb1daa302) }

var fileDescriptor_e73f50fbb1daa302 = []byte{
	// 337 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x91, 0x3f, 0x4b, 0xc3, 0x40,
	0x18, 0xc6, 0x4d, 0xfd, 0xcb, 0x61, 0x1d, 0x82, 0xd4, 0x72, 0x94, 0x2a, 0x55, 0x50, 0x1c, 0x12,
	0x69, 0xb7, 0x8e, 0x6d, 0x71, 0x71, 0x29, 0x76, 0x70, 0xf5, 0x4c, 0x5e, 0xea, 0x41, 0x93, 0x37,
	0xde, 0x5d, 0x82, 0x59, 0x9d, 0xc4, 0xc5, 0x41, 0x10, 0x3f, 0x80, 0xdf, 0xc8, 0xaf, 0xe0, 0xea,
	0xa2, 0xbb, 0x48, 0x73, 0x97, 0xea, 0x29, 0x3a, 0xb9, 0xe6, 0xf7, 0x3c, 0x4f, 0x9e, 0x7b, 0x1f,
	0x52, 0x83, 0x38, 0xc3, 0xdc, 0x67, 0x09, 0xf7, 0xb3, 0xb6, 0x1f, 0x84, 0xd2, 0x4b, 0x04, 0x2a,
	0x74, 0x57, 0x8b, 0xef, 0x1e, 0x4b, 0xb8, 0x97, 0xb5, 0x69, 0xc3, 0x52, 0x85, 0x5c, 0x06, 0x98,
	0x81, 0xc8, 0xb5, 0x96, 0x36, 0xc6, 0x88, 0xe3, 0x09, 0x14, 0x98, 0xc5, 0x31, 0x2a, 0xa6, 0x38,
	0xc6, 0x26, 0x89, 0x6e, 0x19, 0xef, 0x27, 0xf0, 0x05, 0x48, 0x4c, 0x45, 0x00, 0x46, 0xd1, 0x4c,
	0xc3, 0x84, 0x59, 0x82, 0x88, 0x8f, 0x05, 0x53, 0x25, 0xa7, 0x76, 0xc7, 0x49, 0x2a, 0x15, 0x08,
	0xcd, 0x5a, 0x84, 0xac, 0xf4, 0x43, 0x39, 0x48, 0xa3, 0x28, 0x6f, 0xbf, 0x54, 0xc8, 0x46, 0x5f,
	0xd3, 0x41, 0x59, 0x71, 0x04, 0x22, 0xe3, 0x01, 0xb8, 0x27, 0x64, 0x6d, 0xa4, 0x04, 0xb0, 0xc8,
	0x08, 0xa4, 0xdb, 0xf4, 0xbe, 0x3e, 0xd1, 0x9b, 0x39, 0x8e, 0xe1, 0x22, 0x05, 0xa9, 0xe8, 0xe6,
	0xaf, 0x5c, 0x26, 0x18, 0x4b, 0x68, 0xcd, 0xed, 0x39, 0x07, 0x8e, 0x7b, 0x4a, 0xaa, 0x03, 0x98,
	0x28, 0x36, 0xcb, 0xdd, 0xfe, 0xe6, 0x9b, 0xc2, 0x1f, 0xe1, 0x3b, 0x7f, 0x8b, 0xac, 0x3f, 0xe4,
	0xa4, 0x7a, 0x08, 0x2a, 0x38, 0xff, 0xbf, 0xe6, 0xbb, 0x57, 0x4f, 0xcf, 0x77, 0x95, 0x7a, 0xab,
	0x66, 0xad, 0xd9, 0x35, 0x97, 0x95, 0x05, 0x9d, 0xef, 0x3a, 0xfb, 0xb4, 0x71, 0xf3, 0x78, 0xff,
	0xb6, 0x5c, 0x23, 0xeb, 0x56, 0xa0, 0x29, 0xd2, 0x3b, 0x7a, 0x7d, 0x78, 0xbf, 0x5d, 0xa4, 0x6e,
	0x5d, 0x53, 0xa9, 0x4f, 0xed, 0x95, 0x03, 0x65, 0x1d, 0x42, 0x39, 0xea, 0x2e, 0x89, 0xc0, 0xcb,
	0xdc, 0xaa, 0xd5, 0x9b, 0xee, 0x36, 0x9c, 0x6e, 0x38, 0x74, 0xae, 0x1d, 0x67, 0xb8, 0x70, 0xb6,
	0x54, 0x2c, 0xda, 0xf9, 0x08, 0x00, 0x00, 0xff, 0xff, 0x77, 0xbf, 0xdf, 0x11, 0x93, 0x02, 0x00,
	0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// ClusterDiscoveryServiceClient is the client API for ClusterDiscoveryService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ClusterDiscoveryServiceClient interface {
	StreamClusters(ctx context.Context, opts ...grpc.CallOption) (ClusterDiscoveryService_StreamClustersClient, error)
	DeltaClusters(ctx context.Context, opts ...grpc.CallOption) (ClusterDiscoveryService_DeltaClustersClient, error)
	FetchClusters(ctx context.Context, in *DiscoveryRequest, opts ...grpc.CallOption) (*DiscoveryResponse, error)
}

type clusterDiscoveryServiceClient struct {
	cc *grpc.ClientConn
}

func NewClusterDiscoveryServiceClient(cc *grpc.ClientConn) ClusterDiscoveryServiceClient {
	return &clusterDiscoveryServiceClient{cc}
}

func (c *clusterDiscoveryServiceClient) StreamClusters(ctx context.Context, opts ...grpc.CallOption) (ClusterDiscoveryService_StreamClustersClient, error) {
	stream, err := c.cc.NewStream(ctx, &_ClusterDiscoveryService_serviceDesc.Streams[0], "/envoy.api.v2.ClusterDiscoveryService/StreamClusters", opts...)
	if err != nil {
		return nil, err
	}
	x := &clusterDiscoveryServiceStreamClustersClient{stream}
	return x, nil
}

type ClusterDiscoveryService_StreamClustersClient interface {
	Send(*DiscoveryRequest) error
	Recv() (*DiscoveryResponse, error)
	grpc.ClientStream
}

type clusterDiscoveryServiceStreamClustersClient struct {
	grpc.ClientStream
}

func (x *clusterDiscoveryServiceStreamClustersClient) Send(m *DiscoveryRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *clusterDiscoveryServiceStreamClustersClient) Recv() (*DiscoveryResponse, error) {
	m := new(DiscoveryResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *clusterDiscoveryServiceClient) DeltaClusters(ctx context.Context, opts ...grpc.CallOption) (ClusterDiscoveryService_DeltaClustersClient, error) {
	stream, err := c.cc.NewStream(ctx, &_ClusterDiscoveryService_serviceDesc.Streams[1], "/envoy.api.v2.ClusterDiscoveryService/DeltaClusters", opts...)
	if err != nil {
		return nil, err
	}
	x := &clusterDiscoveryServiceDeltaClustersClient{stream}
	return x, nil
}

type ClusterDiscoveryService_DeltaClustersClient interface {
	Send(*DeltaDiscoveryRequest) error
	Recv() (*DeltaDiscoveryResponse, error)
	grpc.ClientStream
}

type clusterDiscoveryServiceDeltaClustersClient struct {
	grpc.ClientStream
}

func (x *clusterDiscoveryServiceDeltaClustersClient) Send(m *DeltaDiscoveryRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *clusterDiscoveryServiceDeltaClustersClient) Recv() (*DeltaDiscoveryResponse, error) {
	m := new(DeltaDiscoveryResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *clusterDiscoveryServiceClient) FetchClusters(ctx context.Context, in *DiscoveryRequest, opts ...grpc.CallOption) (*DiscoveryResponse, error) {
	out := new(DiscoveryResponse)
	err := c.cc.Invoke(ctx, "/envoy.api.v2.ClusterDiscoveryService/FetchClusters", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ClusterDiscoveryServiceServer is the server API for ClusterDiscoveryService service.
type ClusterDiscoveryServiceServer interface {
	StreamClusters(ClusterDiscoveryService_StreamClustersServer) error
	DeltaClusters(ClusterDiscoveryService_DeltaClustersServer) error
	FetchClusters(context.Context, *DiscoveryRequest) (*DiscoveryResponse, error)
}

// UnimplementedClusterDiscoveryServiceServer can be embedded to have forward compatible implementations.
type UnimplementedClusterDiscoveryServiceServer struct {
}

func (*UnimplementedClusterDiscoveryServiceServer) StreamClusters(srv ClusterDiscoveryService_StreamClustersServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamClusters not implemented")
}
func (*UnimplementedClusterDiscoveryServiceServer) DeltaClusters(srv ClusterDiscoveryService_DeltaClustersServer) error {
	return status.Errorf(codes.Unimplemented, "method DeltaClusters not implemented")
}
func (*UnimplementedClusterDiscoveryServiceServer) FetchClusters(ctx context.Context, req *DiscoveryRequest) (*DiscoveryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchClusters not implemented")
}

func RegisterClusterDiscoveryServiceServer(s *grpc.Server, srv ClusterDiscoveryServiceServer) {
	s.RegisterService(&_ClusterDiscoveryService_serviceDesc, srv)
}

func _ClusterDiscoveryService_StreamClusters_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ClusterDiscoveryServiceServer).StreamClusters(&clusterDiscoveryServiceStreamClustersServer{stream})
}

type ClusterDiscoveryService_StreamClustersServer interface {
	Send(*DiscoveryResponse) error
	Recv() (*DiscoveryRequest, error)
	grpc.ServerStream
}

type clusterDiscoveryServiceStreamClustersServer struct {
	grpc.ServerStream
}

func (x *clusterDiscoveryServiceStreamClustersServer) Send(m *DiscoveryResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *clusterDiscoveryServiceStreamClustersServer) Recv() (*DiscoveryRequest, error) {
	m := new(DiscoveryRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _ClusterDiscoveryService_DeltaClusters_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ClusterDiscoveryServiceServer).DeltaClusters(&clusterDiscoveryServiceDeltaClustersServer{stream})
}

type ClusterDiscoveryService_DeltaClustersServer interface {
	Send(*DeltaDiscoveryResponse) error
	Recv() (*DeltaDiscoveryRequest, error)
	grpc.ServerStream
}

type clusterDiscoveryServiceDeltaClustersServer struct {
	grpc.ServerStream
}

func (x *clusterDiscoveryServiceDeltaClustersServer) Send(m *DeltaDiscoveryResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *clusterDiscoveryServiceDeltaClustersServer) Recv() (*DeltaDiscoveryRequest, error) {
	m := new(DeltaDiscoveryRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _ClusterDiscoveryService_FetchClusters_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DiscoveryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterDiscoveryServiceServer).FetchClusters(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/envoy.api.v2.ClusterDiscoveryService/FetchClusters",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterDiscoveryServiceServer).FetchClusters(ctx, req.(*DiscoveryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _ClusterDiscoveryService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "envoy.api.v2.ClusterDiscoveryService",
	HandlerType: (*ClusterDiscoveryServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "FetchClusters",
			Handler:    _ClusterDiscoveryService_FetchClusters_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamClusters",
			Handler:       _ClusterDiscoveryService_StreamClusters_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "DeltaClusters",
			Handler:       _ClusterDiscoveryService_DeltaClusters_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "envoy/api/v2/cds.proto",
}
