/*
Copyright 2018 The Kubernetes Authors.

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
// source: k8s.io/kubernetes/vendor/k8s.io/api/storage/v1/generated.proto
// DO NOT EDIT!

/*
	Package v1 is a generated protocol buffer package.

	It is generated from these files:
		k8s.io/kubernetes/vendor/k8s.io/api/storage/v1/generated.proto

	It has these top-level messages:
		StorageClass
		StorageClassList
*/
package v1

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"

import k8s_io_api_core_v1 "k8s.io/api/core/v1"

import github_com_gogo_protobuf_sortkeys "github.com/gogo/protobuf/sortkeys"

import strings "strings"
import reflect "reflect"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

func (m *StorageClass) Reset()                    { *m = StorageClass{} }
func (*StorageClass) ProtoMessage()               {}
func (*StorageClass) Descriptor() ([]byte, []int) { return fileDescriptorGenerated, []int{0} }

func (m *StorageClassList) Reset()                    { *m = StorageClassList{} }
func (*StorageClassList) ProtoMessage()               {}
func (*StorageClassList) Descriptor() ([]byte, []int) { return fileDescriptorGenerated, []int{1} }

func init() {
	proto.RegisterType((*StorageClass)(nil), "k8s.io.api.storage.v1.StorageClass")
	proto.RegisterType((*StorageClassList)(nil), "k8s.io.api.storage.v1.StorageClassList")
}
func (m *StorageClass) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *StorageClass) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	dAtA[i] = 0xa
	i++
	i = encodeVarintGenerated(dAtA, i, uint64(m.ObjectMeta.Size()))
	n1, err := m.ObjectMeta.MarshalTo(dAtA[i:])
	if err != nil {
		return 0, err
	}
	i += n1
	dAtA[i] = 0x12
	i++
	i = encodeVarintGenerated(dAtA, i, uint64(len(m.Provisioner)))
	i += copy(dAtA[i:], m.Provisioner)
	if len(m.Parameters) > 0 {
		keysForParameters := make([]string, 0, len(m.Parameters))
		for k := range m.Parameters {
			keysForParameters = append(keysForParameters, string(k))
		}
		github_com_gogo_protobuf_sortkeys.Strings(keysForParameters)
		for _, k := range keysForParameters {
			dAtA[i] = 0x1a
			i++
			v := m.Parameters[string(k)]
			mapSize := 1 + len(k) + sovGenerated(uint64(len(k))) + 1 + len(v) + sovGenerated(uint64(len(v)))
			i = encodeVarintGenerated(dAtA, i, uint64(mapSize))
			dAtA[i] = 0xa
			i++
			i = encodeVarintGenerated(dAtA, i, uint64(len(k)))
			i += copy(dAtA[i:], k)
			dAtA[i] = 0x12
			i++
			i = encodeVarintGenerated(dAtA, i, uint64(len(v)))
			i += copy(dAtA[i:], v)
		}
	}
	if m.ReclaimPolicy != nil {
		dAtA[i] = 0x22
		i++
		i = encodeVarintGenerated(dAtA, i, uint64(len(*m.ReclaimPolicy)))
		i += copy(dAtA[i:], *m.ReclaimPolicy)
	}
	if len(m.MountOptions) > 0 {
		for _, s := range m.MountOptions {
			dAtA[i] = 0x2a
			i++
			l = len(s)
			for l >= 1<<7 {
				dAtA[i] = uint8(uint64(l)&0x7f | 0x80)
				l >>= 7
				i++
			}
			dAtA[i] = uint8(l)
			i++
			i += copy(dAtA[i:], s)
		}
	}
	if m.AllowVolumeExpansion != nil {
		dAtA[i] = 0x30
		i++
		if *m.AllowVolumeExpansion {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i++
	}
	if m.VolumeBindingMode != nil {
		dAtA[i] = 0x3a
		i++
		i = encodeVarintGenerated(dAtA, i, uint64(len(*m.VolumeBindingMode)))
		i += copy(dAtA[i:], *m.VolumeBindingMode)
	}
	return i, nil
}

