// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: heimdallv2/clerk/tx.proto

package types

import (
	context "context"
	fmt "fmt"
	types "github.com/0xPolygon/heimdall-v2/types"
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

type MsgEventRecordRequest struct {
	From            string             `protobuf:"bytes,1,opt,name=from,proto3" json:"from,omitempty"`
	TxHash          types.HeimdallHash `protobuf:"bytes,2,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash"`
	LogIndex        uint64             `protobuf:"varint,3,opt,name=log_index,json=logIndex,proto3" json:"log_index,omitempty"`
	BlockNumber     uint64             `protobuf:"varint,4,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	ContractAddress string             `protobuf:"bytes,5,opt,name=contract_address,json=contractAddress,proto3" json:"contract_address,omitempty"`
	Data            types.HexBytes     `protobuf:"bytes,6,opt,name=data,proto3" json:"data"`
	ID              uint64             `protobuf:"varint,7,opt,name=i_d,json=iD,proto3" json:"i_d,omitempty"`
	ChainID         string             `protobuf:"bytes,8,opt,name=chain_i_d,json=chainID,proto3" json:"chain_i_d,omitempty"`
}

func (m *MsgEventRecordRequest) Reset()         { *m = MsgEventRecordRequest{} }
func (m *MsgEventRecordRequest) String() string { return proto.CompactTextString(m) }
func (*MsgEventRecordRequest) ProtoMessage()    {}
func (*MsgEventRecordRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_bb0d5312093e6ca2, []int{1}
}
func (m *MsgEventRecordRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgEventRecordRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgEventRecordRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgEventRecordRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgEventRecordRequest.Merge(m, src)
}
func (m *MsgEventRecordRequest) XXX_Size() int {
	return m.Size()
}
func (m *MsgEventRecordRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgEventRecordRequest.DiscardUnknown(m)
}

var xxx_messageInfo_MsgEventRecordRequest proto.InternalMessageInfo

func init() {
	proto.RegisterType((*MsgEventRecordResponse)(nil), "heimdallv2.clerk.MsgEventRecordResponse")
	proto.RegisterType((*MsgEventRecordRequest)(nil), "heimdallv2.clerk.MsgEventRecordRequest")
}

func init() { proto.RegisterFile("heimdallv2/clerk/tx.proto", fileDescriptor_bb0d5312093e6ca2) }

var fileDescriptor_bb0d5312093e6ca2 = []byte{
	// 491 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x92, 0x3f, 0x6f, 0xd3, 0x40,
	0x18, 0xc6, 0xed, 0xc6, 0xcd, 0x9f, 0x2b, 0x52, 0xcb, 0x29, 0x80, 0xeb, 0x0a, 0x27, 0x74, 0x21,
	0x42, 0xaa, 0x8f, 0x06, 0x26, 0x24, 0x06, 0xa2, 0x22, 0xb9, 0x43, 0x11, 0x32, 0x1b, 0x8b, 0x75,
	0xb1, 0x8f, 0xb3, 0x55, 0xfb, 0x2e, 0xf8, 0x2e, 0xc1, 0xd9, 0x3a, 0x32, 0xf2, 0x11, 0x3a, 0x32,
	0x32, 0xf0, 0x21, 0x3a, 0x56, 0x4c, 0x4c, 0x08, 0x25, 0x03, 0x7c, 0x0c, 0x74, 0x67, 0x47, 0x94,
	0x2a, 0x12, 0x5d, 0x6c, 0xbf, 0xcf, 0xef, 0xb5, 0xfd, 0x3c, 0xf7, 0xbe, 0x60, 0x37, 0x21, 0x69,
	0x1e, 0xe3, 0x2c, 0x9b, 0x0d, 0x51, 0x94, 0x91, 0xe2, 0x14, 0xc9, 0xd2, 0x9b, 0x14, 0x5c, 0x72,
	0xb8, 0xf3, 0x17, 0x79, 0x1a, 0x39, 0xb7, 0x71, 0x9e, 0x32, 0x8e, 0xf4, 0xb5, 0x6a, 0x72, 0xee,
	0x45, 0x5c, 0xe4, 0x5c, 0xa0, 0x5c, 0x50, 0x34, 0x3b, 0x54, 0xb7, 0x1a, 0xec, 0x56, 0x20, 0xd4,
	0x15, 0xaa, 0x8a, 0x1a, 0x75, 0x29, 0xa7, 0xbc, 0xd2, 0xd5, 0x53, 0xad, 0xee, 0x5d, 0x71, 0x22,
	0xe7, 0x13, 0x22, 0x50, 0x82, 0x45, 0x52, 0xc1, 0x7d, 0x1b, 0xdc, 0x3d, 0x11, 0xf4, 0xe5, 0x8c,
	0x30, 0x19, 0x90, 0x88, 0x17, 0x71, 0x40, 0xc4, 0x84, 0x33, 0x41, 0xf6, 0xcf, 0x1a, 0xe0, 0xce,
	0x75, 0xf4, 0x7e, 0x4a, 0x84, 0x84, 0x87, 0xc0, 0x7a, 0x57, 0xf0, 0xdc, 0x36, 0xfb, 0xe6, 0xa0,
	0x33, 0xba, 0xff, 0xed, 0xeb, 0x41, 0xb7, 0xb6, 0xf1, 0x22, 0x8e, 0x0b, 0x22, 0xc4, 0x1b, 0x59,
	0xa4, 0x8c, 0x7e, 0xfe, 0xf5, 0xe5, 0x91, 0x19, 0xe8, 0x56, 0xf8, 0x1c, 0xb4, 0x64, 0x19, 0xaa,
	0xff, 0xda, 0x1b, 0x7d, 0x73, 0xb0, 0x35, 0x74, 0xbd, 0x2b, 0x87, 0xa0, 0x5d, 0x79, 0x7e, 0x2d,
	0xf8, 0x58, 0x24, 0x23, 0xeb, 0xe2, 0x47, 0xcf, 0x08, 0x9a, 0xb2, 0x54, 0x15, 0xdc, 0x03, 0x9d,
	0x8c, 0xd3, 0x30, 0x65, 0x31, 0x29, 0xed, 0x46, 0xdf, 0x1c, 0x58, 0x41, 0x3b, 0xe3, 0xf4, 0x58,
	0xd5, 0xf0, 0x01, 0xb8, 0x35, 0xce, 0x78, 0x74, 0x1a, 0xb2, 0x69, 0x3e, 0x26, 0x85, 0x6d, 0x69,
	0xbe, 0xa5, 0xb5, 0x57, 0x5a, 0x82, 0x3e, 0xd8, 0x89, 0x38, 0x93, 0x05, 0x8e, 0x64, 0x88, 0x2b,
	0x8f, 0xf6, 0xe6, 0x4d, 0xdc, 0x6f, 0xaf, 0x5e, 0xab, 0x19, 0x7c, 0x0a, 0xac, 0x18, 0x4b, 0x6c,
	0x37, 0x75, 0x0a, 0x67, 0x5d, 0x8a, 0x72, 0x34, 0x97, 0x44, 0xd4, 0x09, 0x74, 0x37, 0xdc, 0x06,
	0x8d, 0x34, 0x8c, 0xed, 0x96, 0x76, 0xb6, 0x91, 0x1e, 0x41, 0x07, 0x74, 0xa2, 0x04, 0xa7, 0x2c,
	0x54, 0x72, 0x5b, 0x39, 0x09, 0x5a, 0x5a, 0x38, 0x3e, 0x7a, 0xd6, 0xfe, 0x78, 0xde, 0x33, 0x7e,
	0x9f, 0xf7, 0x8c, 0xe1, 0x07, 0xd0, 0x38, 0x11, 0x14, 0xa6, 0xa0, 0xeb, 0x63, 0x16, 0x67, 0xe4,
	0xdf, 0x71, 0xc0, 0x87, 0xde, 0xf5, 0x45, 0xf2, 0xd6, 0x0e, 0xcc, 0x19, 0xfc, 0xbf, 0xb1, 0x1a,
	0xba, 0xb3, 0x79, 0xa6, 0x62, 0x8f, 0xfc, 0x8b, 0x85, 0x6b, 0x5e, 0x2e, 0x5c, 0xf3, 0xe7, 0xc2,
	0x35, 0x3f, 0x2d, 0x5d, 0xe3, 0x72, 0xe9, 0x1a, 0xdf, 0x97, 0xae, 0xf1, 0xd6, 0xa3, 0xa9, 0x4c,
	0xa6, 0x63, 0x2f, 0xe2, 0x39, 0x7a, 0x5c, 0xbe, 0xe6, 0xd9, 0x9c, 0x72, 0x86, 0x56, 0x9f, 0x3f,
	0x98, 0x0d, 0x51, 0xb9, 0x5a, 0x77, 0x75, 0x1c, 0xe3, 0xa6, 0x5e, 0xb3, 0x27, 0x7f, 0x02, 0x00,
	0x00, 0xff, 0xff, 0xd7, 0x54, 0xbd, 0x9a, 0x0f, 0x03, 0x00, 0x00,
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
	HandleMsgEventRecord(ctx context.Context, in *MsgEventRecordRequest, opts ...grpc.CallOption) (*MsgEventRecordResponse, error)
}

type msgClient struct {
	cc grpc1.ClientConn
}

func NewMsgClient(cc grpc1.ClientConn) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) HandleMsgEventRecord(ctx context.Context, in *MsgEventRecordRequest, opts ...grpc.CallOption) (*MsgEventRecordResponse, error) {
	out := new(MsgEventRecordResponse)
	err := c.cc.Invoke(ctx, "/heimdallv2.clerk.Msg/HandleMsgEventRecord", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
type MsgServer interface {
	HandleMsgEventRecord(context.Context, *MsgEventRecordRequest) (*MsgEventRecordResponse, error)
}

// UnimplementedMsgServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (*UnimplementedMsgServer) HandleMsgEventRecord(ctx context.Context, req *MsgEventRecordRequest) (*MsgEventRecordResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HandleMsgEventRecord not implemented")
}

func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

func _Msg_HandleMsgEventRecord_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgEventRecordRequest)
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
		return srv.(MsgServer).HandleMsgEventRecord(ctx, req.(*MsgEventRecordRequest))
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

func (m *MsgEventRecordRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgEventRecordRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgEventRecordRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.ChainID) > 0 {
		i -= len(m.ChainID)
		copy(dAtA[i:], m.ChainID)
		i = encodeVarintTx(dAtA, i, uint64(len(m.ChainID)))
		i--
		dAtA[i] = 0x42
	}
	if m.ID != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.ID))
		i--
		dAtA[i] = 0x38
	}
	{
		size, err := m.Data.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x32
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
	{
		size, err := m.TxHash.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintTx(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x12
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

func (m *MsgEventRecordRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.From)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = m.TxHash.Size()
	n += 1 + l + sovTx(uint64(l))
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
	l = m.Data.Size()
	n += 1 + l + sovTx(uint64(l))
	if m.ID != 0 {
		n += 1 + sovTx(uint64(m.ID))
	}
	l = len(m.ChainID)
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
func (m *MsgEventRecordRequest) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgEventRecordRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgEventRecordRequest: illegal tag %d (wire type %d)", fieldNum, wire)
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
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.TxHash.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
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
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
				return ErrInvalidLengthTx
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthTx
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Data.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			m.ID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ID |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ChainID", wireType)
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
			m.ChainID = string(dAtA[iNdEx:postIndex])
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
