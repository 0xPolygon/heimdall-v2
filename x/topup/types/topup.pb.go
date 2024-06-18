// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: heimdallv2/topup/v1/topup.proto

package types

import (
	cosmossdk_io_math "cosmossdk.io/math"
	fmt "fmt"
	types "github.com/0xPolygon/heimdall-v2/types"
	_ "github.com/cosmos/cosmos-proto"
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

// MsgTopup defines a high level transaction model for the topup module
type MsgTopup struct {
	FromAddress string                `protobuf:"bytes,1,opt,name=from_address,json=fromAddress,proto3" json:"from_address,omitempty"`
	User        string                `protobuf:"bytes,2,opt,name=user,proto3" json:"user,omitempty"`
	Fee         cosmossdk_io_math.Int `protobuf:"bytes,3,opt,name=fee,proto3,customtype=cosmossdk.io/math.Int" json:"fee"`
	TxHash      *types.TxHash         `protobuf:"bytes,4,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
	LogIndex    uint64                `protobuf:"varint,5,opt,name=log_index,json=logIndex,proto3" json:"log_index,omitempty"`
	BlockNumber uint64                `protobuf:"varint,6,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
}

func (m *MsgTopup) Reset()         { *m = MsgTopup{} }
func (m *MsgTopup) String() string { return proto.CompactTextString(m) }
func (*MsgTopup) ProtoMessage()    {}
func (*MsgTopup) Descriptor() ([]byte, []int) {
	return fileDescriptor_6d306c785079c66b, []int{0}
}
func (m *MsgTopup) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgTopup) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgTopup.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgTopup) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgTopup.Merge(m, src)
}
func (m *MsgTopup) XXX_Size() int {
	return m.Size()
}
func (m *MsgTopup) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgTopup.DiscardUnknown(m)
}

var xxx_messageInfo_MsgTopup proto.InternalMessageInfo

func (m *MsgTopup) GetFromAddress() string {
	if m != nil {
		return m.FromAddress
	}
	return ""
}

func (m *MsgTopup) GetUser() string {
	if m != nil {
		return m.User
	}
	return ""
}

func (m *MsgTopup) GetTxHash() *types.TxHash {
	if m != nil {
		return m.TxHash
	}
	return nil
}

func (m *MsgTopup) GetLogIndex() uint64 {
	if m != nil {
		return m.LogIndex
	}
	return 0
}

func (m *MsgTopup) GetBlockNumber() uint64 {
	if m != nil {
		return m.BlockNumber
	}
	return 0
}

// MsgWithdrawFee defines a high level transaction for the withdrawal of fees in
// topup module
type MsgWithdrawFee struct {
	FromAddress string                `protobuf:"bytes,1,opt,name=from_address,json=fromAddress,proto3" json:"from_address,omitempty"`
	Amount      cosmossdk_io_math.Int `protobuf:"bytes,2,opt,name=amount,proto3,customtype=cosmossdk.io/math.Int" json:"amount"`
}

func (m *MsgWithdrawFee) Reset()         { *m = MsgWithdrawFee{} }
func (m *MsgWithdrawFee) String() string { return proto.CompactTextString(m) }
func (*MsgWithdrawFee) ProtoMessage()    {}
func (*MsgWithdrawFee) Descriptor() ([]byte, []int) {
	return fileDescriptor_6d306c785079c66b, []int{1}
}
func (m *MsgWithdrawFee) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgWithdrawFee) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgWithdrawFee.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgWithdrawFee) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgWithdrawFee.Merge(m, src)
}
func (m *MsgWithdrawFee) XXX_Size() int {
	return m.Size()
}
func (m *MsgWithdrawFee) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgWithdrawFee.DiscardUnknown(m)
}

var xxx_messageInfo_MsgWithdrawFee proto.InternalMessageInfo

func (m *MsgWithdrawFee) GetFromAddress() string {
	if m != nil {
		return m.FromAddress
	}
	return ""
}

func init() {
	proto.RegisterType((*MsgTopup)(nil), "heimdallv2.topup.v1.MsgTopup")
	proto.RegisterType((*MsgWithdrawFee)(nil), "heimdallv2.topup.v1.MsgWithdrawFee")
}

func init() { proto.RegisterFile("heimdallv2/topup/v1/topup.proto", fileDescriptor_6d306c785079c66b) }

