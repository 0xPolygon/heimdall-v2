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
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
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

// Checkpoint function handles the checkpoint msg
func (srv msgServer) Checkpoint(ctx context.Context, msg *types.MsgCheckpoint) (*types.MsgCheckpointResponse, error) {
	logger := srv.Logger(ctx)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	timeStamp := uint64(sdkCtx.BlockTime().Unix())

	params, err := srv.GetParams(ctx)
	if err != nil {
		logger.Error("error in fetching checkpoint parameter")
		return nil, types.ErrCheckpointParams
	}

	checkpointBuffer, err := srv.GetCheckpointFromBuffer(ctx)
	if err == nil {
		checkpointBufferTime := uint64(params.CheckpointBufferTime.Seconds())

		if checkpointBuffer.Timestamp == 0 || ((timeStamp > checkpointBuffer.Timestamp) && (timeStamp-checkpointBuffer.Timestamp) >= checkpointBufferTime) {
			logger.Debug("checkpoint has been timed out. flushing buffer.", "checkpointTimestamp", timeStamp, "prevCheckpointTimestamp", checkpointBuffer.Timestamp)
			err := srv.FlushCheckpointBuffer(ctx)
			if err != nil {
				logger.Error("error in flushing the checkpoint buffer")
				return nil, types.ErrBufferFlush
			}
		} else {
			expiryTime := checkpointBuffer.Timestamp + checkpointBufferTime
			logger.Error("checkpoint already exits in buffer", "checkpoint", checkpointBuffer.String(), "expires", expiryTime)
			return nil, errorsmod.Wrap(types.ErrNoAck, fmt.Sprint("checkpoint already exits in buffer", "checkpoint", checkpointBuffer.String(), "expires", expiryTime))
		}
	}

	// fetch last checkpoint from store
	if lastCheckpoint, err := srv.GetLastCheckpoint(ctx); err == nil {
		// make sure new checkpoint is after tip
		if lastCheckpoint.EndBlock > msg.StartBlock {
			logger.Error("checkpoint already exists",
				"currentTip", lastCheckpoint.EndBlock,
				"startBlock", msg.StartBlock,
			)

			return nil, types.ErrOldCheckpoint
		}

		// check if new checkpoint's start block start from current tip
		if lastCheckpoint.EndBlock+1 != msg.StartBlock {
			logger.Error("checkpoint not in continuity",
				"currentTip", lastCheckpoint.EndBlock,
				"startBlock", msg.StartBlock)

			return nil, types.ErrDiscontinuousCheckpoint
		}
	} else if err.Error() == types.ErrNoCheckpointFound.Error() && msg.StartBlock != 0 {
		logger.Error("first checkpoint to start from block 0", "checkpoint start block", msg.StartBlock, "error", err)
		return nil, errorsmod.Wrap(types.ErrBadBlockDetails, fmt.Sprint("first checkpoint to start from block 0", "checkpoint start block", msg.StartBlock))
	}

	// Make sure latest AccountRootHash matches
	// Calculate new account root hash
	dividendAccounts, err := srv.topupKeeper.GetAllDividendAccounts(ctx)
	if err != nil {
		logger.Error("error while fetching dividends accounts", "error", err)
		return nil, errorsmod.Wrap(types.ErrBadBlockDetails, fmt.Sprint("error while fetching dividends accounts"))
	}

	logger.Debug("dividendAccounts of all validators", "dividendAccountsLength", len(dividendAccounts))

	// Get account root hash from dividend accounts
	accountRoot, err := hmTypes.GetAccountRootHash(dividendAccounts)
	if err != nil {
		logger.Error("error while fetching account root hash", "error", err)
		return nil, errorsmod.Wrap(types.ErrAccountHash, fmt.Sprint("error while fetching account root hash"))
	}

	logger.Debug("Validator account root hash generated", "accountRootHash", common.Bytes2Hex(accountRoot))

	// Compare stored root hash to msg root hash
	if !bytes.Equal(accountRoot, msg.AccountRootHash) {
		logger.Error(
			"AccountRootHash of current state doesn't match from msg",
			"hash", common.Bytes2Hex(accountRoot),
			"msgHash", msg.AccountRootHash,
		)
		return nil, errorsmod.Wrap(types.ErrBadBlockDetails, fmt.Sprint("accountRootHash of current state doesn't match from msg"))
	}

	// Check proposer in message
	validatorSet, err := srv.stakeKeeper.GetValidatorSet(ctx)
	if err != nil {
		logger.Error("no proposer in validator set", "msgProposer", msg.Proposer)
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, fmt.Sprint("no proposer stored in validator set"))
	}

	if validatorSet.Proposer == nil {
		logger.Error("no proposer in validator set", "msgProposer", msg.Proposer)
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, fmt.Sprint("no proposer stored in validator set"))
	}

	if msg.Proposer != validatorSet.Proposer.Signer {
		logger.Error(
			"invalid proposer in msg",
			"proposer", validatorSet.Proposer.Signer,
			"msgProposer", msg.Proposer,
		)

		return nil, errorsmod.Wrap(types.ErrInvalidMsg, fmt.Sprint("invalid proposer in msg"))
	}

	// Emit event for checkpoint
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCheckpoint,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyRootHash, common.Bytes2Hex(msg.RootHash)),
			sdk.NewAttribute(types.AttributeKeyAccountHash, common.Bytes2Hex(msg.AccountRootHash)),
		),
	})

	return &types.MsgCheckpointResponse{}, nil
}

