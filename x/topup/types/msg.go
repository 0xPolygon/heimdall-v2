package types

import (
	"errors"
	"math"

	sdkmath "cosmossdk.io/math"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"

	util "github.com/0xPolygon/heimdall-v2/common/address"
)

var (
	_ sdk.Msg = &MsgTopupTx{}
	_ sdk.Msg = &MsgWithdrawFeeTx{}
)

// NewMsgTopupTx creates and returns a new MsgTopupTx.
func NewMsgTopupTx(proposer, user string, fee sdkmath.Int, txHash []byte, index, blockNumber uint64) *MsgTopupTx {
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
func NewMsgWithdrawFeeTx(proposer string, amount sdkmath.Int) *MsgWithdrawFeeTx {
	return &MsgWithdrawFeeTx{
		Proposer: util.FormatAddress(proposer),
		Amount:   amount,
	}
}

// Type returns the type of the x/topup MsgWithdrawFeeTx.
func (msg MsgWithdrawFeeTx) Type() string {
	return EventTypeWithdraw
}

func (data MsgTopupTx) ValidateBasic() error {
	if data.Fee.IsNegative() {
		return errors.New("fee cannot be negative")
	}
	ac := addresscodec.NewHexCodec()
	_, err := ac.StringToBytes(data.Proposer)
	if err != nil {
		return errors.New("invalid proposer")
	}
	_, err = ac.StringToBytes(data.User)
	if err != nil {
		return errors.New("invalid user")
	}
	if data.LogIndex > math.MaxUint64 {
		return errors.New("log index is out of range")
	}
	if data.BlockNumber > math.MaxUint64 {
		return errors.New("log index is out of range")
	}
	if len(data.TxHash) != 32 {
		return errors.New("invalid tx hash")
	}

	return nil
}

func (data MsgWithdrawFeeTx) ValidateBasic() error {
	if data.Amount.IsNegative() {
		return errors.New("amount cannot be negative")
	}
	ac := addresscodec.NewHexCodec()
	_, err := ac.StringToBytes(data.Proposer)
	if err != nil {
		return errors.New("invalid proposer")
	}
	return nil
}
