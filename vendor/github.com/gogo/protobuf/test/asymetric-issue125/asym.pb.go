// Code generated by protoc-gen-gogo.
// source: asym.proto
// DO NOT EDIT!

/*
Package asym is a generated protocol buffer package.

It is generated from these files:
	asym.proto

It has these top-level messages:
	M
	MyType
*/
package asym

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

type M struct {
	Arr              []MyType `protobuf:"bytes,1,rep,name=arr,customtype=MyType" json:"arr"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *M) Reset()                    { *m = M{} }
func (m *M) String() string            { return proto.CompactTextString(m) }
func (*M) ProtoMessage()               {}
func (*M) Descriptor() ([]byte, []int) { return fileDescriptorAsym, []int{0} }

type MyType struct {
	XXX_unrecognized []byte `json:"-"`
}

func (m *MyType) Reset()                    { *m = MyType{} }
func (m *MyType) String() string            { return proto.CompactTextString(m) }
func (*MyType) ProtoMessage()               {}
func (*MyType) Descriptor() ([]byte, []int) { return fileDescriptorAsym, []int{1} }

func init() {
	proto.RegisterType((*M)(nil), "asym.M")
	proto.RegisterType((*MyType)(nil), "asym.MyType")
}
func (this *M) VerboseEqual(that interface{}) error {
	if that == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that == nil && this != nil")
	}

	that1, ok := that.(*M)
	if !ok {
		that2, ok := that.(M)
		if ok {
			that1 = &that2
		} else {
			return fmt.Errorf("that is not of type *M")
		}
	}
	if that1 == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that is type *M but is nil && this != nil")
	} else if this == nil {
		return fmt.Errorf("that is type *M but is not nil && this == nil")
	}
	if len(this.Arr) != len(that1.Arr) {
		return fmt.Errorf("Arr this(%v) Not Equal that(%v)", len(this.Arr), len(that1.Arr))
	}
	for i := range this.Arr {
		if !this.Arr[i].Equal(that1.Arr[i]) {
			return fmt.Errorf("Arr this[%v](%v) Not Equal that[%v](%v)", i, this.Arr[i], i, that1.Arr[i])
		}
	}
	if !bytes.Equal(this.XXX_unrecognized, that1.XXX_unrecognized) {
		return fmt.Errorf("XXX_unrecognized this(%v) Not Equal that(%v)", this.XXX_unrecognized, that1.XXX_unrecognized)
	}
	return nil
}
func (this *M) Equal(that interface{}) bool {
	if that == nil {
		if this == nil {
			return true
		}
		return false
	}

	that1, ok := that.(*M)
	if !ok {
		that2, ok := that.(M)
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
	if len(this.Arr) != len(that1.Arr) {
		return false
	}
	for i := range this.Arr {
		if !this.Arr[i].Equal(that1.Arr[i]) {
			return false
		}
	}
	if !bytes.Equal(this.XXX_unrecognized, that1.XXX_unrecognized) {
		return false
	}
	return true
}
func (this *MyType) VerboseEqual(that interface{}) error {
	if that == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that == nil && this != nil")
	}

	that1, ok := that.(*MyType)
	if !ok {
		that2, ok := that.(MyType)
		if ok {
			that1 = &that2
		} else {
			return fmt.Errorf("that is not of type *MyType")
		}
	}
	if that1 == nil {
		if this == nil {
			return nil
		}
		return fmt.Errorf("that is type *MyType but is nil && this != nil")
	} else if this == nil {
		return fmt.Errorf("that is type *MyType but is not nil && this == nil")
	}
	if !bytes.Equal(this.XXX_unrecognized, that1.XXX_unrecognized) {
		return fmt.Errorf("XXX_unrecognized this(%v) Not Equal that(%v)", this.XXX_unrecognized, that1.XXX_unrecognized)
	}
	return nil
}
func (this *MyType) Equal(that interface{}) bool {
	if that == nil {
		if this == nil {
			return true
		}
		return false
	}

	that1, ok := that.(*MyType)
	if !ok {
		that2, ok := that.(MyType)
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
	if !bytes.Equal(this.XXX_unrecognized, that1.XXX_unrecognized) {
		return false
	}
	return true
}
func (m *M) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *M) MarshalTo(data []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Arr) > 0 {
		for _, msg := range m.Arr {
			data[i] = 0xa
			i++
			i = encodeVarintAsym(data, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(data[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}

func encodeFixed64Asym(data []byte, offset int, v uint64) int {
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
func encodeFixed32Asym(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintAsym(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
func NewPopulatedM(r randyAsym, easy bool) *M {
	this := &M{}
	if r.Intn(10) != 0 {
		v1 := r.Intn(10)
		this.Arr = make([]MyType, v1)
		for i := 0; i < v1; i++ {
			v2 := NewPopulatedMyType(r)
			this.Arr[i] = *v2
		}
	}
	if !easy && r.Intn(10) != 0 {
		this.XXX_unrecognized = randUnrecognizedAsym(r, 2)
	}
	return this
}

type randyAsym interface {
	Float32() float32
	Float64() float64
	Int63() int64
	Int31() int32
	Uint32() uint32
	Intn(n int) int
}

func randUTF8RuneAsym(r randyAsym) rune {
	ru := r.Intn(62)
	if ru < 10 {
		return rune(ru + 48)
	} else if ru < 36 {
		return rune(ru + 55)
	}
	return rune(ru + 61)
}
func randStringAsym(r randyAsym) string {
	v3 := r.Intn(100)
	tmps := make([]rune, v3)
	for i := 0; i < v3; i++ {
		tmps[i] = randUTF8RuneAsym(r)
	}
	return string(tmps)
}
func randUnrecognizedAsym(r randyAsym, maxFieldNumber int) (data []byte) {
	l := r.Intn(5)
	for i := 0; i < l; i++ {
		wire := r.Intn(4)
		if wire == 3 {
			wire = 5
		}
		fieldNumber := maxFieldNumber + r.Intn(100)
		data = randFieldAsym(data, r, fieldNumber, wire)
	}
	return data
}
func randFieldAsym(data []byte, r randyAsym, fieldNumber int, wire int) []byte {
	key := uint32(fieldNumber)<<3 | uint32(wire)
	switch wire {
	case 0:
		data = encodeVarintPopulateAsym(data, uint64(key))
		v4 := r.Int63()
		if r.Intn(2) == 0 {
			v4 *= -1
		}
		data = encodeVarintPopulateAsym(data, uint64(v4))
	case 1:
		data = encodeVarintPopulateAsym(data, uint64(key))
		data = append(data, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	case 2:
		data = encodeVarintPopulateAsym(data, uint64(key))
		ll := r.Intn(100)
		data = encodeVarintPopulateAsym(data, uint64(ll))
		for j := 0; j < ll; j++ {
			data = append(data, byte(r.Intn(256)))
		}
	default:
		data = encodeVarintPopulateAsym(data, uint64(key))
		data = append(data, byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)), byte(r.Intn(256)))
	}
	return data
}
func encodeVarintPopulateAsym(data []byte, v uint64) []byte {
	for v >= 1<<7 {
		data = append(data, uint8(uint64(v)&0x7f|0x80))
		v >>= 7
	}
	data = append(data, uint8(v))
	return data
}
func (m *M) Size() (n int) {
	var l int
	_ = l
	if len(m.Arr) > 0 {
		for _, e := range m.Arr {
			l = e.Size()
			n += 1 + l + sovAsym(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovAsym(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozAsym(x uint64) (n int) {
	return sovAsym(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *M) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAsym
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
			return fmt.Errorf("proto: M: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: M: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Arr", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAsym
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
				return ErrInvalidLengthAsym
			}
			postIndex := iNdEx + byteLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var v MyType
			m.Arr = append(m.Arr, v)
			if err := m.Arr[len(m.Arr)-1].Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAsym(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthAsym
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
func (m *MyType) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAsym
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
			return fmt.Errorf("proto: MyType: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MyType: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipAsym(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthAsym
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
func skipAsym(data []byte) (n int, err error) {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowAsym
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
					return 0, ErrIntOverflowAsym
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
					return 0, ErrIntOverflowAsym
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
				return 0, ErrInvalidLengthAsym
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowAsym
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
				next, err := skipAsym(data[start:])
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
	ErrInvalidLengthAsym = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowAsym   = fmt.Errorf("proto: integer overflow")
)

var fileDescriptorAsym = []byte{
	// 151 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xe2, 0x4a, 0x2c, 0xae, 0xcc,
	0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x01, 0xb1, 0xa5, 0x74, 0xd3, 0x33, 0x4b, 0x32,
	0x4a, 0x93, 0xf4, 0x92, 0xf3, 0x73, 0xf5, 0xd3, 0xf3, 0xd3, 0xf3, 0xf5, 0xc1, 0x92, 0x49, 0xa5,
	0x69, 0x60, 0x1e, 0x98, 0x03, 0x66, 0x41, 0x34, 0x29, 0xa9, 0x72, 0x31, 0xfa, 0x0a, 0x29, 0x70,
	0x31, 0x27, 0x16, 0x15, 0x49, 0x30, 0x2a, 0x30, 0x6b, 0xf0, 0x38, 0xf1, 0x9d, 0xb8, 0x27, 0xcf,
	0x70, 0xeb, 0x9e, 0x3c, 0x9b, 0x6f, 0x65, 0x48, 0x65, 0x41, 0x6a, 0x10, 0x48, 0x4a, 0x49, 0x8a,
	0x0b, 0xca, 0xb5, 0x12, 0xd8, 0xb1, 0x40, 0x9e, 0xe1, 0x07, 0x10, 0x77, 0x2c, 0x94, 0x67, 0x58,
	0x00, 0xc4, 0x4e, 0x32, 0x0f, 0x1e, 0xca, 0x31, 0xfe, 0x00, 0xe2, 0x15, 0x8f, 0xe4, 0x18, 0x77,
	0x00, 0xf1, 0x09, 0x20, 0xbe, 0x00, 0xc4, 0x0f, 0x80, 0x18, 0x10, 0x00, 0x00, 0xff, 0xff, 0x9c,
	0x0b, 0x12, 0x6c, 0xa2, 0x00, 0x00, 0x00,
}
