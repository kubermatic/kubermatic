// Code generated by protoc-gen-go. DO NOT EDIT.
// source: envoy/api/v2/core/config_source.proto

package envoy_api_v2_core

import (
	fmt "fmt"
	_ "github.com/cncf/udpa/go/udpa/annotations"
	_ "github.com/envoyproxy/go-control-plane/envoy/annotations"
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

type ApiVersion int32

const (
	ApiVersion_AUTO ApiVersion = 0
	ApiVersion_V2   ApiVersion = 1
	ApiVersion_V3   ApiVersion = 2
)

var ApiVersion_name = map[int32]string{
	0: "AUTO",
	1: "V2",
	2: "V3",
}

var ApiVersion_value = map[string]int32{
	"AUTO": 0,
	"V2":   1,
	"V3":   2,
}

func (x ApiVersion) String() string {
	return proto.EnumName(ApiVersion_name, int32(x))
}

func (ApiVersion) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_1ffcc55cf4c30535, []int{0}
}

type ApiConfigSource_ApiType int32

const (
	ApiConfigSource_UNSUPPORTED_REST_LEGACY ApiConfigSource_ApiType = 0 // Deprecated: Do not use.
	ApiConfigSource_REST                    ApiConfigSource_ApiType = 1
	ApiConfigSource_GRPC                    ApiConfigSource_ApiType = 2
	ApiConfigSource_DELTA_GRPC              ApiConfigSource_ApiType = 3
)

var ApiConfigSource_ApiType_name = map[int32]string{
	0: "UNSUPPORTED_REST_LEGACY",
	1: "REST",
	2: "GRPC",
	3: "DELTA_GRPC",
}

var ApiConfigSource_ApiType_value = map[string]int32{
	"UNSUPPORTED_REST_LEGACY": 0,
	"REST":                    1,
	"GRPC":                    2,
	"DELTA_GRPC":              3,
}

func (x ApiConfigSource_ApiType) String() string {
	return proto.EnumName(ApiConfigSource_ApiType_name, int32(x))
}

func (ApiConfigSource_ApiType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_1ffcc55cf4c30535, []int{0, 0}
}

type ApiConfigSource struct {
	ApiType                   ApiConfigSource_ApiType `protobuf:"varint,1,opt,name=api_type,json=apiType,proto3,enum=envoy.api.v2.core.ApiConfigSource_ApiType" json:"api_type,omitempty"`
	TransportApiVersion       ApiVersion              `protobuf:"varint,8,opt,name=transport_api_version,json=transportApiVersion,proto3,enum=envoy.api.v2.core.ApiVersion" json:"transport_api_version,omitempty"`
	ClusterNames              []string                `protobuf:"bytes,2,rep,name=cluster_names,json=clusterNames,proto3" json:"cluster_names,omitempty"`
	GrpcServices              []*GrpcService          `protobuf:"bytes,4,rep,name=grpc_services,json=grpcServices,proto3" json:"grpc_services,omitempty"`
	RefreshDelay              *duration.Duration      `protobuf:"bytes,3,opt,name=refresh_delay,json=refreshDelay,proto3" json:"refresh_delay,omitempty"`
	RequestTimeout            *duration.Duration      `protobuf:"bytes,5,opt,name=request_timeout,json=requestTimeout,proto3" json:"request_timeout,omitempty"`
	RateLimitSettings         *RateLimitSettings      `protobuf:"bytes,6,opt,name=rate_limit_settings,json=rateLimitSettings,proto3" json:"rate_limit_settings,omitempty"`
	SetNodeOnFirstMessageOnly bool                    `protobuf:"varint,7,opt,name=set_node_on_first_message_only,json=setNodeOnFirstMessageOnly,proto3" json:"set_node_on_first_message_only,omitempty"`
	XXX_NoUnkeyedLiteral      struct{}                `json:"-"`
	XXX_unrecognized          []byte                  `json:"-"`
	XXX_sizecache             int32                   `json:"-"`
}