// CheckpointAck handles the checkpoint ack msg
func (srv msgServer) CheckpointAck(ctx context.Context, msg *types.MsgCheckpointAck) (*types.MsgCheckpointAckResponse, error) {
	logger := srv.Logger(ctx)

	// get last checkpoint from buffer
	headerBlock, err := srv.GetCheckpointFromBuffer(ctx)
	if err != nil {
		logger.Error("unable to get checkpoint", "error", err)
		return nil, errorsmod.Wrap(types.ErrBadAck, fmt.Sprint("unable to get checkpoint"))
	}

	if msg.StartBlock != headerBlock.StartBlock {
		logger.Error("invalid start block", "startExpected", headerBlock.StartBlock, "startReceived", msg.StartBlock)
		return nil, errorsmod.Wrap(types.ErrBadAck, fmt.Sprint("invalid start block", "startExpected", headerBlock.StartBlock, "startReceived", msg.StartBlock))
	}

	// return err if start and end match but contract root hash doesn't match
	if msg.StartBlock == headerBlock.StartBlock &&
		msg.EndBlock == headerBlock.EndBlock &&
		!bytes.Equal(msg.RootHash, headerBlock.RootHash) {
		logger.Error("Invalid ACK",
			"startExpected", headerBlock.StartBlock,
			"startReceived", msg.StartBlock,
			"endExpected", headerBlock.EndBlock,
			"endReceived", msg.StartBlock,
			"rootExpected", common.Bytes2Hex(headerBlock.RootHash),
			"rootReceived", common.Bytes2Hex(msg.RootHash),
		)
		return nil, types.ErrBadAck
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

// CheckpointNoAck handles checkpoint no-ack msg
func (srv msgServer) CheckpointNoAck(ctx context.Context, msg *types.MsgCheckpointNoAck) (*types.MsgCheckpointNoAckResponse, error) {
	logger := srv.Logger(ctx)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get current block time
	currentTime := sdkCtx.BlockTime()

	// Get buffer time from params
	params, err := srv.GetParams(ctx)
	if err != nil {
		logger.Error("error in fetching checkpoint parameter", "error", err)
		return nil, errorsmod.Wrap(types.ErrCheckpointParams, "error in fetching checkpoint parameter")
	}

	bufferTime := params.CheckpointBufferTime

	var lastCheckpointTime time.Time

	lastCheckpoint, err := srv.GetLastCheckpoint(ctx)
	if err != nil {
		lastCheckpointTime = time.Unix(0, 0)
	} else {
		lastCheckpointTime = time.Unix(int64(lastCheckpoint.Timestamp), 0)
	}

	// If last checkpoint is not present or last checkpoint happens before checkpoint buffer time,throw an error
	if lastCheckpointTime.After(currentTime) || (currentTime.Sub(lastCheckpointTime) < bufferTime) {
		logger.Debug("invalid no ack, waiting for last checkpoint ack",
			"lastCheckpointTime", lastCheckpointTime,
			"current time", currentTime,
			"buffer Time", bufferTime.String(),
		)

		return nil, errorsmod.Wrap(types.ErrInvalidNoAck, "time has not expired until now")
	}

	timeDiff := currentTime.Sub(lastCheckpointTime)

	// count value is calculated based on the time passed since the last checkpoint
	count := math.Floor(timeDiff.Seconds() / bufferTime.Seconds())

	isProposer := false

	currentValidatorSet, err := srv.stakeKeeper.GetValidatorSet(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error while fetching validator set")
	}

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
		return nil, types.ErrInvalidNoAckProposer
	}

	// Check last no ack - prevents repetitive no-ack
	lastNoAck, err := srv.GetLastNoAck(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error while fetching last no ack")
	}

	if lastNoAck > math.MaxInt64 {
		return nil, errorsmod.Wrap(types.ErrNoAck, "last no-ack timestamp is too large")
	}

	lastNoAckTime := time.Unix(int64(lastNoAck), 0)

	if lastNoAckTime.After(currentTime) || (currentTime.Sub(lastNoAckTime) < bufferTime) {
		logger.Debug("too many no-ack", "lastNoAckTime", lastNoAckTime, "current time", currentTime,
			"buffer Time", bufferTime.String())

		return nil, types.ErrTooManyNoAck
	}

	// Set new last no-ack
	newLastNoAck := uint64(currentTime.Unix())
	err = srv.SetLastNoAck(ctx, newLastNoAck)
	if err != nil {
		return nil, types.ErrNoAck
	}
	logger.Debug("last no-ack time set", "lastNoAck", newLastNoAck)

	// increment accum (selects new proposer)
	err = srv.stakeKeeper.IncrementAccum(ctx, 1)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error in incrementing the accum number")

	}

	// get new proposer
	vs, err := srv.stakeKeeper.GetValidatorSet(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error in fetching the validator set")

	}

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
