package types

import (
	"bytes"
	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg                            = &MsgValidatorJoin{}
	_ codectypes.UnpackInterfacesMessage = (*MsgValidatorJoin)(nil)
	_ sdk.Msg                            = &MsgStakeUpdate{}
	_ sdk.Msg                            = &MsgSignerUpdate{}
	_ sdk.Msg                            = &MsgValidatorExit{}
)

// NewMsgValidatorJoin creates a new MsgCreateValidator instance.
func NewMsgValidatorJoin(
	from string, id uint64, activationEpoch uint64,
	amount math.Int, pubKey cryptotypes.PubKey, txHash []byte, logIndex uint64,
	blockNumber uint64, nonce uint64,
) (*MsgValidatorJoin, error) {

	var pkAny *codectypes.Any
	if pubKey != nil {
		var err error
		if pkAny, err = codectypes.NewAnyWithValue(pubKey); err != nil {
			return nil, err
		}
	}
	return &MsgValidatorJoin{
		From:            from,
		ValId:           id,
		ActivationEpoch: activationEpoch,
		Amount:          amount,
		SignerPubKey:    pkAny,
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

	pk, ok := msg.SignerPubKey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return ErrInvalidMsg.Wrap("error in unwrapping the public key")
	}

	// TODO HV2: Should we implement the check for the size here
	if bytes.Equal(pk.Bytes(), ZeroPubKey[:]) {
		return ErrInvalidMsg.Wrap("signer public key can't be of zero bytes")
	}

	return nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (msg MsgValidatorJoin) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pubKey cryptotypes.PubKey
	return unpacker.UnpackAny(msg.SignerPubKey, &pubKey)
}

// NewMsgStakeUpdate creates a new MsgStakeUpdate instance
func NewMsgStakeUpdate(from string, id uint64,
	newAmount math.Int, txHash []byte, logIndex uint64,
	blockNumber uint64, nonce uint64) (*MsgStakeUpdate, error) {
	return &MsgStakeUpdate{
		From:        from,
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
	pubKey cryptotypes.PubKey, txHash []byte, logIndex uint64,
	blockNumber uint64, nonce uint64) (*MsgSignerUpdate, error) {
	var pkAny *codectypes.Any
	if pubKey != nil {
		var err error
		if pkAny, err = codectypes.NewAnyWithValue(pubKey); err != nil {
			return nil, err
		}
	}

	return &MsgSignerUpdate{
		From:            from,
		ValId:           id,
		NewSignerPubKey: pkAny,
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

	pk, ok := msg.NewSignerPubKey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return ErrInvalidMsg.Wrap("error in unwrapping the public key")
	}

	// TODO HV2: Should we implement the check for the size here
	if bytes.Equal(pk.Bytes(), ZeroPubKey[:]) {
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
		From:              from,
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