func (m *ApiConfigSource) Reset()         { *m = ApiConfigSource{} }
func (m *ApiConfigSource) String() string { return proto.CompactTextString(m) }
func (*ApiConfigSource) ProtoMessage()    {}
func (*ApiConfigSource) Descriptor() ([]byte, []int) {
	return fileDescriptor_1ffcc55cf4c30535, []int{0}
}

func (m *ApiConfigSource) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ApiConfigSource.Unmarshal(m, b)
}
func (m *ApiConfigSource) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ApiConfigSource.Marshal(b, m, deterministic)
}
func (m *ApiConfigSource) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ApiConfigSource.Merge(m, src)
}
func (m *ApiConfigSource) XXX_Size() int {
	return xxx_messageInfo_ApiConfigSource.Size(m)
}
func (m *ApiConfigSource) XXX_DiscardUnknown() {
	xxx_messageInfo_ApiConfigSource.DiscardUnknown(m)
}

var xxx_messageInfo_ApiConfigSource proto.InternalMessageInfo

func (m *ApiConfigSource) GetApiType() ApiConfigSource_ApiType {
	if m != nil {
		return m.ApiType
	}
	return ApiConfigSource_UNSUPPORTED_REST_LEGACY
}

func (m *ApiConfigSource) GetTransportApiVersion() ApiVersion {
	if m != nil {
		return m.TransportApiVersion
	}
	return ApiVersion_AUTO
}

func (m *ApiConfigSource) GetClusterNames() []string {
	if m != nil {
		return m.ClusterNames
	}
	return nil
}

func (m *ApiConfigSource) GetGrpcServices() []*GrpcService {
	if m != nil {
		return m.GrpcServices
	}
	return nil
}

func (m *ApiConfigSource) GetRefreshDelay() *duration.Duration {
	if m != nil {
		return m.RefreshDelay
	}
	return nil
}

func (m *ApiConfigSource) GetRequestTimeout() *duration.Duration {
	if m != nil {
		return m.RequestTimeout
	}
	return nil
}

func (m *ApiConfigSource) GetRateLimitSettings() *RateLimitSettings {
	if m != nil {
		return m.RateLimitSettings
	}
	return nil
}

func (m *ApiConfigSource) GetSetNodeOnFirstMessageOnly() bool {
	if m != nil {
		return m.SetNodeOnFirstMessageOnly
	}
	return false
}

type AggregatedConfigSource struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AggregatedConfigSource) Reset()         { *m = AggregatedConfigSource{} }
func (m *AggregatedConfigSource) String() string { return proto.CompactTextString(m) }
func (*AggregatedConfigSource) ProtoMessage()    {}
func (*AggregatedConfigSource) Descriptor() ([]byte, []int) {
	return fileDescriptor_1ffcc55cf4c30535, []int{1}
}

func (m *AggregatedConfigSource) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AggregatedConfigSource.Unmarshal(m, b)
}
func (m *AggregatedConfigSource) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AggregatedConfigSource.Marshal(b, m, deterministic)
}
func (m *AggregatedConfigSource) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AggregatedConfigSource.Merge(m, src)
}
func (m *AggregatedConfigSource) XXX_Size() int {
	return xxx_messageInfo_AggregatedConfigSource.Size(m)
}
func (m *AggregatedConfigSource) XXX_DiscardUnknown() {
	xxx_messageInfo_AggregatedConfigSource.DiscardUnknown(m)
}

var xxx_messageInfo_AggregatedConfigSource proto.InternalMessageInfo

type SelfConfigSource struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SelfConfigSource) Reset()         { *m = SelfConfigSource{} }
func (m *SelfConfigSource) String() string { return proto.CompactTextString(m) }
func (*SelfConfigSource) ProtoMessage()    {}
func (*SelfConfigSource) Descriptor() ([]byte, []int) {
	return fileDescriptor_1ffcc55cf4c30535, []int{2}
}

