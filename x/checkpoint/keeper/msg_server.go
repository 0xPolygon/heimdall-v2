package keeper

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
	hmerrors "github.com/0xPolygon/heimdall-v2/x/types/error"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the checkpoint MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreateValidator defines a method for creating a new validator
func (k msgServer) CheckpointAdjust(ctx context.Context, msg *types.MsgCheckpointAdjust) (*types.MsgCheckpointAdjustResponse, error) {
	logger := k.Logger(ctx)

	checkpointBuffer, err := k.GetCheckpointFromBuffer(ctx)
	if checkpointBuffer != nil {
		logger.Error("checkpoint buffer exists", "error", err)
		return nil, errorsmod.Wrap(hmerrors.ErrCheckpointBufferFound, "Checkpoint buffer not found")
	}

	checkpointObj, err := k.GetCheckpointByNumber(ctx, msg.HeaderIndex)
	if err != nil {
		logger.Error("Unable to get checkpoint from db", "header index", msg.HeaderIndex, "error", err)
		return nil, errorsmod.Wrap(hmerrors.ErrNoCheckpointFound, "Checkpoint not found in db")
	}

	if checkpointObj.EndBlock == msg.EndBlock && checkpointObj.StartBlock == msg.StartBlock && bytes.Equal(checkpointObj.RootHash.Bytes(), msg.RootHash.Bytes()) && strings.ToLower(checkpointObj.Proposer) == strings.ToLower(msg.Proposer) {
		logger.Error("Same Checkpoint in DB")
		return nil, errorsmod.Wrap(hmerrors.ErrCheckpointAlreadyExists, "Checkpoint already exist in db")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCheckpointAdjust,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyHeaderIndex, strconv.FormatUint(msg.HeaderIndex, 10)),
			sdk.NewAttribute(types.AttributeKeyStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyRootHash, msg.RootHash.String()),
		),
	})

	return &types.MsgCheckpointAdjustResponse{}, nil
}

