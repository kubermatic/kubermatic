/*
Copyright 2016 The Kubernetes Authors.

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

// Code generated by protoc-gen-gogo.
// source: k8s.io/kubernetes/pkg/runtime/generated.proto
// DO NOT EDIT!

/*
	Package runtime is a generated protocol buffer package.

	It is generated from these files:
		k8s.io/kubernetes/pkg/runtime/generated.proto

	It has these top-level messages:
		RawExtension
		TypeMeta
		Unknown
*/
package runtime

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

import strings "strings"
import reflect "reflect"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.GoGoProtoPackageIsVersion1

func (m *RawExtension) Reset()                    { *m = RawExtension{} }
func (*RawExtension) ProtoMessage()               {}
func (*RawExtension) Descriptor() ([]byte, []int) { return fileDescriptorGenerated, []int{0} }

func (m *TypeMeta) Reset()                    { *m = TypeMeta{} }
func (*TypeMeta) ProtoMessage()               {}
func (*TypeMeta) Descriptor() ([]byte, []int) { return fileDescriptorGenerated, []int{1} }

func (m *Unknown) Reset()                    { *m = Unknown{} }
func (*Unknown) ProtoMessage()               {}
func (*Unknown) Descriptor() ([]byte, []int) { return fileDescriptorGenerated, []int{2} }

func init() {
	proto.RegisterType((*RawExtension)(nil), "k8s.io.client-go.pkg.runtime.RawExtension")
	proto.RegisterType((*TypeMeta)(nil), "k8s.io.client-go.pkg.runtime.TypeMeta")
	proto.RegisterType((*Unknown)(nil), "k8s.io.client-go.pkg.runtime.Unknown")
}
func (m *RawExtension) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *RawExtension) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Raw != nil {
		data[i] = 0xa
		i++
		i = encodeVarintGenerated(data, i, uint64(len(m.Raw)))
		i += copy(data[i:], m.Raw)
	}
	return i, nil
}

func (m *TypeMeta) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *TypeMeta) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0xa
	i++
	i = encodeVarintGenerated(data, i, uint64(len(m.APIVersion)))
	i += copy(data[i:], m.APIVersion)
	data[i] = 0x12
	i++
	i = encodeVarintGenerated(data, i, uint64(len(m.Kind)))
	i += copy(data[i:], m.Kind)
	return i, nil
}

func (m *Unknown) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *Unknown) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0xa
	i++
	i = encodeVarintGenerated(data, i, uint64(m.TypeMeta.Size()))
	n1, err := m.TypeMeta.MarshalTo(data[i:])
	if err != nil {
		return 0, err
	}
	i += n1
	if m.Raw != nil {
		data[i] = 0x12
		i++
		i = encodeVarintGenerated(data, i, uint64(len(m.Raw)))
		i += copy(data[i:], m.Raw)
	}
	data[i] = 0x1a
	i++
	i = encodeVarintGenerated(data, i, uint64(len(m.ContentEncoding)))
	i += copy(data[i:], m.ContentEncoding)
	data[i] = 0x22
	i++
	i = encodeVarintGenerated(data, i, uint64(len(m.ContentType)))
	i += copy(data[i:], m.ContentType)
	return i, nil
}

