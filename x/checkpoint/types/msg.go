package types

import (
	"bytes"
	"math/big"
	"strconv"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/types"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
)

var (
	_ sdk.Msg = &MsgCheckpoint{}
	_ sdk.Msg = &MsgCheckpointAck{}
	_ sdk.Msg = &MsgCheckpointNoAck{}
)

// NewMsgCheckpointBlock creates new checkpoint message using mentioned arguments
func NewMsgCheckpointBlock(
	proposer string,
	startBlock uint64,
	endBlock uint64,
	rootHash hmTypes.HeimdallHash,
	accountRootHash hmTypes.HeimdallHash,
	borChainID string,
) *MsgCheckpoint {
	return &MsgCheckpoint{
		Proposer:        proposer,
		StartBlock:      startBlock,
		EndBlock:        endBlock,
		RootHash:        rootHash,
		AccountRootHash: accountRootHash,
		BorChainId:      borChainID,
	}
}

func (msg MsgCheckpoint) ValidateBasic(ac address.Codec) error {
	if bytes.Equal(msg.RootHash.GetHash(), ZeroHeimdallHash.GetHash()) {
		return ErrInvalidMsg.Wrapf("Invalid roothash %v", msg.RootHash.String())
	}

	if len(msg.RootHash.Hash) != types.HashLength {
		return ErrInvalidMsg.Wrapf("Invalid roothash length %v", len(msg.RootHash.Hash))
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
		return ErrInvalidMsg.Wrapf("End should be greater than to start block start block=%d,end block=%d", msg.StartBlock, msg.EndBlock)
	}

	return nil
}

// GetSideSignBytes returns side sign bytes
func (msg MsgCheckpoint) GetSideSignBytes() []byte {
	// keccak256(abi.encoded(proposer, startBlock, endBlock, rootHash, accountRootHash, bor chain id))
	borChainID, _ := strconv.ParseUint(msg.BorChainId, 10, 64)

	return types.AppendBytes32(
		[]byte(msg.Proposer),
		new(big.Int).SetUint64(msg.StartBlock).Bytes(),
		new(big.Int).SetUint64(msg.EndBlock).Bytes(),
		msg.RootHash.GetHash(),
		msg.AccountRootHash.GetHash(),
		new(big.Int).SetUint64(borChainID).Bytes(),
	)
}

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

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.Proposer)
	}

	if bytes.Equal(msg.RootHash.GetHash(), ZeroHeimdallHash.GetHash()) {
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
