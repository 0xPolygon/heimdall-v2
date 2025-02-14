// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: heimdallv2/checkpoint/checkpoint_signatures.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type CheckpointSignature struct {
	ValidatorAddress []byte `protobuf:"bytes,1,opt,name=validator_address,json=validatorAddress,proto3" json:"validator_address,omitempty"`
	Signature        []byte `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (m *CheckpointSignature) Reset()         { *m = CheckpointSignature{} }
func (m *CheckpointSignature) String() string { return proto.CompactTextString(m) }
func (*CheckpointSignature) ProtoMessage()    {}
func (*CheckpointSignature) Descriptor() ([]byte, []int) {
	return fileDescriptor_dc6f26830ee7391c, []int{0}
}
func (m *CheckpointSignature) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *CheckpointSignature) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_CheckpointSignature.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *CheckpointSignature) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CheckpointSignature.Merge(m, src)
}
func (m *CheckpointSignature) XXX_Size() int {
	return m.Size()
}
func (m *CheckpointSignature) XXX_DiscardUnknown() {
	xxx_messageInfo_CheckpointSignature.DiscardUnknown(m)
}

var xxx_messageInfo_CheckpointSignature proto.InternalMessageInfo

func (m *CheckpointSignature) GetValidatorAddress() []byte {
	if m != nil {
		return m.ValidatorAddress
	}
	return nil
}

func (m *CheckpointSignature) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

type CheckpointSignatures struct {
	Signatures []CheckpointSignature `protobuf:"bytes,1,rep,name=signatures,proto3" json:"signatures"`
}

func (m *CheckpointSignatures) Reset()         { *m = CheckpointSignatures{} }
func (m *CheckpointSignatures) String() string { return proto.CompactTextString(m) }
func (*CheckpointSignatures) ProtoMessage()    {}
func (*CheckpointSignatures) Descriptor() ([]byte, []int) {
	return fileDescriptor_dc6f26830ee7391c, []int{1}
}
func (m *CheckpointSignatures) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *CheckpointSignatures) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_CheckpointSignatures.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *CheckpointSignatures) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CheckpointSignatures.Merge(m, src)
}
func (m *CheckpointSignatures) XXX_Size() int {
	return m.Size()
}
func (m *CheckpointSignatures) XXX_DiscardUnknown() {
	xxx_messageInfo_CheckpointSignatures.DiscardUnknown(m)
}

var xxx_messageInfo_CheckpointSignatures proto.InternalMessageInfo

func (m *CheckpointSignatures) GetSignatures() []CheckpointSignature {
	if m != nil {
		return m.Signatures
	}
	return nil
}

func init() {
	proto.RegisterType((*CheckpointSignature)(nil), "heimdallv2.checkpoint.CheckpointSignature")
	proto.RegisterType((*CheckpointSignatures)(nil), "heimdallv2.checkpoint.CheckpointSignatures")
}

func init() {
	proto.RegisterFile("heimdallv2/checkpoint/checkpoint_signatures.proto", fileDescriptor_dc6f26830ee7391c)
}

var fileDescriptor_dc6f26830ee7391c = []byte{
	// 270 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x32, 0xcc, 0x48, 0xcd, 0xcc,
	0x4d, 0x49, 0xcc, 0xc9, 0x29, 0x33, 0xd2, 0x4f, 0xce, 0x48, 0x4d, 0xce, 0x2e, 0xc8, 0xcf, 0xcc,
	0x2b, 0x41, 0x62, 0xc6, 0x17, 0x67, 0xa6, 0xe7, 0x25, 0x96, 0x94, 0x16, 0xa5, 0x16, 0xeb, 0x15,
	0x14, 0xe5, 0x97, 0xe4, 0x0b, 0x89, 0x22, 0xb4, 0xe8, 0x21, 0xd4, 0x49, 0x09, 0x26, 0xe6, 0x66,
	0xe6, 0xe5, 0xeb, 0x83, 0x49, 0x88, 0x4a, 0x29, 0x91, 0xf4, 0xfc, 0xf4, 0x7c, 0x30, 0x53, 0x1f,
	0xc4, 0x82, 0x88, 0x2a, 0xe5, 0x71, 0x09, 0x3b, 0xc3, 0xb5, 0x05, 0xc3, 0x4c, 0x17, 0x32, 0xe2,
	0x12, 0x2c, 0x4b, 0xcc, 0xc9, 0x4c, 0x49, 0x2c, 0xc9, 0x2f, 0x8a, 0x4f, 0x4c, 0x49, 0x29, 0x4a,
	0x2d, 0x2e, 0x96, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x71, 0x62, 0x5d, 0xf1, 0x7c, 0x83, 0x16, 0x63,
	0x90, 0x00, 0x5c, 0xde, 0x11, 0x22, 0x2d, 0xa4, 0xcc, 0xc5, 0x09, 0x77, 0x9e, 0x04, 0x13, 0xb2,
	0x5a, 0x84, 0xb8, 0x52, 0x2e, 0x97, 0x08, 0x16, 0xfb, 0x8a, 0x85, 0x42, 0xb9, 0xb8, 0x10, 0x7e,
	0x93, 0x60, 0x54, 0x60, 0xd6, 0xe0, 0x36, 0xd2, 0xd2, 0xc3, 0xea, 0x39, 0x3d, 0x2c, 0x06, 0x38,
	0x71, 0x9e, 0xb8, 0x27, 0xcf, 0x00, 0xb1, 0x0d, 0xc9, 0x20, 0x27, 0xdf, 0x13, 0x8f, 0xe4, 0x18,
	0x2f, 0x3c, 0x92, 0x63, 0x7c, 0xf0, 0x48, 0x8e, 0x71, 0xc2, 0x63, 0x39, 0x86, 0x0b, 0x8f, 0xe5,
	0x18, 0x6e, 0x3c, 0x96, 0x63, 0x88, 0x32, 0x4e, 0xcf, 0x2c, 0xc9, 0x28, 0x4d, 0xd2, 0x4b, 0xce,
	0xcf, 0xd5, 0x37, 0xa8, 0x08, 0xc8, 0xcf, 0xa9, 0x4c, 0xcf, 0xcf, 0xd3, 0x87, 0x59, 0xa8, 0x5b,
	0x66, 0xa4, 0x5f, 0x81, 0x1c, 0x07, 0x25, 0x95, 0x05, 0xa9, 0xc5, 0x49, 0x6c, 0xe0, 0x40, 0x33,
	0x06, 0x04, 0x00, 0x00, 0xff, 0xff, 0x7b, 0x3e, 0x68, 0xd1, 0xa9, 0x01, 0x00, 0x00,
}

func (m *CheckpointSignature) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *CheckpointSignature) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *CheckpointSignature) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Signature) > 0 {
		i -= len(m.Signature)
		copy(dAtA[i:], m.Signature)
		i = encodeVarintCheckpointSignatures(dAtA, i, uint64(len(m.Signature)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.ValidatorAddress) > 0 {
		i -= len(m.ValidatorAddress)
		copy(dAtA[i:], m.ValidatorAddress)
		i = encodeVarintCheckpointSignatures(dAtA, i, uint64(len(m.ValidatorAddress)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *CheckpointSignatures) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *CheckpointSignatures) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *CheckpointSignatures) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Signatures) > 0 {
		for iNdEx := len(m.Signatures) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Signatures[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintCheckpointSignatures(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintCheckpointSignatures(dAtA []byte, offset int, v uint64) int {
	offset -= sovCheckpointSignatures(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *CheckpointSignature) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ValidatorAddress)
	if l > 0 {
		n += 1 + l + sovCheckpointSignatures(uint64(l))
	}
	l = len(m.Signature)
	if l > 0 {
		n += 1 + l + sovCheckpointSignatures(uint64(l))
	}
	return n
}

func (m *CheckpointSignatures) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Signatures) > 0 {
		for _, e := range m.Signatures {
			l = e.Size()
			n += 1 + l + sovCheckpointSignatures(uint64(l))
		}
	}
	return n
}

func sovCheckpointSignatures(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozCheckpointSignatures(x uint64) (n int) {
	return sovCheckpointSignatures(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *CheckpointSignature) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCheckpointSignatures
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
			return fmt.Errorf("proto: CheckpointSignature: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: CheckpointSignature: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ValidatorAddress", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpointSignatures
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthCheckpointSignatures
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpointSignatures
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ValidatorAddress = append(m.ValidatorAddress[:0], dAtA[iNdEx:postIndex]...)
			if m.ValidatorAddress == nil {
				m.ValidatorAddress = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signature", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpointSignatures
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthCheckpointSignatures
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpointSignatures
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Signature = append(m.Signature[:0], dAtA[iNdEx:postIndex]...)
			if m.Signature == nil {
				m.Signature = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipCheckpointSignatures(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthCheckpointSignatures
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
func (m *CheckpointSignatures) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCheckpointSignatures
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
			return fmt.Errorf("proto: CheckpointSignatures: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: CheckpointSignatures: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signatures", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCheckpointSignatures
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
				return ErrInvalidLengthCheckpointSignatures
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthCheckpointSignatures
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Signatures = append(m.Signatures, CheckpointSignature{})
			if err := m.Signatures[len(m.Signatures)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipCheckpointSignatures(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthCheckpointSignatures
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
func skipCheckpointSignatures(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowCheckpointSignatures
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
					return 0, ErrIntOverflowCheckpointSignatures
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowCheckpointSignatures
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
				return 0, ErrInvalidLengthCheckpointSignatures
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupCheckpointSignatures
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthCheckpointSignatures
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthCheckpointSignatures        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowCheckpointSignatures          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupCheckpointSignatures = fmt.Errorf("proto: unexpected end of group")
)
