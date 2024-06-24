package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgProposeSpanRequest{}

// NewMsgProposeSpanRequest creates a new MsgProposeSpanRequest instance
func NewMsgProposeSpanRequest(
	spanID uint64,
	proposer string,
	startBlock uint64,
	endBlock uint64,
	chainId string,
	seed []byte,
) *MsgProposeSpanRequest {
	return &MsgProposeSpanRequest{
		SpanId:     spanID,
		Proposer:   proposer,
		StartBlock: startBlock,
		EndBlock:   endBlock,
		ChainId:    chainId,
		Seed:       seed,
	}
}

// Type returns the type of the x/bor MsgTopupTx.
func (msg MsgProposeSpanRequest) Type() string {
	return EventTypeProposeSpan
}