func (m *SelfConfigSource) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SelfConfigSource.Unmarshal(m, b)
}
func (m *SelfConfigSource) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SelfConfigSource.Marshal(b, m, deterministic)
}
func (m *SelfConfigSource) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SelfConfigSource.Merge(m, src)
}
func (m *SelfConfigSource) XXX_Size() int {
	return xxx_messageInfo_SelfConfigSource.Size(m)
}
func (m *SelfConfigSource) XXX_DiscardUnknown() {
	xxx_messageInfo_SelfConfigSource.DiscardUnknown(m)
}

var xxx_messageInfo_SelfConfigSource proto.InternalMessageInfo

type RateLimitSettings struct {
	MaxTokens            *wrappers.UInt32Value `protobuf:"bytes,1,opt,name=max_tokens,json=maxTokens,proto3" json:"max_tokens,omitempty"`
	FillRate             *wrappers.DoubleValue `protobuf:"bytes,2,opt,name=fill_rate,json=fillRate,proto3" json:"fill_rate,omitempty"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *RateLimitSettings) Reset()         { *m = RateLimitSettings{} }
func (m *RateLimitSettings) String() string { return proto.CompactTextString(m) }
func (*RateLimitSettings) ProtoMessage()    {}
func (*RateLimitSettings) Descriptor() ([]byte, []int) {
	return fileDescriptor_1ffcc55cf4c30535, []int{3}
}

func (m *RateLimitSettings) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RateLimitSettings.Unmarshal(m, b)
}
func (m *RateLimitSettings) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RateLimitSettings.Marshal(b, m, deterministic)
}
func (m *RateLimitSettings) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RateLimitSettings.Merge(m, src)
}
func (m *RateLimitSettings) XXX_Size() int {
	return xxx_messageInfo_RateLimitSettings.Size(m)
}
func (m *RateLimitSettings) XXX_DiscardUnknown() {
	xxx_messageInfo_RateLimitSettings.DiscardUnknown(m)
}

var xxx_messageInfo_RateLimitSettings proto.InternalMessageInfo

func (m *RateLimitSettings) GetMaxTokens() *wrappers.UInt32Value {
	if m != nil {
		return m.MaxTokens
	}
	return nil
}

func (m *RateLimitSettings) GetFillRate() *wrappers.DoubleValue {
	if m != nil {
		return m.FillRate
	}
	return nil
}

type ConfigSource struct {
	// Types that are valid to be assigned to ConfigSourceSpecifier:
	//	*ConfigSource_Path
	//	*ConfigSource_ApiConfigSource
	//	*ConfigSource_Ads
	//	*ConfigSource_Self
	ConfigSourceSpecifier isConfigSource_ConfigSourceSpecifier `protobuf_oneof:"config_source_specifier"`
	InitialFetchTimeout   *duration.Duration                   `protobuf:"bytes,4,opt,name=initial_fetch_timeout,json=initialFetchTimeout,proto3" json:"initial_fetch_timeout,omitempty"`
	ResourceApiVersion    ApiVersion                           `protobuf:"varint,6,opt,name=resource_api_version,json=resourceApiVersion,proto3,enum=envoy.api.v2.core.ApiVersion" json:"resource_api_version,omitempty"`
	XXX_NoUnkeyedLiteral  struct{}                             `json:"-"`
	XXX_unrecognized      []byte                               `json:"-"`
	XXX_sizecache         int32                                `json:"-"`
}

func (m *ConfigSource) Reset()         { *m = ConfigSource{} }
func (m *ConfigSource) String() string { return proto.CompactTextString(m) }
func (*ConfigSource) ProtoMessage()    {}
func (*ConfigSource) Descriptor() ([]byte, []int) {
	return fileDescriptor_1ffcc55cf4c30535, []int{4}
}

func (m *ConfigSource) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ConfigSource.Unmarshal(m, b)
}
func (m *ConfigSource) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ConfigSource.Marshal(b, m, deterministic)
}
func (m *ConfigSource) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ConfigSource.Merge(m, src)
}
func (m *ConfigSource) XXX_Size() int {
	return xxx_messageInfo_ConfigSource.Size(m)
}
func (m *ConfigSource) XXX_DiscardUnknown() {
	xxx_messageInfo_ConfigSource.DiscardUnknown(m)
}

var xxx_messageInfo_ConfigSource proto.InternalMessageInfo

type isConfigSource_ConfigSourceSpecifier interface {
	isConfigSource_ConfigSourceSpecifier()
}

type ConfigSource_Path struct {
	Path string `protobuf:"bytes,1,opt,name=path,proto3,oneof"`
}

type ConfigSource_ApiConfigSource struct {
	ApiConfigSource *ApiConfigSource `protobuf:"bytes,2,opt,name=api_config_source,json=apiConfigSource,proto3,oneof"`
}

type ConfigSource_Ads struct {
	Ads *AggregatedConfigSource `protobuf:"bytes,3,opt,name=ads,proto3,oneof"`
}

type ConfigSource_Self struct {
	Self *SelfConfigSource `protobuf:"bytes,5,opt,name=self,proto3,oneof"`
}

func (*ConfigSource_Path) isConfigSource_ConfigSourceSpecifier() {}

func (*ConfigSource_ApiConfigSource) isConfigSource_ConfigSourceSpecifier() {}

func (*ConfigSource_Ads) isConfigSource_ConfigSourceSpecifier() {}

func (*ConfigSource_Self) isConfigSource_ConfigSourceSpecifier() {}

func (m *ConfigSource) GetConfigSourceSpecifier() isConfigSource_ConfigSourceSpecifier {
	if m != nil {
		return m.ConfigSourceSpecifier
	}
	return nil
}

func (m *ConfigSource) GetPath() string {
	if x, ok := m.GetConfigSourceSpecifier().(*ConfigSource_Path); ok {
		return x.Path
	}
	return ""
}

func (m *ConfigSource) GetApiConfigSource() *ApiConfigSource {
	if x, ok := m.GetConfigSourceSpecifier().(*ConfigSource_ApiConfigSource); ok {
		return x.ApiConfigSource
	}
	return nil
}

func (m *ConfigSource) GetAds() *AggregatedConfigSource {
	if x, ok := m.GetConfigSourceSpecifier().(*ConfigSource_Ads); ok {
		return x.Ads
	}
	return nil
}

func (m *ConfigSource) GetSelf() *SelfConfigSource {
	if x, ok := m.GetConfigSourceSpecifier().(*ConfigSource_Self); ok {
		return x.Self
	}
	return nil
}

func (m *ConfigSource) GetInitialFetchTimeout() *duration.Duration {
	if m != nil {
		return m.InitialFetchTimeout
	}
	return nil
}

func (m *ConfigSource) GetResourceApiVersion() ApiVersion {
	if m != nil {
		return m.ResourceApiVersion
	}
	return ApiVersion_AUTO
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*ConfigSource) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*ConfigSource_Path)(nil),
		(*ConfigSource_ApiConfigSource)(nil),
		(*ConfigSource_Ads)(nil),
		(*ConfigSource_Self)(nil),
	}
}

func init() {
	proto.RegisterEnum("envoy.api.v2.core.ApiVersion", ApiVersion_name, ApiVersion_value)
	proto.RegisterEnum("envoy.api.v2.core.ApiConfigSource_ApiType", ApiConfigSource_ApiType_name, ApiConfigSource_ApiType_value)
	proto.RegisterType((*ApiConfigSource)(nil), "envoy.api.v2.core.ApiConfigSource")
	proto.RegisterType((*AggregatedConfigSource)(nil), "envoy.api.v2.core.AggregatedConfigSource")
	proto.RegisterType((*SelfConfigSource)(nil), "envoy.api.v2.core.SelfConfigSource")
	proto.RegisterType((*RateLimitSettings)(nil), "envoy.api.v2.core.RateLimitSettings")
	proto.RegisterType((*ConfigSource)(nil), "envoy.api.v2.core.ConfigSource")
}

func init() {
	proto.RegisterFile("envoy/api/v2/core/config_source.proto", fileDescriptor_1ffcc55cf4c30535)
}

var fileDescriptor_1ffcc55cf4c30535 = []byte{
	// 859 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x54, 0xcf, 0x6f, 0xe3, 0x44,
	0x14, 0x8e, 0x93, 0xb4, 0x4d, 0xa7, 0xbf, 0x92, 0x69, 0x77, 0xeb, 0xad, 0xa0, 0x84, 0x74, 0x17,
	0x85, 0x1e, 0x1c, 0x29, 0x3d, 0x21, 0x04, 0x52, 0xd2, 0x74, 0xdb, 0x95, 0xba, 0x4d, 0xe4, 0xb8,
	0x95, 0x56, 0x1c, 0x46, 0x53, 0xfb, 0xd9, 0x1d, 0xe1, 0x78, 0xcc, 0xcc, 0x38, 0x34, 0x57, 0xae,
	0x1c, 0xb8, 0x22, 0x71, 0x47, 0x88, 0x33, 0x27, 0xfe, 0x2c, 0x2e, 0xa0, 0x3d, 0x20, 0x34, 0xb6,
	0xb3, 0x9b, 0x34, 0x59, 0x55, 0xe4, 0x12, 0xcf, 0x7b, 0xdf, 0xf7, 0xcd, 0xfb, 0x39, 0xe8, 0x05,
	0x44, 0x63, 0x3e, 0x69, 0xd1, 0x98, 0xb5, 0xc6, 0xed, 0x96, 0xcb, 0x05, 0xb4, 0x5c, 0x1e, 0xf9,
	0x2c, 0x20, 0x92, 0x27, 0xc2, 0x05, 0x2b, 0x16, 0x5c, 0x71, 0x5c, 0x4b, 0x61, 0x16, 0x8d, 0x99,
	0x35, 0x6e, 0x5b, 0x1a, 0x76, 0xf0, 0x7c, 0x91, 0x19, 0x88, 0xd8, 0x25, 0x12, 0xc4, 0x98, 0x4d,
	0x89, 0x07, 0x87, 0x01, 0xe7, 0x41, 0x08, 0xad, 0xf4, 0x74, 0x9b, 0xf8, 0x2d, 0x2f, 0x11, 0x54,
	0x31, 0x1e, 0x7d, 0xc8, 0xff, 0xbd, 0xa0, 0x71, 0x0c, 0x42, 0xe6, 0xfe, 0xa3, 0xfc, 0x96, 0x28,
	0xe2, 0x2a, 0xe5, 0xc9, 0x96, 0x07, 0xb1, 0x00, 0x77, 0x4e, 0x24, 0xf1, 0x62, 0x3a, 0x87, 0x19,
	0xb1, 0x40, 0x50, 0x35, 0x0d, 0x62, 0x7f, 0x4c, 0x43, 0xe6, 0x51, 0x05, 0xad, 0xe9, 0x47, 0xe6,
	0x68, 0xfc, 0xba, 0x82, 0x76, 0x3a, 0x31, 0x3b, 0x4d, 0x33, 0x1e, 0xa6, 0x09, 0xe3, 0x3e, 0xaa,
	0xd0, 0x98, 0x11, 0x35, 0x89, 0xc1, 0x34, 0xea, 0x46, 0x73, 0xbb, 0x7d, 0x6c, 0x2d, 0x64, 0x6f,
	0x3d, 0x60, 0xe9, 0xb3, 0x33, 0x89, 0xa1, 0x5b, 0x79, 0xdb, 0x5d, 0xf9, 0xc1, 0x28, 0x56, 0x0d,
	0x7b, 0x8d, 0x66, 0x26, 0xfc, 0x0d, 0x7a, 0xa2, 0x04, 0x8d, 0x64, 0xcc, 0x85, 0x22, 0x5a, 0x7a,
	0x0c, 0x42, 0x32, 0x1e, 0x99, 0x95, 0x54, 0xfd, 0xe3, 0xe5, 0xea, 0x37, 0x19, 0x68, 0x46, 0x70,
	0xf7, 0x9d, 0xca, 0x7b, 0x37, 0x3e, 0x42, 0x5b, 0x6e, 0x98, 0x48, 0x05, 0x82, 0x44, 0x74, 0x04,
	0xd2, 0x2c, 0xd6, 0x4b, 0xcd, 0x75, 0x7b, 0x33, 0x37, 0x5e, 0x69, 0x1b, 0x3e, 0x45, 0x5b, 0xb3,
	0xad, 0x91, 0x66, 0xb9, 0x5e, 0x6a, 0x6e, 0xb4, 0x0f, 0x97, 0xdc, 0x7c, 0x2e, 0x62, 0x77, 0x98,
	0xc1, 0xec, 0xcd, 0xe0, 0xfd, 0x41, 0xe2, 0xaf, 0xd1, 0x96, 0x00, 0x5f, 0x80, 0xbc, 0x23, 0x1e,
	0x84, 0x74, 0x62, 0x96, 0xea, 0x46, 0x73, 0xa3, 0xfd, 0xcc, 0xca, 0x3a, 0x68, 0x4d, 0x3b, 0x68,
	0xf5, 0xf2, 0x0e, 0xdb, 0x9b, 0x39, 0xbe, 0xa7, 0xe1, 0xf8, 0x12, 0xed, 0x08, 0xf8, 0x2e, 0x01,
	0xa9, 0x88, 0x62, 0x23, 0xe0, 0x89, 0x32, 0x57, 0x1e, 0x51, 0x48, 0x93, 0xff, 0xdd, 0x28, 0x1e,
	0x17, 0xec, 0xed, 0x9c, 0xeb, 0x64, 0x54, 0xec, 0xa0, 0x5d, 0xdd, 0x60, 0x12, 0xb2, 0x11, 0x53,
	0x44, 0x82, 0x52, 0x2c, 0x0a, 0xa4, 0xb9, 0x9a, 0x2a, 0x3e, 0x5f, 0x92, 0x98, 0x4d, 0x15, 0x5c,
	0x6a, 0xf0, 0x30, 0xc7, 0xda, 0x35, 0xf1, 0xd0, 0x84, 0x3b, 0xe8, 0x50, 0x82, 0x22, 0x11, 0xf7,
	0x80, 0xf0, 0x88, 0xf8, 0x4c, 0x48, 0x45, 0x46, 0x20, 0x25, 0x0d, 0xb4, 0x21, 0x9c, 0x98, 0x6b,
	0x75, 0xa3, 0x59, 0xb1, 0x9f, 0x49, 0x50, 0x57, 0xdc, 0x83, 0x7e, 0xf4, 0x52, 0x43, 0x5e, 0x67,
	0x88, 0x7e, 0x14, 0x4e, 0x1a, 0x0e, 0x5a, 0xcb, 0x67, 0x01, 0xbf, 0x40, 0xfb, 0xd7, 0x57, 0xc3,
	0xeb, 0xc1, 0xa0, 0x6f, 0x3b, 0x67, 0x3d, 0x62, 0x9f, 0x0d, 0x1d, 0x72, 0x79, 0x76, 0xde, 0x39,
	0x7d, 0x53, 0x2d, 0x1c, 0x54, 0x7e, 0xfb, 0xfb, 0x8f, 0x1f, 0x8b, 0x46, 0xc5, 0xc0, 0x15, 0x54,
	0xd6, 0xae, 0x6a, 0xfa, 0x75, 0x6e, 0x0f, 0x4e, 0xab, 0x45, 0xbc, 0x8d, 0x50, 0xef, 0xec, 0xd2,
	0xe9, 0x90, 0xf4, 0x5c, 0x6a, 0x98, 0xe8, 0x69, 0x27, 0x08, 0x04, 0x04, 0x54, 0x81, 0x37, 0x3b,
	0x78, 0x0d, 0x8c, 0xaa, 0x43, 0x08, 0xfd, 0x39, 0xdb, 0x2f, 0x06, 0xaa, 0x2d, 0xe4, 0x8b, 0xbf,
	0x44, 0x68, 0x44, 0xef, 0x89, 0xe2, 0xdf, 0x42, 0x24, 0xd3, 0xd1, 0xde, 0x68, 0x7f, 0xb4, 0x50,
	0xfb, 0xeb, 0x57, 0x91, 0x3a, 0x69, 0xdf, 0xd0, 0x30, 0x01, 0x7b, 0x7d, 0x44, 0xef, 0x9d, 0x14,
	0x8e, 0x5f, 0xa1, 0x75, 0x9f, 0x85, 0x21, 0xd1, 0x35, 0x33, 0x8b, 0x1f, 0xe0, 0xf6, 0x78, 0x72,
	0x1b, 0x42, 0xca, 0xed, 0x6e, 0xbf, 0xed, 0x6e, 0xe0, 0xf5, 0x4f, 0x0b, 0xf9, 0xcf, 0xae, 0x68,
	0xba, 0x0e, 0xaa, 0xf1, 0x67, 0x09, 0x6d, 0xce, 0x6d, 0xdc, 0x1e, 0x2a, 0xc7, 0x54, 0xdd, 0xa5,
	0x21, 0xad, 0x5f, 0x14, 0xec, 0xf4, 0x84, 0x07, 0xa8, 0xa6, 0x97, 0x65, 0xee, 0x35, 0xca, 0x6f,
	0x6e, 0x3c, 0xbe, 0x90, 0x17, 0x05, 0x7b, 0x87, 0x3e, 0xd8, 0xec, 0xaf, 0x50, 0x89, 0x7a, 0x32,
	0x9f, 0xdb, 0xcf, 0x97, 0x69, 0x2c, 0x2d, 0xf1, 0x45, 0xc1, 0xd6, 0x3c, 0xfc, 0x05, 0x2a, 0x4b,
	0x08, 0xfd, 0x7c, 0x6a, 0x8f, 0x96, 0xf0, 0x1f, 0x36, 0x42, 0xe7, 0xa2, 0x29, 0xf8, 0x35, 0x7a,
	0xc2, 0x22, 0xa6, 0x18, 0x0d, 0x89, 0x0f, 0xca, 0xbd, 0x7b, 0xb7, 0x01, 0xe5, 0xc7, 0x76, 0x68,
	0x37, 0xe7, 0xbd, 0xd4, 0xb4, 0xe9, 0xf0, 0xbf, 0x41, 0x7b, 0x02, 0xb2, 0x8a, 0xcc, 0x3d, 0x28,
	0xab, 0xff, 0xef, 0x41, 0xc1, 0x53, 0x91, 0x19, 0xef, 0x21, 0xda, 0x9f, 0xab, 0x38, 0x91, 0x31,
	0xb8, 0xcc, 0x67, 0x20, 0x70, 0xe9, 0x9f, 0xae, 0x71, 0xfc, 0x19, 0x42, 0x33, 0xaf, 0x4f, 0x05,
	0x95, 0x3b, 0xd7, 0x4e, 0xbf, 0x5a, 0xc0, 0xab, 0xa8, 0x78, 0xd3, 0xae, 0x1a, 0xe9, 0xff, 0x49,
	0xb5, 0xd8, 0xb5, 0xff, 0xfa, 0xf9, 0xdf, 0x9f, 0x56, 0x9e, 0xe2, 0xbd, 0x2c, 0x96, 0x4c, 0x33,
	0x8b, 0x65, 0x7c, 0x82, 0x3e, 0x61, 0x3c, 0x0b, 0x32, 0x16, 0xfc, 0x7e, 0xb2, 0x18, 0x6f, 0xb7,
	0x36, 0x5b, 0xc6, 0x81, 0xae, 0xca, 0xc0, 0xb8, 0x5d, 0x4d, 0xcb, 0x73, 0xf2, 0x5f, 0x00, 0x00,
	0x00, 0xff, 0xff, 0x4e, 0x83, 0x38, 0x04, 0xb4, 0x06, 0x00, 0x00,
}
