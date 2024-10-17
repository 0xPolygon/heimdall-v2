// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: heimdallv2/bor/bor.proto

package types

import (
	fmt "fmt"
	types "github.com/0xPolygon/heimdall-v2/x/stake/types"
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

// Span represents a range of block numbers in bor
type Span struct {
	Id                uint64             `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	StartBlock        uint64             `protobuf:"varint,2,opt,name=start_block,json=startBlock,proto3" json:"start_block,omitempty"`
	EndBlock          uint64             `protobuf:"varint,3,opt,name=end_block,json=endBlock,proto3" json:"end_block,omitempty"`
	ValidatorSet      types.ValidatorSet `protobuf:"bytes,4,opt,name=validator_set,json=validatorSet,proto3" json:"validator_set"`
	SelectedProducers []types.Validator  `protobuf:"bytes,5,rep,name=selected_producers,json=selectedProducers,proto3" json:"selected_producers"`
	ChainId           string             `protobuf:"bytes,6,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
}

func (m *Span) Reset()         { *m = Span{} }
func (m *Span) String() string { return proto.CompactTextString(m) }
func (*Span) ProtoMessage()    {}
func (*Span) Descriptor() ([]byte, []int) {
	return fileDescriptor_ed6109dea23871eb, []int{0}
}
func (m *Span) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Span) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Span.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Span) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Span.Merge(m, src)
}
func (m *Span) XXX_Size() int {
	return m.Size()
}
func (m *Span) XXX_DiscardUnknown() {
	xxx_messageInfo_Span.DiscardUnknown(m)
}

var xxx_messageInfo_Span proto.InternalMessageInfo

func (m *Span) GetId() uint64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *Span) GetStartBlock() uint64 {
	if m != nil {
		return m.StartBlock
	}
	return 0
}

func (m *Span) GetEndBlock() uint64 {
	if m != nil {
		return m.EndBlock
	}
	return 0
}

func (m *Span) GetValidatorSet() types.ValidatorSet {
	if m != nil {
		return m.ValidatorSet
	}
	return types.ValidatorSet{}
}

func (m *Span) GetSelectedProducers() []types.Validator {
	if m != nil {
		return m.SelectedProducers
	}
	return nil
}

func (m *Span) GetChainId() string {
	if m != nil {
		return m.ChainId
	}
	return ""
}

// Params represents the parameters for the bor module
type Params struct {
	SprintDuration uint64 `protobuf:"varint,1,opt,name=sprint_duration,json=sprintDuration,proto3" json:"sprint_duration,omitempty"`
	SpanDuration   uint64 `protobuf:"varint,2,opt,name=span_duration,json=spanDuration,proto3" json:"span_duration,omitempty"`
	ProducerCount  uint64 `protobuf:"varint,3,opt,name=producer_count,json=producerCount,proto3" json:"producer_count,omitempty"`
}

