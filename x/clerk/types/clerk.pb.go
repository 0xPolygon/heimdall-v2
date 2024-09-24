// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: heimdallv2/clerk/clerk.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	github_com_cosmos_gogoproto_types "github.com/cosmos/gogoproto/types"
	_ "google.golang.org/protobuf/types/known/timestamppb"
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

type EventRecord struct {
	Id         uint64    `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Contract   string    `protobuf:"bytes,2,opt,name=contract,proto3" json:"contract,omitempty"`
	Data       []byte    `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
	TxHash     string    `protobuf:"bytes,4,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
	LogIndex   uint64    `protobuf:"varint,5,opt,name=log_index,json=logIndex,proto3" json:"log_index,omitempty"`
	BorChainId string    `protobuf:"bytes,6,opt,name=bor_chain_id,json=borChainId,proto3" json:"bor_chain_id,omitempty"`
	RecordTime time.Time `protobuf:"bytes,7,opt,name=record_time,json=recordTime,proto3,stdtime" json:"record_time"`
}

func (m *EventRecord) Reset()         { *m = EventRecord{} }
func (m *EventRecord) String() string { return proto.CompactTextString(m) }
func (*EventRecord) ProtoMessage()    {}
func (*EventRecord) Descriptor() ([]byte, []int) {
	return fileDescriptor_94e73a41bf24ef06, []int{0}
}
func (m *EventRecord) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *EventRecord) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_EventRecord.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *EventRecord) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EventRecord.Merge(m, src)
}
func (m *EventRecord) XXX_Size() int {
	return m.Size()
}
func (m *EventRecord) XXX_DiscardUnknown() {
	xxx_messageInfo_EventRecord.DiscardUnknown(m)
}

var xxx_messageInfo_EventRecord proto.InternalMessageInfo

func init() {
	proto.RegisterType((*EventRecord)(nil), "heimdallv2.clerk.EventRecord")
}

func init() { proto.RegisterFile("heimdallv2/clerk/clerk.proto", fileDescriptor_94e73a41bf24ef06) }

var fileDescriptor_94e73a41bf24ef06 = []byte{
	// 403 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x44, 0x51, 0x3f, 0x6f, 0xd4, 0x30,
	0x14, 0x8f, 0x8f, 0xe3, 0x7a, 0xf5, 0x55, 0x08, 0xac, 0x4a, 0x84, 0x03, 0x92, 0x88, 0x29, 0x42,
	0x6a, 0x8c, 0x8e, 0x09, 0x36, 0x0e, 0x55, 0x6a, 0x37, 0x14, 0x98, 0x58, 0x22, 0x27, 0x36, 0x8e,
	0x45, 0x92, 0x77, 0xb2, 0xdd, 0x53, 0xfa, 0x0d, 0x18, 0xbb, 0xb0, 0x77, 0x64, 0x64, 0xe0, 0x43,
	0x74, 0xac, 0x98, 0x98, 0x00, 0xdd, 0x0d, 0xf0, 0x31, 0x50, 0x9c, 0x1c, 0x5d, 0xac, 0xf7, 0xfb,
	0xf3, 0xfc, 0xd3, 0x4f, 0x0f, 0x3f, 0x2a, 0x85, 0xaa, 0x39, 0xab, 0xaa, 0xf5, 0x82, 0x16, 0x95,
	0xd0, 0x1f, 0xfb, 0x37, 0x59, 0x69, 0xb0, 0x40, 0xee, 0xde, 0xa8, 0x89, 0xe3, 0xe7, 0xf7, 0x58,
	0xad, 0x1a, 0xa0, 0xee, 0xed, 0x4d, 0xf3, 0x07, 0x05, 0x98, 0x1a, 0x4c, 0xe6, 0x10, 0xed, 0xc1,
	0x20, 0x1d, 0x4a, 0x90, 0xd0, 0xf3, 0xdd, 0x34, 0xb0, 0xa1, 0x04, 0x90, 0x95, 0xa0, 0x0e, 0xe5,
	0x67, 0x1f, 0xa8, 0x55, 0xb5, 0x30, 0x96, 0xd5, 0xab, 0xde, 0xf0, 0xe4, 0xf3, 0x08, 0xcf, 0x8e,
	0xd7, 0xa2, 0xb1, 0xa9, 0x28, 0x40, 0x73, 0x72, 0x07, 0x8f, 0x14, 0xf7, 0x51, 0x84, 0xe2, 0x71,
	0x3a, 0x52, 0x9c, 0xbc, 0xc0, 0xd3, 0x02, 0x1a, 0xab, 0x59, 0x61, 0xfd, 0x51, 0x84, 0xe2, 0xfd,
	0xe5, 0xe3, 0xef, 0xdf, 0x8e, 0x0e, 0x87, 0xe8, 0x57, 0x9c, 0x6b, 0x61, 0xcc, 0x5b, 0xab, 0x55,
	0x23, 0xbf, 0xfc, 0xf9, 0xfa, 0x14, 0xa5, 0xff, 0xed, 0x84, 0xe0, 0x31, 0x67, 0x96, 0xf9, 0xb7,
	0x22, 0x14, 0x1f, 0xa4, 0x6e, 0x26, 0xf7, 0xf1, 0x9e, 0x6d, 0xb3, 0x92, 0x99, 0xd2, 0x1f, 0x77,
	0xbf, 0xa5, 0x13, 0xdb, 0x9e, 0x30, 0x53, 0x92, 0x87, 0x78, 0xbf, 0x02, 0x99, 0xa9, 0x86, 0x8b,
	0xd6, 0xbf, 0xed, 0xe2, 0xa7, 0x15, 0xc8, 0xd3, 0x0e, 0x93, 0x08, 0x1f, 0xe4, 0xa0, 0xb3, 0xa2,
	0x64, 0xaa, 0xc9, 0x14, 0xf7, 0x27, 0x6e, 0x15, 0xe7, 0xa0, 0x5f, 0x77, 0xd4, 0x29, 0x27, 0xc7,
	0x78, 0xa6, 0x5d, 0x81, 0xac, 0x2b, 0xe8, 0xef, 0x45, 0x28, 0x9e, 0x2d, 0xe6, 0x49, 0xdf, 0x3e,
	0xd9, 0xb5, 0x4f, 0xde, 0xed, 0xda, 0x2f, 0xa7, 0x57, 0x3f, 0x43, 0xef, 0xe2, 0x57, 0x88, 0x52,
	0xdc, 0x2f, 0x76, 0xd2, 0xcb, 0xe9, 0xa7, 0xcb, 0xd0, 0xfb, 0x7b, 0x19, 0x7a, 0xcb, 0x93, 0xab,
	0x4d, 0x80, 0xae, 0x37, 0x01, 0xfa, 0xbd, 0x09, 0xd0, 0xc5, 0x36, 0xf0, 0xae, 0xb7, 0x81, 0xf7,
	0x63, 0x1b, 0x78, 0xef, 0x13, 0xa9, 0x6c, 0x79, 0x96, 0x27, 0x05, 0xd4, 0xf4, 0x59, 0xfb, 0x06,
	0xaa, 0x73, 0x09, 0x0d, 0xdd, 0x5d, 0xef, 0x68, 0xbd, 0xa0, 0xed, 0x70, 0x5e, 0x7b, 0xbe, 0x12,
	0x26, 0x9f, 0xb8, 0xf4, 0xe7, 0xff, 0x02, 0x00, 0x00, 0xff, 0xff, 0xc9, 0x2a, 0x8c, 0x7d, 0xff,
	0x01, 0x00, 0x00,
}

func (m *EventRecord) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *EventRecord) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *EventRecord) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	n1, err1 := github_com_cosmos_gogoproto_types.StdTimeMarshalTo(m.RecordTime, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdTime(m.RecordTime):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintClerk(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0x3a
	if len(m.BorChainId) > 0 {
		i -= len(m.BorChainId)
		copy(dAtA[i:], m.BorChainId)
		i = encodeVarintClerk(dAtA, i, uint64(len(m.BorChainId)))
		i--
		dAtA[i] = 0x32
	}
	if m.LogIndex != 0 {
		i = encodeVarintClerk(dAtA, i, uint64(m.LogIndex))
		i--
		dAtA[i] = 0x28
	}
	if len(m.TxHash) > 0 {
		i -= len(m.TxHash)
		copy(dAtA[i:], m.TxHash)
		i = encodeVarintClerk(dAtA, i, uint64(len(m.TxHash)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.Data) > 0 {
		i -= len(m.Data)
		copy(dAtA[i:], m.Data)
		i = encodeVarintClerk(dAtA, i, uint64(len(m.Data)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Contract) > 0 {
		i -= len(m.Contract)
		copy(dAtA[i:], m.Contract)
		i = encodeVarintClerk(dAtA, i, uint64(len(m.Contract)))
		i--
		dAtA[i] = 0x12
	}
	if m.Id != 0 {
		i = encodeVarintClerk(dAtA, i, uint64(m.Id))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintClerk(dAtA []byte, offset int, v uint64) int {
	offset -= sovClerk(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *EventRecord) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Id != 0 {
		n += 1 + sovClerk(uint64(m.Id))
	}
	l = len(m.Contract)
	if l > 0 {
		n += 1 + l + sovClerk(uint64(l))
	}
	l = len(m.Data)
	if l > 0 {
		n += 1 + l + sovClerk(uint64(l))
	}
	l = len(m.TxHash)
	if l > 0 {
		n += 1 + l + sovClerk(uint64(l))
	}
	if m.LogIndex != 0 {
		n += 1 + sovClerk(uint64(m.LogIndex))
	}
	l = len(m.BorChainId)
	if l > 0 {
		n += 1 + l + sovClerk(uint64(l))
	}
	l = github_com_cosmos_gogoproto_types.SizeOfStdTime(m.RecordTime)
	n += 1 + l + sovClerk(uint64(l))
	return n
}

func sovClerk(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozClerk(x uint64) (n int) {
	return sovClerk(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *EventRecord) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowClerk
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
			return fmt.Errorf("proto: EventRecord: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: EventRecord: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			m.Id = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowClerk
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
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Contract", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowClerk
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
				return ErrInvalidLengthClerk
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthClerk
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Contract = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Data", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowClerk
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
				return ErrInvalidLengthClerk
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthClerk
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Data = append(m.Data[:0], dAtA[iNdEx:postIndex]...)
			if m.Data == nil {
				m.Data = []byte{}
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TxHash", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowClerk
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
				return ErrInvalidLengthClerk
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthClerk
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TxHash = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LogIndex", wireType)
			}
			m.LogIndex = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowClerk
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
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field BorChainId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowClerk
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
				return ErrInvalidLengthClerk
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthClerk
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.BorChainId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RecordTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowClerk
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
				return ErrInvalidLengthClerk
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthClerk
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_cosmos_gogoproto_types.StdTimeUnmarshal(&m.RecordTime, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipClerk(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthClerk
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
func skipClerk(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowClerk
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
					return 0, ErrIntOverflowClerk
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
					return 0, ErrIntOverflowClerk
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
				return 0, ErrInvalidLengthClerk
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupClerk
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthClerk
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthClerk        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowClerk          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupClerk = fmt.Errorf("proto: unexpected end of group")
)
