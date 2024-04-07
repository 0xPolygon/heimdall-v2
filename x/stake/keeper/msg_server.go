package keeper

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	errorsmod "cosmossdk.io/errors"

	"github.com/0xPolygon/heimdall-v2/helper"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	hmerrors "github.com/0xPolygon/heimdall-v2/x/types/error"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the staking MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreateValidator defines a method for creating a new validator
func (k msgServer) ValidatorJoin(ctx context.Context, msg *types.MsgValidatorJoin) (*types.MsgValidatorJoinResponse, error) {
	k.Logger(ctx).Debug("✅ Validating validator join msg",
		"validatorId", msg.ValId,
		"activationEpoch", msg.ActivationEpoch,
		"amount", msg.Amount,
		"SignerPubkey", msg.SignerPubKey.String(),
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	// Generate PubKey from Pubkey in message and signer
	pubkey := msg.SignerPubKey
	pk, ok := pubkey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "Error in interfacing out pub key")
	}

	//TODO H2 Can any attack possible about it?
	//String directly coming from it is not of correct length
	signer := strings.ToLower(pk.Address().String())

	// Check if validator has been validator before
	if _, ok := k.GetSignerFromValidatorID(ctx, msg.ValId); ok {
		k.Logger(ctx).Error("validator has been validator beforeV, cannot join with same ID", "validatorId", msg.ValId)
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "validator has been validator before")
	}

	// get validator by signer
	checkVal, err := k.GetValidatorInfo(ctx, signer)
	if err == nil || checkVal.Signer == signer {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "validator already exist")
	}

	// validate voting power
	_, err = helper.GetPowerFromAmount(msg.Amount.BigInt())
	if err != nil {
		return nil, errorsmod.Wrap(hmerrors.ErrInvalidMsg, fmt.Sprintf("Invalid amount %v for validator %v", msg.Amount, msg.ValId))
	}

	// sequence id
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if k.HasStakingSequence(ctx, sequence.String()) {
		k.Logger(ctx).Error("Older invalid tx found")
		return nil, errorsmod.Wrap(hmerrors.ErrOldTx, "Older invalid tx found")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Emit event join
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

// EditValidator defines a method for editing an existing validator
func (k msgServer) StakeUpdate(ctx context.Context, msg *types.MsgStakeUpdate) (*types.MsgStakeUpdateResponse, error) {
	k.Logger(ctx).Debug("✅ Validating stake update msg",
		"validatorID", msg.ValId,
		"newAmount", msg.NewAmount,
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	// pull validator from store
	_, ok := k.GetValidatorFromValID(ctx, msg.ValId)
	if !ok {
		k.Logger(ctx).Error("Fetching of validator from store failed", "validatorId", msg.ValId)
		return nil, errorsmod.Wrap(hmerrors.ErrNoValidator, "Fetching of validator from store failed")
	}

	// sequence id
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if k.HasStakingSequence(ctx, sequence.String()) {
		k.Logger(ctx).Error("Older invalid tx found")
		return nil, errorsmod.Wrap(hmerrors.ErrInvalidMsg, "Older invalid tx found")
	}

	// pull validator from store
	validator, ok := k.GetValidatorFromValID(ctx, msg.ValId)
	if !ok {
		k.Logger(ctx).Error("Fetching of validator from store failed", "validatorId", msg.ValId)
		return nil, errorsmod.Wrap(hmerrors.ErrNoValidator, "Fetching of validator from store failed")
	}

	if msg.Nonce != validator.Nonce+1 {
		k.Logger(ctx).Error("Incorrect validator nonce")
		return nil, errorsmod.Wrap(hmerrors.ErrInvalidNonce, "Incorrect validator nonce")
	}

	// set validator amount
	_, err := helper.GetPowerFromAmount(msg.NewAmount.BigInt())
	if err != nil {
		return nil, errorsmod.Wrap(hmerrors.ErrInvalidMsg, fmt.Sprintf("Invalid amount %v for validator %v", msg.NewAmount, msg.ValId))
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

// Delegate defines a method for performing a delegation of coins from a delegator to a validator
func (k msgServer) SignerUpdate(ctx context.Context, msg *types.MsgSignerUpdate) (*types.MsgSignerUpdateResponse, error) {
	k.Logger(ctx).Debug("✅ Validating signer update msg",
		"validatorID", msg.ValId,
		"NewSignerPubkey", msg.NewSignerPubKey.String(),
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	// Generate PubKey from Pubkey in message and signer
	pubkey := msg.NewSignerPubKey
	pk, ok := pubkey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "Error in interfacing out pub key")
	}

	newSigner := strings.ToLower(pk.Address().String())

	// pull validator from store
	validator, ok := k.GetValidatorFromValID(ctx, msg.ValId)
	if !ok {
		k.Logger(ctx).Error("Fetching of validator from store failed", "validatorId", msg.ValId)
		return nil, errorsmod.Wrap(hmerrors.ErrNoValidator, "Fetching of validator from store failed")
	}

	// sequence id
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if k.HasStakingSequence(ctx, sequence.String()) {
		k.Logger(ctx).Error("Older invalid tx found")
		return nil, errorsmod.Wrap(hmerrors.ErrInvalidMsg, "Older invalid tx found")
	}

	// check if new signer address is same as existing signer
	if newSigner == validator.Signer {
		// No signer change
		k.Logger(ctx).Error("NewSigner same as OldSigner.")
		return nil, errorsmod.Wrap(hmerrors.ErrNoSignerChange, "NewSigner same as OldSigner")

	}

	// check nonce validity
	if msg.Nonce != validator.Nonce+1 {
		k.Logger(ctx).Error("Incorrect validator nonce")
		return nil, errorsmod.Wrap(hmerrors.ErrInvalidNonce, "Incorrect validator nonce")
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

// BeginRedelegate defines a method for performing a redelegation of coins from a source validator to a destination validator of given delegator
func (k msgServer) ValidatorExit(ctx context.Context, msg *types.MsgValidatorExit) (*types.MsgValidatorExitResponse, error) {
	k.Logger(ctx).Debug("✅ Validating validator exit msg",
		"validatorID", msg.ValId,
		"deactivatonEpoch", msg.DeactivationEpoch,
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	validator, ok := k.GetValidatorFromValID(ctx, msg.ValId)
	if !ok {
		k.Logger(ctx).Error("Fetching of validator from store failed", "validatorID", msg.ValId)
		return nil, errorsmod.Wrap(hmerrors.ErrNoValidator, "Fetching of validator from store failed")
	}

	k.Logger(ctx).Debug("validator in store", "validator", validator)
	// check if validator deactivation period is set
	if validator.EndEpoch != 0 {
		k.Logger(ctx).Error("Validator already unbonded")
		return nil, errorsmod.Wrap(hmerrors.ErrValUnbonded, "Validator already unbonded")
	}

	// sequence id
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if incoming tx is older
	if k.HasStakingSequence(ctx, sequence.String()) {
		k.Logger(ctx).Error("Older invalid tx found")
		return nil, errorsmod.Wrap(hmerrors.ErrInvalidMsg, "Older invalid tx found")
	}

	// check nonce validity
	if msg.Nonce != validator.Nonce+1 {
		k.Logger(ctx).Error("Incorrect validator nonce")
		return nil, errorsmod.Wrap(hmerrors.ErrInvalidNonce, "Incorrect validator nonce")
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
