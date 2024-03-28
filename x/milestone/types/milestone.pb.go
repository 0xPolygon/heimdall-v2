// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: heimdallv2/milestone/v1/milestone.proto

package types

import (
	fmt "fmt"
	types "github.com/0xPolygon/heimdall-v2/types"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	github_com_cosmos_gogoproto_types "github.com/cosmos/gogoproto/types"
	_ "google.golang.org/protobuf/types/known/durationpb"
	io "io"
	math "math"
	math_bits "math/bits"
	time "time"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf
var _ = time.Kitchen

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// Milestone representes the milestone struct
type Milestone struct {
	Proposer    string             `protobuf:"bytes,1,opt,name=proposer,proto3" json:"proposer,omitempty"`
	StartBlock  uint64             `protobuf:"varint,2,opt,name=start_block,json=startBlock,proto3" json:"start_block,omitempty"`
	EndBlock    uint64             `protobuf:"varint,3,opt,name=end_block,json=endBlock,proto3" json:"end_block,omitempty"`
	Hash        types.HeimdallHash `protobuf:"bytes,4,opt,name=hash,proto3" json:"hash"`
	BorChainID  string             `protobuf:"bytes,5,opt,name=bor_chain_i_d,json=borChainID,proto3" json:"bor_chain_i_d,omitempty"`
	MilestoneID string             `protobuf:"bytes,6,opt,name=milestone_i_d,json=milestoneID,proto3" json:"milestone_i_d,omitempty"`
	TimeStamp   uint64             `protobuf:"varint,7,opt,name=time_stamp,json=timeStamp,proto3" json:"time_stamp,omitempty"`
}

func (m *Milestone) Reset()         { *m = Milestone{} }
func (m *Milestone) String() string { return proto.CompactTextString(m) }
func (*Milestone) ProtoMessage()    {}
func (*Milestone) Descriptor() ([]byte, []int) {
	return fileDescriptor_f3bb318b16edbffd, []int{0}
}
func (m *Milestone) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Milestone) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Milestone.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Milestone) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Milestone.Merge(m, src)
}
func (m *Milestone) XXX_Size() int {
	return m.Size()
}
func (m *Milestone) XXX_DiscardUnknown() {
	xxx_messageInfo_Milestone.DiscardUnknown(m)
}

var xxx_messageInfo_Milestone proto.InternalMessageInfo

func (m *Milestone) GetProposer() string {
	if m != nil {
		return m.Proposer
	}
	return ""
}

func (m *Milestone) GetStartBlock() uint64 {
	if m != nil {
		return m.StartBlock
	}
	return 0
}

func (m *Milestone) GetEndBlock() uint64 {
	if m != nil {
		return m.EndBlock
	}
	return 0
}

func (m *Milestone) GetHash() types.HeimdallHash {
	if m != nil {
		return m.Hash
	}
	return types.HeimdallHash{}
}

func (m *Milestone) GetBorChainID() string {
	if m != nil {
		return m.BorChainID
	}
	return ""
}

func (m *Milestone) GetMilestoneID() string {
	if m != nil {
		return m.MilestoneID
	}
	return ""
}

func (m *Milestone) GetTimeStamp() uint64 {
	if m != nil {
		return m.TimeStamp
	}
	return 0
}

// Params represents the milestone paramters
type Params struct {
	MinMilestoneLength       uint64        `protobuf:"varint,1,opt,name=min_milestone_length,json=minMilestoneLength,proto3" json:"min_milestone_length,omitempty"`
	MilestoneBufferTime      time.Duration `protobuf:"bytes,2,opt,name=milestone_buffer_time,json=milestoneBufferTime,proto3,stdduration" json:"milestone_buffer_time"`
	MilestoneBufferLength    uint64        `protobuf:"varint,3,opt,name=milestone_buffer_length,json=milestoneBufferLength,proto3" json:"milestone_buffer_length,omitempty"`
	MilestoneTxConfirmations uint64        `protobuf:"varint,4,opt,name=milestone_tx_confirmations,json=milestoneTxConfirmations,proto3" json:"milestone_tx_confirmations,omitempty"`
}