func (m *StorageClassList) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *StorageClassList) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	dAtA[i] = 0xa
	i++
	i = encodeVarintGenerated(dAtA, i, uint64(m.ListMeta.Size()))
	n2, err := m.ListMeta.MarshalTo(dAtA[i:])
	if err != nil {
		return 0, err
	}
	i += n2
	if len(m.Items) > 0 {
		for _, msg := range m.Items {
			dAtA[i] = 0x12
			i++
			i = encodeVarintGenerated(dAtA, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(dAtA[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	return i, nil
}

func encodeFixed64Generated(dAtA []byte, offset int, v uint64) int {
	dAtA[offset] = uint8(v)
	dAtA[offset+1] = uint8(v >> 8)
	dAtA[offset+2] = uint8(v >> 16)
	dAtA[offset+3] = uint8(v >> 24)
	dAtA[offset+4] = uint8(v >> 32)
	dAtA[offset+5] = uint8(v >> 40)
	dAtA[offset+6] = uint8(v >> 48)
	dAtA[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Generated(dAtA []byte, offset int, v uint32) int {
	dAtA[offset] = uint8(v)
	dAtA[offset+1] = uint8(v >> 8)
	dAtA[offset+2] = uint8(v >> 16)
	dAtA[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintGenerated(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *StorageClass) Size() (n int) {
	var l int
	_ = l
	l = m.ObjectMeta.Size()
	n += 1 + l + sovGenerated(uint64(l))
	l = len(m.Provisioner)
	n += 1 + l + sovGenerated(uint64(l))
	if len(m.Parameters) > 0 {
		for k, v := range m.Parameters {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovGenerated(uint64(len(k))) + 1 + len(v) + sovGenerated(uint64(len(v)))
			n += mapEntrySize + 1 + sovGenerated(uint64(mapEntrySize))
		}
	}
	if m.ReclaimPolicy != nil {
		l = len(*m.ReclaimPolicy)
		n += 1 + l + sovGenerated(uint64(l))
	}
	if len(m.MountOptions) > 0 {
		for _, s := range m.MountOptions {
			l = len(s)
			n += 1 + l + sovGenerated(uint64(l))
		}
	}
	if m.AllowVolumeExpansion != nil {
		n += 2
	}
	if m.VolumeBindingMode != nil {
		l = len(*m.VolumeBindingMode)
		n += 1 + l + sovGenerated(uint64(l))
	}
	return n
}

func (m *StorageClassList) Size() (n int) {
	var l int
	_ = l
	l = m.ListMeta.Size()
	n += 1 + l + sovGenerated(uint64(l))
	if len(m.Items) > 0 {
		for _, e := range m.Items {
			l = e.Size()
			n += 1 + l + sovGenerated(uint64(l))
		}
	}
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
func (this *StorageClass) String() string {
	if this == nil {
		return "nil"
	}
	keysForParameters := make([]string, 0, len(this.Parameters))
	for k := range this.Parameters {
		keysForParameters = append(keysForParameters, k)
	}
	github_com_gogo_protobuf_sortkeys.Strings(keysForParameters)
	mapStringForParameters := "map[string]string{"
	for _, k := range keysForParameters {
		mapStringForParameters += fmt.Sprintf("%v: %v,", k, this.Parameters[k])
	}
	mapStringForParameters += "}"
	s := strings.Join([]string{`&StorageClass{`,
		`ObjectMeta:` + strings.Replace(strings.Replace(this.ObjectMeta.String(), "ObjectMeta", "k8s_io_apimachinery_pkg_apis_meta_v1.ObjectMeta", 1), `&`, ``, 1) + `,`,
		`Provisioner:` + fmt.Sprintf("%v", this.Provisioner) + `,`,
		`Parameters:` + mapStringForParameters + `,`,
		`ReclaimPolicy:` + valueToStringGenerated(this.ReclaimPolicy) + `,`,
		`MountOptions:` + fmt.Sprintf("%v", this.MountOptions) + `,`,
		`AllowVolumeExpansion:` + valueToStringGenerated(this.AllowVolumeExpansion) + `,`,
		`VolumeBindingMode:` + valueToStringGenerated(this.VolumeBindingMode) + `,`,
		`}`,
	}, "")
	return s
}
func (this *StorageClassList) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&StorageClassList{`,
		`ListMeta:` + strings.Replace(strings.Replace(this.ListMeta.String(), "ListMeta", "k8s_io_apimachinery_pkg_apis_meta_v1.ListMeta", 1), `&`, ``, 1) + `,`,
		`Items:` + strings.Replace(strings.Replace(fmt.Sprintf("%v", this.Items), "StorageClass", "StorageClass", 1), `&`, ``, 1) + `,`,
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
func (m *StorageClass) Unmarshal(dAtA []byte) error {
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
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: StorageClass: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: StorageClass: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ObjectMeta", wireType)
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
			if err := m.ObjectMeta.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Provisioner", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
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
			m.Provisioner = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Parameters", wireType)
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
			var keykey uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				keykey |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			var stringLenmapkey uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLenmapkey |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLenmapkey := int(stringLenmapkey)
			if intStringLenmapkey < 0 {
				return ErrInvalidLengthGenerated
			}
			postStringIndexmapkey := iNdEx + intStringLenmapkey
			if postStringIndexmapkey > l {
				return io.ErrUnexpectedEOF
			}
			mapkey := string(dAtA[iNdEx:postStringIndexmapkey])
			iNdEx = postStringIndexmapkey
			if m.Parameters == nil {
				m.Parameters = make(map[string]string)
			}
			if iNdEx < postIndex {
				var valuekey uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowGenerated
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					valuekey |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				var stringLenmapvalue uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowGenerated
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					stringLenmapvalue |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				intStringLenmapvalue := int(stringLenmapvalue)
				if intStringLenmapvalue < 0 {
					return ErrInvalidLengthGenerated
				}
				postStringIndexmapvalue := iNdEx + intStringLenmapvalue
				if postStringIndexmapvalue > l {
					return io.ErrUnexpectedEOF
				}
				mapvalue := string(dAtA[iNdEx:postStringIndexmapvalue])
				iNdEx = postStringIndexmapvalue
				m.Parameters[mapkey] = mapvalue
			} else {
				var mapvalue string
				m.Parameters[mapkey] = mapvalue
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ReclaimPolicy", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
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
			s := k8s_io_api_core_v1.PersistentVolumeReclaimPolicy(dAtA[iNdEx:postIndex])
			m.ReclaimPolicy = &s
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MountOptions", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
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
			m.MountOptions = append(m.MountOptions, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field AllowVolumeExpansion", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			b := bool(v != 0)
			m.AllowVolumeExpansion = &b
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field VolumeBindingMode", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenerated
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
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
			s := VolumeBindingMode(dAtA[iNdEx:postIndex])
			m.VolumeBindingMode = &s
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
func (m *StorageClassList) Unmarshal(dAtA []byte) error {
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
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: StorageClassList: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: StorageClassList: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
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
			if err := m.ListMeta.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
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
			m.Items = append(m.Items, StorageClass{})
			if err := m.Items[len(m.Items)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
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

func init() {
	proto.RegisterFile("k8s.io/kubernetes/vendor/k8s.io/api/storage/v1/generated.proto", fileDescriptorGenerated)
}

var fileDescriptorGenerated = []byte{
	// 623 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x52, 0x4f, 0x6f, 0xd3, 0x3c,
	0x18, 0x6f, 0xda, 0xb7, 0x2f, 0x9b, 0xbb, 0x89, 0x2e, 0x0c, 0x29, 0xea, 0x21, 0xa9, 0xc6, 0xa5,
	0x9a, 0x84, 0xb3, 0x6e, 0x03, 0x4d, 0x48, 0x20, 0x11, 0x34, 0x09, 0xa4, 0x4d, 0xab, 0x82, 0x34,
	0x21, 0xc4, 0x01, 0x37, 0x7d, 0xc8, 0x4c, 0x13, 0x3b, 0xb2, 0x9d, 0x40, 0x6f, 0x7c, 0x04, 0xce,
	0x7c, 0x14, 0x3e, 0xc1, 0x8e, 0x3b, 0xee, 0x14, 0xb1, 0xf0, 0x2d, 0x76, 0x42, 0x49, 0xca, 0x9a,
	0xad, 0x9d, 0xd8, 0x2d, 0xfe, 0xfd, 0xb3, 0x9f, 0x27, 0x3f, 0xf4, 0x62, 0xbc, 0x27, 0x31, 0xe5,
	0xf6, 0x38, 0x1e, 0x82, 0x60, 0xa0, 0x40, 0xda, 0x09, 0xb0, 0x11, 0x17, 0xf6, 0x94, 0x20, 0x11,
	0xb5, 0xa5, 0xe2, 0x82, 0xf8, 0x60, 0x27, 0x7d, 0xdb, 0x07, 0x06, 0x82, 0x28, 0x18, 0xe1, 0x48,
	0x70, 0xc5, 0xf5, 0x87, 0xa5, 0x0c, 0x93, 0x88, 0xe2, 0xa9, 0x0c, 0x27, 0xfd, 0xce, 0x63, 0x9f,
	0xaa, 0x93, 0x78, 0x88, 0x3d, 0x1e, 0xda, 0x3e, 0xf7, 0xb9, 0x5d, 0xa8, 0x87, 0xf1, 0xa7, 0xe2,
	0x54, 0x1c, 0x8a, 0xaf, 0x32, 0xa5, 0xb3, 0xb9, 0xf0, 0xb2, 0x21, 0x28, 0x32, 0x77, 0x63, 0x67,
	0x77, 0xa6, 0x0d, 0x89, 0x77, 0x42, 0x19, 0x88, 0x89, 0x1d, 0x8d, 0xfd, 0x1c, 0x90, 0x76, 0x08,
	0x8a, 0x2c, 0x78, 0x67, 0xc7, 0xbe, 0xcd, 0x25, 0x62, 0xa6, 0x68, 0x08, 0x73, 0x86, 0xa7, 0xff,
	0x32, 0x48, 0xef, 0x04, 0x42, 0x32, 0xe7, 0xdb, 0xb9, 0xcd, 0x17, 0x2b, 0x1a, 0xd8, 0x94, 0x29,
	0xa9, 0xc4, 0x4d, 0xd3, 0xc6, 0x8f, 0x26, 0x5a, 0x79, 0x5b, 0xce, 0xfd, 0x2a, 0x20, 0x52, 0xea,
	0x1f, 0xd1, 0x52, 0x3e, 0xc9, 0x88, 0x28, 0x62, 0x68, 0x5d, 0xad, 0xd7, 0xda, 0xde, 0xc2, 0xb3,
	0x4d, 0x5f, 0x05, 0xe3, 0x68, 0xec, 0xe7, 0x80, 0xc4, 0xb9, 0x1a, 0x27, 0x7d, 0x7c, 0x34, 0xfc,
	0x0c, 0x9e, 0x3a, 0x04, 0x45, 0x1c, 0xfd, 0x34, 0xb5, 0x6a, 0x59, 0x6a, 0xa1, 0x19, 0xe6, 0x5e,
	0xa5, 0xea, 0x4f, 0x50, 0x2b, 0x12, 0x3c, 0xa1, 0x92, 0x72, 0x06, 0xc2, 0xa8, 0x77, 0xb5, 0xde,
	0xb2, 0xf3, 0x60, 0x6a, 0x69, 0x0d, 0x66, 0x94, 0x5b, 0xd5, 0xe9, 0x3e, 0x42, 0x11, 0x11, 0x24,
	0x04, 0x05, 0x42, 0x1a, 0x8d, 0x6e, 0xa3, 0xd7, 0xda, 0xde, 0xc1, 0x0b, 0x4b, 0x80, 0xab, 0x13,
	0xe1, 0xc1, 0x95, 0x6b, 0x9f, 0x29, 0x31, 0x99, 0xbd, 0x6e, 0x46, 0xb8, 0x95, 0x68, 0x7d, 0x8c,
	0x56, 0x05, 0x78, 0x01, 0xa1, 0xe1, 0x80, 0x07, 0xd4, 0x9b, 0x18, 0xff, 0x15, 0x2f, 0xdc, 0xcf,
	0x52, 0x6b, 0xd5, 0xad, 0x12, 0x97, 0xa9, 0xb5, 0x55, 0xa9, 0x8f, 0xc7, 0x45, 0xde, 0x1d, 0x3c,
	0x00, 0x21, 0xa9, 0x54, 0xc0, 0xd4, 0x31, 0x0f, 0xe2, 0x10, 0xae, 0x79, 0xdc, 0xeb, 0xd9, 0xfa,
	0x2e, 0x5a, 0x09, 0x79, 0xcc, 0xd4, 0x51, 0xa4, 0x28, 0x67, 0xd2, 0x68, 0x76, 0x1b, 0xbd, 0x65,
	0xa7, 0x9d, 0xa5, 0xd6, 0xca, 0x61, 0x05, 0x77, 0xaf, 0xa9, 0xf4, 0x03, 0xb4, 0x4e, 0x82, 0x80,
	0x7f, 0x29, 0x2f, 0xd8, 0xff, 0x1a, 0x11, 0x96, 0x6f, 0xc9, 0xf8, 0xbf, 0xab, 0xf5, 0x96, 0x1c,
	0x23, 0x4b, 0xad, 0xf5, 0x97, 0x0b, 0x78, 0x77, 0xa1, 0x4b, 0x7f, 0x87, 0xd6, 0x92, 0x02, 0x72,
	0x28, 0x1b, 0x51, 0xe6, 0x1f, 0xf2, 0x11, 0x18, 0xf7, 0x8a, 0xa1, 0x37, 0xb3, 0xd4, 0x5a, 0x3b,
	0xbe, 0x49, 0x5e, 0x2e, 0x02, 0xdd, 0xf9, 0x90, 0xce, 0x73, 0x74, 0xff, 0xc6, 0xf6, 0xf5, 0x36,
	0x6a, 0x8c, 0x61, 0x52, 0x54, 0x6b, 0xd9, 0xcd, 0x3f, 0xf5, 0x75, 0xd4, 0x4c, 0x48, 0x10, 0x43,
	0xd9, 0x04, 0xb7, 0x3c, 0x3c, 0xab, 0xef, 0x69, 0x1b, 0x3f, 0x35, 0xd4, 0xae, 0xfe, 0xca, 0x03,
	0x2a, 0x95, 0xfe, 0x61, 0xae, 0xa0, 0xf8, 0x6e, 0x05, 0xcd, 0xdd, 0x45, 0x3d, 0xdb, 0xd3, 0x02,
	0x2c, 0xfd, 0x45, 0x2a, 0xe5, 0x7c, 0x8d, 0x9a, 0x54, 0x41, 0x28, 0x8d, 0x7a, 0x51, 0xb0, 0x47,
	0x77, 0x28, 0x98, 0xb3, 0x3a, 0xcd, 0x6b, 0xbe, 0xc9, 0x9d, 0x6e, 0x19, 0xe0, 0xf4, 0x4e, 0x2f,
	0xcc, 0xda, 0xd9, 0x85, 0x59, 0x3b, 0xbf, 0x30, 0x6b, 0xdf, 0x32, 0x53, 0x3b, 0xcd, 0x4c, 0xed,
	0x2c, 0x33, 0xb5, 0xf3, 0xcc, 0xd4, 0x7e, 0x65, 0xa6, 0xf6, 0xfd, 0xb7, 0x59, 0x7b, 0x5f, 0x4f,
	0xfa, 0x7f, 0x02, 0x00, 0x00, 0xff, 0xff, 0xee, 0x56, 0xcc, 0xfd, 0x0a, 0x05, 0x00, 0x00,
}
