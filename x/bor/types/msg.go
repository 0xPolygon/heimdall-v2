package types

import (
	util "github.com/0xPolygon/heimdall-v2/common/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgProposeSpan{}

// NewMsgProposeSpan creates a new MsgProposeSpan instance
func NewMsgProposeSpan(
	spanID uint64,
	proposer string,
	startBlock uint64,
	endBlock uint64,
	chainID string,
	seed []byte,
) *MsgProposeSpan {
	return &MsgProposeSpan{
		SpanId:     spanID,
		Proposer:   util.FormatAddress(proposer),
		StartBlock: startBlock,
		EndBlock:   endBlock,
		ChainId:    chainID,
		Seed:       seed,
	}
}

// Type returns the type of the x/bor MsgProposeSpan.
func (msg MsgProposeSpan) Type() string {
	return EventTypeProposeSpan
}
