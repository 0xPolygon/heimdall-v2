package types

import (
	"bytes"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
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
	amount math.Int, pubKey cryptotypes.PubKey, txHash hmTypes.TxHash, logIndex uint64,
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

// Validate validates the MsgValidatorJoin sdk msg.
func (msg MsgValidatorJoin) Validate(ac address.Codec) error {
	if msg.ValId == uint64(0) {
		return ErrInvalidMsg.Wrapf("Invalid validator ID %v", msg.ValId)
	}

	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return ErrInvalidMsg.Wrapf("Invalid proposer %v", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid proposer %v", msg.From)
	}

	if msg.SignerPubKey == nil {
		return ErrInvalidMsg.Wrapf("Signer public key can't be nil")
	}

	pk, ok := msg.SignerPubKey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return ErrInvalidMsg.Wrapf("Error in unwrapping the public key")
	}

	// TODO HV2: Should we implement the check for the size here
	if bytes.Equal(pk.Bytes(), ZeroPubKey[:]) {
		return ErrInvalidMsg.Wrapf("Signer public key can't be of zero bytes")
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
	newAmount math.Int, txHash hmTypes.TxHash, logIndex uint64,
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

func (msg MsgStakeUpdate) Validate(ac address.Codec) error {
	if msg.ValId == uint64(0) {
		return ErrInvalidMsg.Wrapf("Invalid validator ID %v", msg.ValId)
	}

	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return ErrInvalidMsg.Wrapf("Invalid proposer %v", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid proposer %v", msg.From)
	}

	return nil
}

// NewMsgSignerUpdate creates a new MsgSignerUpdate instance.
func NewMsgSignerUpdate(from string, id uint64,
	pubKey cryptotypes.PubKey, txHash hmTypes.TxHash, logIndex uint64,
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

func (msg MsgSignerUpdate) Validate(ac address.Codec) error {
	if msg.ValId == uint64(0) {
		return ErrInvalidMsg.Wrapf("Invalid validator ID %v", msg.ValId)
	}

	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return ErrInvalidMsg.Wrapf("Invalid proposer %v", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid proposer %v", msg.From)
	}

	if msg.NewSignerPubKey == nil {
		return ErrInvalidMsg.Wrapf("Signer public key can't be nil")
	}

	pk, ok := msg.NewSignerPubKey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return ErrInvalidMsg.Wrapf("Error in unwrapping the public key")
	}

	// TODO HV2: Should we implement the check for the size here
	if bytes.Equal(pk.Bytes(), ZeroPubKey[:]) {
		return ErrInvalidMsg.Wrapf("New signer public key can't be of zero bytes")
	}

	return nil
}

// NewMsgValidatorExit creates a new MsgValidatorExit instance.
func NewMsgValidatorExit(
	from string, id uint64, deactivationEpoch uint64,
	txHash hmTypes.TxHash, logIndex uint64,
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

func (msg MsgValidatorExit) Validate(ac address.Codec) error {
	if msg.ValId == uint64(0) {
		return ErrInvalidMsg.Wrapf("Invalid validator ID %v", msg.ValId)
	}

	addrBytes, err := ac.StringToBytes(msg.From)
	if err != nil {
		return ErrInvalidMsg.Wrapf("Invalid proposer %v", msg.From)
	}

	accAddr := sdk.AccAddress(addrBytes)

	if accAddr.Empty() {
		return ErrInvalidMsg.Wrapf("Invalid proposer %v", msg.From)
	}

	return nil
}
