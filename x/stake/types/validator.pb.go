// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: heimdallv2/stake/validator.proto

package types

import (
	bytes "bytes"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
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

// Validators define the validator structure
type Validator struct {
	ValId            uint64 `protobuf:"varint,1,opt,name=val_id,json=valId,proto3" json:"val_id,omitempty"`
	StartEpoch       uint64 `protobuf:"varint,2,opt,name=start_epoch,json=startEpoch,proto3" json:"start_epoch,omitempty"`
	EndEpoch         uint64 `protobuf:"varint,3,opt,name=end_epoch,json=endEpoch,proto3" json:"end_epoch,omitempty"`
	Nonce            uint64 `protobuf:"varint,4,opt,name=nonce,proto3" json:"nonce,omitempty"`
	VotingPower      int64  `protobuf:"varint,5,opt,name=voting_power,json=votingPower,proto3" json:"voting_power,omitempty"`
	PubKey           []byte `protobuf:"bytes,6,opt,name=pub_key,json=pubKey,proto3" json:"pub_key,omitempty"`
	Signer           string `protobuf:"bytes,7,opt,name=signer,proto3" json:"signer,omitempty"`
	LastUpdated      string `protobuf:"bytes,8,opt,name=last_updated,json=lastUpdated,proto3" json:"last_updated,omitempty"`
	Jailed           bool   `protobuf:"varint,9,opt,name=jailed,proto3" json:"jailed,omitempty"`
	ProposerPriority int64  `protobuf:"varint,10,opt,name=proposer_priority,json=proposerPriority,proto3" json:"proposer_priority,omitempty"`
}

func (m *Validator) Reset()         { *m = Validator{} }
func (m *Validator) String() string { return proto.CompactTextString(m) }
func (*Validator) ProtoMessage()    {}
func (*Validator) Descriptor() ([]byte, []int) {
	return fileDescriptor_a505b479e81213bc, []int{0}
}
func (m *Validator) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Validator) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Validator.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Validator) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Validator.Merge(m, src)
}
func (m *Validator) XXX_Size() int {
	return m.Size()
}
func (m *Validator) XXX_DiscardUnknown() {
	xxx_messageInfo_Validator.DiscardUnknown(m)
}

var xxx_messageInfo_Validator proto.InternalMessageInfo

func (m *Validator) GetValId() uint64 {
	if m != nil {
		return m.ValId
	}
	return 0
}

func (m *Validator) GetStartEpoch() uint64 {
	if m != nil {
		return m.StartEpoch
	}
	return 0
}

func (m *Validator) GetEndEpoch() uint64 {
	if m != nil {
		return m.EndEpoch
	}
	return 0
}

func (m *Validator) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *Validator) GetVotingPower() int64 {
	if m != nil {
		return m.VotingPower
	}
	return 0
}

func (m *Validator) GetPubKey() []byte {
	if m != nil {
		return m.PubKey
	}
	return nil
}

func (m *Validator) GetSigner() string {
	if m != nil {
		return m.Signer
	}
	return ""
}

func (m *Validator) GetLastUpdated() string {
	if m != nil {
		return m.LastUpdated
	}
	return ""
}

func (m *Validator) GetJailed() bool {
	if m != nil {
		return m.Jailed
	}
	return false
}

func (m *Validator) GetProposerPriority() int64 {
	if m != nil {
		return m.ProposerPriority
	}
	return 0
}

// ValidatorSet defines the set of validator
type ValidatorSet struct {
	Validators []*Validator `protobuf:"bytes,1,rep,name=validators,proto3" json:"validators,omitempty"`
	Proposer   *Validator   `protobuf:"bytes,2,opt,name=proposer,proto3" json:"proposer,omitempty"`
	// total voting power denotes the total power of all the active validators in
	// the validator set
	TotalVotingPower int64 `protobuf:"varint,3,opt,name=total_voting_power,json=totalVotingPower,proto3" json:"total_voting_power,omitempty"`
}

