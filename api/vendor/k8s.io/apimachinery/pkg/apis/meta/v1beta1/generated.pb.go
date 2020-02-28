/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: k8s.io/kubernetes/vendor/k8s.io/apimachinery/pkg/apis/meta/v1beta1/generated.proto

package v1beta1

import (
	fmt "fmt"

	io "io"

	proto "github.com/gogo/protobuf/proto"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	math "math"
	math_bits "math/bits"
	reflect "reflect"
	strings "strings"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

func (m *PartialObjectMetadataList) Reset()      { *m = PartialObjectMetadataList{} }
func (*PartialObjectMetadataList) ProtoMessage() {}
func (*PartialObjectMetadataList) Descriptor() ([]byte, []int) {
	return fileDescriptor_90ec10f86b91f9a8, []int{0}
}
func (m *PartialObjectMetadataList) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *PartialObjectMetadataList) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *PartialObjectMetadataList) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PartialObjectMetadataList.Merge(m, src)
}
func (m *PartialObjectMetadataList) XXX_Size() int {
	return m.Size()
}
func (m *PartialObjectMetadataList) XXX_DiscardUnknown() {
	xxx_messageInfo_PartialObjectMetadataList.DiscardUnknown(m)
}

var xxx_messageInfo_PartialObjectMetadataList proto.InternalMessageInfo

func init() {
	proto.RegisterType((*PartialObjectMetadataList)(nil), "k8s.io.apimachinery.pkg.apis.meta.v1beta1.PartialObjectMetadataList")
}

func init() {
	proto.RegisterFile("k8s.io/kubernetes/vendor/k8s.io/apimachinery/pkg/apis/meta/v1beta1/generated.proto", fileDescriptor_90ec10f86b91f9a8)
}

var fileDescriptor_90ec10f86b91f9a8 = []byte{
	// 321 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x91, 0x41, 0x4b, 0xf3, 0x30,
	0x18, 0xc7, 0x9b, 0xf7, 0x65, 0x38, 0x3a, 0x04, 0xd9, 0x69, 0xee, 0x90, 0x0d, 0x4f, 0xf3, 0xb0,
	0x84, 0x0d, 0x11, 0xc1, 0xdb, 0x6e, 0x82, 0xa2, 0xec, 0x28, 0x1e, 0x4c, 0xbb, 0xc7, 0x2e, 0xd6,
	0x34, 0x25, 0x79, 0x3a, 0xf0, 0xe6, 0x47, 0xf0, 0x63, 0xed, 0xb8, 0xe3, 0x40, 0x18, 0xae, 0x7e,
	0x11, 0x49, 0x57, 0x45, 0xa6, 0x62, 0x6f, 0x79, 0xfe, 0xe1, 0xf7, 0xcb, 0x3f, 0x89, 0x3f, 0x8e,
	0x4f, 0x2c, 0x93, 0x9a, 0xc7, 0x59, 0x00, 0x26, 0x01, 0x04, 0xcb, 0x67, 0x90, 0x4c, 0xb4, 0xe1,
	0xe5, 0x86, 0x48, 0xa5, 0x12, 0xe1, 0x54, 0x26, 0x60, 0x1e, 0x79, 0x1a, 0x47, 0x2e, 0xb0, 0x5c,
	0x01, 0x0a, 0x3e, 0x1b, 0x04, 0x80, 0x62, 0xc0, 0x23, 0x48, 0xc0, 0x08, 0x84, 0x09, 0x4b, 0x8d,
	0x46, 0xdd, 0x3c, 0xdc, 0xa0, 0xec, 0x2b, 0xca, 0xd2, 0x38, 0x72, 0x81, 0x65, 0x0e, 0x65, 0x25,
	0xda, 0xee, 0x47, 0x12, 0xa7, 0x59, 0xc0, 0x42, 0xad, 0x78, 0xa4, 0x23, 0xcd, 0x0b, 0x43, 0x90,
	0xdd, 0x15, 0x53, 0x31, 0x14, 0xab, 0x8d, 0xb9, 0x7d, 0x54, 0xa5, 0xd4, 0x76, 0x9f, 0xf6, 0xaf,
	0x57, 0x31, 0x59, 0x82, 0x52, 0xc1, 0x37, 0xe0, 0xf8, 0x2f, 0xc0, 0x86, 0x53, 0x50, 0x62, 0x9b,
	0x3b, 0x78, 0x21, 0xfe, 0xfe, 0x95, 0x30, 0x28, 0xc5, 0xc3, 0x65, 0x70, 0x0f, 0x21, 0x5e, 0x00,
	0x8a, 0x89, 0x40, 0x71, 0x2e, 0x2d, 0x36, 0x6f, 0xfc, 0xba, 0x2a, 0xe7, 0xd6, 0xbf, 0x2e, 0xe9,
	0x35, 0x86, 0x8c, 0x55, 0x79, 0x29, 0xe6, 0x68, 0x67, 0x1a, 0xed, 0xcd, 0x57, 0x1d, 0x2f, 0x5f,
	0x75, 0xea, 0x1f, 0xc9, 0xf8, 0xd3, 0xd8, 0xbc, 0xf5, 0x6b, 0x12, 0x41, 0xd9, 0x16, 0xe9, 0xfe,
	0xef, 0x35, 0x86, 0xa7, 0xd5, 0xd4, 0x3f, 0xb6, 0x1d, 0xed, 0x96, 0xe7, 0xd4, 0xce, 0x9c, 0x71,
	0xbc, 0x11, 0x8f, 0xfa, 0xf3, 0x35, 0xf5, 0x16, 0x6b, 0xea, 0x2d, 0xd7, 0xd4, 0x7b, 0xca, 0x29,
	0x99, 0xe7, 0x94, 0x2c, 0x72, 0x4a, 0x96, 0x39, 0x25, 0xaf, 0x39, 0x25, 0xcf, 0x6f, 0xd4, 0xbb,
	0xde, 0x29, 0xbf, 0xf6, 0x3d, 0x00, 0x00, 0xff, 0xff, 0xc6, 0x7e, 0x00, 0x08, 0x5a, 0x02, 0x00,
	0x00,
}