var fileDescriptor_6d306c785079c66b = []byte{
	// 421 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x52, 0x4f, 0x6b, 0xd4, 0x40,
	0x14, 0xdf, 0x69, 0x63, 0x6c, 0x67, 0x8b, 0x87, 0x58, 0x61, 0x6c, 0x21, 0x59, 0x7b, 0x5a, 0xd0,
	0x9d, 0x71, 0xd7, 0x9b, 0xe2, 0xc1, 0x0a, 0xd2, 0x3d, 0x54, 0x24, 0x16, 0x04, 0x2f, 0x21, 0xd9,
	0x4c, 0x27, 0xa1, 0x49, 0x5e, 0xc8, 0x4c, 0x62, 0xfa, 0x2d, 0x3c, 0x7a, 0xf4, 0x0b, 0x78, 0xeb,
	0x87, 0xe8, 0xb1, 0xf4, 0x24, 0x1e, 0x8a, 0xec, 0x5e, 0xfc, 0x18, 0x92, 0x99, 0x88, 0x7b, 0x13,
	0xe9, 0x29, 0xef, 0xfd, 0xfe, 0xbc, 0xcc, 0xfb, 0xf1, 0xb0, 0x97, 0xf0, 0x34, 0x8f, 0xc3, 0x2c,
	0x6b, 0x66, 0x4c, 0x41, 0x59, 0x97, 0xac, 0x99, 0x9a, 0x82, 0x96, 0x15, 0x28, 0x70, 0xee, 0xff,
	0x15, 0x50, 0x83, 0x37, 0xd3, 0xbd, 0x5d, 0x01, 0x02, 0x34, 0xcf, 0xba, 0xca, 0x48, 0xf7, 0x1e,
	0x2e, 0x40, 0xe6, 0x20, 0x03, 0x43, 0x98, 0xa6, 0xa7, 0xf6, 0xd7, 0x7f, 0x73, 0x5e, 0x72, 0xc9,
	0x92, 0x50, 0x26, 0x86, 0x3c, 0xf8, 0xb6, 0x81, 0xb7, 0x8e, 0xa5, 0x38, 0xe9, 0xa6, 0x3b, 0x2f,
	0xf0, 0xce, 0x69, 0x05, 0x79, 0x10, 0xc6, 0x71, 0xc5, 0xa5, 0x24, 0x68, 0x84, 0xc6, 0xdb, 0x87,
	0xe4, 0xfa, 0x62, 0xb2, 0xdb, 0x4f, 0x7c, 0x65, 0x98, 0xf7, 0xaa, 0x4a, 0x0b, 0xe1, 0x0f, 0x3b,
	0x75, 0x0f, 0x39, 0x4f, 0xb0, 0x55, 0x4b, 0x5e, 0x91, 0x8d, 0x7f, 0x98, 0xb4, 0xca, 0x79, 0x89,
	0x37, 0x4f, 0x39, 0x27, 0x9b, 0x5a, 0xfc, 0xf8, 0xf2, 0xc6, 0x1b, 0xfc, 0xb8, 0xf1, 0x1e, 0x18,
	0x83, 0x8c, 0xcf, 0x68, 0x0a, 0x2c, 0x0f, 0x55, 0x42, 0xe7, 0x85, 0xba, 0xbe, 0x98, 0xe0, 0x7e,
	0xd2, 0xbc, 0x50, 0x7e, 0xe7, 0x73, 0xa6, 0xf8, 0xae, 0x6a, 0x83, 0x6e, 0x0f, 0x62, 0x8d, 0xd0,
	0x78, 0x38, 0x23, 0x74, 0x3d, 0xab, 0x6e, 0x4b, 0x7a, 0xd2, 0x1e, 0x85, 0x32, 0xf1, 0x6d, 0xa5,
	0xbf, 0xce, 0x3e, 0xde, 0xce, 0x40, 0x04, 0x69, 0x11, 0xf3, 0x96, 0xdc, 0x19, 0xa1, 0xb1, 0xe5,
	0x6f, 0x65, 0x20, 0xe6, 0x5d, 0xef, 0x3c, 0xc2, 0x3b, 0x51, 0x06, 0x8b, 0xb3, 0xa0, 0xa8, 0xf3,
	0x88, 0x57, 0xc4, 0xd6, 0xfc, 0x50, 0x63, 0x6f, 0x35, 0xf4, 0xdc, 0xfa, 0xf5, 0xd5, 0x43, 0x07,
	0x5f, 0x10, 0xbe, 0x77, 0x2c, 0xc5, 0x87, 0x54, 0x25, 0x71, 0x15, 0x7e, 0x7a, 0xc3, 0xf9, 0xed,
	0x52, 0x7b, 0x8d, 0xed, 0x30, 0x87, 0xba, 0x50, 0x7d, 0x6e, 0xff, 0x15, 0x45, 0x6f, 0x35, 0x4f,
	0x3b, 0x3c, 0xba, 0x5c, 0xba, 0xe8, 0x6a, 0xe9, 0xa2, 0x9f, 0x4b, 0x17, 0x7d, 0x5e, 0xb9, 0x83,
	0xab, 0x95, 0x3b, 0xf8, 0xbe, 0x72, 0x07, 0x1f, 0xa9, 0x48, 0x55, 0x52, 0x47, 0x74, 0x01, 0x39,
	0x7b, 0xda, 0xbe, 0x83, 0xec, 0x5c, 0x40, 0xc1, 0xfe, 0x04, 0x36, 0x69, 0x66, 0xac, 0xed, 0x0f,
	0x50, 0x27, 0x17, 0xd9, 0xfa, 0x36, 0x9e, 0xfd, 0x0e, 0x00, 0x00, 0xff, 0xff, 0x3a, 0x40, 0xf0,
	0x74, 0xa1, 0x02, 0x00, 0x00,
}

