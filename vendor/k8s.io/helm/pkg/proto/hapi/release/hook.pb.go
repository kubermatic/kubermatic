// Code generated by protoc-gen-go.
// source: hapi/release/hook.proto
// DO NOT EDIT!

/*
Package release is a generated protocol buffer package.

It is generated from these files:
	hapi/release/hook.proto
	hapi/release/info.proto
	hapi/release/release.proto
	hapi/release/status.proto

It has these top-level messages:
	Hook
	Info
	Release
	Status
*/
package release

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/golang/protobuf/ptypes/timestamp"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Hook_Event int32

const (
	Hook_UNKNOWN       Hook_Event = 0
	Hook_PRE_INSTALL   Hook_Event = 1
	Hook_POST_INSTALL  Hook_Event = 2
	Hook_PRE_DELETE    Hook_Event = 3
	Hook_POST_DELETE   Hook_Event = 4
	Hook_PRE_UPGRADE   Hook_Event = 5
	Hook_POST_UPGRADE  Hook_Event = 6
	Hook_PRE_ROLLBACK  Hook_Event = 7
	Hook_POST_ROLLBACK Hook_Event = 8
)

var Hook_Event_name = map[int32]string{
	0: "UNKNOWN",
	1: "PRE_INSTALL",
	2: "POST_INSTALL",
	3: "PRE_DELETE",
	4: "POST_DELETE",
	5: "PRE_UPGRADE",
	6: "POST_UPGRADE",
	7: "PRE_ROLLBACK",
	8: "POST_ROLLBACK",
}
var Hook_Event_value = map[string]int32{
	"UNKNOWN":       0,
	"PRE_INSTALL":   1,
	"POST_INSTALL":  2,
	"PRE_DELETE":    3,
	"POST_DELETE":   4,
	"PRE_UPGRADE":   5,
	"POST_UPGRADE":  6,
	"PRE_ROLLBACK":  7,
	"POST_ROLLBACK": 8,
}

func (x Hook_Event) String() string {
	return proto.EnumName(Hook_Event_name, int32(x))
}
func (Hook_Event) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

// Hook defines a hook object.
type Hook struct {
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// Kind is the Kubernetes kind.
	Kind string `protobuf:"bytes,2,opt,name=kind" json:"kind,omitempty"`
	// Path is the chart-relative path to the template.
	Path string `protobuf:"bytes,3,opt,name=path" json:"path,omitempty"`
	// Manifest is the manifest contents.
	Manifest string `protobuf:"bytes,4,opt,name=manifest" json:"manifest,omitempty"`
	// Events are the events that this hook fires on.
	Events []Hook_Event `protobuf:"varint,5,rep,packed,name=events,enum=hapi.release.Hook_Event" json:"events,omitempty"`
	// LastRun indicates the date/time this was last run.
	LastRun *google_protobuf.Timestamp `protobuf:"bytes,6,opt,name=last_run,json=lastRun" json:"last_run,omitempty"`
}

func (m *Hook) Reset()                    { *m = Hook{} }
func (m *Hook) String() string            { return proto.CompactTextString(m) }
func (*Hook) ProtoMessage()               {}
func (*Hook) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Hook) GetLastRun() *google_protobuf.Timestamp {
	if m != nil {
		return m.LastRun
	}
	return nil
}

func init() {
	proto.RegisterType((*Hook)(nil), "hapi.release.Hook")
	proto.RegisterEnum("hapi.release.Hook_Event", Hook_Event_name, Hook_Event_value)
}

