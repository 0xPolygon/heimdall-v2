package types

import (
	"bytes"
	"math/big"
	"strconv"

	"cosmossdk.io/core/address"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	heimdallError "github.com/0xPolygon/heimdall-v2/types/error"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgMilestone{}

// NewMsgMilestoneBlock creates new milestone message using mentioned arguments
func NewMsgMilestoneBlock(
	proposer string,
	startBlock uint64,
	endBlock uint64,
	hash hmTypes.HeimdallHash,
	borChainID string,
	milestoneID string,
) MsgMilestone {
	return MsgMilestone{
		Proposer:    proposer,
		StartBlock:  startBlock,
		EndBlock:    endBlock,
		Hash:        hash,
		BorChainID:  borChainID,
		MilestoneID: milestoneID,
	}
}

// Type returns message type
func (msg MsgMilestone) Type() string {
	return EventTypeMilestone
}

func (msg MsgMilestone) Route() string {
	return RouterKey
}

// GetSigners returns address of the signer
func (msg MsgMilestone) GetSigners() string {
	return msg.Proposer
}

func (msg MsgMilestone) ValidateBasic(ac address.Codec) error {
	if bytes.Equal(msg.Hash.Bytes(), hmTypes.ZeroHeimdallHash) {
		return heimdallError.ErrInvalidMsg.Wrapf("Invalid roothash %v", msg.Hash.String())
	}

	addrBytes, err := ac.StringToBytes(msg.Proposer)
	if err != nil {
		return heimdallError.ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.Proposer)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return heimdallError.ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.Proposer)
	}

	if msg.StartBlock >= msg.EndBlock || msg.EndBlock == 0 {
		return heimdallError.ErrInvalidMsg.Wrapf("End should be greater than to start block start block=%s,end block=%s", msg.StartBlock, msg.EndBlock)
	}

	return nil
}

// GetSideSignBytes returns side sign bytes
func (msg MsgMilestone) GetSideSignBytes() []byte {
	// keccak256(abi.encoded(proposer, startBlock, endBlock, rootHash, accountRootHash, bor chain id))
	borChainID, _ := strconv.ParseUint(msg.BorChainID, 10, 64)

	return appendBytes32(
		[]byte(msg.Proposer),
		new(big.Int).SetUint64(msg.StartBlock).Bytes(),
		new(big.Int).SetUint64(msg.EndBlock).Bytes(),
		msg.Hash.Bytes(),
		new(big.Int).SetUint64(borChainID).Bytes(),
		[]byte(msg.MilestoneID),
	)
}

var _ sdk.Msg = &MsgMilestoneTimeout{}

func NewMsgMilestoneTimeout(from string) MsgMilestoneTimeout {
	return MsgMilestoneTimeout{
		From: from,
	}
}

func (msg MsgMilestoneTimeout) Type() string {
	return "milestone-timeout"
}

func (msg MsgMilestoneTimeout) Route() string {
	return RouterKey
}

func (msg MsgMilestoneTimeout) GetSigners() string {
	return msg.From
}

func (msg MsgMilestoneTimeout) ValidateBasic(ac address.Codec) error {
	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return heimdallError.ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return heimdallError.ErrInvalidMsg.Wrapf("Invalid proposer %s", msg.From)
	}

	return nil
}
