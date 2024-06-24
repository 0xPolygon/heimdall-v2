package keeper

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	addrCodec "github.com/cosmos/cosmos-sdk/codec/address"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

type msgServer struct {
	k *Keeper
}

// NewMsgServerImpl returns an implementation of the staking MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

// ValidatorJoin defines a method for new validator's joining
func (m msgServer) ValidatorJoin(ctx context.Context, msg *types.MsgValidatorJoin) (*types.MsgValidatorJoinResponse, error) {
	m.k.Logger(ctx).Debug("✅ Validating validator join msg",
		"validatorId", msg.ValId,
		"activationEpoch", msg.ActivationEpoch,
		"amount", msg.Amount,
		"SignerPubKey", msg.SignerPubKey.String(),
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	// Generate PubKey from PubKey in message and signer
	pubKey := msg.SignerPubKey
	pk, ok := pubKey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "error in interfacing out pub key")
	}

	// TODO HV2: is any attack possible here?
	signer, err := addrCodec.NewHexCodec().BytesToString(pk.Address())
	if err != nil {
		m.k.Logger(ctx).Error("signer is invalid", "error", err)
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, "signer is invalid")
	}

	// check if validator has been validator before
	if ok, err = m.k.DoValIdExist(ctx, msg.ValId); ok {
		m.k.Logger(ctx).Error("validator has been a validator before, hence cannot join with same id", "validatorId", msg.ValId, "err", err)
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "validator corresponding to the val id already exists in store")
	}

	// get validator by signer
	checkVal, err := m.k.GetValidatorInfo(ctx, signer)
	if err == nil && checkVal.Signer == signer {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, fmt.Sprintf("validator %s already exists", signer))
	}

	// validate voting power
	_, err = helper.GetPowerFromAmount(msg.Amount.BigInt())
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, fmt.Sprintf("Invalid amount %s for validator %d", msg.Amount, msg.ValId))
	}

	// add sequence
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if m.k.HasStakingSequence(ctx, sequence.String()) {
		m.k.Logger(ctx).Error("older invalid tx found", "sequence", sequence.String())
		return nil, errorsmod.Wrap(types.ErrOldTx, "older invalid tx found")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeValidatorJoin,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(msg.ValId, 10)),
			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
		),
	})

	return &types.MsgValidatorJoinResponse{}, nil
}

// StakeUpdate defines a method for updating the stake of a validator
func (m msgServer) StakeUpdate(ctx context.Context, msg *types.MsgStakeUpdate) (*types.MsgStakeUpdateResponse, error) {
	m.k.Logger(ctx).Debug("✅ Validating stake update msg",
		"validatorID", msg.ValId,
		"newAmount", msg.NewAmount,
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	// pull validator from store
	_, err := m.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		m.k.Logger(ctx).Error("failed to fetch validator from store", "validatorId", msg.ValId, "error", err)
		return nil, errorsmod.Wrap(types.ErrNoValidator, "failed to fetch validator from store")
	}

	// add sequence
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if m.k.HasStakingSequence(ctx, sequence.String()) {
		m.k.Logger(ctx).Error("older invalid tx found", "sequence", sequence.String())
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, "older invalid tx found")
	}

	// pull validator from store
	validator, err := m.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		m.k.Logger(ctx).Error("failed to fetch validator from store", "validatorId", msg.ValId, "error", err)
		return nil, errorsmod.Wrap(types.ErrNoValidator, "failed to fetch validator from store")
	}

	if msg.Nonce != validator.Nonce+1 {
		m.k.Logger(ctx).Error("incorrect validator nonce")
		return nil, errorsmod.Wrap(types.ErrInvalidNonce, "incorrect validator nonce")
	}

	// set validator amount
	_, err = helper.GetPowerFromAmount(msg.NewAmount.BigInt())
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, fmt.Sprintf("invalid amount %s for validator %d", msg.NewAmount, msg.ValId))
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeStakeUpdate,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(validator.ValId, 10)),
			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
		),
	})

	return &types.MsgStakeUpdateResponse{}, nil
}

// SignerUpdate defines a method for updating the validator's signer
func (m msgServer) SignerUpdate(ctx context.Context, msg *types.MsgSignerUpdate) (*types.MsgSignerUpdateResponse, error) {
	m.k.Logger(ctx).Debug("✅ Validating signer update msg",
		"validatorID", msg.ValId,
		"NewSignerPubKey", msg.NewSignerPubKey.String(),
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	// Generate PubKey from PubKey in message and signer
	pubKey := msg.NewSignerPubKey
	pk, ok := pubKey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "error in interfacing out pub key")
	}

	newSigner, err := addrCodec.NewHexCodec().BytesToString(pk.Address())
	if err != nil {
		m.k.Logger(ctx).Error("signer is invalid", "error", err)
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, "signer is invalid")
	}

	// pull validator from store
	validator, err := m.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		m.k.Logger(ctx).Error("Fetching of validator from store failed", "validatorId", msg.ValId, "error", err)
		return nil, errorsmod.Wrap(types.ErrNoValidator, "Fetching of validator from store failed")
	}

	// add sequence
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if m.k.HasStakingSequence(ctx, sequence.String()) {
		m.k.Logger(ctx).Error("older invalid tx found", "sequence", sequence.String())
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, "older invalid tx found")
	}

	// check if new signer address is same as existing signer
	if newSigner == validator.Signer {
		// No signer change
		m.k.Logger(ctx).Error("new signer is the same as old signer")
		return nil, errorsmod.Wrap(types.ErrNoSignerChange, "newSigner same as oldSigner")

	}

	// check nonce validity
	if msg.Nonce != validator.Nonce+1 {
		m.k.Logger(ctx).Error("incorrect validator nonce")
		return nil, errorsmod.Wrap(types.ErrInvalidNonce, "incorrect validator nonce")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeSignerUpdate,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(validator.ValId, 10)),
			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
		),
	})

	return &types.MsgSignerUpdateResponse{}, nil
}

// ValidatorExit defines a method for exiting the validator from the validator set
func (m msgServer) ValidatorExit(ctx context.Context, msg *types.MsgValidatorExit) (*types.MsgValidatorExitResponse, error) {
	m.k.Logger(ctx).Debug("✅ Validating validator exit msg",
		"validatorID", msg.ValId,
		"deactivationEpoch", msg.DeactivationEpoch,
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	validator, err := m.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		m.k.Logger(ctx).Error("failed to fetch validator from store", "validatorID", msg.ValId, "error", err)
		return nil, errorsmod.Wrap(types.ErrNoValidator, "failed to fetch validator from store")
	}

	m.k.Logger(ctx).Debug("validator in store", "validator", validator)
	// check if validator deactivation period is set
	if validator.EndEpoch != 0 {
		m.k.Logger(ctx).Error("validator already unbonded")
		return nil, errorsmod.Wrap(types.ErrValUnbonded, "validator already unbonded")
	}

	// add sequence
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if m.k.HasStakingSequence(ctx, sequence.String()) {
		m.k.Logger(ctx).Error("older invalid tx found", "sequence", sequence.String())
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, "older invalid tx found")
	}

	// check nonce validity
	if msg.Nonce != validator.Nonce+1 {
		m.k.Logger(ctx).Error("incorrect validator nonce")
		return nil, errorsmod.Wrap(types.ErrInvalidNonce, "incorrect validator nonce")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeValidatorExit,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyValidatorID, strconv.FormatUint(validator.ValId, 10)),
			sdk.NewAttribute(types.AttributeKeyValidatorNonce, strconv.FormatUint(msg.Nonce, 10)),
		),
	})

	return &types.MsgValidatorExitResponse{}, nil
}
