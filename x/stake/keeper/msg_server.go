package keeper

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
	addrCodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	util "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics/api"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
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
func (srv msgServer) ValidatorJoin(ctx context.Context, msg *types.MsgValidatorJoin) (*types.MsgValidatorJoinResponse, error) {
	var err error
	startTime := time.Now()
	defer recordStakeTransactionMetric(api.ValidatorJoinMethod, startTime, &err)

	srv.k.Logger(ctx).Debug(helper.LogValidatingExternalCall("ValidatorJoin"),
		"validatorId", msg.ValId,
		"activationEpoch", msg.ActivationEpoch,
		"amount", msg.Amount,
		"SignerPubKey", common.Bytes2Hex(msg.SignerPubKey),
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	err = msg.ValidateBasic()
	if err != nil {
		srv.k.Logger(ctx).Error(hmTypes.ErrMsgValidationFailed, hmTypes.LogKeyError, err)
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, hmTypes.ErrMsgValidationFailed)
	}

	// Generate PubKey from PubKey in the message and signer
	pubKey := msg.SignerPubKey
	pk := secp256k1.PubKey{Key: pubKey}

	if pk.Type() != types.Secp256k1Type {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "pub key is invalid")
	}

	signer, err := addrCodec.NewHexCodec().BytesToString(pk.Address())
	if err != nil {
		srv.k.Logger(ctx).Error("Signer is invalid for ValidatorJoin", "error", err)
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, "signer is invalid for ValidatorJoin")
	}

	// check if the validator has been validator before
	if ok, err := srv.k.DoesValIdExist(ctx, msg.ValId); ok {
		srv.k.Logger(ctx).Error(hmTypes.ErrMsgValidatorAlreadyExists, hmTypes.LogKeyValidatorID, msg.ValId, hmTypes.LogKeyError, err)
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "validator corresponding to the val id already exists in store")
	}

	signer = util.FormatAddress(signer)
	// get validator by signer
	checkVal, err := srv.k.GetValidatorInfo(ctx, signer)
	// not returning error if validator not found because it is a new validator
	if err == nil && strings.Compare(util.FormatAddress(checkVal.Signer), signer) == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, fmt.Sprintf("validator %s already exists", signer))
	}

	// validate voting power
	_, err = helper.GetPowerFromAmount(msg.Amount.BigInt())
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, fmt.Sprintf("%s %s for validator %d", hmTypes.ErrMsgInvalidAmount, msg.Amount, msg.ValId))
	}

	// add the sequence
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if the event has already been processed
	if srv.k.HasStakingSequence(ctx, sequence.String()) {
		srv.k.Logger(ctx).Error(hmTypes.ErrMsgEventAlreadyProcessed, hmTypes.LogKeySequence, sequence.String())
		return nil, errors.Wrapf(sdkerrors.ErrConflict, hmTypes.ErrMsgOldEventsNotAllowed)
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
func (srv msgServer) StakeUpdate(ctx context.Context, msg *types.MsgStakeUpdate) (*types.MsgStakeUpdateResponse, error) {
	var err error
	startTime := time.Now()
	defer recordStakeTransactionMetric(api.StakeUpdateMethod, startTime, &err)

	srv.k.Logger(ctx).Debug(helper.LogValidatingExternalCall("StakeUpdate"),
		"validatorID", msg.ValId,
		"newAmount", msg.NewAmount,
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	err = msg.ValidateBasic()
	if err != nil {
		srv.k.Logger(ctx).Error(hmTypes.ErrMsgValidationFailed, hmTypes.LogKeyError, err)
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, hmTypes.ErrMsgValidationFailed)
	}

	// pull validator from store
	_, err = srv.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		srv.k.Logger(ctx).Error(hmTypes.ErrMsgFailedToFetchValidator, "validatorId", msg.ValId, "error", err)
		return nil, errorsmod.Wrap(types.ErrNoValidator, hmTypes.ErrMsgFailedToFetchValidator)
	}

	// add the sequence
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if the event has already been processed
	if srv.k.HasStakingSequence(ctx, sequence.String()) {
		srv.k.Logger(ctx).Error(hmTypes.ErrMsgEventAlreadyProcessed, hmTypes.LogKeySequence, sequence.String())
		return nil, errors.Wrapf(sdkerrors.ErrConflict, hmTypes.ErrMsgOldEventsNotAllowed)
	}

	// set validator amount
	_, err = helper.GetPowerFromAmount(msg.NewAmount.BigInt())
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, fmt.Sprintf("invalid amount %s for validator %d", msg.NewAmount, msg.ValId))
	}

	return &types.MsgStakeUpdateResponse{}, nil
}

