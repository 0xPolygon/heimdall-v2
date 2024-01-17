package types

import (
	context "context"
	fmt "fmt"
	io "io"
	math "math"
	math_bits "math/bits"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/cosmos-sdk/types/msgservice"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	grpc1 "github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	_ "google.golang.org/protobuf/types/known/timestamppb"
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

type MsgValidatorJoin struct {
	From            hmTypes.HeimdallAddress `protobuf:"bytes,1,opt,name=from,proto3" json:"from,omitempty"`
	ID              hmTypes.ValidatorID     `protobuf:"bytes,2,opt,name=id,json=id,proto3" json:"id,omitempty"`
	ActivationEpoch uint64                  `protobuf:"varint,3,opt,name=activation_epoch,json=activationEpoch,proto3" json:"activation_epoch,omitempty"`
	Amount          sdk.IntProto            `protobuf:"bytes,4,opt,name=amount,proto3" json:"amount,omitempty"`
	SignerPubKey    hmTypes.PubKey          `protobuf:"bytes,5,opt,name=signer_pub_key,json=signerPubKey,proto3" json:"signer_pub_key,omitempty"`
	TxHash          hmTypes.HeimdallHash    `protobuf:"bytes,6,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
	LogIndex        uint64                  `protobuf:"varint,7,opt,name=log_index,json=logIndex,proto3" json:"log_index,omitempty"`
	BlockNumber     uint64                  `protobuf:"varint,8,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	Nonce           uint64                  `protobuf:"varint,9,opt,name=nonce,proto3" json:"nonce,omitempty"`
}

func (m *MsgValidatorJoin) Reset()         { *m = MsgValidatorJoin{} }
func (m *MsgValidatorJoin) String() string { return proto.CompactTextString(m) }
func (*MsgValidatorJoin) ProtoMessage()    {}
func (*MsgValidatorJoin) Descriptor() ([]byte, []int) {
	return fileDescriptor_0926ef28816b35ab, []int{0}
}
func (m *MsgValidatorJoin) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgValidatorJoin) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgValidatorJoin.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgValidatorJoin) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgValidatorJoin.Merge(m, src)
}
func (m *MsgValidatorJoin) XXX_Size() int {
	return m.Size()
}
func (m *MsgValidatorJoin) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgValidatorJoin.DiscardUnknown(m)
}

var xxx_messageInfo_MsgValidatorJoin proto.InternalMessageInfo

func (m *MsgValidatorJoin) GetFrom() hmTypes.HeimdallAddress {
	if m != nil {
		return m.From
	}
	return hmTypes.HeimdallAddress{}
}

func (m *MsgValidatorJoin) GetSignerPubKey() hmTypes.PubKey {
	if m != nil {
		return m.SignerPubKey
	}
	return hmTypes.PubKey{}
}

func (m *MsgValidatorJoin) GetTxHash() hmTypes.HeimdallHash {
	if m != nil {
		return m.TxHash
	}
	return hmTypes.HeimdallHash{}
}

func (m *MsgValidatorJoin) GetLogIndex() uint64 {
	if m != nil {
		return m.LogIndex
	}
	return 0
}

func (m *MsgValidatorJoin) GetBlockNumber() uint64 {
	if m != nil {
		return m.BlockNumber
	}
	return 0
}

func (m *MsgValidatorJoin) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

// MsgValidatorJoinResponse defines the Msg/ValidatorJoin response type.
type MsgValidatorJoinResponse struct {
}

func (m *MsgValidatorJoinResponse) Reset()         { *m = MsgValidatorJoinResponse{} }
func (m *MsgValidatorJoinResponse) String() string { return proto.CompactTextString(m) }
func (*MsgValidatorJoinResponse) ProtoMessage()    {}
func (*MsgValidatorJoinResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_0926ef28816b35ab, []int{1}
}
func (m *MsgValidatorJoinResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgValidatorJoinResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgValidatorJoinResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgValidatorJoinResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgValidatorJoinResponse.Merge(m, src)
}
func (m *MsgValidatorJoinResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgValidatorJoinResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgValidatorJoinResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgValidatorJoinResponse proto.InternalMessageInfo

// MsgDelegate defines a SDK message for performing a delegation of coins
// from a delegator to a validator.
type MsgStakeUpdate struct {
	From        hmTypes.HeimdallAddress `protobuf:"bytes,1,opt,name=from,proto3" json:"from,omitempty"`
	ID          hmTypes.ValidatorID     `protobuf:"bytes,2,opt,name=i_d,json=iD,proto3" json:"i_d,omitempty"`
	NewAmount   sdk.IntProto            `protobuf:"bytes,3,opt,name=new_amount,json=newAmount,proto3" json:"new_amount,omitempty"`
	TxHash      hmTypes.HeimdallHash    `protobuf:"bytes,4,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
	LogIndex    uint64                  `protobuf:"varint,5,opt,name=log_index,json=logIndex,proto3" json:"log_index,omitempty"`
	BlockNumber uint64                  `protobuf:"varint,6,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	Nonce       uint64                  `protobuf:"varint,7,opt,name=nonce,proto3" json:"nonce,omitempty"`
}

func (m *MsgStakeUpdate) Reset()         { *m = MsgStakeUpdate{} }
func (m *MsgStakeUpdate) String() string { return proto.CompactTextString(m) }
func (*MsgStakeUpdate) ProtoMessage()    {}
func (*MsgStakeUpdate) Descriptor() ([]byte, []int) {
	return fileDescriptor_0926ef28816b35ab, []int{2}
}
func (m *MsgStakeUpdate) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgStakeUpdate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgStakeUpdate.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgStakeUpdate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgStakeUpdate.Merge(m, src)
}
func (m *MsgStakeUpdate) XXX_Size() int {
	return m.Size()
}
func (m *MsgStakeUpdate) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgStakeUpdate.DiscardUnknown(m)
}

