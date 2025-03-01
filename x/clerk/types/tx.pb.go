// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: heimdallv2/clerk/tx.proto

package types

import (
	context "context"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types/msgservice"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	grpc1 "github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

type MsgEventRecordResponse struct {
}

func (m *MsgEventRecordResponse) Reset()         { *m = MsgEventRecordResponse{} }
func (m *MsgEventRecordResponse) String() string { return proto.CompactTextString(m) }
func (*MsgEventRecordResponse) ProtoMessage()    {}
func (*MsgEventRecordResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb0d5312093e6ca2, []int{0}
}
func (m *MsgEventRecordResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgEventRecordResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgEventRecordResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgEventRecordResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgEventRecordResponse.Merge(m, src)
}
func (m *MsgEventRecordResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgEventRecordResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgEventRecordResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgEventRecordResponse proto.InternalMessageInfo

type MsgEventRecord struct {
	From            string `protobuf:"bytes,1,opt,name=from,proto3" json:"from,omitempty"`
	TxHash          string `protobuf:"bytes,2,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
	LogIndex        uint64 `protobuf:"varint,3,opt,name=log_index,json=logIndex,proto3" json:"log_index,omitempty"`
	BlockNumber     uint64 `protobuf:"varint,4,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	ContractAddress string `protobuf:"bytes,5,opt,name=contract_address,json=contractAddress,proto3" json:"contract_address,omitempty"`
	Data            []byte `protobuf:"bytes,6,opt,name=data,proto3" json:"data,omitempty"`
	Id              uint64 `protobuf:"varint,7,opt,name=id,proto3" json:"id,omitempty"`
	ChainId         string `protobuf:"bytes,8,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
}

func (m *MsgEventRecord) Reset()         { *m = MsgEventRecord{} }
func (m *MsgEventRecord) String() string { return proto.CompactTextString(m) }
func (*MsgEventRecord) ProtoMessage()    {}
func (*MsgEventRecord) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb0d5312093e6ca2, []int{1}
}
func (m *MsgEventRecord) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgEventRecord) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgEventRecord.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgEventRecord) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgEventRecord.Merge(m, src)
}
func (m *MsgEventRecord) XXX_Size() int {
	return m.Size()
}
func (m *MsgEventRecord) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgEventRecord.DiscardUnknown(m)
}

var xxx_messageInfo_MsgEventRecord proto.InternalMessageInfo

func init() {
	proto.RegisterType((*MsgEventRecordResponse)(nil), "heimdallv2.clerk.MsgEventRecordResponse")
	proto.RegisterType((*MsgEventRecord)(nil), "heimdallv2.clerk.MsgEventRecord")
}

func init() { proto.RegisterFile("heimdallv2/clerk/tx.proto", fileDescriptor_bb0d5312093e6ca2) }

var fileDescriptor_bb0d5312093e6ca2 = []byte{
	// 466 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x92, 0x3f, 0x8f, 0xd3, 0x3c,
	0x00, 0xc6, 0x93, 0xfe, 0x3f, 0xbf, 0xa7, 0x97, 0xc3, 0x2a, 0x90, 0x56, 0x22, 0x8d, 0x3a, 0x45,
	0x95, 0x2e, 0xe6, 0x8a, 0x58, 0xd8, 0x38, 0x09, 0xa9, 0x37, 0x1c, 0x42, 0x65, 0x63, 0x89, 0x9c,
	0xd8, 0x38, 0xd6, 0x25, 0x76, 0x15, 0xfb, 0xaa, 0xdc, 0x86, 0x18, 0x10, 0x62, 0xe2, 0x23, 0xdc,
	0xc8, 0xd8, 0x81, 0x0f, 0xc1, 0x78, 0x62, 0x62, 0x44, 0xed, 0x50, 0x3e, 0x06, 0xaa, 0xd3, 0x88,
	0xf6, 0x18, 0x60, 0x49, 0xe2, 0xe7, 0xf7, 0x38, 0x7e, 0x6c, 0x3f, 0xa0, 0x97, 0x50, 0x9e, 0x11,
	0x9c, 0xa6, 0xf3, 0x31, 0x8a, 0x53, 0x9a, 0x5f, 0x20, 0x5d, 0x04, 0xb3, 0x5c, 0x6a, 0x09, 0x8f,
	0x7e, 0xa3, 0xc0, 0xa0, 0xfe, 0x5d, 0x9c, 0x71, 0x21, 0x91, 0x79, 0x96, 0xa6, 0xfe, 0x83, 0x58,
	0xaa, 0x4c, 0x2a, 0x94, 0x29, 0x86, 0xe6, 0x27, 0x9b, 0xd7, 0x16, 0xf4, 0x4a, 0x10, 0x9a, 0x11,
	0x2a, 0x07, 0x5b, 0xd4, 0x65, 0x92, 0xc9, 0x52, 0xdf, 0x7c, 0x95, 0xea, 0xd0, 0x01, 0xf7, 0xcf,
	0x15, 0x7b, 0x3e, 0xa7, 0x42, 0x4f, 0x69, 0x2c, 0x73, 0x32, 0xa5, 0x6a, 0x26, 0x85, 0xa2, 0xc3,
	0xf7, 0x75, 0xf0, 0xff, 0x3e, 0x82, 0x27, 0xa0, 0xf1, 0x26, 0x97, 0x99, 0x63, 0x7b, 0xb6, 0x7f,
	0x70, 0xfa, 0xf0, 0xdb, 0x97, 0xe3, 0xee, 0x76, 0x89, 0x67, 0x84, 0xe4, 0x54, 0xa9, 0x57, 0x3a,
	0xe7, 0x82, 0x7d, 0x5e, 0x2f, 0x46, 0xf6, 0xd4, 0x58, 0xa1, 0x0b, 0xda, 0xba, 0x08, 0x13, 0xac,
	0x12, 0xa7, 0x66, 0x66, 0x35, 0x4b, 0xda, 0xd2, 0xc5, 0x04, 0xab, 0x04, 0x0e, 0xc1, 0x41, 0x2a,
	0x59, 0xc8, 0x05, 0xa1, 0x85, 0x53, 0xf7, 0x6c, 0xbf, 0x51, 0x39, 0x3a, 0xa9, 0x64, 0x67, 0x1b,
	0x19, 0xfa, 0xe0, 0x30, 0x4a, 0x65, 0x7c, 0x11, 0x8a, 0xcb, 0x2c, 0xa2, 0xb9, 0xd3, 0xd8, 0xb5,
	0xfd, 0x67, 0xd0, 0x0b, 0x43, 0xe0, 0x04, 0x1c, 0xc5, 0x52, 0xe8, 0x1c, 0xc7, 0x3a, 0xc4, 0x65,
	0x24, 0xa7, 0xf9, 0x2f, 0x61, 0xef, 0x54, 0xd3, 0xb6, 0x0c, 0xf6, 0x40, 0x83, 0x60, 0x8d, 0x9d,
	0x96, 0x67, 0xfb, 0x87, 0xd5, 0x5a, 0x46, 0x82, 0xf7, 0x40, 0x8d, 0x13, 0xa7, 0xbd, 0x1b, 0xa2,
	0xc6, 0x09, 0xf4, 0x40, 0x27, 0x4e, 0x30, 0x17, 0x21, 0x27, 0x4e, 0x67, 0x77, 0xab, 0x6d, 0x23,
	0x9f, 0x91, 0xa7, 0x4f, 0x3e, 0x5c, 0x0f, 0xac, 0x9f, 0xd7, 0x03, 0xeb, 0xdd, 0x7a, 0x31, 0x32,
	0xc7, 0xf3, 0x71, 0xbd, 0x18, 0x0d, 0xfe, 0xe8, 0xc2, 0xfe, 0xa9, 0x8f, 0x67, 0xa0, 0x7e, 0xae,
	0x18, 0x8c, 0x40, 0x77, 0x82, 0x05, 0x49, 0xe9, 0xad, 0x4b, 0xf1, 0x82, 0xdb, 0x8d, 0x09, 0xf6,
	0x1d, 0x7d, 0xff, 0x6f, 0x8e, 0xea, 0xce, 0xfb, 0xcd, 0xb7, 0x9b, 0xc4, 0xa7, 0x93, 0xaf, 0x4b,
	0xd7, 0xbe, 0x59, 0xba, 0xf6, 0x8f, 0xa5, 0x6b, 0x7f, 0x5a, 0xb9, 0xd6, 0xcd, 0xca, 0xb5, 0xbe,
	0xaf, 0x5c, 0xeb, 0x75, 0xc0, 0xb8, 0x4e, 0x2e, 0xa3, 0x20, 0x96, 0x19, 0x7a, 0x54, 0xbc, 0x94,
	0xe9, 0x15, 0x93, 0x02, 0x55, 0xbf, 0x3f, 0x9e, 0x8f, 0x51, 0x51, 0x15, 0xfa, 0x6a, 0x46, 0x55,
	0xd4, 0x32, 0x2d, 0x7b, 0xfc, 0x2b, 0x00, 0x00, 0xff, 0xff, 0x4b, 0xf8, 0x43, 0x7d, 0xf1, 0x02,
	0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MsgClient interface {
	// HandleMsgEventRecord defines a clerk operation for handling an event record
	HandleMsgEventRecord(ctx context.Context, in *MsgEventRecord, opts ...grpc.CallOption) (*MsgEventRecordResponse, error)
}

type msgClient struct {
	cc grpc1.ClientConn
}

func NewMsgClient(cc grpc1.ClientConn) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) HandleMsgEventRecord(ctx context.Context, in *MsgEventRecord, opts ...grpc.CallOption) (*MsgEventRecordResponse, error) {
	out := new(MsgEventRecordResponse)
	err := c.cc.Invoke(ctx, "/heimdallv2.clerk.Msg/HandleMsgEventRecord", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
type MsgServer interface {
	// HandleMsgEventRecord defines a clerk operation for handling an event record
	HandleMsgEventRecord(context.Context, *MsgEventRecord) (*MsgEventRecordResponse, error)
}

// UnimplementedMsgServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (*UnimplementedMsgServer) HandleMsgEventRecord(ctx context.Context, req *MsgEventRecord) (*MsgEventRecordResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HandleMsgEventRecord not implemented")
}

func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

func _Msg_HandleMsgEventRecord_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgEventRecord)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).HandleMsgEventRecord(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/heimdallv2.clerk.Msg/HandleMsgEventRecord",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).HandleMsgEventRecord(ctx, req.(*MsgEventRecord))
	}
	return interceptor(ctx, in, info, handler)
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "heimdallv2.clerk.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "HandleMsgEventRecord",
			Handler:    _Msg_HandleMsgEventRecord_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "heimdallv2/clerk/tx.proto",
}

func (m *MsgEventRecordResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgEventRecordResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgEventRecordResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *MsgEventRecord) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgEventRecord) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgEventRecord) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ChainId) > 0 {
		i -= len(m.ChainId)
		copy(dAtA[i:], m.ChainId)
		i = encodeVarintTx(dAtA, i, uint64(len(m.ChainId)))
		i--
		dAtA[i] = 0x42
	}
	if m.Id != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.Id))
		i--
		dAtA[i] = 0x38
	}
	if len(m.Data) > 0 {
		i -= len(m.Data)
		copy(dAtA[i:], m.Data)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Data)))
		i--
		dAtA[i] = 0x32
	}
	if len(m.ContractAddress) > 0 {
		i -= len(m.ContractAddress)
		copy(dAtA[i:], m.ContractAddress)
		i = encodeVarintTx(dAtA, i, uint64(len(m.ContractAddress)))
		i--
		dAtA[i] = 0x2a
	}
	if m.BlockNumber != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.BlockNumber))
		i--
		dAtA[i] = 0x20
	}
	if m.LogIndex != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.LogIndex))
		i--
		dAtA[i] = 0x18
	}
	if len(m.TxHash) > 0 {
		i -= len(m.TxHash)
		copy(dAtA[i:], m.TxHash)
		i = encodeVarintTx(dAtA, i, uint64(len(m.TxHash)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.From) > 0 {
		i -= len(m.From)
		copy(dAtA[i:], m.From)
		i = encodeVarintTx(dAtA, i, uint64(len(m.From)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintTx(dAtA []byte, offset int, v uint64) int {
	offset -= sovTx(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *MsgEventRecordResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *MsgEventRecord) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.From)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.TxHash)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	if m.LogIndex != 0 {
		n += 1 + sovTx(uint64(m.LogIndex))
	}
	if m.BlockNumber != 0 {
		n += 1 + sovTx(uint64(m.BlockNumber))
	}
	l = len(m.ContractAddress)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.Data)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	if m.Id != 0 {
		n += 1 + sovTx(uint64(m.Id))
	}
	l = len(m.ChainId)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	return n
}

func sovTx(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTx(x uint64) (n int) {
	return sovTx(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgEventRecordResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgEventRecordResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgEventRecordResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func (m *MsgEventRecord) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowTx
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
			return fmt.Errorf("proto: MsgEventRecord: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgEventRecord: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field From", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.From = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TxHash", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TxHash = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LogIndex", wireType)
			}
			m.LogIndex = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field BlockNumber", wireType)
			}
			m.BlockNumber = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ContractAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ContractAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Data", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Data = append(m.Data[:0], dAtA[iNdEx:postIndex]...)
			if m.Data == nil {
				m.Data = []byte{}
			}
			iNdEx = postIndex
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			m.Id = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ChainId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ChainId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipTx(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthTx
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
func skipTx(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowTx
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
					return 0, ErrIntOverflowTx
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
					return 0, ErrIntOverflowTx
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
				return 0, ErrInvalidLengthTx
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupTx
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthTx
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthTx        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowTx          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupTx = fmt.Errorf("proto: unexpected end of group")
)