// SignerUpdate defines a method for updating the validator's signer
func (srv msgServer) SignerUpdate(ctx context.Context, msg *types.MsgSignerUpdate) (*types.MsgSignerUpdateResponse, error) {
	var err error
	startTime := time.Now()
	defer recordStakeTransactionMetric(api.SignerUpdateMethod, startTime, &err)

	srv.k.Logger(ctx).Debug(helper.LogValidatingExternalCall("SignerUpdate"),
		"validatorID", msg.ValId,
		"NewSignerPubKey", common.Bytes2Hex(msg.NewSignerPubKey),
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	err = msg.ValidateBasic()
	if err != nil {
		srv.k.Logger(ctx).Error(hmTypes.ErrMsgValidationFailed, hmTypes.LogKeyError, err)
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, hmTypes.ErrMsgValidationFailed)
	}

	// Generate PubKey from PubKey in the message and signer
	pubKey := msg.NewSignerPubKey
	pk := &secp256k1.PubKey{Key: pubKey}

	if pk.Type() != types.Secp256k1Type {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "pub key is invalid")
	}

	newSigner, err := addrCodec.NewHexCodec().BytesToString(pk.Address())
	if err != nil {
		srv.k.Logger(ctx).Error("New signer is invalid for signerUpdate", "error", err, "newSigner", newSigner)
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, "new signer is invalid for signerUpdate")
	}

	// pull validator from store
	validator, err := srv.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		srv.k.Logger(ctx).Error("Fetching of validator from store failed", "validatorId", msg.ValId, "error", err)
		return nil, errorsmod.Wrap(types.ErrNoValidator, "Fetching of validator from store failed")
	}

	// make oldSigner address compatible with newSigner address
	oldSigner := util.FormatAddress(validator.Signer)

	// add the sequence
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if the event has already been processed
	if srv.k.HasStakingSequence(ctx, sequence.String()) {
		srv.k.Logger(ctx).Error(hmTypes.ErrMsgEventAlreadyProcessed, hmTypes.LogKeySequence, sequence.String())
		return nil, errors.Wrapf(sdkerrors.ErrConflict, hmTypes.ErrMsgOldEventsNotAllowed)
	}

	// check if the new signer address is the same as the existing signer
	if newSigner == oldSigner {
		// No signer change
		srv.k.Logger(ctx).Error("New signer is the same as old signer in signerUpdate", "validatorId", msg.ValId, "signer", newSigner)
		return nil, errorsmod.Wrap(types.ErrNoSignerChange, "newSigner same as oldSigner in signerUpdate")
	}

	return &types.MsgSignerUpdateResponse{}, nil
}

// ValidatorExit defines a method for exiting the validator from the validator set
func (srv msgServer) ValidatorExit(ctx context.Context, msg *types.MsgValidatorExit) (*types.MsgValidatorExitResponse, error) {
	var err error
	startTime := time.Now()
	defer recordStakeTransactionMetric(api.ValidatorExitMethod, startTime, &err)

	srv.k.Logger(ctx).Debug(helper.LogValidatingExternalCall("ValidatorExit"),
		"validatorID", msg.ValId,
		"deactivationEpoch", msg.DeactivationEpoch,
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	err = msg.ValidateBasic()
	if err != nil {
		srv.k.Logger(ctx).Error(hmTypes.ErrMsgValidationFailed, hmTypes.LogKeyError, err)
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, hmTypes.ErrMsgValidationFailed)
	}

	validator, err := srv.k.GetValidatorFromValID(ctx, msg.ValId)
	if err != nil {
		srv.k.Logger(ctx).Error(hmTypes.ErrMsgFailedToFetchValidator, "validatorID", msg.ValId, "error", err)
		return nil, errorsmod.Wrap(types.ErrNoValidator, hmTypes.ErrMsgFailedToFetchValidator)
	}

	srv.k.Logger(ctx).Debug("Validator in store", "validator", validator)
	// check if the validator deactivation period is set
	if validator.EndEpoch != 0 {
		srv.k.Logger(ctx).Error("Validator already unBonded", "validatorID", msg.ValId, "endEpoch", validator.EndEpoch)
		return nil, errorsmod.Wrap(types.ErrValUnBonded, "validator already unBonded")
	}

	// add the sequence
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// check if the event has already been processed
	if srv.k.HasStakingSequence(ctx, sequence.String()) {
		srv.k.Logger(ctx).Error(hmTypes.ErrMsgEventAlreadyProcessed, hmTypes.LogKeySequence, sequence.String())
		return nil, errors.Wrapf(sdkerrors.ErrConflict, hmTypes.ErrMsgOldEventsNotAllowed)
	}

	return &types.MsgValidatorExitResponse{}, nil
}

func recordStakeTransactionMetric(method string, start time.Time, err *error) {
	success := *err == nil
	api.RecordAPICallWithStart(api.StakeSubsystem, method, api.TxType, success, start)
}