var xxx_messageInfo_MsgStakeUpdate proto.InternalMessageInfo

func (m *MsgStakeUpdate) GetFrom() hmTypes.HeimdallAddress {
	if m != nil {
		return m.From
	}
	return hmTypes.HeimdallAddress{}
}

func (m *MsgStakeUpdate) GetNewAmount() sdk.IntProto {
	if m != nil {
		return m.NewAmount
	}
	return sdk.IntProto{}
}

func (m *MsgStakeUpdate) GetTxHash() hmTypes.HeimdallHash {
	if m != nil {
		return m.TxHash
	}
	return hmTypes.HeimdallHash{}
}

func (m *MsgStakeUpdate) GetLogIndex() uint64 {
	if m != nil {
		return m.LogIndex
	}
	return 0
}

func (m *MsgStakeUpdate) GetBlockNumber() uint64 {
	if m != nil {
		return m.BlockNumber
	}
	return 0
}

func (m *MsgStakeUpdate) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

// MsgDelegateResponse defines the Msg/Delegate response type.
type MsgStakeUpdateResponse struct {
}

func (m *MsgStakeUpdateResponse) Reset()         { *m = MsgStakeUpdateResponse{} }
func (m *MsgStakeUpdateResponse) String() string { return proto.CompactTextString(m) }
func (*MsgStakeUpdateResponse) ProtoMessage()    {}
func (*MsgStakeUpdateResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_0926ef28816b35ab, []int{3}
}
func (m *MsgStakeUpdateResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgStakeUpdateResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgStakeUpdateResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgStakeUpdateResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgStakeUpdateResponse.Merge(m, src)
}
func (m *MsgStakeUpdateResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgStakeUpdateResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgStakeUpdateResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgStakeUpdateResponse proto.InternalMessageInfo

// MsgSignerUpdate defines a SDK message for updating signer of the existing
// validator
type MsgSignerUpdate struct {
	From            hmTypes.HeimdallAddress `protobuf:"bytes,1,opt,name=from,proto3" json:"from,omitempty"`
	ID              hmTypes.ValidatorID     `protobuf:"bytes,2,opt,name=i_d,json=iD,proto3" json:"i_d,omitempty"`
	NewSignerPubKey hmTypes.PubKey          `protobuf:"bytes,3,opt,name=new_signer_pub_key,json=newSignerPubKey,proto3" json:"new_signer_pub_key,omitempty"`
	TxHash          hmTypes.HeimdallHash    `protobuf:"bytes,4,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
	LogIndex        uint64                  `protobuf:"varint,5,opt,name=log_index,json=logIndex,proto3" json:"log_index,omitempty"`
	BlockNumber     uint64                  `protobuf:"varint,6,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	Nonce           uint64                  `protobuf:"varint,7,opt,name=nonce,proto3" json:"nonce,omitempty"`
}

func (m *MsgSignerUpdate) Reset()         { *m = MsgSignerUpdate{} }
func (m *MsgSignerUpdate) String() string { return proto.CompactTextString(m) }
func (*MsgSignerUpdate) ProtoMessage()    {}
func (*MsgSignerUpdate) Descriptor() ([]byte, []int) {
	return fileDescriptor_0926ef28816b35ab, []int{4}
}
func (m *MsgSignerUpdate) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgSignerUpdate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgSignerUpdate.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgSignerUpdate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgSignerUpdate.Merge(m, src)
}
func (m *MsgSignerUpdate) XXX_Size() int {
	return m.Size()
}
func (m *MsgSignerUpdate) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgSignerUpdate.DiscardUnknown(m)
}

var xxx_messageInfo_MsgSignerUpdate proto.InternalMessageInfo

func (m *MsgSignerUpdate) GetFrom() hmTypes.HeimdallAddress {
	if m != nil {
		return m.From
	}
	return hmTypes.HeimdallAddress{}
}

func (m *MsgSignerUpdate) GetTxHash() hmTypes.HeimdallHash {
	if m != nil {
		return m.TxHash
	}
	return hmTypes.HeimdallHash{}
}

func (m *MsgSignerUpdate) GetLogIndex() uint64 {
	if m != nil {
		return m.LogIndex
	}
	return 0
}

