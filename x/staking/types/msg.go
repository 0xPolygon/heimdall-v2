package types

import (
	"bytes"

	"cosmossdk.io/math"
	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
	heimdallError "github.com/0xPolygon/heimdall-v2/x/types/error"

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
// Delegator address and validator address are the same.
func NewMsgValidatorJoin(
	from hmTypes.HeimdallAddress, id hmTypes.ValidatorID, activationEpoch uint64,
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
		ID:              id,
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
func (msg MsgValidatorJoin) Validate() error {
	if msg.ID.GetID() == uint64(0) {
		return heimdallError.ErrInvalidMsg.Wrapf("Invalid validator ID %v", msg.ID.GetID())
	}

	if msg.From.Empty() {
		return heimdallError.ErrInvalidMsg.Wrapf("Invalid proposer %v", msg.From.String())
	}

	if msg.SignerPubKey == nil {
		return heimdallError.ErrInvalidMsg.Wrapf("Signer public key can't be nil")
	}

	pk, ok := msg.SignerPubKey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return heimdallError.ErrInvalidMsg.Wrapf("Error in unwrapping the public key")
	}

	//TODO H2: Should we implement the check for the size here
	if bytes.Equal(pk.Bytes(), hmTypes.ZeroPubKey.Bytes()) {
		return heimdallError.ErrInvalidMsg.Wrapf("Signer public key can't be of zero bytes")
	}

	return nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (msg MsgValidatorJoin) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pubKey cryptotypes.PubKey
	return unpacker.UnpackAny(msg.SignerPubKey, &pubKey)
}

// NewMsgStakeUpdate creates a new MsgStakeUpdate instance
func NewMsgStakeUpdate(from hmTypes.HeimdallAddress, id hmTypes.ValidatorID,
	newAmount math.Int, txHash hmTypes.TxHash, logIndex uint64,
	blockNumber uint64, nonce uint64) (*MsgStakeUpdate, error) {
	return &MsgStakeUpdate{
		From:        from,
		ID:          id,
		NewAmount:   newAmount,
		TxHash:      txHash,
		LogIndex:    logIndex,
		BlockNumber: blockNumber,
		Nonce:       nonce,
	}, nil
}

func (msg MsgStakeUpdate) Validate() error {
	if msg.ID.GetID() == uint64(0) {
		return heimdallError.ErrInvalidMsg.Wrapf("Invalid validator ID %v", msg.ID.GetID())
	}

	if msg.From.Empty() {
		return heimdallError.ErrInvalidMsg.Wrapf("Invalid proposer %v", msg.From.String())
	}

	return nil
}

// NewMsgDelegate creates a new MsgDelegate instance.
func NewMsgSignerUpdate(from hmTypes.HeimdallAddress, id hmTypes.ValidatorID,
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
		ID:              id,
		NewSignerPubKey: pkAny,
		TxHash:          txHash,
		LogIndex:        logIndex,
		BlockNumber:     blockNumber,
		Nonce:           nonce,
	}, nil
}

func (msg MsgSignerUpdate) Validate() error {
	if msg.ID.GetID() == uint64(0) {
		return heimdallError.ErrInvalidMsg.Wrapf("Invalid validator ID %v", msg.ID.GetID())
	}

	if msg.From.Empty() {
		return heimdallError.ErrInvalidMsg.Wrapf("Invalid proposer %v", msg.From.String())
	}

	if msg.NewSignerPubKey == nil {
		return heimdallError.ErrInvalidMsg.Wrapf("Signer public key can't be nil")
	}

	pk, ok := msg.NewSignerPubKey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return heimdallError.ErrInvalidMsg.Wrapf("Error in unwrapping the public key")
	}

	//TODO H2: Should we implement the check for the size here
	if bytes.Equal(pk.Bytes(), hmTypes.ZeroPubKey.Bytes()) {
		return heimdallError.ErrInvalidMsg.Wrapf("New signer public key can't be of zero bytes")
	}

	return nil
}

// NewMsgBeginRedelegate creates a new MsgBeginRedelegate instance.
func NewMsgValidatorExit(
	from hmTypes.HeimdallAddress, id hmTypes.ValidatorID, deactivationEpoch uint64,
	pubKey cryptotypes.PubKey, txHash hmTypes.TxHash, logIndex uint64,
	blockNumber uint64, nonce uint64,
) (*MsgValidatorExit, error) {
	return &MsgValidatorExit{
		From:              from,
		ID:                id,
		DeactivationEpoch: deactivationEpoch,
		TxHash:            txHash,
		LogIndex:          logIndex,
		BlockNumber:       blockNumber,
		Nonce:             nonce,
	}, nil
}

func (msg MsgValidatorExit) Validate() error {
	if msg.ID.GetID() == uint64(0) {
		return heimdallError.ErrInvalidMsg.Wrapf("Invalid validator ID %v", msg.ID.GetID())
	}

	if msg.From.Empty() {
		return heimdallError.ErrInvalidMsg.Wrapf("Invalid proposer %v", msg.From.String())
	}

	return nil
}
