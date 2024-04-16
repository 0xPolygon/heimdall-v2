package types

import (
	"cosmossdk.io/math"
	"github.com/0xPolygon/heimdall-v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgTopupTx{}
var _ sdk.Msg = &MsgWithdrawFeeTx{}

// NewMsgTopupTx creates and returns a new MsgTopupTx.
func NewMsgTopupTx(proposer, user string, fee math.Int, txHash types.TxHash, index, blockNumber uint64) *MsgTopupTx {
	return &MsgTopupTx{
		Proposer:    proposer,
		User:        user,
		Fee:         fee,
		TxHash:      txHash,
		LogIndex:    index,
		BlockNumber: blockNumber,
	}
}

// Type returns the type of the x/topup MsgTopupTx.
func (msg MsgTopupTx) Type() string {
	return EventTypeTopup
}

// NewMsgWithdrawFeeTx creates and returns a new MsgWithdrawFeeTx.
func NewMsgWithdrawFeeTx(proposer string, amount math.Int) *MsgWithdrawFeeTx {
	return &MsgWithdrawFeeTx{
		Proposer: proposer,
		Amount:   amount,
	}
}

// Type returns the type of the x/topup MsgWithdrawFeeTx.
func (msg MsgWithdrawFeeTx) Type() string {
	return EventTypeWithdraw
}