func (m *MsgSignerUpdate) GetBlockNumber() uint64 {
	if m != nil {
		return m.BlockNumber
	}
	return 0
}

func (m *MsgSignerUpdate) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

// MsgSignerUpdate defines the Msg/SignerUpdate response type.
type MsgSignerUpdateResponse struct {
}

func (m *MsgSignerUpdateResponse) Reset()         { *m = MsgSignerUpdateResponse{} }
func (m *MsgSignerUpdateResponse) String() string { return proto.CompactTextString(m) }
func (*MsgSignerUpdateResponse) ProtoMessage()    {}
func (*MsgSignerUpdateResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_0926ef28816b35ab, []int{5}
}
func (m *MsgSignerUpdateResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgSignerUpdateResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgSignerUpdateResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgSignerUpdateResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgSignerUpdateResponse.Merge(m, src)
}
func (m *MsgSignerUpdateResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgSignerUpdateResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgSignerUpdateResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgSignerUpdateResponse proto.InternalMessageInfo

// MsgValidatorExit defines a SDK message for exiting the validator
type MsgValidatorExit struct {
	From              hmTypes.HeimdallAddress `protobuf:"bytes,1,opt,name=from,proto3" json:"from,omitempty"`
	ID                hmTypes.ValidatorID     `protobuf:"bytes,2,opt,name=i_d,json=iD,proto3" json:"i_d,omitempty"`
	DeactivationEpoch uint64                  `protobuf:"varint,3,opt,name=deactivation_epoch,json=deactivationEpoch,proto3" json:"deactivation_epoch,omitempty"`
	TxHash            hmTypes.HeimdallHash    `protobuf:"bytes,6,opt,name=tx_hash,json=txHash,proto3" json:"tx_hash,omitempty"`
	LogIndex          uint64                  `protobuf:"varint,7,opt,name=log_index,json=logIndex,proto3" json:"log_index,omitempty"`
	BlockNumber       uint64                  `protobuf:"varint,8,opt,name=block_number,json=blockNumber,proto3" json:"block_number,omitempty"`
	Nonce             uint64                  `protobuf:"varint,9,opt,name=nonce,proto3" json:"nonce,omitempty"`
}

func (m *MsgValidatorExit) Reset()         { *m = MsgValidatorExit{} }
func (m *MsgValidatorExit) String() string { return proto.CompactTextString(m) }
func (*MsgValidatorExit) ProtoMessage()    {}
func (*MsgValidatorExit) Descriptor() ([]byte, []int) {
	return fileDescriptor_0926ef28816b35ab, []int{6}
}
func (m *MsgValidatorExit) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgValidatorExit) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgValidatorExit.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgValidatorExit) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgValidatorExit.Merge(m, src)
}
func (m *MsgValidatorExit) XXX_Size() int {
	return m.Size()
}
func (m *MsgValidatorExit) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgValidatorExit.DiscardUnknown(m)
}

var xxx_messageInfo_MsgValidatorExit proto.InternalMessageInfo

func (m *MsgValidatorExit) GetFrom() hmTypes.HeimdallAddress {
	if m != nil {
		return m.From
	}
	return hmTypes.HeimdallAddress{}
}

func (m *MsgValidatorExit) GetTxHash() hmTypes.HeimdallHash {
	if m != nil {
		return m.TxHash
	}
	return hmTypes.HeimdallHash{}
}

func (m *MsgValidatorExit) GetLogIndex() uint64 {
	if m != nil {
		return m.LogIndex
	}
	return 0
}

func (m *MsgValidatorExit) GetBlockNumber() uint64 {
	if m != nil {
		return m.BlockNumber
	}
	return 0
}

func (m *MsgValidatorExit) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

// MsgValidatorExit defines the Msg/ValidatorExit response type.
type MsgValidatorExitResponse struct {
}

func (m *MsgValidatorExitResponse) Reset()         { *m = MsgValidatorExitResponse{} }
func (m *MsgValidatorExitResponse) String() string { return proto.CompactTextString(m) }
func (*MsgValidatorExitResponse) ProtoMessage()    {}
func (*MsgValidatorExitResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_0926ef28816b35ab, []int{7}
}
func (m *MsgValidatorExitResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MsgValidatorExitResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MsgValidatorExitResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MsgValidatorExitResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MsgValidatorExitResponse.Merge(m, src)
}
func (m *MsgValidatorExitResponse) XXX_Size() int {
	return m.Size()
}
func (m *MsgValidatorExitResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MsgValidatorExitResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MsgValidatorExitResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*MsgValidatorJoin)(nil), "cosmos.staking.v1beta1.MsgValidatorJoin")
	proto.RegisterType((*MsgValidatorJoinResponse)(nil), "cosmos.staking.v1beta1.MsgValidatorJoinResponse")
	proto.RegisterType((*MsgStakeUpdate)(nil), "cosmos.staking.v1beta1.MsgStakeUpdate")
	proto.RegisterType((*MsgStakeUpdateResponse)(nil), "cosmos.staking.v1beta1.MsgStakeUpdateResponse")
	proto.RegisterType((*MsgSignerUpdate)(nil), "cosmos.staking.v1beta1.MsgSignerUpdate")
	proto.RegisterType((*MsgSignerUpdateResponse)(nil), "cosmos.staking.v1beta1.MsgSignerUpdateResponse")
	proto.RegisterType((*MsgValidatorExit)(nil), "cosmos.staking.v1beta1.MsgValidatorExit")
	proto.RegisterType((*MsgValidatorExitResponse)(nil), "cosmos.staking.v1beta1.MsgValidatorExitResponse")
}