func (m *Params) Reset()         { *m = Params{} }
func (m *Params) String() string { return proto.CompactTextString(m) }
func (*Params) ProtoMessage()    {}
func (*Params) Descriptor() ([]byte, []int) {
	return fileDescriptor_f3bb318b16edbffd, []int{1}
}
func (m *Params) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Params) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Params.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Params) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Params.Merge(m, src)
}
func (m *Params) XXX_Size() int {
	return m.Size()
}
func (m *Params) XXX_DiscardUnknown() {
	xxx_messageInfo_Params.DiscardUnknown(m)
}

var xxx_messageInfo_Params proto.InternalMessageInfo

func (m *Params) GetMinMilestoneLength() uint64 {
	if m != nil {
		return m.MinMilestoneLength
	}
	return 0
}

func (m *Params) GetMilestoneBufferTime() time.Duration {
	if m != nil {
		return m.MilestoneBufferTime
	}
	return 0
}

func (m *Params) GetMilestoneBufferLength() uint64 {
	if m != nil {
		return m.MilestoneBufferLength
	}
	return 0
}

func (m *Params) GetMilestoneTxConfirmations() uint64 {
	if m != nil {
		return m.MilestoneTxConfirmations
	}
	return 0
}

func init() {
	proto.RegisterType((*Milestone)(nil), "heimdallv2.milestone.v1.Milestone")
	proto.RegisterType((*Params)(nil), "heimdallv2.milestone.v1.Params")
}

func init() {
	proto.RegisterFile("heimdallv2/milestone/v1/milestone.proto", fileDescriptor_f3bb318b16edbffd)
}

