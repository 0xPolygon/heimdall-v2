package types

import (
	"bytes"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

var _ sdk.Msg = &MsgMilestone{}
var _ sdk.Msg = &MsgMilestoneTimeout{}

// NewMsgMilestoneBlock creates new milestone message using mentioned arguments
func NewMsgMilestoneBlock(
	proposer string,
	startBlock uint64,
	endBlock uint64,
	hash []byte,
	borChainID string,
	milestoneID string,
) MsgMilestone {
	return MsgMilestone{
		Proposer:    proposer,
		StartBlock:  startBlock,
		EndBlock:    endBlock,
		Hash:        hash,
		BorChainId:  borChainID,
		MilestoneId: milestoneID,
	}
}

func (msg MsgMilestone) ValidateBasic(ac address.Codec) error {
	if bytes.Equal(msg.Hash, common.Hash{}.Bytes()) {
		return ErrInvalidMsg.Wrapf("invalid roothash %v", string(msg.Hash))
	}

	addrBytes, err := ac.StringToBytes(msg.Proposer)
	if err != nil {
		return ErrInvalidMsg.Wrapf("invalid proposer %s", msg.Proposer)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("invalid proposer %s", msg.Proposer)
	}

	if msg.StartBlock >= msg.EndBlock || msg.EndBlock == 0 {
		return ErrInvalidMsg.Wrapf("end should be greater than to start block start block=%d,end block=%d", msg.StartBlock, msg.EndBlock)
	}

	return nil
}

var _ sdk.Msg = &MsgMilestoneTimeout{}

func NewMsgMilestoneTimeout(from string) MsgMilestoneTimeout {
	return MsgMilestoneTimeout{
		From: from,
	}
}

func (msg MsgMilestoneTimeout) ValidateBasic(ac address.Codec) error {
	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return ErrInvalidMsg.Wrapf("invalid proposer %s", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("invalid proposer %s", msg.From)
	}

	return nil
}