func (m *PartialObjectMetadataList) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *PartialObjectMetadataList) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *PartialObjectMetadataList) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.ListMeta.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintGenerated(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.Items) > 0 {
		for iNdEx := len(m.Items) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Items[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenerated(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintGenerated(dAtA []byte, offset int, v uint64) int {
	offset -= sovGenerated(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *PartialObjectMetadataList) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Items) > 0 {
		for _, e := range m.Items {
			l = e.Size()
			n += 1 + l + sovGenerated(uint64(l))
		}
	}
	l = m.ListMeta.Size()
	n += 1 + l + sovGenerated(uint64(l))
	return n
}

func sovGenerated(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozGenerated(x uint64) (n int) {
	return sovGenerated(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *PartialObjectMetadataList) String() string {
	if this == nil {
		return "nil"
	}
	repeatedStringForItems := "[]PartialObjectMetadata{"
	for _, f := range this.Items {
		repeatedStringForItems += fmt.Sprintf("%v", f) + ","
	}
	repeatedStringForItems += "}"
	s := strings.Join([]string{`&PartialObjectMetadataList{`,
		`Items:` + repeatedStringForItems + `,`,
		`ListMeta:` + strings.Replace(strings.Replace(fmt.Sprintf("%v", this.ListMeta), "ListMeta", "v1.ListMeta", 1), `&`, ``, 1) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringGenerated(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *PartialObjectMetadataList) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: PartialObjectMetadataList: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PartialObjectMetadataList: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Items", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Items = append(m.Items, v1.PartialObjectMetadata{})
			if err := m.Items[len(m.Items)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ListMeta", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenerated
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ListMeta.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenerated(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthGenerated
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthGenerated
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipGenerated(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGenerated
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthGenerated
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthGenerated
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowGenerated
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipGenerated(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthGenerated
				}
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthGenerated = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGenerated   = fmt.Errorf("proto: integer overflow")
)