var fileDescriptor_f3bb318b16edbffd = []byte{
	// 556 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x93, 0xb1, 0x6f, 0xd3, 0x4e,
	0x14, 0xc7, 0x73, 0xfd, 0xa5, 0xf9, 0x25, 0x17, 0x65, 0xe0, 0x48, 0x55, 0x37, 0x08, 0x27, 0x8a,
	0x10, 0x04, 0xa4, 0xda, 0x34, 0x0c, 0x08, 0xa4, 0x0e, 0x38, 0x19, 0x5a, 0xa9, 0x48, 0x15, 0xed,
	0x84, 0x90, 0x2c, 0x3b, 0xbe, 0xd8, 0x27, 0x7c, 0x77, 0xd1, 0xdd, 0x25, 0x4a, 0xff, 0x03, 0x46,
	0x46, 0xc6, 0x8e, 0x2c, 0x48, 0x0c, 0xfc, 0x11, 0x1d, 0x2b, 0x06, 0xc4, 0x04, 0x28, 0x19, 0xe0,
	0xcf, 0x40, 0x3e, 0x3b, 0xb6, 0x81, 0x25, 0xca, 0xdd, 0xf7, 0xf3, 0xde, 0xfb, 0xde, 0x7b, 0xcf,
	0xf0, 0x5e, 0x84, 0x09, 0x0d, 0xbc, 0x38, 0x5e, 0x0c, 0x6d, 0x4a, 0x62, 0x2c, 0x15, 0x67, 0xd8,
	0x5e, 0x1c, 0x14, 0x07, 0x6b, 0x26, 0xb8, 0xe2, 0x68, 0xb7, 0x00, 0xad, 0x42, 0x5b, 0x1c, 0x74,
	0xda, 0x21, 0x0f, 0xb9, 0x66, 0xec, 0xe4, 0x5f, 0x8a, 0x77, 0xcc, 0x90, 0xf3, 0x30, 0xc6, 0xb6,
	0x3e, 0xf9, 0xf3, 0xa9, 0x1d, 0xcc, 0x85, 0xa7, 0x08, 0x67, 0x99, 0xbe, 0x37, 0xe1, 0x92, 0x72,
	0xe9, 0xa6, 0x81, 0xe9, 0x21, 0x93, 0x6e, 0x78, 0x94, 0x30, 0x6e, 0xeb, 0xdf, 0xec, 0xea, 0x56,
	0xc9, 0xa5, 0xba, 0x98, 0x61, 0x69, 0x47, 0x9e, 0x8c, 0x52, 0xb1, 0xff, 0x65, 0x0b, 0x36, 0x9e,
	0x6f, 0x1c, 0xa1, 0x27, 0xb0, 0x3e, 0x13, 0x7c, 0xc6, 0x25, 0x16, 0x06, 0xe8, 0x81, 0x41, 0xc3,
	0xb9, 0xfd, 0xf9, 0xd3, 0x7e, 0x3b, 0xab, 0xf0, 0x2c, 0x08, 0x04, 0x96, 0xf2, 0x4c, 0x09, 0xc2,
	0xc2, 0xf7, 0x3f, 0x3f, 0x3e, 0x00, 0x2f, 0x72, 0x1c, 0xdd, 0x85, 0x4d, 0xa9, 0x3c, 0xa1, 0x5c,
	0x3f, 0xe6, 0x93, 0xd7, 0xc6, 0x56, 0x0f, 0x0c, 0xaa, 0xce, 0x76, 0x4a, 0x41, 0xad, 0x38, 0x89,
	0x80, 0xfa, 0xb0, 0x81, 0x59, 0x90, 0x51, 0xff, 0x95, 0xa9, 0x3a, 0x66, 0x41, 0xca, 0x1c, 0xc2,
	0x6a, 0x62, 0xd1, 0xa8, 0xf6, 0xc0, 0xa0, 0x39, 0x34, 0xad, 0x52, 0xf7, 0xf4, 0x03, 0xac, 0xa3,
	0xec, 0xe2, 0xc8, 0x93, 0x91, 0xd3, 0xb8, 0xfa, 0xd6, 0xad, 0xa4, 0x29, 0x74, 0x18, 0x1a, 0xc0,
	0x96, 0xcf, 0x85, 0x3b, 0x89, 0x3c, 0xc2, 0x5c, 0xe2, 0x06, 0xc6, 0xb6, 0x7e, 0xca, 0xc6, 0x8c,
	0xcf, 0xc5, 0x28, 0x91, 0x8e, 0xc7, 0xe8, 0x3e, 0x6c, 0xe5, 0xe3, 0xd0, 0x64, 0xad, 0x4c, 0x36,
	0x73, 0xed, 0x78, 0x8c, 0xee, 0x40, 0xa8, 0x08, 0xc5, 0xae, 0x54, 0x1e, 0x9d, 0x19, 0xff, 0x97,
	0x8d, 0x37, 0x12, 0xe1, 0x2c, 0xb9, 0x7f, 0x5a, 0x7f, 0x73, 0xd9, 0x05, 0xbf, 0x2e, 0xbb, 0xa0,
	0xff, 0x61, 0x0b, 0xd6, 0x4e, 0x3d, 0xe1, 0x51, 0x89, 0x1e, 0xc3, 0x36, 0x25, 0xcc, 0x2d, 0x2a,
	0xc5, 0x98, 0x85, 0x2a, 0xd2, 0x1d, 0xce, 0x93, 0x20, 0x4a, 0x58, 0x3e, 0x88, 0x13, 0x0d, 0xa0,
	0x57, 0x70, 0xa7, 0x08, 0xf2, 0xe7, 0xd3, 0x29, 0x16, 0x6e, 0x52, 0x4b, 0x77, 0xb7, 0x39, 0xdc,
	0xb3, 0xd2, 0x3d, 0xb1, 0x36, 0x7b, 0x62, 0x8d, 0xb3, 0x3d, 0x71, 0x5a, 0x49, 0x4f, 0xde, 0x7d,
	0xef, 0x82, 0x34, 0xf9, 0xcd, 0x3c, 0x8d, 0xa3, 0xb3, 0x9c, 0x13, 0x8a, 0xd1, 0x21, 0xdc, 0xfd,
	0x27, 0x7b, 0xe6, 0xec, 0x8f, 0xb9, 0xec, 0xfc, 0x15, 0x9c, 0x99, 0x1b, 0xc1, 0x4e, 0x11, 0xae,
	0x96, 0xee, 0x84, 0xb3, 0x29, 0x11, 0x54, 0xd7, 0x97, 0x7a, 0x74, 0x79, 0x06, 0x23, 0x07, 0xcf,
	0x97, 0xa3, 0x32, 0x56, 0xf4, 0xcb, 0x39, 0xb9, 0x5a, 0x99, 0xe0, 0x7a, 0x65, 0x82, 0x1f, 0x2b,
	0x13, 0xbc, 0x5d, 0x9b, 0x95, 0xeb, 0xb5, 0x59, 0xf9, 0xba, 0x36, 0x2b, 0x2f, 0x87, 0x21, 0x51,
	0xd1, 0xdc, 0xb7, 0x26, 0x9c, 0xda, 0x0f, 0x97, 0xa7, 0x3c, 0xbe, 0x08, 0x39, 0xb3, 0x37, 0x3b,
	0xb1, 0xbf, 0x18, 0xda, 0xcb, 0xd2, 0xd7, 0xa7, 0x17, 0xc4, 0xaf, 0xe9, 0x96, 0x3c, 0xfa, 0x1d,
	0x00, 0x00, 0xff, 0xff, 0xdd, 0x64, 0x9a, 0xd6, 0xa2, 0x03, 0x00, 0x00,
}