func (m *ValidatorSet) Reset()         { *m = ValidatorSet{} }
func (m *ValidatorSet) String() string { return proto.CompactTextString(m) }
func (*ValidatorSet) ProtoMessage()    {}
func (*ValidatorSet) Descriptor() ([]byte, []int) {
	return fileDescriptor_a505b479e81213bc, []int{1}
}
func (m *ValidatorSet) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ValidatorSet) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ValidatorSet.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ValidatorSet) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ValidatorSet.Merge(m, src)
}
func (m *ValidatorSet) XXX_Size() int {
	return m.Size()
}
func (m *ValidatorSet) XXX_DiscardUnknown() {
	xxx_messageInfo_ValidatorSet.DiscardUnknown(m)
}

var xxx_messageInfo_ValidatorSet proto.InternalMessageInfo

func init() {
	proto.RegisterType((*Validator)(nil), "heimdallv2.stake.Validator")
	proto.RegisterType((*ValidatorSet)(nil), "heimdallv2.stake.ValidatorSet")
}

func init() { proto.RegisterFile("heimdallv2/stake/validator.proto", fileDescriptor_a505b479e81213bc) }

var fileDescriptor_a505b479e81213bc = []byte{
	// 509 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x93, 0x3f, 0x6f, 0xd3, 0x40,
	0x14, 0xc0, 0x73, 0xa4, 0x71, 0x93, 0x4b, 0x86, 0xf6, 0x54, 0x89, 0xa3, 0xa5, 0x8e, 0x95, 0x01,
	0x59, 0x48, 0xb5, 0x51, 0x2a, 0x16, 0x36, 0x22, 0x81, 0x40, 0x2c, 0x51, 0x2b, 0x3a, 0xb0, 0x58,
	0x97, 0xdc, 0xc9, 0x39, 0xea, 0xf8, 0x4e, 0x77, 0x17, 0xd3, 0x7c, 0x03, 0x46, 0x24, 0xbe, 0x40,
	0x47, 0x46, 0x06, 0x3e, 0x44, 0xc7, 0x8a, 0x09, 0x16, 0x84, 0x92, 0x01, 0x3e, 0x06, 0xf2, 0xd9,
	0xce, 0x1f, 0xa6, 0x2e, 0x96, 0xfd, 0x7e, 0xbf, 0x77, 0xcf, 0xcf, 0xef, 0x19, 0x7a, 0x13, 0xc6,
	0xa7, 0x94, 0x24, 0x49, 0xd6, 0x0f, 0xb5, 0x21, 0x97, 0x2c, 0xcc, 0x48, 0xc2, 0x29, 0x31, 0x42,
	0x05, 0x52, 0x09, 0x23, 0xd0, 0xde, 0xda, 0x08, 0xac, 0x71, 0x78, 0x10, 0x8b, 0x58, 0x58, 0x18,
	0xe6, 0x77, 0x85, 0x77, 0xf8, 0x60, 0x2c, 0xf4, 0x54, 0xe8, 0xa8, 0x00, 0xc5, 0x43, 0x89, 0xf6,
	0xc9, 0x94, 0xa7, 0x22, 0xb4, 0xd7, 0x22, 0xd4, 0xfb, 0x5c, 0x87, 0xad, 0x8b, 0xaa, 0x12, 0x7a,
	0x08, 0x9d, 0x8c, 0x24, 0x11, 0xa7, 0x18, 0x78, 0xc0, 0xdf, 0x19, 0x34, 0xbe, 0xfc, 0xf9, 0xfa,
	0x18, 0x9c, 0x35, 0x32, 0x92, 0xbc, 0xa6, 0xe8, 0x11, 0x6c, 0x6b, 0x43, 0x94, 0x89, 0x98, 0x14,
	0xe3, 0x09, 0xbe, 0xb7, 0xa9, 0x40, 0x4b, 0x5e, 0xe4, 0x00, 0xf5, 0x60, 0x8b, 0xa5, 0xb4, 0xb4,
	0xea, 0x9b, 0x56, 0x93, 0xa5, 0xb4, 0x70, 0x8e, 0x60, 0x23, 0x15, 0xe9, 0x98, 0xe1, 0x9d, 0xad,
	0x42, 0x36, 0x86, 0x7c, 0xd8, 0xc9, 0x84, 0xe1, 0x69, 0x1c, 0x49, 0xf1, 0x81, 0x29, 0xdc, 0xf0,
	0x80, 0x5f, 0xaf, 0x9c, 0x76, 0x81, 0x86, 0x39, 0x41, 0xf7, 0xe1, 0xae, 0x9c, 0x8d, 0xa2, 0x4b,
	0x36, 0xc7, 0x8e, 0x07, 0xfc, 0xce, 0x99, 0x23, 0x67, 0xa3, 0x37, 0x6c, 0x8e, 0x9e, 0x42, 0x47,
	0xf3, 0x38, 0x65, 0x0a, 0xef, 0x7a, 0xc0, 0x6f, 0x0d, 0x8e, 0xbf, 0x7f, 0x3b, 0x39, 0x28, 0x3f,
	0xc6, 0x73, 0x4a, 0x15, 0xd3, 0xfa, 0xdc, 0x28, 0x9e, 0xc6, 0xc5, 0xa1, 0xa5, 0x9c, 0x57, 0x4e,
	0x88, 0x36, 0xd1, 0x4c, 0x52, 0x62, 0x18, 0xc5, 0x4d, 0x9b, 0x5c, 0x55, 0xce, 0xd1, 0xdb, 0x82,
	0xa0, 0x63, 0xe8, 0xbc, 0x27, 0x3c, 0x61, 0x14, 0xb7, 0x3c, 0xe0, 0x37, 0x2b, 0xa7, 0x0c, 0xa2,
	0x3e, 0xdc, 0x97, 0x4a, 0x48, 0xa1, 0x99, 0x8a, 0xa4, 0xe2, 0x42, 0x71, 0x33, 0xc7, 0x70, 0xb3,
	0x8f, 0xbd, 0x8a, 0x0f, 0x4b, 0xfc, 0xac, 0xf9, 0xf1, 0xba, 0x0b, 0xfe, 0x5e, 0x77, 0x41, 0xef,
	0x27, 0x80, 0x9d, 0xd5, 0x54, 0xce, 0x99, 0x41, 0x2f, 0x21, 0x5c, 0xed, 0x83, 0xc6, 0xc0, 0xab,
	0xfb, 0xed, 0xfe, 0x51, 0xf0, 0xff, 0x46, 0x04, 0xab, 0x9c, 0x41, 0xeb, 0xe6, 0x57, 0x17, 0x94,
	0xa3, 0x59, 0x67, 0xa2, 0x01, 0x6c, 0x56, 0x65, 0xed, 0xfc, 0xee, 0x7e, 0xca, 0x2a, 0x0f, 0x9d,
	0x42, 0x64, 0x84, 0x21, 0x49, 0xb4, 0x35, 0xa3, 0xfa, 0x56, 0x6f, 0x56, 0xb8, 0x58, 0x0f, 0xca,
	0xf6, 0x56, 0xcb, 0x7b, 0x1b, 0xbc, 0xba, 0x59, 0xb8, 0xe0, 0x76, 0xe1, 0x82, 0xdf, 0x0b, 0x17,
	0x7c, 0x5a, 0xba, 0xb5, 0xdb, 0xa5, 0x5b, 0xfb, 0xb1, 0x74, 0x6b, 0xef, 0x82, 0x98, 0x9b, 0xc9,
	0x6c, 0x14, 0x8c, 0xc5, 0x34, 0x7c, 0x72, 0x35, 0x14, 0xc9, 0x3c, 0x16, 0x69, 0x58, 0xbd, 0xde,
	0x49, 0xd6, 0x0f, 0xaf, 0xca, 0x7f, 0xc3, 0xcc, 0x25, 0xd3, 0x23, 0xc7, 0xae, 0xf0, 0xe9, 0xbf,
	0x00, 0x00, 0x00, 0xff, 0xff, 0x15, 0xaf, 0x4d, 0x10, 0x3c, 0x03, 0x00, 0x00,
}