func encodeFixed64Generated(data []byte, offset int, v uint64) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	data[offset+4] = uint8(v >> 32)
	data[offset+5] = uint8(v >> 40)
	data[offset+6] = uint8(v >> 48)
	data[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Generated(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintGenerated(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func (m *RawExtension) Size() (n int) {
	var l int
	_ = l
	if m.Raw != nil {
		l = len(m.Raw)
		n += 1 + l + sovGenerated(uint64(l))
	}
	return n
}

func (m *TypeMeta) Size() (n int) {
	var l int
	_ = l
	l = len(m.APIVersion)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Kind)
	n += 1 + l + sovGenerated(uint64(l))
	return n
}

func (m *Unknown) Size() (n int) {
	var l int
	_ = l
	l = m.TypeMeta.Size()
	n += 1 + l + sovGenerated(uint64(l))
	if m.Raw != nil {
		l = len(m.Raw)
		n += 1 + l + sovGenerated(uint64(l))
	}
	l = len(m.ContentEncoding)
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.ContentType)
	n += 1 + l + sovGenerated(uint64(l))
	return n
}

func sovGenerated(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozGenerated(x uint64) (n int) {
	return sovGenerated(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *RawExtension) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&RawExtension{`,
		`Raw:` + valueToStringGenerated(this.Raw) + `,`,
		`}`,
	}, "")
	return s
}
func (this *TypeMeta) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&TypeMeta{`,
		`APIVersion:` + fmt.Sprintf("%v", this.APIVersion) + `,`,
		`Kind:` + fmt.Sprintf("%v", this.Kind) + `,`,
		`}`,
	}, "")
	return s
}
func (this *Unknown) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&Unknown{`,
		`TypeMeta:` + strings.Replace(strings.Replace(this.TypeMeta.String(), "TypeMeta", "TypeMeta", 1), `&`, ``, 1) + `,`,
		`Raw:` + valueToStringGenerated(this.Raw) + `,`,
		`ContentEncoding:` + fmt.Sprintf("%v", this.ContentEncoding) + `,`,
		`ContentType:` + fmt.Sprintf("%v", this.ContentType) + `,`,
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
func (m *RawExtension) Unmarshal(data []byte) error {
	l := len(data)
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
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: RawExtension: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RawExtension: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Raw", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Raw = append(m.Raw[:0], data[iNdEx:postIndex]...)
			if m.Raw == nil {
				m.Raw = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenerated(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
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
func (m *TypeMeta) Unmarshal(data []byte) error {
	l := len(data)
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
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: TypeMeta: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: TypeMeta: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field APIVersion", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.APIVersion = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Kind", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Kind = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenerated(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
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
func (m *Unknown) Unmarshal(data []byte) error {
	l := len(data)
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
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Unknown: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Unknown: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TypeMeta", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.TypeMeta.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Raw", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Raw = append(m.Raw[:0], data[iNdEx:postIndex]...)
			if m.Raw == nil {
				m.Raw = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ContentEncoding", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ContentEncoding = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ContentType", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthGenerated
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ContentType = string(data[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenerated(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
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
func skipGenerated(data []byte) (n int, err error) {
	l := len(data)
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
			b := data[iNdEx]
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
				if data[iNdEx-1] < 0x80 {
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
				b := data[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
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
					b := data[iNdEx]
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
				next, err := skipGenerated(data[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
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

var fileDescriptorGenerated = []byte{
	// 388 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x7c, 0x90, 0x3f, 0x8f, 0xda, 0x30,
	0x18, 0xc6, 0x13, 0x40, 0x82, 0x1a, 0x24, 0x2a, 0x77, 0x68, 0x8a, 0x54, 0x83, 0x58, 0x5a, 0x06,
	0x6c, 0x15, 0xa9, 0x52, 0x57, 0x82, 0x18, 0xaa, 0xaa, 0x52, 0x65, 0x95, 0x0e, 0x9d, 0x1a, 0x12,
	0x37, 0xb5, 0x52, 0xec, 0xc8, 0x71, 0x94, 0x76, 0xeb, 0x47, 0xe8, 0xc7, 0x62, 0x64, 0xbc, 0x09,
	0x1d, 0xb9, 0xcf, 0x70, 0xfb, 0x09, 0x63, 0xfe, 0x1c, 0xa0, 0xdb, 0xe2, 0xf7, 0xf9, 0x3d, 0xcf,
	0xfb, 0xbc, 0x01, 0xc3, 0xe4, 0x43, 0x86, 0xb9, 0x24, 0x49, 0x3e, 0x67, 0x4a, 0x30, 0xcd, 0x32,
	0x92, 0x26, 0x31, 0x51, 0xb9, 0xd0, 0x7c, 0xc1, 0x48, 0xcc, 0x04, 0x53, 0x81, 0x66, 0x11, 0x4e,
	0x95, 0xd4, 0x12, 0xbe, 0xde, 0xe1, 0xf8, 0x88, 0xe3, 0x34, 0x89, 0xb1, 0xc5, 0x3b, 0xc3, 0x98,
	0xeb, 0x5f, 0xf9, 0x1c, 0x87, 0x72, 0x41, 0x62, 0x19, 0x4b, 0x62, 0x5c, 0xf3, 0xfc, 0xa7, 0x79,
	0x99, 0x87, 0xf9, 0xda, 0xa5, 0x75, 0x46, 0xd7, 0x97, 0x07, 0x29, 0x27, 0x8a, 0x65, 0x32, 0x57,
	0xe1, 0x45, 0x83, 0xce, 0xbb, 0xeb, 0x9e, 0x5c, 0xf3, 0xdf, 0x84, 0x0b, 0x9d, 0x69, 0x75, 0x6e,
	0xe9, 0x0f, 0x40, 0x8b, 0x06, 0xc5, 0xf4, 0x8f, 0x66, 0x22, 0xe3, 0x52, 0xc0, 0x57, 0xa0, 0xaa,
	0x82, 0xc2, 0x73, 0x7b, 0xee, 0xdb, 0x96, 0x5f, 0x2f, 0xd7, 0xdd, 0x2a, 0x0d, 0x0a, 0xba, 0x9d,
	0xf5, 0x7f, 0x80, 0xc6, 0xd7, 0xbf, 0x29, 0xfb, 0xcc, 0x74, 0x00, 0x47, 0x00, 0x04, 0x29, 0xff,
	0xc6, 0xd4, 0xd6, 0x64, 0xe8, 0x67, 0x3e, 0x5c, 0xae, 0xbb, 0x4e, 0xb9, 0xee, 0x82, 0xf1, 0x97,
	0x8f, 0x56, 0xa1, 0x27, 0x14, 0xec, 0x81, 0x5a, 0xc2, 0x45, 0xe4, 0x55, 0x0c, 0xdd, 0xb2, 0x74,
	0xed, 0x13, 0x17, 0x11, 0x35, 0x4a, 0xff, 0xde, 0x05, 0xf5, 0x99, 0x48, 0x84, 0x2c, 0x04, 0x9c,
	0x81, 0x86, 0xb6, 0xdb, 0x4c, 0x7e, 0x73, 0xf4, 0x06, 0x3f, 0xf9, 0x83, 0xf1, 0xbe, 0x9c, 0xff,
	0xdc, 0x46, 0x1f, 0xea, 0xd2, 0x43, 0xd4, 0xfe, 0xbe, 0xca, 0xe5, 0x7d, 0x70, 0x0c, 0xda, 0xa1,
	0x14, 0x9a, 0x09, 0x3d, 0x15, 0xa1, 0x8c, 0xb8, 0x88, 0xbd, 0xaa, 0xa9, 0xfa, 0xd2, 0xe6, 0xb5,
	0x27, 0x8f, 0x65, 0x7a, 0xce, 0xc3, 0xf7, 0xa0, 0x69, 0x47, 0xdb, 0xd5, 0x5e, 0xcd, 0xd8, 0x5f,
	0x58, 0x7b, 0x73, 0x72, 0x94, 0xe8, 0x29, 0xe7, 0x0f, 0x96, 0x1b, 0xe4, 0xac, 0x36, 0xc8, 0xb9,
	0xd9, 0x20, 0xe7, 0x5f, 0x89, 0xdc, 0x65, 0x89, 0xdc, 0x55, 0x89, 0xdc, 0xdb, 0x12, 0xb9, 0xff,
	0xef, 0x90, 0xf3, 0xbd, 0x6e, 0x8f, 0x7c, 0x08, 0x00, 0x00, 0xff, 0xff, 0x32, 0xc1, 0x73, 0x2d,
	0x94, 0x02, 0x00, 0x00,
}