func init() { proto.RegisterFile("cosmos/staking/v1beta1/tx.proto", fileDescriptor_0926ef28816b35ab) }

var fileDescriptor_0926ef28816b35ab = []byte{
	// 711 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xcc, 0x96, 0x3d, 0x6f, 0xd3, 0x40,
	0x18, 0xc7, 0xe3, 0xbc, 0xb5, 0xb9, 0xbe, 0x9f, 0x50, 0xeb, 0xb8, 0x22, 0xa9, 0x22, 0xd4, 0x46,
	0x05, 0xec, 0xb6, 0x30, 0xc1, 0x44, 0x45, 0x11, 0x2f, 0x2a, 0x42, 0xad, 0x60, 0x60, 0xb1, 0xce,
	0xf1, 0xd5, 0x39, 0x39, 0xbe, 0xb3, 0x72, 0x97, 0x36, 0xd9, 0x10, 0x13, 0x82, 0x85, 0x0f, 0xc0,
	0xd0, 0x91, 0x81, 0xa1, 0x1b, 0x5f, 0x81, 0x8d, 0x8e, 0x8c, 0xa8, 0x1d, 0xca, 0xc7, 0x40, 0x7e,
	0x49, 0x38, 0x5b, 0xa5, 0xb5, 0xba, 0xc0, 0x92, 0x58, 0xff, 0xff, 0xdf, 0xf7, 0x9c, 0x7e, 0x7e,
	0xfc, 0x9c, 0x41, 0xbd, 0xc5, 0xb8, 0xc7, 0xb8, 0xc1, 0x05, 0x72, 0x09, 0x75, 0x8c, 0xfd, 0x75,
	0x0b, 0x0b, 0xb4, 0x6e, 0x88, 0xbe, 0xee, 0x77, 0x99, 0x60, 0x70, 0x3e, 0x0a, 0xe8, 0x71, 0x40,
	0x8f, 0x03, 0x5a, 0xd5, 0x61, 0xcc, 0xe9, 0x60, 0x23, 0x4c, 0x59, 0xbd, 0x3d, 0x03, 0xd1, 0x41,
	0x74, 0x8b, 0x56, 0x4f, 0x5b, 0x82, 0x78, 0x98, 0x0b, 0xe4, 0xf9, 0x71, 0xe0, 0x9a, 0xc3, 0x1c,
	0x16, 0x5e, 0x1a, 0xc1, 0x55, 0xac, 0x56, 0xa3, 0x4a, 0x66, 0x64, 0xc4, 0x65, 0x23, 0x6b, 0x21,
	0xde, 0xa5, 0xc7, 0x83, 0x1d, 0x06, 0x7f, 0xb1, 0x31, 0x87, 0x3c, 0x42, 0x99, 0x11, 0xfe, 0x46,
	0x52, 0xe3, 0x43, 0x01, 0xcc, 0x6e, 0x73, 0xe7, 0x15, 0xea, 0x10, 0x1b, 0x09, 0xd6, 0x7d, 0xca,
	0x08, 0x85, 0x55, 0x50, 0xdc, 0xeb, 0x32, 0x4f, 0x55, 0x96, 0x94, 0x66, 0x65, 0xb3, 0xf4, 0xf9,
	0xec, 0x68, 0x55, 0xd9, 0x09, 0x25, 0x38, 0x0f, 0x0a, 0xc4, 0xb4, 0xd5, 0xbc, 0xec, 0xe4, 0xc9,
	0x43, 0xb8, 0x06, 0x66, 0x51, 0x4b, 0x90, 0x7d, 0x24, 0x08, 0xa3, 0x26, 0xf6, 0x59, 0xab, 0xad,
	0x16, 0x96, 0x94, 0x66, 0x71, 0x18, 0x9a, 0xf9, 0x63, 0x6f, 0x05, 0x2e, 0xbc, 0x0e, 0xca, 0xc8,
	0x63, 0x3d, 0x2a, 0xd4, 0xa2, 0xbc, 0x58, 0x2c, 0xc2, 0x9b, 0x60, 0x9a, 0x13, 0x87, 0xe2, 0xae,
	0xe9, 0xf7, 0x2c, 0xd3, 0xc5, 0x03, 0xb5, 0x24, 0xc7, 0x26, 0x23, 0xf3, 0x45, 0xcf, 0x7a, 0x86,
	0x07, 0xb0, 0x06, 0xc6, 0x44, 0xdf, 0x6c, 0x23, 0xde, 0x56, 0xcb, 0x89, 0xc5, 0x44, 0xff, 0x31,
	0xe2, 0x6d, 0xd8, 0x00, 0x95, 0x0e, 0x73, 0x4c, 0x42, 0x6d, 0xdc, 0x57, 0xc7, 0xe4, 0x6d, 0x8d,
	0x77, 0x98, 0xf3, 0x24, 0x90, 0x61, 0x13, 0x4c, 0x5a, 0x1d, 0xd6, 0x72, 0x4d, 0xda, 0xf3, 0x2c,
	0xdc, 0x55, 0xc7, 0xe5, 0xd8, 0x44, 0x68, 0x3d, 0x0f, 0x1d, 0xb8, 0x08, 0x4a, 0x94, 0xd1, 0x16,
	0x56, 0x2b, 0x72, 0x24, 0xd2, 0xee, 0xdd, 0x7f, 0x77, 0x58, 0x57, 0x7e, 0x1d, 0xd6, 0x73, 0x6f,
	0xcf, 0x8e, 0x56, 0xe7, 0xf6, 0x87, 0x5c, 0x4d, 0x64, 0xdb, 0x5d, 0xcc, 0xf9, 0xfb, 0xb3, 0xa3,
	0x55, 0x75, 0xd8, 0x3e, 0x69, 0xf0, 0x0d, 0x0d, 0xa8, 0x69, 0x6d, 0x07, 0x73, 0x9f, 0x51, 0x8e,
	0x1b, 0x9f, 0xf2, 0x60, 0x7a, 0x9b, 0x3b, 0xbb, 0x02, 0xb9, 0xf8, 0xa5, 0x6f, 0x23, 0x81, 0xaf,
	0xf2, 0x9c, 0x6e, 0x00, 0x40, 0xf1, 0x81, 0x19, 0x93, 0x2f, 0xc8, 0x76, 0x85, 0xe2, 0x83, 0x07,
	0x11, 0x7c, 0x89, 0x67, 0xf1, 0x52, 0x9e, 0xa5, 0x6c, 0x3c, 0xcb, 0x97, 0xf3, 0x1c, 0x3b, 0x87,
	0xe7, 0xf2, 0x90, 0x67, 0x40, 0x2e, 0x6e, 0xfa, 0xdb, 0xdc, 0x76, 0x8d, 0x24, 0x8b, 0x86, 0x0a,
	0xe6, 0x93, 0xca, 0x08, 0xdc, 0x97, 0x3c, 0x98, 0x09, 0xac, 0xb0, 0x61, 0xae, 0x4e, 0x6e, 0x03,
	0xc0, 0x80, 0x5c, 0xaa, 0x29, 0x13, 0x04, 0x67, 0x28, 0x3e, 0xd8, 0xfd, 0x4b, 0x5f, 0xfe, 0x6b,
	0x8e, 0x2b, 0x32, 0x47, 0x2d, 0xc5, 0x51, 0x42, 0xd3, 0xa8, 0x82, 0x85, 0x94, 0x34, 0x22, 0xf9,
	0x3d, 0x9f, 0x1c, 0x16, 0x5b, 0x7d, 0x22, 0xae, 0x82, 0xf2, 0x2e, 0x80, 0x36, 0xbe, 0x78, 0x5c,
	0xcc, 0xc9, 0x81, 0x4c, 0x03, 0xe3, 0x3f, 0x9a, 0x01, 0x4d, 0x99, 0xf5, 0x62, 0x92, 0x75, 0x02,
	0x5e, 0xfa, 0x85, 0x0f, 0xb4, 0x21, 0xed, 0x8d, 0xaf, 0x05, 0x50, 0xd8, 0xe6, 0x0e, 0x74, 0xc1,
	0x54, 0x30, 0x08, 0x46, 0x21, 0xd8, 0xd4, 0xcf, 0x3f, 0x65, 0xf4, 0xf4, 0xec, 0xd0, 0xd6, 0xb2,
	0x26, 0x87, 0x45, 0x21, 0x06, 0x13, 0xf2, 0x84, 0x59, 0xbe, 0x60, 0x01, 0x29, 0xa7, 0xe9, 0xd9,
	0x72, 0xa3, 0x32, 0x6d, 0x30, 0x99, 0x78, 0x1f, 0x57, 0x2e, 0xba, 0x5f, 0x0a, 0x6a, 0x46, 0xc6,
	0xe0, 0xa8, 0x92, 0x0b, 0xa6, 0x92, 0xfd, 0x9a, 0x89, 0x5e, 0x90, 0xcc, 0x46, 0x4f, 0x7e, 0x64,
	0x5a, 0xe9, 0x4d, 0xd0, 0x06, 0x9b, 0x8f, 0xbe, 0x9d, 0xd4, 0x94, 0xe3, 0x93, 0x9a, 0xf2, 0xf3,
	0xa4, 0xa6, 0x7c, 0x3c, 0xad, 0xe5, 0x8e, 0x4f, 0x6b, 0xb9, 0x1f, 0xa7, 0xb5, 0xdc, 0xeb, 0x5b,
	0x0e, 0x11, 0xed, 0x9e, 0xa5, 0xb7, 0x98, 0x17, 0x9f, 0xd9, 0x86, 0xd4, 0x1e, 0xfd, 0xd1, 0x87,
	0x85, 0x18, 0xf8, 0x98, 0x5b, 0xe5, 0xf0, 0x8c, 0xbe, 0xf3, 0x3b, 0x00, 0x00, 0xff, 0xff, 0x5b,
	0xe3, 0x2f, 0xbc, 0x77, 0x08, 0x00, 0x00,
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
	// JoinValidator defines a method for joining a new validator.
	JoinValidator(ctx context.Context, in *MsgValidatorJoin, opts ...grpc.CallOption) (*MsgValidatorJoinResponse, error)
	// StakeUpdate defines a method for updating an existing validator's stake.
	StakeUpdate(ctx context.Context, in *MsgStakeUpdate, opts ...grpc.CallOption) (*MsgStakeUpdateResponse, error)
	// v defines a method for updating an existing validator's signer.
	SignerUpdate(ctx context.Context, in *MsgSignerUpdate, opts ...grpc.CallOption) (*MsgSignerUpdateResponse, error)
	// ValidatorExit defines a method for exiting an existing validator
	ValidatorExit(ctx context.Context, in *MsgValidatorExit, opts ...grpc.CallOption) (*MsgValidatorExitResponse, error)
}

