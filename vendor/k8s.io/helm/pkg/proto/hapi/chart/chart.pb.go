// Code generated by protoc-gen-go.
// source: hapi/chart/chart.proto
// DO NOT EDIT!

/*
Package chart is a generated protocol buffer package.

It is generated from these files:
	hapi/chart/chart.proto
	hapi/chart/config.proto
	hapi/chart/metadata.proto
	hapi/chart/template.proto

It has these top-level messages:
	Chart
	Config
	Value
	Maintainer
	Metadata
	Template
*/
package chart

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/golang/protobuf/ptypes/any"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// 	Chart is a helm package that contains metadata, a default config, zero or more
// 	optionally parameterizable templates, and zero or more charts (dependencies).
type Chart struct {
	// Contents of the Chartfile.
	Metadata *Metadata `protobuf:"bytes,1,opt,name=metadata" json:"metadata,omitempty"`
	// Templates for this chart.
	Templates []*Template `protobuf:"bytes,2,rep,name=templates" json:"templates,omitempty"`
	// Charts that this chart depends on.
	Dependencies []*Chart `protobuf:"bytes,3,rep,name=dependencies" json:"dependencies,omitempty"`
	// Default config for this template.
	Values *Config `protobuf:"bytes,4,opt,name=values" json:"values,omitempty"`
	// Miscellaneous files in a chart archive,
	// e.g. README, LICENSE, etc.
	Files []*google_protobuf.Any `protobuf:"bytes,5,rep,name=files" json:"files,omitempty"`
}

func (m *Chart) Reset()                    { *m = Chart{} }
func (m *Chart) String() string            { return proto.CompactTextString(m) }
func (*Chart) ProtoMessage()               {}
func (*Chart) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Chart) GetMetadata() *Metadata {
	if m != nil {
		return m.Metadata
	}
	return nil
}

func (m *Chart) GetTemplates() []*Template {
	if m != nil {
		return m.Templates
	}
	return nil
}

func (m *Chart) GetDependencies() []*Chart {
	if m != nil {
		return m.Dependencies
	}
	return nil
}

func (m *Chart) GetValues() *Config {
	if m != nil {
		return m.Values
	}
	return nil
}

func (m *Chart) GetFiles() []*google_protobuf.Any {
	if m != nil {
		return m.Files
	}
	return nil
}

func init() {
	proto.RegisterType((*Chart)(nil), "hapi.chart.Chart")
}

func init() { proto.RegisterFile("hapi/chart/chart.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 242 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x6c, 0x90, 0xb1, 0x4e, 0xc3, 0x30,
	0x10, 0x86, 0x15, 0x4a, 0x0a, 0x1c, 0x2c, 0x58, 0x08, 0x4c, 0xa7, 0x8a, 0x09, 0x75, 0x70, 0x50,
	0x11, 0x0f, 0x00, 0xcc, 0x2c, 0x16, 0x13, 0xdb, 0xb5, 0xb9, 0xa4, 0x91, 0x52, 0x3b, 0xaa, 0x5d,
	0xa4, 0xbe, 0x3b, 0x03, 0xea, 0xd9, 0xa6, 0x09, 0xea, 0x12, 0x29, 0xf7, 0x7d, 0xff, 0xe5, 0xbf,
	0xc0, 0xed, 0x0a, 0xbb, 0xa6, 0x58, 0xae, 0x70, 0xe3, 0xc3, 0x53, 0x75, 0x1b, 0xeb, 0xad, 0x80,
	0xfd, 0x5c, 0xf1, 0x64, 0x72, 0xd7, 0x77, 0xac, 0xa9, 0x9a, 0x3a, 0x48, 0x93, 0xfb, 0x1e, 0x58,
	0x93, 0xc7, 0x12, 0x3d, 0x1e, 0x41, 0x9e, 0xd6, 0x5d, 0x8b, 0x9e, 0x12, 0xaa, 0xad, 0xad, 0x5b,
	0x2a, 0xf8, 0x6d, 0xb1, 0xad, 0x0a, 0x34, 0xbb, 0x80, 0x1e, 0x7e, 0x32, 0xc8, 0xdf, 0xf7, 0x19,
	0xf1, 0x04, 0xe7, 0x69, 0xa3, 0xcc, 0xa6, 0xd9, 0xe3, 0xe5, 0xfc, 0x46, 0x1d, 0x2a, 0xa9, 0x8f,
	0xc8, 0xf4, 0x9f, 0x25, 0xe6, 0x70, 0x91, 0x3e, 0xe4, 0xe4, 0xc9, 0x74, 0xf4, 0x3f, 0xf2, 0x19,
	0xa1, 0x3e, 0x68, 0xe2, 0x05, 0xae, 0x4a, 0xea, 0xc8, 0x94, 0x64, 0x96, 0x0d, 0x39, 0x39, 0xe2,
	0xd8, 0x75, 0x3f, 0xc6, 0x75, 0xf4, 0x40, 0x13, 0x33, 0x18, 0x7f, 0x63, 0xbb, 0x25, 0x27, 0x4f,
	0xb9, 0x9a, 0x18, 0x04, 0xf8, 0x0f, 0xe9, 0x68, 0x88, 0x19, 0xe4, 0x55, 0xd3, 0x92, 0x93, 0x79,
	0xac, 0x14, 0xae, 0x57, 0xe9, 0x7a, 0xf5, 0x6a, 0x76, 0x3a, 0x28, 0x6f, 0x67, 0x5f, 0x39, 0xef,
	0x58, 0x8c, 0x99, 0x3e, 0xff, 0x06, 0x00, 0x00, 0xff, 0xff, 0xe9, 0x70, 0x34, 0x75, 0x9e, 0x01,
	0x00, 0x00,
}