func init() { proto.RegisterFile("hapi/release/hook.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 321 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x4c, 0x8f, 0x4d, 0x6e, 0xea, 0x30,
	0x14, 0x85, 0x5f, 0x20, 0x24, 0x70, 0xe1, 0xd1, 0xd4, 0x93, 0x5a, 0x4c, 0x8a, 0x18, 0x31, 0x72,
	0x2a, 0xaa, 0x2e, 0x00, 0x8a, 0xd5, 0x56, 0x44, 0x01, 0x99, 0xa0, 0x4a, 0x9d, 0x20, 0xa3, 0x1a,
	0x88, 0x20, 0x71, 0x44, 0x4c, 0xd7, 0xd3, 0xf5, 0x75, 0x15, 0x95, 0x9d, 0x1f, 0x75, 0x76, 0xfd,
	0xdd, 0xcf, 0xc7, 0x3e, 0x70, 0x77, 0xe4, 0x59, 0xec, 0x5f, 0xc4, 0x59, 0xf0, 0x5c, 0xf8, 0x47,
	0x29, 0x4f, 0x24, 0xbb, 0x48, 0x25, 0x51, 0x4f, 0x2f, 0x48, 0xb9, 0x18, 0xdc, 0x1f, 0xa4, 0x3c,
	0x9c, 0x85, 0x6f, 0x76, 0xbb, 0xeb, 0xde, 0x57, 0x71, 0x22, 0x72, 0xc5, 0x93, 0xac, 0xd0, 0x47,
	0x3f, 0x0d, 0xb0, 0x5f, 0xa5, 0x3c, 0x21, 0x04, 0x76, 0xca, 0x13, 0x81, 0xad, 0xa1, 0x35, 0xee,
	0x30, 0x33, 0x6b, 0x76, 0x8a, 0xd3, 0x4f, 0xdc, 0x28, 0x98, 0x9e, 0x35, 0xcb, 0xb8, 0x3a, 0xe2,
	0x66, 0xc1, 0xf4, 0x8c, 0x06, 0xd0, 0x4e, 0x78, 0x1a, 0xef, 0x45, 0xae, 0xb0, 0x6d, 0x78, 0x7d,
	0x46, 0x0f, 0xe0, 0x88, 0x2f, 0x91, 0xaa, 0x1c, 0xb7, 0x86, 0xcd, 0x71, 0x7f, 0x82, 0xc9, 0xdf,
	0x0f, 0x12, 0xfd, 0x36, 0xa1, 0x5a, 0x60, 0xa5, 0x87, 0x9e, 0xa0, 0x7d, 0xe6, 0xb9, 0xda, 0x5e,
	0xae, 0x29, 0x76, 0x86, 0xd6, 0xb8, 0x3b, 0x19, 0x90, 0xa2, 0x06, 0xa9, 0x6a, 0x90, 0xa8, 0xaa,
	0xc1, 0x5c, 0xed, 0xb2, 0x6b, 0x3a, 0xfa, 0xb6, 0xa0, 0x65, 0x82, 0x50, 0x17, 0xdc, 0x4d, 0xb8,
	0x08, 0x97, 0xef, 0xa1, 0xf7, 0x0f, 0xdd, 0x40, 0x77, 0xc5, 0xe8, 0xf6, 0x2d, 0x5c, 0x47, 0xd3,
	0x20, 0xf0, 0x2c, 0xe4, 0x41, 0x6f, 0xb5, 0x5c, 0x47, 0x35, 0x69, 0xa0, 0x3e, 0x80, 0x56, 0xe6,
	0x34, 0xa0, 0x11, 0xf5, 0x9a, 0xe6, 0x8a, 0x36, 0x4a, 0x60, 0x57, 0x19, 0x9b, 0xd5, 0x0b, 0x9b,
	0xce, 0xa9, 0xd7, 0xaa, 0x33, 0x2a, 0xe2, 0x18, 0xc2, 0xe8, 0x96, 0x2d, 0x83, 0x60, 0x36, 0x7d,
	0x5e, 0x78, 0x2e, 0xba, 0x85, 0xff, 0xc6, 0xa9, 0x51, 0x7b, 0xd6, 0xf9, 0x70, 0xcb, 0xde, 0x3b,
	0xc7, 0x54, 0x79, 0xfc, 0x0d, 0x00, 0x00, 0xff, 0xff, 0xa4, 0x2e, 0x6f, 0xbd, 0xc8, 0x01, 0x00,
	0x00,
}