type msgClient struct {
	cc grpc1.ClientConn
}

func NewMsgClient(cc grpc1.ClientConn) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) JoinValidator(ctx context.Context, in *MsgValidatorJoin, opts ...grpc.CallOption) (*MsgValidatorJoinResponse, error) {
	out := new(MsgValidatorJoinResponse)
	err := c.cc.Invoke(ctx, "/cosmos.staking.v1beta1.Msg/JoinValidator", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) StakeUpdate(ctx context.Context, in *MsgStakeUpdate, opts ...grpc.CallOption) (*MsgStakeUpdateResponse, error) {
	out := new(MsgStakeUpdateResponse)
	err := c.cc.Invoke(ctx, "/cosmos.staking.v1beta1.Msg/StakeUpdate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) SignerUpdate(ctx context.Context, in *MsgSignerUpdate, opts ...grpc.CallOption) (*MsgSignerUpdateResponse, error) {
	out := new(MsgSignerUpdateResponse)
	err := c.cc.Invoke(ctx, "/cosmos.staking.v1beta1.Msg/SignerUpdate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) ValidatorExit(ctx context.Context, in *MsgValidatorExit, opts ...grpc.CallOption) (*MsgValidatorExitResponse, error) {
	out := new(MsgValidatorExitResponse)
	err := c.cc.Invoke(ctx, "/cosmos.staking.v1beta1.Msg/ValidatorExit", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
type MsgServer interface {
	// JoinValidator defines a method for joining a new validator.
	JoinValidator(context.Context, *MsgValidatorJoin) (*MsgValidatorJoinResponse, error)
	// StakeUpdate defines a method for updating an existing validator's stake.
	StakeUpdate(context.Context, *MsgStakeUpdate) (*MsgStakeUpdateResponse, error)
	// v defines a method for updating an existing validator's signer.
	SignerUpdate(context.Context, *MsgSignerUpdate) (*MsgSignerUpdateResponse, error)
	// ValidatorExit defines a method for exiting an existing validator
	ValidatorExit(context.Context, *MsgValidatorExit) (*MsgValidatorExitResponse, error)
}

// UnimplementedMsgServer can be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (*UnimplementedMsgServer) JoinValidator(ctx context.Context, req *MsgValidatorJoin) (*MsgValidatorJoinResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method JoinValidator not implemented")
}
func (*UnimplementedMsgServer) StakeUpdate(ctx context.Context, req *MsgStakeUpdate) (*MsgStakeUpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StakeUpdate not implemented")
}
func (*UnimplementedMsgServer) SignerUpdate(ctx context.Context, req *MsgSignerUpdate) (*MsgSignerUpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SignerUpdate not implemented")
}
func (*UnimplementedMsgServer) ValidatorExit(ctx context.Context, req *MsgValidatorExit) (*MsgValidatorExitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ValidatorExit not implemented")
}

func RegisterMsgServer(s grpc1.Server, srv MsgServer) {
	s.RegisterService(&_Msg_serviceDesc, srv)
}

func _Msg_JoinValidator_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgValidatorJoin)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).JoinValidator(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cosmos.staking.v1beta1.Msg/JoinValidator",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).JoinValidator(ctx, req.(*MsgValidatorJoin))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_StakeUpdate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgStakeUpdate)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).StakeUpdate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cosmos.staking.v1beta1.Msg/StakeUpdate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).StakeUpdate(ctx, req.(*MsgStakeUpdate))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_SignerUpdate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSignerUpdate)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).SignerUpdate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cosmos.staking.v1beta1.Msg/SignerUpdate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).SignerUpdate(ctx, req.(*MsgSignerUpdate))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_ValidatorExit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgValidatorExit)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ValidatorExit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cosmos.staking.v1beta1.Msg/ValidatorExit",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ValidatorExit(ctx, req.(*MsgValidatorExit))
	}
	return interceptor(ctx, in, info, handler)
}

var _Msg_serviceDesc = grpc.ServiceDesc{
	ServiceName: "cosmos.staking.v1beta1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "JoinValidator",
			Handler:    _Msg_JoinValidator_Handler,
		},
		{
			MethodName: "StakeUpdate",
			Handler:    _Msg_StakeUpdate_Handler,
		},
		{
			MethodName: "SignerUpdate",
			Handler:    _Msg_SignerUpdate_Handler,
		},
		{
			MethodName: "ValidatorExit",
			Handler:    _Msg_ValidatorExit_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cosmos/staking/v1beta1/tx.proto",
}

func (m *MsgValidatorJoin) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgValidatorJoin) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgValidatorJoin) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Nonce != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.Nonce))
		i--
		dAtA[i] = 0x48
	}
	if m.BlockNumber != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.BlockNumber))
		i--
		dAtA[i] = 0x40
	}
	if m.LogIndex != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.LogIndex))
		i--
		dAtA[i] = 0x38
	}
	if len(m.TxHash) > 0 {
		i -= len(m.TxHash)
		copy(dAtA[i:], m.TxHash)
		i = encodeVarintTx(dAtA, i, uint64(len(m.TxHash)))
		i--
		dAtA[i] = 0x32
	}
	if len(m.SignerPubKey) > 0 {
		i -= len(m.SignerPubKey)
		copy(dAtA[i:], m.SignerPubKey)
		i = encodeVarintTx(dAtA, i, uint64(len(m.SignerPubKey)))
		i--
		dAtA[i] = 0x2a
	}
	if len(m.Amount) > 0 {
		i -= len(m.Amount)
		copy(dAtA[i:], m.Amount)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Amount)))
		i--
		dAtA[i] = 0x22
	}
	if m.ActivationEpoch != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.ActivationEpoch))
		i--
		dAtA[i] = 0x18
	}
	if len(m.ID) > 0 {
		i -= len(m.ID)
		copy(dAtA[i:], m.ID)
		i = encodeVarintTx(dAtA, i, uint64(len(m.ID)))
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

func (m *MsgValidatorJoinResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgValidatorJoinResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgValidatorJoinResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *MsgStakeUpdate) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgStakeUpdate) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgStakeUpdate) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Nonce != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.Nonce))
		i--
		dAtA[i] = 0x38
	}
	if m.BlockNumber != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.BlockNumber))
		i--
		dAtA[i] = 0x30
	}
	if m.LogIndex != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.LogIndex))
		i--
		dAtA[i] = 0x28
	}
	if len(m.TxHash) > 0 {
		i -= len(m.TxHash)
		copy(dAtA[i:], m.TxHash)
		i = encodeVarintTx(dAtA, i, uint64(len(m.TxHash)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.NewAmount) > 0 {
		i -= len(m.NewAmount)
		copy(dAtA[i:], m.NewAmount)
		i = encodeVarintTx(dAtA, i, uint64(len(m.NewAmount)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.ID) > 0 {
		i -= len(m.ID)
		copy(dAtA[i:], m.ID)
		i = encodeVarintTx(dAtA, i, uint64(len(m.ID)))
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

func (m *MsgStakeUpdateResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgStakeUpdateResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgStakeUpdateResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *MsgSignerUpdate) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgSignerUpdate) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgSignerUpdate) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Nonce != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.Nonce))
		i--
		dAtA[i] = 0x38
	}
	if m.BlockNumber != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.BlockNumber))
		i--
		dAtA[i] = 0x30
	}
	if m.LogIndex != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.LogIndex))
		i--
		dAtA[i] = 0x28
	}
	if len(m.TxHash) > 0 {
		i -= len(m.TxHash)
		copy(dAtA[i:], m.TxHash)
		i = encodeVarintTx(dAtA, i, uint64(len(m.TxHash)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.NewSignerPubKey) > 0 {
		i -= len(m.NewSignerPubKey)
		copy(dAtA[i:], m.NewSignerPubKey)
		i = encodeVarintTx(dAtA, i, uint64(len(m.NewSignerPubKey)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.ID) > 0 {
		i -= len(m.ID)
		copy(dAtA[i:], m.ID)
		i = encodeVarintTx(dAtA, i, uint64(len(m.ID)))
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

func (m *MsgSignerUpdateResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgSignerUpdateResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgSignerUpdateResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	return len(dAtA) - i, nil
}

func (m *MsgValidatorExit) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgValidatorExit) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgValidatorExit) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Nonce != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.Nonce))
		i--
		dAtA[i] = 0x48
	}
	if m.BlockNumber != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.BlockNumber))
		i--
		dAtA[i] = 0x40
	}
	if m.LogIndex != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.LogIndex))
		i--
		dAtA[i] = 0x38
	}
	if len(m.TxHash) > 0 {
		i -= len(m.TxHash)
		copy(dAtA[i:], m.TxHash)
		i = encodeVarintTx(dAtA, i, uint64(len(m.TxHash)))
		i--
		dAtA[i] = 0x32
	}
	if len(m.Amount) > 0 {
		i -= len(m.Amount)
		copy(dAtA[i:], m.Amount)
		i = encodeVarintTx(dAtA, i, uint64(len(m.Amount)))
		i--
		dAtA[i] = 0x22
	}
	if m.DeactivationEpoch != 0 {
		i = encodeVarintTx(dAtA, i, uint64(m.DeactivationEpoch))
		i--
		dAtA[i] = 0x18
	}
	if len(m.ID) > 0 {
		i -= len(m.ID)
		copy(dAtA[i:], m.ID)
		i = encodeVarintTx(dAtA, i, uint64(len(m.ID)))
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

func (m *MsgValidatorExitResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MsgValidatorExitResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *MsgValidatorExitResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
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
func (m *MsgValidatorJoin) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.From)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.ID)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	if m.ActivationEpoch != 0 {
		n += 1 + sovTx(uint64(m.ActivationEpoch))
	}
	l = len(m.Amount)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.SignerPubKey)
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
	if m.Nonce != 0 {
		n += 1 + sovTx(uint64(m.Nonce))
	}
	return n
}

