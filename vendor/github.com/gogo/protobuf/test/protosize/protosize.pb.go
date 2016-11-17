// Code generated by protoc-gen-gogo.
// source: protosize.proto
// DO NOT EDIT!

/*
	Package protosize is a generated protocol buffer package.

	It is generated from these files:
		protosize.proto

	It has these top-level messages:
		SizeMessage
*/
package protosize

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/gogo/protobuf/gogoproto"

import bytes "bytes"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.GoGoProtoPackageIsVersion1

type SizeMessage struct {
	Size             *int64  `protobuf:"varint,1,opt,name=size" json:"size,omitempty"`
	ProtoSize_       *int64  `protobuf:"varint,2,opt,name=proto_size,json=protoSize" json:"proto_size,omitempty"`
	Equal_           *bool   `protobuf:"varint,3,opt,name=Equal,json=equal" json:"Equal,omitempty"`
	String_          *string `protobuf:"bytes,4,opt,name=String,json=string" json:"String,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *SizeMessage) Reset()                    { *m = SizeMessage{} }
func (m *SizeMessage) String() string            { return proto.CompactTextString(m) }
func (*SizeMessage) ProtoMessage()               {}
func (*SizeMessage) Descriptor() ([]byte, []int) { return fileDescriptorProtosize, []int{0} }

func (m *SizeMessage) GetSize() int64 {
	if m != nil && m.Size != nil {
		return *m.Size
	}
	return 0
}

func (m *SizeMessage) GetProtoSize_() int64 {
	if m != nil && m.ProtoSize_ != nil {
		return *m.ProtoSize_
	}
	return 0
}

func (m *SizeMessage) GetEqual_() bool {
	if m != nil && m.Equal_ != nil {
		return *m.Equal_
	}
	return false
}

func (m *SizeMessage) GetString_() string {
	if m != nil && m.String_ != nil {
		return *m.String_
	}
	return ""
}

func init() {
	proto.RegisterType((*SizeMessage)(nil), "protosize.SizeMessage")
}
func (this *SizeMessage) Equal(that interface{}) bool {
	if that == nil {
		if this == nil {
			return true
		}
		return false
	}

	that1, ok := that.(*SizeMessage)
	if !ok {
		that2, ok := that.(SizeMessage)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		if this == nil {
			return true
		}
		return false
	} else if this == nil {
		return false
	}
	if this.Size != nil && that1.Size != nil {
		if *this.Size != *that1.Size {
			return false
		}
	} else if this.Size != nil {
		return false
	} else if that1.Size != nil {
		return false
	}
	if this.ProtoSize_ != nil && that1.ProtoSize_ != nil {
		if *this.ProtoSize_ != *that1.ProtoSize_ {
			return false
		}
	} else if this.ProtoSize_ != nil {
		return false
	} else if that1.ProtoSize_ != nil {
		return false
	}
	if this.Equal_ != nil && that1.Equal_ != nil {
		if *this.Equal_ != *that1.Equal_ {
			return false
		}
	} else if this.Equal_ != nil {
		return false
	} else if that1.Equal_ != nil {
		return false
	}
	if this.String_ != nil && that1.String_ != nil {
		if *this.String_ != *that1.String_ {
			return false
		}
	} else if this.String_ != nil {
		return false
	} else if that1.String_ != nil {
		return false
	}
	if !bytes.Equal(this.XXX_unrecognized, that1.XXX_unrecognized) {
		return false
	}
	return true
}
func (m *SizeMessage) Marshal() (data []byte, err error) {
	size := m.ProtoSize()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *SizeMessage) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Size != nil {
		data[i] = 0x8
		i++
		i = encodeVarintProtosize(data, i, uint64(*m.Size))
	}
	if m.ProtoSize_ != nil {
		data[i] = 0x10
		i++
		i = encodeVarintProtosize(data, i, uint64(*m.ProtoSize_))
	}
	if m.Equal_ != nil {
		data[i] = 0x18
		i++
		if *m.Equal_ {
			data[i] = 1
		} else {
			data[i] = 0
		}
		i++
	}
	if m.String_ != nil {
		data[i] = 0x22
		i++
		i = encodeVarintProtosize(data, i, uint64(len(*m.String_)))
		i += copy(data[i:], *m.String_)
	}
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}

func encodeFixed64Protosize(data []byte, offset int, v uint64) int {
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
func encodeFixed32Protosize(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintProtosize(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func NewPopulatedSizeMessage(r randyProtosize, easy bool) *SizeMessage {
	this := &SizeMessage{}
	if r.Intn(10) != 0 {
		v1 := int64(r.Int63())
		if r.Intn(2) == 0 {
			v1 *= -1
		}
		this.Size = &v1
	}
	if r.Intn(10) != 0 {
		v2 := int64(r.Int63())
		if r.Intn(2) == 0 {
			v2 *= -1
		}
		this.ProtoSize_ = &v2
	}
	if r.Intn(10) != 0 {
		v3 := bool(bool(r.Intn(2) == 0))
		this.Equal_ = &v3
	}
	if r.Intn(10) != 0 {
		v4 := randStringProtosize(r)
		this.String_ = &v4
	}
	if !easy && r.Intn(10) != 0 {
		this.XXX_unrecognized = randUnrecognizedProtosize(r, 5)
	}
	return this
}

type randyProtosize interface {
	Float32() float32
	Float64() float64
	Int63() int64
	Int31() int32
	Uint32() uint32
	Intn(n int) int
}

func randUTF8RuneProtosize(r randyProtosize) rune {
	ru := r.Intn(62)
	if ru < 10 {
		return rune(ru + 48)
	} else if ru < 36 {
		return rune(ru + 55)
	}
	return rune(ru + 61)
}
func randStringProtosize(r randyProtosize) string {
	v5 := r.Intn(100)
	tmps := make([]rune, v5)
	for i := 0; i < v5; i++ {
		tmps[i] = randUTF8RuneProtosize(r)
	}
	return string(tmps)
}
func randUnrecognizedProtosize(r randyProtosize, maxFieldNumber int) (data []byte) {
	l := r.Intn(5)
	for i := 0; i < l; i++ {
		wire := r.Intn(4)
		if wire == 3 {
			wire = 5
		}
		fieldNumber := maxFieldNumber + r.Intn(100)
		data = randFieldProtosize(data, r, fieldNumber, wire)
	}
	return data
}
func randFieldProtosize(data []byte, r randyProtosize, fieldNumber int, wire int) []byte {
	key := uint32(fieldNumber)<<3 | uint32(wire)
	switch wire {
	case 0:
		data = encodeVarintPopulateProtosize(data, uint64(key))
		v6 := r.Int63()
		if r.Intn(2) == 0 {
			v6 *= -1
		}
		data = encodeVarintPopulateProtosize(data, uint64(v6))
	case 1:
		data = encodeVarintPopulateProtosize(data, uint64(key))
		data = append(data, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	case 2:
		data = encodeVarintPopulateProtosize(data, uint64(key))
		ll := r.Intn(100)
		data = encodeVarintPopulateProtosize(data, uint64(ll))
		for j := 0; j < ll; j++ {
			data = append(data, byte(r.Intn(256)))
		}
	default:
		data = encodeVarintPopulateProtosize(data, uint64(key))
		data = append(data, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	}
	return data
}
func encodeVarintPopulateProtosize(data []byte, v uint64) []byte {
	for v >= 1<<7 {
		data = append(data, uint8(uint64(v)&0x7f|0x80))
		v >>= 7
	}
	data = append(data, uint8(v))
	return data
}
func (m *SizeMessage) ProtoSize() (n int) {
	var l int
	_ = l
	if m.Size != nil {
		n += 1 + sovProtosize(uint64(*m.Size))
	}
	if m.ProtoSize_ != nil {
		n += 1 + sovProtosize(uint64(*m.ProtoSize_))
	}
	if m.Equal_ != nil {
		n += 2
	}
	if m.String_ != nil {
		l = len(*m.String_)
		n += 1 + l + sovProtosize(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovProtosize(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozProtosize(x uint64) (n int) {
	return sovProtosize(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *SizeMessage) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowProtosize
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
			return fmt.Errorf("proto: SizeMessage: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: SizeMessage: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Size", wireType)
			}
			var v int64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtosize
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Size = &v
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProtoSize_", wireType)
			}
			var v int64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtosize
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.ProtoSize_ = &v
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Equal_", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtosize
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			b := bool(v != 0)
			m.Equal_ = &b
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field String_", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowProtosize
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
				return ErrInvalidLengthProtosize
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			s := string(data[iNdEx:postIndex])
			m.String_ = &s
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipProtosize(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthProtosize
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipProtosize(data []byte) (n int, err error) {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowProtosize
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
					return 0, ErrIntOverflowProtosize
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
					return 0, ErrIntOverflowProtosize
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
				return 0, ErrInvalidLengthProtosize
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowProtosize
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
				next, err := skipProtosize(data[start:])
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
	ErrInvalidLengthProtosize = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowProtosize   = fmt.Errorf("proto: integer overflow")
)

var fileDescriptorProtosize = []byte{
	// 177 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xe2, 0x2f, 0x28, 0xca, 0x2f,
	0xc9, 0x2f, 0xce, 0xac, 0x4a, 0xd5, 0x03, 0xb3, 0x84, 0x38, 0xe1, 0x02, 0x52, 0xba, 0xe9, 0x99,
	0x25, 0x19, 0xa5, 0x49, 0x7a, 0xc9, 0xf9, 0xb9, 0xfa, 0xe9, 0xf9, 0xe9, 0xf9, 0xfa, 0x60, 0xa9,
	0xa4, 0xd2, 0x34, 0x30, 0x0f, 0xcc, 0x01, 0xb3, 0x20, 0x3a, 0x95, 0xf2, 0xb8, 0xb8, 0x83, 0x81,
	0xda, 0x7c, 0x53, 0x8b, 0x8b, 0x13, 0xd3, 0x53, 0x85, 0x84, 0xb8, 0x58, 0x40, 0xa6, 0x48, 0x30,
	0x2a, 0x30, 0x6a, 0x30, 0x07, 0x81, 0xd9, 0x42, 0xb2, 0x5c, 0x5c, 0x60, 0xb5, 0xf1, 0x60, 0x19,
	0x26, 0xb0, 0x0c, 0xc4, 0x42, 0x90, 0x4e, 0x21, 0x11, 0x2e, 0x56, 0xd7, 0xc2, 0xd2, 0xc4, 0x1c,
	0x09, 0x66, 0xa0, 0x0c, 0x47, 0x10, 0x6b, 0x2a, 0x88, 0x23, 0x24, 0xc6, 0xc5, 0x16, 0x5c, 0x52,
	0x94, 0x99, 0x97, 0x2e, 0xc1, 0x02, 0x14, 0xe6, 0x0c, 0x62, 0x2b, 0x06, 0xf3, 0x9c, 0x24, 0x7e,
	0x3c, 0x94, 0x63, 0x5c, 0xf1, 0x48, 0x8e, 0x71, 0x07, 0x10, 0x9f, 0x00, 0xe2, 0x0b, 0x40, 0xbc,
	0xe0, 0xb1, 0x1c, 0x23, 0x20, 0x00, 0x00, 0xff, 0xff, 0xdf, 0x5d, 0x65, 0x12, 0xd5, 0x00, 0x00,
	0x00,
}