func (this *MsgTopup) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*MsgTopup)
	if !ok {
		that2, ok := that.(MsgTopup)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.FromAddress != that1.FromAddress {
		return false
	}
	if this.User != that1.User {
		return false
	}
	if !this.Fee.Equal(that1.Fee) {
		return false
	}
	if !this.TxHash.Equal(that1.TxHash) {
		return false
	}
	if this.LogIndex != that1.LogIndex {
		return false
	}
	if this.BlockNumber != that1.BlockNumber {
		return false
	}
	return true
}
func (this *MsgWithdrawFee) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*MsgWithdrawFee)
	if !ok {
		that2, ok := that.(MsgWithdrawFee)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.FromAddress != that1.FromAddress {
		return false
	}
	if !this.Amount.Equal(that1.Amount) {
		return false
	}
	return true
}
func (m *MsgTopup) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgTopup) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgTopup) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.BlockNumber != 0 {
		i = encodeVarintTopup(dAtA, i, uint64(m.BlockNumber))
		i--
		dAtA[i] = 0x30
	}
	if m.LogIndex != 0 {
		i = encodeVarintTopup(dAtA, i, uint64(m.LogIndex))
		i--
		dAtA[i] = 0x28
	}
	if m.TxHash != nil {
		{
			size, err := m.TxHash.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintTopup(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x22
	}
	{
		size := m.Fee.Size()
		i -= size
		if _, err := m.Fee.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTopup(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x1a
	if len(m.User) > 0 {
		i -= len(m.User)
		copy(dAtA[i:], m.User)
		i = encodeVarintTopup(dAtA, i, uint64(len(m.User)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.FromAddress) > 0 {
		i -= len(m.FromAddress)
		copy(dAtA[i:], m.FromAddress)
		i = encodeVarintTopup(dAtA, i, uint64(len(m.FromAddress)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *MsgWithdrawFee) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgWithdrawFee) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgWithdrawFee) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.Amount.Size()
		i -= size
		if _, err := m.Amount.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintTopup(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
	if len(m.FromAddress) > 0 {
		i -= len(m.FromAddress)
		copy(dAtA[i:], m.FromAddress)
		i = encodeVarintTopup(dAtA, i, uint64(len(m.FromAddress)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintTopup(dAtA []byte, offset int, v uint64) int {
	offset -= sovTopup(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgTopup) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.FromAddress)
	if l > 0 {
		n += 1 + l + sovTopup(uint64(l))
	}
	l = len(m.User)
	if l > 0 {
		n += 1 + l + sovTopup(uint64(l))
	}
	l = m.Fee.Size()
	n += 1 + l + sovTopup(uint64(l))
	if m.TxHash != nil {
		l = m.TxHash.Size()
		n += 1 + l + sovTopup(uint64(l))
	}
	if m.LogIndex != 0 {
		n += 1 + sovTopup(uint64(m.LogIndex))
	}
	if m.BlockNumber != 0 {
		n += 1 + sovTopup(uint64(m.BlockNumber))
	}
	return n
}

func (m *MsgWithdrawFee) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.FromAddress)
	if l > 0 {
		n += 1 + l + sovTopup(uint64(l))
	}
	l = m.Amount.Size()
	n += 1 + l + sovTopup(uint64(l))
	return n
}

func sovTopup(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTopup(x uint64) (n int) {
	return sovTopup(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgTopup) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTopup
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
			return fmt.Errorf("proto: MsgTopup: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgTopup: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FromAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTopup
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTopup
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTopup
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.FromAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field User", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTopup
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTopup
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTopup
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.User = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Fee", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTopup
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTopup
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTopup
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Fee.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TxHash", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTopup
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
				return ErrInvalidLengthTopup
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTopup
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.TxHash == nil {
				m.TxHash = &types.TxHash{}
			}
			if err := m.TxHash.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LogIndex", wireType)
			}
			m.LogIndex = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTopup
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.LogIndex |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field BlockNumber", wireType)
			}
			m.BlockNumber = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTopup
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.BlockNumber |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipTopup(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTopup
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
func (m *MsgWithdrawFee) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTopup
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
			return fmt.Errorf("proto: MsgWithdrawFee: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgWithdrawFee: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FromAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTopup
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTopup
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTopup
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.FromAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Amount", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTopup
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthTopup
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTopup
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Amount.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTopup(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTopup
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
func skipTopup(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTopup
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
					return 0, ErrIntOverflowTopup
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
					return 0, ErrIntOverflowTopup
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
				return 0, ErrInvalidLengthTopup
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTopup
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTopup
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTopup        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTopup          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTopup = fmt.Errorf("proto: unexpected end of group")
)