func (m *MsgValidatorJoinResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *MsgStakeUpdate) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.From)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.ID)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.NewAmount)
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
	if m.Nonce != 0 {
		n += 1 + sovTx(uint64(m.Nonce))
	}
	return n
}

func (m *MsgStakeUpdateResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *MsgSignerUpdate) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.From)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.ID)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.NewSignerPubKey)
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
	if m.Nonce != 0 {
		n += 1 + sovTx(uint64(m.Nonce))
	}
	return n
}

func (m *MsgSignerUpdateResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func (m *MsgValidatorExit) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.From)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	l = len(m.ID)
	if l > 0 {
		n += 1 + l + sovTx(uint64(l))
	}
	if m.DeactivationEpoch != 0 {
		n += 1 + sovTx(uint64(m.DeactivationEpoch))
	}
	l = len(m.Amount)
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
	if m.Nonce != 0 {
		n += 1 + sovTx(uint64(m.Nonce))
	}
	return n
}

func (m *MsgValidatorExitResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	return n
}

func sovTx(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozTx(x uint64) (n int) {
	return sovTx(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MsgValidatorJoin) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgValidatorJoin: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgValidatorJoin: illegal tag %d (wire type %d)", fieldNum, wire)
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
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
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
			m.ID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ActivationEpoch", wireType)
			}
			m.ActivationEpoch = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ActivationEpoch |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Amount", wireType)
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
			m.Amount = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field SignerPubKey", wireType)
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
			m.SignerPubKey = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
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
		case 7:
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
		case 8:
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
		case 9:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nonce", wireType)
			}
			m.Nonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