func (this *Milestone) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Milestone)
	if !ok {
		that2, ok := that.(Milestone)
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
	if this.Proposer != that1.Proposer {
		return false
	}
	if this.StartBlock != that1.StartBlock {
		return false
	}
	if this.EndBlock != that1.EndBlock {
		return false
	}
	if !this.Hash.Equal(&that1.Hash) {
		return false
	}
	if this.BorChainID != that1.BorChainID {
		return false
	}
	if this.MilestoneID != that1.MilestoneID {
		return false
	}
	if this.TimeStamp != that1.TimeStamp {
		return false
	}
	return true
}
func (this *Params) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*Params)
	if !ok {
		that2, ok := that.(Params)
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
	if this.MinMilestoneLength != that1.MinMilestoneLength {
		return false
	}
	if this.MilestoneBufferTime != that1.MilestoneBufferTime {
		return false
	}
	if this.MilestoneBufferLength != that1.MilestoneBufferLength {
		return false
	}
	if this.MilestoneTxConfirmations != that1.MilestoneTxConfirmations {
		return false
	}
	return true
}
func (m *Milestone) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Milestone) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Milestone) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.TimeStamp != 0 {
		i = encodeVarintMilestone(dAtA, i, uint64(m.TimeStamp))
		i--
		dAtA[i] = 0x38
	}
	if len(m.MilestoneID) > 0 {
		i -= len(m.MilestoneID)
		copy(dAtA[i:], m.MilestoneID)
		i = encodeVarintMilestone(dAtA, i, uint64(len(m.MilestoneID)))
		i--
		dAtA[i] = 0x32
	}
	if len(m.BorChainID) > 0 {
		i -= len(m.BorChainID)
		copy(dAtA[i:], m.BorChainID)
		i = encodeVarintMilestone(dAtA, i, uint64(len(m.BorChainID)))
		i--
		dAtA[i] = 0x2a
	}
	{
		size, err := m.Hash.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintMilestone(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x22
	if m.EndBlock != 0 {
		i = encodeVarintMilestone(dAtA, i, uint64(m.EndBlock))
		i--
		dAtA[i] = 0x18
	}
	if m.StartBlock != 0 {
		i = encodeVarintMilestone(dAtA, i, uint64(m.StartBlock))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Proposer) > 0 {
		i -= len(m.Proposer)
		copy(dAtA[i:], m.Proposer)
		i = encodeVarintMilestone(dAtA, i, uint64(len(m.Proposer)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *Params) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Params) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Params) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.MilestoneTxConfirmations != 0 {
		i = encodeVarintMilestone(dAtA, i, uint64(m.MilestoneTxConfirmations))
		i--
		dAtA[i] = 0x20
	}
	if m.MilestoneBufferLength != 0 {
		i = encodeVarintMilestone(dAtA, i, uint64(m.MilestoneBufferLength))
		i--
		dAtA[i] = 0x18
	}
	n2, err2 := github_com_cosmos_gogoproto_types.StdDurationMarshalTo(m.MilestoneBufferTime, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.MilestoneBufferTime):])
	if err2 != nil {
		return 0, err2
	}
	i -= n2
	i = encodeVarintMilestone(dAtA, i, uint64(n2))
	i--
	dAtA[i] = 0x12
	if m.MinMilestoneLength != 0 {
		i = encodeVarintMilestone(dAtA, i, uint64(m.MinMilestoneLength))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintMilestone(dAtA []byte, offset int, v uint64) int {
	offset -= sovMilestone(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Milestone) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Proposer)
	if l > 0 {
		n += 1 + l + sovMilestone(uint64(l))
	}
	if m.StartBlock != 0 {
		n += 1 + sovMilestone(uint64(m.StartBlock))
	}
	if m.EndBlock != 0 {
		n += 1 + sovMilestone(uint64(m.EndBlock))
	}
	l = m.Hash.Size()
	n += 1 + l + sovMilestone(uint64(l))
	l = len(m.BorChainID)
	if l > 0 {
		n += 1 + l + sovMilestone(uint64(l))
	}
	l = len(m.MilestoneID)
	if l > 0 {
		n += 1 + l + sovMilestone(uint64(l))
	}
	if m.TimeStamp != 0 {
		n += 1 + sovMilestone(uint64(m.TimeStamp))
	}
	return n
}

