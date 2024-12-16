package types

import (
	"bytes"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	util "github.com/0xPolygon/heimdall-v2/common/address"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgValidatorJoin{}
	_ sdk.Msg = &MsgStakeUpdate{}
	_ sdk.Msg = &MsgSignerUpdate{}
	_ sdk.Msg = &MsgValidatorExit{}
)

// NewMsgValidatorJoin creates a new MsgCreateValidator instance.
func NewMsgValidatorJoin(
	from string, id uint64, activationEpoch uint64,
	amount math.Int, pubKey cryptotypes.PubKey, txHash []byte, logIndex uint64,
	blockNumber uint64, nonce uint64,
) (*MsgValidatorJoin, error) {
	return &MsgValidatorJoin{
		From:            util.FormatAddress(from),
		ValId:           id,
		ActivationEpoch: activationEpoch,
		Amount:          amount,
		SignerPubKey:    pubKey.Bytes(),
		TxHash:          txHash,
		LogIndex:        logIndex,
		BlockNumber:     blockNumber,
		Nonce:           nonce,
	}, nil
}

// Validate validates the validator join msg before it is executed
func (msg MsgValidatorJoin) Validate(ac address.Codec) error {
	if msg.ValId == uint64(0) {
		return ErrInvalidMsg.Wrapf("invalid validator id %v", msg.ValId)
	}

	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return ErrInvalidMsg.Wrapf("invalid proposer %v", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("invalid proposer %v", msg.From)
	}

	if msg.SignerPubKey == nil {
		return ErrInvalidMsg.Wrap("signer public key can't be nil")
	}

	// TODO HV2: Should we implement the check for the size here
	if bytes.Equal(msg.SignerPubKey, EmptyPubKey[:]) {
		return ErrInvalidMsg.Wrap("signer public key can't be of zero bytes")
	}

	return nil
}

// NewMsgStakeUpdate creates a new MsgStakeUpdate instance
func NewMsgStakeUpdate(from string, id uint64,
	newAmount math.Int, txHash []byte, logIndex uint64,
	blockNumber uint64, nonce uint64,
) (*MsgStakeUpdate, error) {
	return &MsgStakeUpdate{
		From:        util.FormatAddress(from),
		ValId:       id,
		NewAmount:   newAmount,
		TxHash:      txHash,
		LogIndex:    logIndex,
		BlockNumber: blockNumber,
		Nonce:       nonce,
	}, nil
}

// Validate validates the stake update msg before it is executed
func (msg MsgStakeUpdate) Validate(ac address.Codec) error {
	if msg.ValId == uint64(0) {
		return ErrInvalidMsg.Wrapf("invalid validator id %v", msg.ValId)
	}

	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return ErrInvalidMsg.Wrapf("invalid proposer %v", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("invalid proposer %v", msg.From)
	}

	return nil
}

// NewMsgSignerUpdate creates a new MsgSignerUpdate instance.
func NewMsgSignerUpdate(from string, id uint64,
	pubKey []byte, txHash []byte, logIndex uint64,
	blockNumber uint64, nonce uint64,
) (*MsgSignerUpdate, error) {
	return &MsgSignerUpdate{
		From:            util.FormatAddress(from),
		ValId:           id,
		NewSignerPubKey: pubKey,
		TxHash:          txHash,
		LogIndex:        logIndex,
		BlockNumber:     blockNumber,
		Nonce:           nonce,
	}, nil
}

// Validate validates the signer update msg before it is executed
func (msg MsgSignerUpdate) Validate(ac address.Codec) error {
	if msg.ValId == uint64(0) {
		return ErrInvalidMsg.Wrapf("invalid validator id %v", msg.ValId)
	}

	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return ErrInvalidMsg.Wrapf("invalid proposer %v", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("invalid proposer %v", msg.From)
	}

	if msg.NewSignerPubKey == nil {
		return ErrInvalidMsg.Wrap("signer public key can't be nil")
	}

	// TODO HV2: Should we implement the check for the size here
	if bytes.Equal(msg.NewSignerPubKey, EmptyPubKey[:]) {
		return ErrInvalidMsg.Wrap("new signer public key can't be of zero bytes")
	}

	return nil
}

// NewMsgValidatorExit creates a new MsgValidatorExit instance.
func NewMsgValidatorExit(
	from string, id uint64, deactivationEpoch uint64,
	txHash []byte, logIndex uint64,
	blockNumber uint64, nonce uint64,
) (*MsgValidatorExit, error) {
	return &MsgValidatorExit{
		From:              util.FormatAddress(from),
		ValId:             id,
		DeactivationEpoch: deactivationEpoch,
		TxHash:            txHash,
		LogIndex:          logIndex,
		BlockNumber:       blockNumber,
		Nonce:             nonce,
	}, nil
}

// Validate validates the validator exit msg before it is executed
func (msg MsgValidatorExit) Validate(ac address.Codec) error {
	if msg.ValId == uint64(0) {
		return ErrInvalidMsg.Wrapf("invalid validator id %v", msg.ValId)
	}

	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return ErrInvalidMsg.Wrapf("invalid proposer %v", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("invalid proposer %v", msg.From)
	}

	return nil
}