func (m *MsgValidatorJoinResponse) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgValidatorJoinResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgValidatorJoinResponse: illegal tag %d (wire type %d)", fieldNum, wire)
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
func (m *MsgStakeUpdate) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgStakeUpdate: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgStakeUpdate: illegal tag %d (wire type %d)", fieldNum, wire)
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
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
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
			m.ID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field NewAmount", wireType)
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
			m.NewAmount = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
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
		case 5:
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
		case 6:
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
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nonce", wireType)
			}
			m.Nonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
func (m *MsgStakeUpdateResponse) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgStakeUpdateResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgStakeUpdateResponse: illegal tag %d (wire type %d)", fieldNum, wire)
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
func (m *MsgSignerUpdate) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgSignerUpdate: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgSignerUpdate: illegal tag %d (wire type %d)", fieldNum, wire)
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
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
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
			m.ID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field NewSignerPubKey", wireType)
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
			m.NewSignerPubKey = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
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
		case 5:
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
		case 6:
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
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nonce", wireType)
			}
			m.Nonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
func (m *MsgSignerUpdateResponse) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgSignerUpdateResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgSignerUpdateResponse: illegal tag %d (wire type %d)", fieldNum, wire)
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
func (m *MsgValidatorExit) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgValidatorExit: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgValidatorExit: illegal tag %d (wire type %d)", fieldNum, wire)
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
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
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
			m.ID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field DeactivationEpoch", wireType)
			}
			m.DeactivationEpoch = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.DeactivationEpoch |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Amount", wireType)
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
			m.Amount = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
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
		case 7:
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
		case 8:
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
		case 9:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Nonce", wireType)
			}
			m.Nonce = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowTx
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
func (m *MsgValidatorExitResponse) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: MsgValidatorExitResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MsgValidatorExitResponse: illegal tag %d (wire type %d)", fieldNum, wire)
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
