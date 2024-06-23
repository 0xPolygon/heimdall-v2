package types

import (
	"bytes"
	"math/big"
	"strconv"

	"cosmossdk.io/core/address"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	types "github.com/0xPolygon/heimdall-v2/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgCheckpointAdjust{}
	_ sdk.Msg = &MsgCheckpoint{}
	_ sdk.Msg = &MsgCheckpointAck{}
	_ sdk.Msg = &MsgCheckpointNoAck{}
)

// NewMsgCheckpointAdjust adjust previous checkpoint
func NewMsgCheckpointAdjust(
	headerIndex uint64,
	startBlock uint64,
	endBlock uint64,
	proposer string,
	from string,
	rootHash hmTypes.HeimdallHash,
) MsgCheckpointAdjust {
	return MsgCheckpointAdjust{
		HeaderIndex: headerIndex,
		StartBlock:  startBlock,
		EndBlock:    endBlock,
		Proposer:    proposer,
		From:        from,
		RootHash:    rootHash,
	}
}

func (msg MsgCheckpointAdjust) GetSignBytes() []byte {
	return nil
}

// Type returns message type
func (msg MsgCheckpointAdjust) Type() string {
	return EventTypeCheckpointAdjust
}

func (msg MsgCheckpointAdjust) ValidateBasic(ac address.Codec) error {
	if bytes.Equal(msg.RootHash.Bytes(), hmTypes.ZeroHeimdallHash) {
		return ErrInvalidMsg.Wrapf("Invalid roothash %v", msg.RootHash.String())
	}

	addrBytes, err := ac.StringToBytes(msg.Proposer)
	if err != nil {
		return ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.Proposer)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.Proposer)
	}

	if msg.StartBlock >= msg.EndBlock || msg.EndBlock == 0 {
		return ErrInvalidMsg.Wrapf("End should be greater than to start block start block=%s,end block=%s", msg.StartBlock, msg.EndBlock)
	}

	return nil
}

// GetSideSignBytes returns side sign bytes
func (msg MsgCheckpointAdjust) GetSideSignBytes() []byte {
	return nil
}

// NewMsgCheckpointBlock creates new checkpoint message using mentioned arguments
func NewMsgCheckpointBlock(
	proposer string,
	startBlock uint64,
	endBlock uint64,
	roothash hmTypes.HeimdallHash,
	accountRootHash hmTypes.HeimdallHash,
	borChainID string,
) MsgCheckpoint {
	return MsgCheckpoint{
		Proposer:        proposer,
		StartBlock:      startBlock,
		EndBlock:        endBlock,
		RootHash:        roothash,
		AccountRootHash: accountRootHash,
		BorChainID:      borChainID,
	}
}

// Type returns message type
func (msg MsgCheckpoint) Type() string {
	return EventTypeCheckpoint
}

func (msg MsgCheckpoint) ValidateBasic(ac address.Codec) error {
	if bytes.Equal(msg.RootHash.Bytes(), hmTypes.ZeroHeimdallHash) {
		return ErrInvalidMsg.Wrapf("Invalid roothash %v", msg.RootHash.String())
	}

	addrBytes, err := ac.StringToBytes(msg.Proposer)
	if err != nil {
		return ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.Proposer)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.Proposer)
	}

	if msg.StartBlock >= msg.EndBlock || msg.EndBlock == 0 {
		return ErrInvalidMsg.Wrapf("End should be greater than to start block start block=%s,end block=%s", msg.StartBlock, msg.EndBlock)
	}

	return nil
}

// GetSideSignBytes returns side sign bytes
func (msg MsgCheckpoint) GetSideSignBytes() []byte {
	// keccak256(abi.encoded(proposer, startBlock, endBlock, rootHash, accountRootHash, bor chain id))
	borChainID, _ := strconv.ParseUint(msg.BorChainID, 10, 64)

	return appendBytes32(
		[]byte(msg.Proposer),
		new(big.Int).SetUint64(msg.StartBlock).Bytes(),
		new(big.Int).SetUint64(msg.EndBlock).Bytes(),
		msg.RootHash.Bytes(),
		msg.AccountRootHash.Bytes(),
		new(big.Int).SetUint64(borChainID).Bytes(),
	)
}

//
// Msg Checkpoint Ack
//

var _ sdk.Msg = &MsgCheckpointAck{}

func NewMsgCheckpointAck(
	from string,
	number uint64,
	proposer string,
	startBlock uint64,
	endBlock uint64,
	rootHash types.HeimdallHash,
	txHash types.HeimdallHash,
	logIndex uint64,
) MsgCheckpointAck {
	return MsgCheckpointAck{
		From:       from,
		Number:     number,
		Proposer:   proposer,
		StartBlock: startBlock,
		EndBlock:   endBlock,
		RootHash:   rootHash,
		TxHash:     txHash,
		LogIndex:   logIndex,
	}
}

func (msg MsgCheckpointAck) Type() string {
	return EventTypeCheckpointAck
}

// ValidateBasic validate basic
func (msg MsgCheckpointAck) ValidateBasic(ac address.Codec) error {
	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return ErrInvalidMsg.Wrapf("Invalid sender %s", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid sender %s", msg.From)
	}

	addrBytes, err = ac.StringToBytes(msg.Proposer)
	if err != nil {
		return ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.Proposer)
	}

	accAddr = sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.Proposer)
	}

	if bytes.Equal(msg.RootHash.Bytes(), hmTypes.ZeroHeimdallHash) {
		return ErrInvalidMsg.Wrapf("Invalid roothash %v", msg.RootHash.String())
	}

	return nil
}

// GetSideSignBytes returns side sign bytes
func (msg MsgCheckpointAck) GetSideSignBytes() []byte {
	return nil
}

var _ sdk.Msg = &MsgCheckpointNoAck{}

func NewMsgCheckpointNoAck(from string) MsgCheckpointNoAck {
	return MsgCheckpointNoAck{
		From: from,
	}
}

func (msg MsgCheckpointNoAck) Type() string {
	return EventTypeCheckpointNoAck
}

func (msg MsgCheckpointNoAck) ValidateBasic(ac address.Codec) error {
	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return ErrInvalidMsg.Wrapf("Invalid sender %s", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid sender %s", msg.From)
	}

	return nil
}