func (this *Validator) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Validator)
	if !ok {
		that2, ok := that.(Validator)
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
	if this.ValId != that1.ValId {
		return false
	}
	if this.StartEpoch != that1.StartEpoch {
		return false
	}
	if this.EndEpoch != that1.EndEpoch {
		return false
	}
	if this.Nonce != that1.Nonce {
		return false
	}
	if this.VotingPower != that1.VotingPower {
		return false
	}
	if !bytes.Equal(this.PubKey, that1.PubKey) {
		return false
	}
	if this.Signer != that1.Signer {
		return false
	}
	if this.LastUpdated != that1.LastUpdated {
		return false
	}
	if this.Jailed != that1.Jailed {
		return false
	}
	if this.ProposerPriority != that1.ProposerPriority {
		return false
	}
	return true
}
func (this *ValidatorSet) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*ValidatorSet)
	if !ok {
		that2, ok := that.(ValidatorSet)
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
	if len(this.Validators) != len(that1.Validators) {
		return false
	}
	for i := range this.Validators {
		if !this.Validators[i].Equal(that1.Validators[i]) {
			return false
		}
	}
	if !this.Proposer.Equal(that1.Proposer) {
		return false
	}
	if this.TotalVotingPower != that1.TotalVotingPower {
		return false
	}
	return true
}
func (m *Validator) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Validator) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Validator) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.ProposerPriority != 0 {
		i = encodeVarintValidator(dAtA, i, uint64(m.ProposerPriority))
		i--
		dAtA[i] = 0x50
	}
	if m.Jailed {
		i--
		if m.Jailed {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x48
	}
	if len(m.LastUpdated) > 0 {
		i -= len(m.LastUpdated)
		copy(dAtA[i:], m.LastUpdated)
		i = encodeVarintValidator(dAtA, i, uint64(len(m.LastUpdated)))
		i--
		dAtA[i] = 0x42
	}
	if len(m.Signer) > 0 {
		i -= len(m.Signer)
		copy(dAtA[i:], m.Signer)
		i = encodeVarintValidator(dAtA, i, uint64(len(m.Signer)))
		i--
		dAtA[i] = 0x3a
	}
	if len(m.PubKey) > 0 {
		i -= len(m.PubKey)
		copy(dAtA[i:], m.PubKey)
		i = encodeVarintValidator(dAtA, i, uint64(len(m.PubKey)))
		i--
		dAtA[i] = 0x32
	}
	if m.VotingPower != 0 {
		i = encodeVarintValidator(dAtA, i, uint64(m.VotingPower))
		i--
		dAtA[i] = 0x28
	}
	if m.Nonce != 0 {
		i = encodeVarintValidator(dAtA, i, uint64(m.Nonce))
		i--
		dAtA[i] = 0x20
	}
	if m.EndEpoch != 0 {
		i = encodeVarintValidator(dAtA, i, uint64(m.EndEpoch))
		i--
		dAtA[i] = 0x18
	}
	if m.StartEpoch != 0 {
		i = encodeVarintValidator(dAtA, i, uint64(m.StartEpoch))
		i--
		dAtA[i] = 0x10
	}
	if m.ValId != 0 {
		i = encodeVarintValidator(dAtA, i, uint64(m.ValId))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *ValidatorSet) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ValidatorSet) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ValidatorSet) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.TotalVotingPower != 0 {
		i = encodeVarintValidator(dAtA, i, uint64(m.TotalVotingPower))
		i--
		dAtA[i] = 0x18
	}
	if m.Proposer != nil {
		{
			size, err := m.Proposer.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintValidator(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if len(m.Validators) > 0 {
		for iNdEx := len(m.Validators) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Validators[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintValidator(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintValidator(dAtA []byte, offset int, v uint64) int {
	offset -= sovValidator(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Validator) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.ValId != 0 {
		n += 1 + sovValidator(uint64(m.ValId))
	}
	if m.StartEpoch != 0 {
		n += 1 + sovValidator(uint64(m.StartEpoch))
	}
	if m.EndEpoch != 0 {
		n += 1 + sovValidator(uint64(m.EndEpoch))
	}
	if m.Nonce != 0 {
		n += 1 + sovValidator(uint64(m.Nonce))
	}
	if m.VotingPower != 0 {
		n += 1 + sovValidator(uint64(m.VotingPower))
	}
	l = len(m.PubKey)
	if l > 0 {
		n += 1 + l + sovValidator(uint64(l))
	}
	l = len(m.Signer)
	if l > 0 {
		n += 1 + l + sovValidator(uint64(l))
	}
	l = len(m.LastUpdated)
	if l > 0 {
		n += 1 + l + sovValidator(uint64(l))
	}
	if m.Jailed {
		n += 2
	}
	if m.ProposerPriority != 0 {
		n += 1 + sovValidator(uint64(m.ProposerPriority))
	}
	return n
}

func (m *ValidatorSet) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Validators) > 0 {
		for _, e := range m.Validators {
			l = e.Size()
			n += 1 + l + sovValidator(uint64(l))
		}
	}
	if m.Proposer != nil {
		l = m.Proposer.Size()
		n += 1 + l + sovValidator(uint64(l))
	}
	if m.TotalVotingPower != 0 {
		n += 1 + sovValidator(uint64(m.TotalVotingPower))
	}
	return n
}

func sovValidator(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozValidator(x uint64) (n int) {
	return sovValidator(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Validator) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowValidator
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
			return fmt.Errorf("proto: Validator: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Validator: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ValId", wireType)
			}
			m.ValId = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowValidator
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ValId |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field StartEpoch", wireType)
			}
			m.StartEpoch = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowValidator
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.StartEpoch |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field EndEpoch", wireType)
			}
			m.EndEpoch = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowValidator
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.EndEpoch |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nonce", wireType)
			}
			m.Nonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowValidator
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Nonce |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field VotingPower", wireType)
			}
			m.VotingPower = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowValidator
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.VotingPower |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PubKey", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowValidator
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
				return ErrInvalidLengthValidator
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthValidator
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.PubKey = append(m.PubKey[:0], dAtA[iNdEx:postIndex]...)
			if m.PubKey == nil {
				m.PubKey = []byte{}
			}
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Signer", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowValidator
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
				return ErrInvalidLengthValidator
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthValidator
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Signer = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastUpdated", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowValidator
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
				return ErrInvalidLengthValidator
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthValidator
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.LastUpdated = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 9:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Jailed", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowValidator
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Jailed = bool(v != 0)
		case 10:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProposerPriority", wireType)
			}
			m.ProposerPriority = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowValidator
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ProposerPriority |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipValidator(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthValidator
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
func (m *ValidatorSet) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowValidator
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
			return fmt.Errorf("proto: ValidatorSet: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ValidatorSet: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Validators", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowValidator
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
				return ErrInvalidLengthValidator
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthValidator
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Validators = append(m.Validators, &Validator{})
			if err := m.Validators[len(m.Validators)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Proposer", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowValidator
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
				return ErrInvalidLengthValidator
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthValidator
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Proposer == nil {
				m.Proposer = &Validator{}
			}
			if err := m.Proposer.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TotalVotingPower", wireType)
			}
			m.TotalVotingPower = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowValidator
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TotalVotingPower |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipValidator(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthValidator
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
func skipValidator(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowValidator
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
					return 0, ErrIntOverflowValidator
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
					return 0, ErrIntOverflowValidator
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
				return 0, ErrInvalidLengthValidator
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupValidator
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthValidator
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthValidator        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowValidator          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupValidator = fmt.Errorf("proto: unexpected end of group")
)