// EditValidator defines a method for editing an existing validator
func (k msgServer) Checkpoint(ctx context.Context, msg *types.MsgCheckpoint) (*types.MsgCheckpointResponse, error) {
	logger := k.Logger(ctx)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	timeStamp := uint64(sdkCtx.BlockTime().Unix())

	params, err := k.GetParams(ctx)
	if err != nil {
		logger.Error("Error in fetching checkpoint parameter")
		return nil, errorsmod.Wrap(hmerrors.ErrCheckpointParams, "Error in fetching checkpoint parameter")
	}

	//
	// Check checkpoint buffer
	//

	checkpointBuffer, err := k.GetCheckpointFromBuffer(ctx)
	if err == nil {
		checkpointBufferTime := uint64(params.CheckpointBufferTime.Seconds())

		if checkpointBuffer.TimeStamp == 0 || ((timeStamp > checkpointBuffer.TimeStamp) && (timeStamp-checkpointBuffer.TimeStamp) >= checkpointBufferTime) {
			logger.Debug("Checkpoint has been timed out. Flushing buffer.", "checkpointTimestamp", timeStamp, "prevCheckpointTimestamp", checkpointBuffer.TimeStamp)
			k.FlushCheckpointBuffer(ctx)
		} else {
			expiryTime := checkpointBuffer.TimeStamp + checkpointBufferTime
			logger.Error("Checkpoint already exits in buffer", "Checkpoint", checkpointBuffer.String(), "Expires", expiryTime)
			return nil, errorsmod.Wrap(hmerrors.ErrNoACK, fmt.Sprint("Checkpoint already exits in buffer", "Checkpoint", checkpointBuffer.String(), "Expires", expiryTime))
		}
	}

	//
	// Validate last checkpoint
	//

	// fetch last checkpoint from store
	if lastCheckpoint, err := k.GetLastCheckpoint(ctx); err == nil {
		// make sure new checkpoint is after tip
		if lastCheckpoint.EndBlock > msg.StartBlock {
			logger.Error("Checkpoint already exists",
				"currentTip", lastCheckpoint.EndBlock,
				"startBlock", msg.StartBlock,
			)

			return nil, errorsmod.Wrap(hmerrors.ErrOldCheckpoint, "Checkpoint already exist for start and end block")
		}

		// check if new checkpoint's start block start from current tip
		if lastCheckpoint.EndBlock+1 != msg.StartBlock {
			logger.Error("Checkpoint not in continuity",
				"currentTip", lastCheckpoint.EndBlock,
				"startBlock", msg.StartBlock)

			return nil, errorsmod.Wrap(hmerrors.ErrDisCountinuousCheckpoint, fmt.Sprint("Checkpoint not in continuity", "currentTip", lastCheckpoint.EndBlock, "startBlock", msg.StartBlock))
		}
	} else if err.Error() == hmerrors.ErrNoCheckpointFound.Error() && msg.StartBlock != 0 {
		logger.Error("First checkpoint to start from block 0", "checkpoint start block", msg.StartBlock, "error", err)
		return nil, errorsmod.Wrap(hmerrors.ErrBadBlockDetails, fmt.Sprint("First checkpoint to start from block 0", "checkpoint start block", msg.StartBlock))
	}

	//
	// Validate account hash
	//

	// Make sure latest AccountRootHash matches
	// Calculate new account root hash
	dividendAccounts := k.moduleCommunicator.GetAllDividendAccounts(ctx)
	logger.Debug("DividendAccounts of all validators", "dividendAccountsLength", len(dividendAccounts))

	// Get account root hash from dividend accounts
	accountRoot, err := types.GetAccountRootHash(dividendAccounts)
	if err != nil {
		logger.Error("Error while fetching account root hash", "error", err)
		return nil, errorsmod.Wrap(hmerrors.ErrBadBlockDetails, fmt.Sprint("Error while fetching account root hash"))
	}

	logger.Debug("Validator account root hash generated", "accountRootHash", hmTypes.BytesToHeimdallHash(accountRoot).HexString())

	// Compare stored root hash to msg root hash
	if !bytes.Equal(accountRoot, msg.AccountRootHash.Bytes()) {
		logger.Error(
			"AccountRootHash of current state doesn't match from msg",
			"hash", hmTypes.BytesToHeimdallHash(accountRoot).HexString(),
			"msgHash", msg.AccountRootHash,
		)
		return nil, errorsmod.Wrap(hmerrors.ErrBadBlockDetails, fmt.Sprint("AccountRootHash of current state doesn't match from msg",
			"hash", hmTypes.BytesToHeimdallHash(accountRoot).HexString(),
			"msgHash", msg.AccountRootHash))
	}

	//
	// Validate proposer
	//

	// Check proposer in message
	validatorSet := k.sk.GetValidatorSet(ctx)
	if validatorSet.Proposer == nil {
		logger.Error("No proposer in validator set", "msgProposer", msg.Proposer)
		return nil, errorsmod.Wrap(hmerrors.ErrInvalidMsg, fmt.Sprint("No proposer in stored validator set"))
	}

	if msg.Proposer != validatorSet.Proposer.Signer {
		logger.Error(
			"Invalid proposer in msg",
			"proposer", validatorSet.Proposer.Signer,
			"msgProposer", msg.Proposer,
		)

		return nil, errorsmod.Wrap(hmerrors.ErrInvalidMsg, fmt.Sprint("Invalid proposer in msg"))
	}

	// Emit event for checkpoint
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCheckpoint,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyRootHash, msg.RootHash.String()),
			sdk.NewAttribute(types.AttributeKeyAccountHash, msg.AccountRootHash.String()),
		),
	})

	return &types.MsgCheckpointResponse{}, nil
}

// Delegate defines a method for performing a delegation of coins from a delegator to a validator
func (k msgServer) CheckpointAck(ctx context.Context, msg *types.MsgCheckpointAck) (*types.MsgCheckpointAckResponse, error) {
	logger := k.Logger(ctx)

	// Get last checkpoint from buffer
	headerBlock, err := k.GetCheckpointFromBuffer(ctx)
	if err != nil {
		logger.Error("Unable to get checkpoint", "error", err)
		return nil, errorsmod.Wrap(hmerrors.ErrBadAck, fmt.Sprint("Unable to get checkpoint"))
	}

	if msg.StartBlock != headerBlock.StartBlock {
		logger.Error("Invalid start block", "startExpected", headerBlock.StartBlock, "startReceived", msg.StartBlock)
		return nil, errorsmod.Wrap(hmerrors.ErrBadAck, fmt.Sprint("Invalid start block", "startExpected", headerBlock.StartBlock, "startReceived", msg.StartBlock))
	}

	// Return err if start and end matches but contract root hash doesn't match
	if msg.StartBlock == headerBlock.StartBlock && msg.EndBlock == headerBlock.EndBlock && !msg.RootHash.Equals(headerBlock.RootHash) {
		logger.Error("Invalid ACK",
			"startExpected", headerBlock.StartBlock,
			"startReceived", msg.StartBlock,
			"endExpected", headerBlock.EndBlock,
			"endReceived", msg.StartBlock,
			"rootExpected", headerBlock.RootHash.String(),
			"rootRecieved", msg.RootHash.String(),
		)
		return nil, errorsmod.Wrap(hmerrors.ErrBadAck, fmt.Sprint("Invalid Ack"))
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCheckpointAck,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyHeaderIndex, strconv.FormatUint(msg.Number, 10)),
		),
	})

	return &types.MsgCheckpointAckResponse{}, nil
}