func (m *Params) Reset()         { *m = Params{} }
func (m *Params) String() string { return proto.CompactTextString(m) }
func (*Params) ProtoMessage()    {}
func (*Params) Descriptor() ([]byte, []int) {
	return fileDescriptor_ed6109dea23871eb, []int{1}
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

func (m *Params) GetSprintDuration() uint64 {
	if m != nil {
		return m.SprintDuration
	}
	return 0
}

func (m *Params) GetSpanDuration() uint64 {
	if m != nil {
		return m.SpanDuration
	}
	return 0
}

func (m *Params) GetProducerCount() uint64 {
	if m != nil {
		return m.ProducerCount
	}
	return 0
}

func init() {
	proto.RegisterType((*Span)(nil), "heimdallv2.bor.Span")
	proto.RegisterType((*Params)(nil), "heimdallv2.bor.Params")
}

func init() { proto.RegisterFile("heimdallv2/bor/bor.proto", fileDescriptor_ed6109dea23871eb) }

var fileDescriptor_ed6109dea23871eb = []byte{
	// 423 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x92, 0x31, 0x6f, 0xd4, 0x30,
	0x14, 0xc7, 0xcf, 0x69, 0x7a, 0xf4, 0x7c, 0xbd, 0x43, 0xb5, 0x40, 0x8a, 0x0e, 0x29, 0x8d, 0x6e,
	0x40, 0x51, 0x55, 0x12, 0x14, 0x36, 0xc6, 0x80, 0x90, 0x58, 0xd0, 0x89, 0x0a, 0x06, 0x96, 0xc8,
	0x89, 0xad, 0x9c, 0xd5, 0xc4, 0x8e, 0x6c, 0x27, 0x6a, 0xbf, 0x05, 0x23, 0x23, 0x23, 0x23, 0xe2,
	0x53, 0x74, 0xec, 0xc8, 0x84, 0xd0, 0xdd, 0x00, 0x1f, 0x03, 0x25, 0xa9, 0x7b, 0xb9, 0x85, 0xc1,
	0x96, 0xf5, 0x7e, 0x7f, 0xff, 0xf5, 0xde, 0x5f, 0x0f, 0x3a, 0x6b, 0xca, 0x4a, 0x82, 0x8b, 0xa2,
	0x89, 0xc2, 0x54, 0xc8, 0xf6, 0x04, 0x95, 0x14, 0x5a, 0xa0, 0xf9, 0x8e, 0x04, 0xa9, 0x90, 0x8b,
	0x47, 0xb9, 0xc8, 0x45, 0x87, 0xc2, 0xf6, 0xd5, 0xab, 0x16, 0xde, 0xe0, 0xbf, 0xd2, 0xf8, 0x92,
	0x86, 0x0d, 0x2e, 0x18, 0xc1, 0xda, 0xf8, 0x2c, 0x4e, 0x70, 0xc9, 0xb8, 0x08, 0xbb, 0xbb, 0x2f,
	0x2d, 0x7f, 0x58, 0xd0, 0xbe, 0xa8, 0x30, 0x47, 0x8f, 0xa1, 0xc5, 0x88, 0x03, 0x3c, 0xe0, 0xdb,
	0xf1, 0xe1, 0xb7, 0x3f, 0xdf, 0xcf, 0xc0, 0x7b, 0x8b, 0x11, 0xf4, 0x14, 0x4e, 0x95, 0xc6, 0x52,
	0x27, 0x69, 0x21, 0xb2, 0x4b, 0xc7, 0x1a, 0x72, 0xd8, 0x91, 0xb8, 0x05, 0x68, 0x09, 0x27, 0x94,
	0x93, 0x3b, 0xd5, 0xc1, 0x50, 0x75, 0x44, 0x39, 0xe9, 0x35, 0xef, 0xe0, 0xec, 0xbe, 0xa3, 0x44,
	0x51, 0xed, 0xd8, 0x1e, 0xf0, 0xa7, 0x91, 0x1b, 0x0c, 0xc6, 0xeb, 0x1a, 0x0f, 0x3e, 0x1a, 0xd9,
	0x05, 0xd5, 0xf1, 0xe4, 0xe6, 0xd7, 0xe9, 0xa8, 0xf7, 0x3a, 0x6e, 0x06, 0x00, 0x7d, 0x80, 0x48,
	0xd1, 0x82, 0x66, 0x9a, 0x92, 0xa4, 0x92, 0x82, 0xd4, 0x19, 0x95, 0xca, 0x39, 0xf4, 0x0e, 0xfc,
	0x69, 0xf4, 0xe4, 0x3f, 0xa6, 0x43, 0xc7, 0x13, 0xe3, 0xb0, 0x32, 0x06, 0xc8, 0x83, 0x47, 0xd9,
	0x1a, 0x33, 0x9e, 0x30, 0xe2, 0x8c, 0x3d, 0xe0, 0x4f, 0xcc, 0x24, 0x0f, 0xba, 0xf2, 0x5b, 0xb2,
	0xfc, 0x02, 0xe0, 0x78, 0x85, 0x25, 0x2e, 0x15, 0x0a, 0xe0, 0x43, 0x55, 0x49, 0xc6, 0x75, 0x42,
	0x6a, 0x89, 0x35, 0x13, 0x7c, 0x3f, 0xc3, 0x79, 0x4f, 0x5f, 0xdf, 0x41, 0x74, 0x06, 0x67, 0xaa,
	0xc2, 0x7c, 0xa7, 0xde, 0x4b, 0xf4, 0xb8, 0x65, 0xf7, 0xda, 0x73, 0x38, 0x37, 0x63, 0x25, 0x99,
	0xa8, 0xb9, 0xde, 0x0f, 0x76, 0x66, 0xe0, 0xab, 0x96, 0xbd, 0xb4, 0xff, 0x7e, 0x3d, 0x05, 0xf1,
	0x9b, 0x9b, 0x8d, 0x0b, 0x6e, 0x37, 0x2e, 0xf8, 0xbd, 0x71, 0xc1, 0xe7, 0xad, 0x3b, 0xba, 0xdd,
	0xba, 0xa3, 0x9f, 0x5b, 0x77, 0xf4, 0xe9, 0x3c, 0x67, 0x7a, 0x5d, 0xa7, 0x41, 0x26, 0xca, 0xf0,
	0xf9, 0xd5, 0x4a, 0x14, 0xd7, 0xb9, 0xe0, 0xa1, 0x49, 0xe9, 0x59, 0x13, 0x85, 0x57, 0xdd, 0xda,
	0xe9, 0xeb, 0x8a, 0xaa, 0x74, 0xdc, 0xad, 0xc7, 0x8b, 0x7f, 0x01, 0x00, 0x00, 0xff, 0xff, 0x3e,
	0xe3, 0x02, 0x97, 0x95, 0x02, 0x00, 0x00,
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
	if this.SprintDuration != that1.SprintDuration {
		return false
	}
	if this.SpanDuration != that1.SpanDuration {
		return false
	}
	if this.ProducerCount != that1.ProducerCount {
		return false
	}
	return true
}
func (m *Span) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Span) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Span) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ChainId) > 0 {
		i -= len(m.ChainId)
		copy(dAtA[i:], m.ChainId)
		i = encodeVarintBor(dAtA, i, uint64(len(m.ChainId)))
		i--
		dAtA[i] = 0x32
	}
	if len(m.SelectedProducers) > 0 {
		for iNdEx := len(m.SelectedProducers) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.SelectedProducers[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintBor(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x2a
		}
	}
	{
		size, err := m.ValidatorSet.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintBor(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x22
	if m.EndBlock != 0 {
		i = encodeVarintBor(dAtA, i, uint64(m.EndBlock))
		i--
		dAtA[i] = 0x18
	}
	if m.StartBlock != 0 {
		i = encodeVarintBor(dAtA, i, uint64(m.StartBlock))
		i--
		dAtA[i] = 0x10
	}
	if m.Id != 0 {
		i = encodeVarintBor(dAtA, i, uint64(m.Id))
		i--
		dAtA[i] = 0x8
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
	if m.ProducerCount != 0 {
		i = encodeVarintBor(dAtA, i, uint64(m.ProducerCount))
		i--
		dAtA[i] = 0x18
	}
	if m.SpanDuration != 0 {
		i = encodeVarintBor(dAtA, i, uint64(m.SpanDuration))
		i--
		dAtA[i] = 0x10
	}
	if m.SprintDuration != 0 {
		i = encodeVarintBor(dAtA, i, uint64(m.SprintDuration))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintBor(dAtA []byte, offset int, v uint64) int {
	offset -= sovBor(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Span) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Id != 0 {
		n += 1 + sovBor(uint64(m.Id))
	}
	if m.StartBlock != 0 {
		n += 1 + sovBor(uint64(m.StartBlock))
	}
	if m.EndBlock != 0 {
		n += 1 + sovBor(uint64(m.EndBlock))
	}
	l = m.ValidatorSet.Size()
	n += 1 + l + sovBor(uint64(l))
	if len(m.SelectedProducers) > 0 {
		for _, e := range m.SelectedProducers {
			l = e.Size()
			n += 1 + l + sovBor(uint64(l))
		}
	}
	l = len(m.ChainId)
	if l > 0 {
		n += 1 + l + sovBor(uint64(l))
	}
	return n
}

func (m *Params) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.SprintDuration != 0 {
		n += 1 + sovBor(uint64(m.SprintDuration))
	}
	if m.SpanDuration != 0 {
		n += 1 + sovBor(uint64(m.SpanDuration))
	}
	if m.ProducerCount != 0 {
		n += 1 + sovBor(uint64(m.ProducerCount))
	}
	return n
}

func sovBor(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozBor(x uint64) (n int) {
	return sovBor(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Span) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowBor
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
			return fmt.Errorf("proto: Span: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Span: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			m.Id = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBor
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Id |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field StartBlock", wireType)
			}
			m.StartBlock = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBor
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
					return ErrIntOverflowBor
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
				return fmt.Errorf("proto: wrong wireType = %d for field ValidatorSet", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBor
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
				return ErrInvalidLengthBor
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthBor
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.ValidatorSet.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SelectedProducers", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBor
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
				return ErrInvalidLengthBor
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthBor
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.SelectedProducers = append(m.SelectedProducers, types.Validator{})
			if err := m.SelectedProducers[len(m.SelectedProducers)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ChainId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBor
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
				return ErrInvalidLengthBor
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthBor
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ChainId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipBor(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthBor
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
				return ErrIntOverflowBor
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
				return fmt.Errorf("proto: wrong wireType = %d for field SprintDuration", wireType)
			}
			m.SprintDuration = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBor
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SprintDuration |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SpanDuration", wireType)
			}
			m.SpanDuration = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBor
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SpanDuration |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProducerCount", wireType)
			}
			m.ProducerCount = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBor
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ProducerCount |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipBor(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthBor
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
func skipBor(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowBor
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
					return 0, ErrIntOverflowBor
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
					return 0, ErrIntOverflowBor
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
				return 0, ErrInvalidLengthBor
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupBor
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthBor
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthBor        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowBor          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupBor = fmt.Errorf("proto: unexpected end of group")
)
