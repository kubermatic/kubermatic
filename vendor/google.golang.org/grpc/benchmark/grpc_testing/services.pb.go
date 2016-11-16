// Code generated by protoc-gen-go.
// source: services.proto
// DO NOT EDIT!

package grpc_testing

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion3

// Client API for BenchmarkService service

type BenchmarkServiceClient interface {
	// One request followed by one response.
	// The server returns the client payload as-is.
	UnaryCall(ctx context.Context, in *SimpleRequest, opts ...grpc.CallOption) (*SimpleResponse, error)
	// One request followed by one response.
	// The server returns the client payload as-is.
	StreamingCall(ctx context.Context, opts ...grpc.CallOption) (BenchmarkService_StreamingCallClient, error)
}

type benchmarkServiceClient struct {
	cc *grpc.ClientConn
}

func NewBenchmarkServiceClient(cc *grpc.ClientConn) BenchmarkServiceClient {
	return &benchmarkServiceClient{cc}
}

func (c *benchmarkServiceClient) UnaryCall(ctx context.Context, in *SimpleRequest, opts ...grpc.CallOption) (*SimpleResponse, error) {
	out := new(SimpleResponse)
	err := grpc.Invoke(ctx, "/grpc.testing.BenchmarkService/UnaryCall", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *benchmarkServiceClient) StreamingCall(ctx context.Context, opts ...grpc.CallOption) (BenchmarkService_StreamingCallClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_BenchmarkService_serviceDesc.Streams[0], c.cc, "/grpc.testing.BenchmarkService/StreamingCall", opts...)
	if err != nil {
		return nil, err
	}
	x := &benchmarkServiceStreamingCallClient{stream}
	return x, nil
}

type BenchmarkService_StreamingCallClient interface {
	Send(*SimpleRequest) error
	Recv() (*SimpleResponse, error)
	grpc.ClientStream
}

type benchmarkServiceStreamingCallClient struct {
	grpc.ClientStream
}

