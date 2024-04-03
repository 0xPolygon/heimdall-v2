package types

import (
	"cosmossdk.io/math"
	"github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	"math/big"
)

// TODO HV2: this file has some extension methods for msg interfaces. Do we need it at all?

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

// Route returns the RouterKey.
func (msg MsgTopupTx) Route() string {
	return RouterKey
}

// Type returns the type of the topup Msg.
func (msg MsgTopupTx) Type() string {
	return EventTypeTopup
}

// ValidateBasic runs stateless checks on the message
func (msg MsgTopupTx) ValidateBasic() error {
	if len(msg.Proposer) == 0 {
		return errors.ErrInvalidAddress
	}
	return nil
}

// GetSigners returns the addresses of signers that must sign.
func (msg MsgTopupTx) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{sdk.AccAddress(msg.Proposer)}
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgTopupTx) GetSignBytes() []byte {
	// TODO HV2: double check this function
	return keeper.AppendBytes32(
		[]byte(msg.Proposer),
		[]byte(msg.User),
		msg.Fee.BigInt().Bytes(),
		msg.TxHash.GetHash(),
		new(big.Int).SetUint64(msg.LogIndex).Bytes(),
		new(big.Int).SetUint64(msg.BlockNumber).Bytes(),
	)
}

// GetSideSignBytes returns the side sign bytes.
func (msg MsgTopupTx) GetSideSignBytes() []byte {
	return nil
}

// NewMsgWithdrawFeeTx creates and returns a new MsgWithdrawFeeTx.
func NewMsgWithdrawFeeTx(proposer string, amount math.Int) *MsgWithdrawFeeTx {
	return &MsgWithdrawFeeTx{
		Proposer: proposer,
		Amount:   amount,
	}
}

// Route returns the RouterKey.
func (msg MsgWithdrawFeeTx) Route() string {
	return RouterKey
}

// Type returns the type of the topup Msg.
func (msg MsgWithdrawFeeTx) Type() string {
	return EventTypeWithdraw
}

// ValidateBasic runs stateless checks on the message
func (msg MsgWithdrawFeeTx) ValidateBasic() error {
	if len(msg.Proposer) == 0 {
		return errors.ErrInvalidAddress
	}
	return nil
}

// GetSigners returns the addresses of signers that must sign.
func (msg MsgWithdrawFeeTx) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{sdk.AccAddress(msg.Proposer)}
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgWithdrawFeeTx) GetSignBytes() []byte {
	// TODO HV2: double check this function
	return keeper.AppendBytes32(
		[]byte(msg.Proposer),
		msg.Amount.BigInt().Bytes(),
	)
}