func (m *Params) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.MinMilestoneLength != 0 {
		n += 1 + sovMilestone(uint64(m.MinMilestoneLength))
	}
	l = github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.MilestoneBufferTime)
	n += 1 + l + sovMilestone(uint64(l))
	if m.MilestoneBufferLength != 0 {
		n += 1 + sovMilestone(uint64(m.MilestoneBufferLength))
	}
	if m.MilestoneTxConfirmations != 0 {
		n += 1 + sovMilestone(uint64(m.MilestoneTxConfirmations))
	}
	return n
}

func sovMilestone(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozMilestone(x uint64) (n int) {
	return sovMilestone(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Milestone) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMilestone
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
			return fmt.Errorf("proto: Milestone: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Milestone: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Proposer", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMilestone
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
				return ErrInvalidLengthMilestone
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMilestone
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Proposer = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field StartBlock", wireType)
			}
			m.StartBlock = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMilestone
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.StartBlock |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field EndBlock", wireType)
			}
			m.EndBlock = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMilestone
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.EndBlock |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Hash", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMilestone
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
				return ErrInvalidLengthMilestone
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMilestone
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Hash.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BorChainID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMilestone
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
				return ErrInvalidLengthMilestone
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMilestone
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.BorChainID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MilestoneID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMilestone
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
				return ErrInvalidLengthMilestone
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthMilestone
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.MilestoneID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TimeStamp", wireType)
			}
			m.TimeStamp = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMilestone
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TimeStamp |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipMilestone(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMilestone
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
func (m *Params) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMilestone
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
			return fmt.Errorf("proto: Params: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Params: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MinMilestoneLength", wireType)
			}
			m.MinMilestoneLength = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMilestone
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MinMilestoneLength |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MilestoneBufferTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMilestone
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
				return ErrInvalidLengthMilestone
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMilestone
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_cosmos_gogoproto_types.StdDurationUnmarshal(&m.MilestoneBufferTime, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MilestoneBufferLength", wireType)
			}
			m.MilestoneBufferLength = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMilestone
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MilestoneBufferLength |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MilestoneTxConfirmations", wireType)
			}
			m.MilestoneTxConfirmations = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMilestone
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MilestoneTxConfirmations |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipMilestone(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMilestone
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
func skipMilestone(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowMilestone
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
					return 0, ErrIntOverflowMilestone
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
					return 0, ErrIntOverflowMilestone
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
				return 0, ErrInvalidLengthMilestone
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupMilestone
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthMilestone
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthMilestone        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowMilestone          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupMilestone = fmt.Errorf("proto: unexpected end of group")
)