func (x *benchmarkServiceStreamingCallClient) Send(m *SimpleRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *benchmarkServiceStreamingCallClient) Recv() (*SimpleResponse, error) {
	m := new(SimpleResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Server API for BenchmarkService service

type BenchmarkServiceServer interface {
	// One request followed by one response.
	// The server returns the client payload as-is.
	UnaryCall(context.Context, *SimpleRequest) (*SimpleResponse, error)
	// One request followed by one response.
	// The server returns the client payload as-is.
	StreamingCall(BenchmarkService_StreamingCallServer) error
}

func RegisterBenchmarkServiceServer(s *grpc.Server, srv BenchmarkServiceServer) {
	s.RegisterService(&_BenchmarkService_serviceDesc, srv)
}

func _BenchmarkService_UnaryCall_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SimpleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BenchmarkServiceServer).UnaryCall(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/grpc.testing.BenchmarkService/UnaryCall",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BenchmarkServiceServer).UnaryCall(ctx, req.(*SimpleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BenchmarkService_StreamingCall_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(BenchmarkServiceServer).StreamingCall(&benchmarkServiceStreamingCallServer{stream})
}

type BenchmarkService_StreamingCallServer interface {
	Send(*SimpleResponse) error
	Recv() (*SimpleRequest, error)
	grpc.ServerStream
}

type benchmarkServiceStreamingCallServer struct {
	grpc.ServerStream
}

func (x *benchmarkServiceStreamingCallServer) Send(m *SimpleResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *benchmarkServiceStreamingCallServer) Recv() (*SimpleRequest, error) {
	m := new(SimpleRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _BenchmarkService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "grpc.testing.BenchmarkService",
	HandlerType: (*BenchmarkServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UnaryCall",
			Handler:    _BenchmarkService_UnaryCall_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamingCall",
			Handler:       _BenchmarkService_StreamingCall_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: fileDescriptor3,
}

// Client API for WorkerService service

type WorkerServiceClient interface {
	// Start server with specified workload.
	// First request sent specifies the ServerConfig followed by ServerStatus
	// response. After that, a "Mark" can be sent anytime to request the latest
	// stats. Closing the stream will initiate shutdown of the test server
	// and once the shutdown has finished, the OK status is sent to terminate
	// this RPC.
	RunServer(ctx context.Context, opts ...grpc.CallOption) (WorkerService_RunServerClient, error)
	// Start client with specified workload.
	// First request sent specifies the ClientConfig followed by ClientStatus
	// response. After that, a "Mark" can be sent anytime to request the latest
	// stats. Closing the stream will initiate shutdown of the test client
	// and once the shutdown has finished, the OK status is sent to terminate
	// this RPC.
	RunClient(ctx context.Context, opts ...grpc.CallOption) (WorkerService_RunClientClient, error)
	// Just return the core count - unary call
	CoreCount(ctx context.Context, in *CoreRequest, opts ...grpc.CallOption) (*CoreResponse, error)
	// Quit this worker
	QuitWorker(ctx context.Context, in *Void, opts ...grpc.CallOption) (*Void, error)
}

type workerServiceClient struct {
	cc *grpc.ClientConn
}

func NewWorkerServiceClient(cc *grpc.ClientConn) WorkerServiceClient {
	return &workerServiceClient{cc}
}

func (c *workerServiceClient) RunServer(ctx context.Context, opts ...grpc.CallOption) (WorkerService_RunServerClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_WorkerService_serviceDesc.Streams[0], c.cc, "/grpc.testing.WorkerService/RunServer", opts...)
	if err != nil {
		return nil, err
	}
	x := &workerServiceRunServerClient{stream}
	return x, nil
}

type WorkerService_RunServerClient interface {
	Send(*ServerArgs) error
	Recv() (*ServerStatus, error)
	grpc.ClientStream
}

type workerServiceRunServerClient struct {
	grpc.ClientStream
}

func (x *workerServiceRunServerClient) Send(m *ServerArgs) error {
	return x.ClientStream.SendMsg(m)
}

func (x *workerServiceRunServerClient) Recv() (*ServerStatus, error) {
	m := new(ServerStatus)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *workerServiceClient) RunClient(ctx context.Context, opts ...grpc.CallOption) (WorkerService_RunClientClient, error) {
	stream, err := grpc.NewClientStream(ctx, &_WorkerService_serviceDesc.Streams[1], c.cc, "/grpc.testing.WorkerService/RunClient", opts...)
	if err != nil {
		return nil, err
	}
	x := &workerServiceRunClientClient{stream}
	return x, nil
}

type WorkerService_RunClientClient interface {
	Send(*ClientArgs) error
	Recv() (*ClientStatus, error)
	grpc.ClientStream
}

type workerServiceRunClientClient struct {
	grpc.ClientStream
}

func (x *workerServiceRunClientClient) Send(m *ClientArgs) error {
	return x.ClientStream.SendMsg(m)
}

func (x *workerServiceRunClientClient) Recv() (*ClientStatus, error) {
	m := new(ClientStatus)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *workerServiceClient) CoreCount(ctx context.Context, in *CoreRequest, opts ...grpc.CallOption) (*CoreResponse, error) {
	out := new(CoreResponse)
	err := grpc.Invoke(ctx, "/grpc.testing.WorkerService/CoreCount", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *workerServiceClient) QuitWorker(ctx context.Context, in *Void, opts ...grpc.CallOption) (*Void, error) {
	out := new(Void)
	err := grpc.Invoke(ctx, "/grpc.testing.WorkerService/QuitWorker", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for WorkerService service

type WorkerServiceServer interface {
	// Start server with specified workload.
	// First request sent specifies the ServerConfig followed by ServerStatus
	// response. After that, a "Mark" can be sent anytime to request the latest
	// stats. Closing the stream will initiate shutdown of the test server
	// and once the shutdown has finished, the OK status is sent to terminate
	// this RPC.
	RunServer(WorkerService_RunServerServer) error
	// Start client with specified workload.
	// First request sent specifies the ClientConfig followed by ClientStatus
	// response. After that, a "Mark" can be sent anytime to request the latest
	// stats. Closing the stream will initiate shutdown of the test client
	// and once the shutdown has finished, the OK status is sent to terminate
	// this RPC.
	RunClient(WorkerService_RunClientServer) error
	// Just return the core count - unary call
	CoreCount(context.Context, *CoreRequest) (*CoreResponse, error)
	// Quit this worker
	QuitWorker(context.Context, *Void) (*Void, error)
}

func RegisterWorkerServiceServer(s *grpc.Server, srv WorkerServiceServer) {
	s.RegisterService(&_WorkerService_serviceDesc, srv)
}

func _WorkerService_RunServer_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(WorkerServiceServer).RunServer(&workerServiceRunServerServer{stream})
}

type WorkerService_RunServerServer interface {
	Send(*ServerStatus) error
	Recv() (*ServerArgs, error)
	grpc.ServerStream
}

type workerServiceRunServerServer struct {
	grpc.ServerStream
}

func (x *workerServiceRunServerServer) Send(m *ServerStatus) error {
	return x.ServerStream.SendMsg(m)
}

func (x *workerServiceRunServerServer) Recv() (*ServerArgs, error) {
	m := new(ServerArgs)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _WorkerService_RunClient_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(WorkerServiceServer).RunClient(&workerServiceRunClientServer{stream})
}

type WorkerService_RunClientServer interface {
	Send(*ClientStatus) error
	Recv() (*ClientArgs, error)
	grpc.ServerStream
}

type workerServiceRunClientServer struct {
	grpc.ServerStream
}

func (x *workerServiceRunClientServer) Send(m *ClientStatus) error {
	return x.ServerStream.SendMsg(m)
}

func (x *workerServiceRunClientServer) Recv() (*ClientArgs, error) {
	m := new(ClientArgs)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _WorkerService_CoreCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CoreRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkerServiceServer).CoreCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/grpc.testing.WorkerService/CoreCount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkerServiceServer).CoreCount(ctx, req.(*CoreRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WorkerService_QuitWorker_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Void)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkerServiceServer).QuitWorker(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/grpc.testing.WorkerService/QuitWorker",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkerServiceServer).QuitWorker(ctx, req.(*Void))
	}
	return interceptor(ctx, in, info, handler)
}

var _WorkerService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "grpc.testing.WorkerService",
	HandlerType: (*WorkerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CoreCount",
			Handler:    _WorkerService_CoreCount_Handler,
		},
		{
			MethodName: "QuitWorker",
			Handler:    _WorkerService_QuitWorker_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "RunServer",
			Handler:       _WorkerService_RunServer_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "RunClient",
			Handler:       _WorkerService_RunClient_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: fileDescriptor3,
}

func init() { proto.RegisterFile("services.proto", fileDescriptor3) }

var fileDescriptor3 = []byte{
	// 254 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xa4, 0x91, 0xc1, 0x4a, 0xc4, 0x30,
	0x10, 0x86, 0xa9, 0x07, 0xa1, 0xc1, 0x2e, 0x92, 0x93, 0x46, 0x1f, 0xc0, 0x53, 0x91, 0xd5, 0x17,
	0x70, 0x8b, 0x1e, 0x05, 0xb7, 0xa8, 0xe7, 0x58, 0x87, 0x1a, 0x36, 0x4d, 0xea, 0xcc, 0x44, 0xf0,
	0x49, 0x7c, 0x07, 0x9f, 0xd2, 0xee, 0x66, 0x0b, 0xb5, 0xe4, 0xb6, 0xc7, 0xf9, 0xbf, 0xe1, 0x23,
	0x7f, 0x46, 0x2c, 0x08, 0xf0, 0xcb, 0x34, 0x40, 0x65, 0x8f, 0x9e, 0xbd, 0x3c, 0x69, 0xb1, 0x6f,
	0x4a, 0x06, 0x62, 0xe3, 0x5a, 0xb5, 0xe8, 0x80, 0x48, 0xb7, 0x23, 0x55, 0x45, 0xe3, 0x1d, 0xa3,
	0xb7, 0x71, 0x5c, 0xfe, 0x66, 0xe2, 0x74, 0x05, 0xae, 0xf9, 0xe8, 0x34, 0x6e, 0xea, 0x28, 0x92,
	0x0f, 0x22, 0x7f, 0x76, 0x1a, 0xbf, 0x2b, 0x6d, 0xad, 0xbc, 0x28, 0xa7, 0xbe, 0xb2, 0x36, 0x5d,
	0x6f, 0x61, 0x0d, 0x9f, 0x61, 0x08, 0xd4, 0x65, 0x1a, 0x52, 0xef, 0x1d, 0x81, 0x7c, 0x14, 0x45,
	0xcd, 0x08, 0xba, 0x1b, 0xd8, 0x81, 0xae, 0xab, 0xec, 0x3a, 0x5b, 0xfe, 0x1c, 0x89, 0xe2, 0xd5,
	0xe3, 0x06, 0x70, 0x7c, 0xe9, 0xbd, 0xc8, 0xd7, 0xc1, 0x6d, 0x27, 0x40, 0x79, 0x36, 0x13, 0xec,
	0xd2, 0x3b, 0x6c, 0x49, 0xa9, 0x14, 0xa9, 0x59, 0x73, 0xa0, 0xad, 0x78, 0xaf, 0xa9, 0xac, 0x01,
	0xc7, 0x73, 0x4d, 0x4c, 0x53, 0x9a, 0x48, 0x26, 0x9a, 0x95, 0xc8, 0x2b, 0x8f, 0x50, 0xf9, 0x30,
	0x68, 0xce, 0x67, 0xcb, 0x03, 0x18, 0x9b, 0xaa, 0x14, 0xda, 0xff, 0xd9, 0xad, 0x10, 0x4f, 0xc1,
	0x70, 0xac, 0x29, 0xe5, 0xff, 0xcd, 0x17, 0x6f, 0xde, 0x55, 0x22, 0x7b, 0x3b, 0xde, 0x5d, 0xf3,
	0xe6, 0x2f, 0x00, 0x00, 0xff, 0xff, 0x3b, 0x84, 0x02, 0xe3, 0x0c, 0x02, 0x00, 0x00,
}
