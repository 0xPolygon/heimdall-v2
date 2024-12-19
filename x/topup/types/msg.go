package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	util "github.com/0xPolygon/heimdall-v2/common/address"
)

var (
	_ sdk.Msg = &MsgTopupTx{}
	_ sdk.Msg = &MsgWithdrawFeeTx{}
)

// NewMsgTopupTx creates and returns a new MsgTopupTx.
func NewMsgTopupTx(proposer, user string, fee math.Int, txHash []byte, index, blockNumber uint64) *MsgTopupTx {
	return &MsgTopupTx{
		Proposer:    util.FormatAddress(proposer),
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
		Proposer: util.FormatAddress(proposer),
		Amount:   amount,
	}
}

// Type returns the type of the x/topup MsgWithdrawFeeTx.
func (msg MsgWithdrawFeeTx) Type() string {
	return EventTypeWithdraw
}