// CheckpointNoAck handles checkpoint no-ack transaction
func (k msgServer) CheckpointNoAck(ctx context.Context, msg *types.MsgCheckpointNoAck) (*types.MsgCheckpointNoAckResponse, error) {
	logger := k.Logger(ctx)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get current block time
	currentTime := sdkCtx.BlockTime()

	// Get buffer time from params
	params, err := k.GetParams(ctx)
	if err != nil {
		logger.Error("Error in fetching checkpoint parameter")
		return nil, errorsmod.Wrap(hmerrors.ErrCheckpointParams, "Error in fetching checkpoint parameter")

	}

	bufferTime := params.CheckpointBufferTime

	// Fetch last checkpoint from store
	// TODO figure out how to handle this error
	lastCheckpoint, _ := k.GetLastCheckpoint(ctx)
	lastCheckpointTime := time.Unix(int64(lastCheckpoint.TimeStamp), 0)

	// If last checkpoint is not present or last checkpoint happens before checkpoint buffer time -- thrown an error
	if lastCheckpointTime.After(currentTime) || (currentTime.Sub(lastCheckpointTime) < bufferTime) {
		logger.Debug("Invalid No ACK -- Waiting for last checkpoint ACK", "lastCheckpointTime", lastCheckpointTime, "current time", currentTime,
			"buffer Time", bufferTime.String(),
		)

		return nil, errorsmod.Wrap(hmerrors.ErrInvalidNoACK, "Time as not expired till now")
	}

	timeDiff := currentTime.Sub(lastCheckpointTime)

	//count value is calculated based on the time passed since the last checkpoint
	count := math.Floor(timeDiff.Seconds() / bufferTime.Seconds())

	var isProposer bool = false

	currentValidatorSet := k.sk.GetValidatorSet(ctx)
	currentValidatorSet.IncrementProposerPriority(1)

	for i := 0; i < int(count); i++ {
		if strings.ToLower(currentValidatorSet.Proposer.Signer) == strings.ToLower(msg.From) {
			isProposer = true
			break
		}

		currentValidatorSet.IncrementProposerPriority(1)
	}

	//If NoAck sender is not the valid proposer, return error
	if !isProposer {
		return nil, errorsmod.Wrap(hmerrors.ErrInvalidNoACK, "Ack proposer is not correct")
	}

	// Check last no ack - prevents repetitive no-ack
	lastNoAck := k.GetLastNoAck(ctx)
	lastNoAckTime := time.Unix(int64(lastNoAck), 0)

	if lastNoAckTime.After(currentTime) || (currentTime.Sub(lastNoAckTime) < bufferTime) {
		logger.Debug("Too many no-ack", "lastNoAckTime", lastNoAckTime, "current time", currentTime,
			"buffer Time", bufferTime.String())

		return nil, errorsmod.Wrap(hmerrors.ErrTooManyNoACK, "Too many no acks")
	}

	// Set new last no-ack
	newLastNoAck := uint64(currentTime.Unix())
	k.SetLastNoAck(ctx, newLastNoAck)
	logger.Debug("Last No-ACK time set", "lastNoAck", newLastNoAck)

	//
	// Update to new proposer
	//

	// Increment accum (selects new proposer)
	k.sk.IncrementAccum(ctx, 1)

	// Get new proposer
	vs := k.sk.GetValidatorSet(ctx)
	newProposer := vs.GetProposer()
	logger.Debug(
		"New proposer selected",
		"validator", newProposer.Signer,
		"signer", newProposer.Signer,
		"power", newProposer.VotingPower,
	)

	// add events
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCheckpointNoAck,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyNewProposer, newProposer.Signer),
		),
	})

	return &types.MsgCheckpointNoAckResponse{}, nil
}
