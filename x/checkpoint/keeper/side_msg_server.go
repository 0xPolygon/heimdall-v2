package keeper

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	util "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics/api"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

type sideMsgServer struct {
	*Keeper
}

var (
	checkpointTypeUrl    = sdk.MsgTypeURL(&types.MsgCheckpoint{})
	checkpointAckTypeUrl = sdk.MsgTypeURL(&types.MsgCpAck{})
)

// NewSideMsgServerImpl returns an implementation of the checkpoint sideMsgServer interface
// for the provided Keeper.
func NewSideMsgServerImpl(keeper *Keeper) sidetxs.SideMsgServer {
	return &sideMsgServer{Keeper: keeper}
}

// SideTxHandler returns a side handler for "checkpoint" type messages.
func (srv *sideMsgServer) SideTxHandler(methodName string) sidetxs.SideTxHandler {
	switch methodName {
	case checkpointTypeUrl:
		return srv.SideHandleMsgCheckpoint
	case checkpointAckTypeUrl:
		return srv.SideHandleMsgCheckpointAck
	default:
		return nil
	}
}

// PostTxHandler returns a post-handler for "checkpoint" type messages.
func (srv *sideMsgServer) PostTxHandler(methodName string) sidetxs.PostTxHandler {
	switch methodName {
	case checkpointTypeUrl:
		return srv.PostHandleMsgCheckpoint
	case checkpointAckTypeUrl:
		return srv.PostHandleMsgCheckpointAck
	default:
		return nil
	}
}

// SideHandleMsgCheckpoint handles checkpoint message
func (srv *sideMsgServer) SideHandleMsgCheckpoint(ctx sdk.Context, sdkMsg sdk.Msg) (result sidetxs.Vote) {
	var err error
	startTime := time.Now()
	defer recordCheckpointMetric(api.SideHandleMsgCheckpointMethod, api.SideType, startTime, &err)

	// logger
	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCheckpoint)
	if !ok {
		logger.Error(helper.ErrTypeMismatch("MsgCheckpoint"))
		return sidetxs.Vote_VOTE_NO
	}

	contractCaller := srv.IContractCaller

	chainParams, err := srv.ck.GetParams(ctx)
	if err != nil {
		logger.Error(hmTypes.ErrMsgErrorInGettingChainManagerParams, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	borChainTxConfirmations := chainParams.BorChainTxConfirmations

	// get params
	params, err := srv.GetParams(ctx)
	if err != nil {
		logger.Error(hmTypes.ErrMsgFailedToFetchParams, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	chainParams, err = srv.ck.GetParams(ctx)
	if err != nil {
		logger.Error(hmTypes.ErrMsgErrorInGettingChainManagerParams, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}
	if !helper.ValidateChainID(msg.BorChainId, chainParams.ChainParams.BorChainId, logger, "checkpoint") {
		return sidetxs.Vote_VOTE_NO
	}

	// validate checkpoint number
	if lastCheckpoint, err := srv.GetLastCheckpoint(ctx); err == nil {
		// check if the new checkpoint's start block starts from the current tip
		if lastCheckpoint.EndBlock+1 != msg.StartBlock {
			logger.Error("Checkpoint not in continuity in checkpoint side handler",
				"currentTip", lastCheckpoint.EndBlock,
				"startBlock", msg.StartBlock)

			return sidetxs.Vote_VOTE_NO
		}
	} else if errors.Is(err, types.ErrNoCheckpointFound) && msg.StartBlock != 0 {
		logger.Error("First checkpoint to start from block 0", "checkpoint start block", msg.StartBlock, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	// Check the proposer in the message
	validatorSet, err := srv.stakeKeeper.GetValidatorSet(ctx)
	if err != nil {
		logger.Error(hmTypes.ErrMsgNoProposerInValidatorSet, "msgProposer", msg.Proposer)
		return sidetxs.Vote_VOTE_NO
	}

	if validatorSet.Proposer == nil {
		logger.Error(hmTypes.ErrMsgNoProposerInValidatorSet, "msgProposer", msg.Proposer)
		return sidetxs.Vote_VOTE_NO
	}

	msgProposer := util.FormatAddress(msg.Proposer)
	valProposer := util.FormatAddress(validatorSet.Proposer.Signer)

	if msgProposer != valProposer {
		logger.Error(
			hmTypes.ErrMsgInvalidProposerInMsg,
			hmTypes.LogKeyProposer, valProposer,
			"msgProposer", msgProposer,
		)

		return sidetxs.Vote_VOTE_NO
	}

	// Make sure the latest AccountRootHash matches
	// Calculate new account root hash
	dividendAccounts, err := srv.topupKeeper.GetAllDividendAccounts(ctx)
	if err != nil {
		logger.Error("Error while fetching dividends accounts", hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	logger.Debug("DividendAccounts of all validators", "dividendAccountsLength", len(dividendAccounts))

	// Get account root hash from dividend accounts
	accountRoot, err := hmTypes.GetAccountRootHash(dividendAccounts)
	if err != nil {
		logger.Error("Error while fetching account root hash", hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	logger.Debug("Validator account root hash generated", "accountRootHash", common.Bytes2Hex(accountRoot))

	// Compare stored root hash to msg root hash
	if !bytes.Equal(accountRoot, msg.AccountRootHash) {
		logger.Error(
			"AccountRootHash of current state doesn't match from msg",
			"hash", common.Bytes2Hex(accountRoot),
			"msgHash", msg.AccountRootHash,
		)
		return sidetxs.Vote_VOTE_NO
	}

	logger.Info("Validating checkpoint proposal",
		"proposer", msg.Proposer,
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"rootHash", common.Bytes2Hex(msg.RootHash),
		"borChainId", msg.BorChainId,
	)

	// validate checkpoint
	validCheckpoint, err := types.IsValidCheckpoint(msg.StartBlock, msg.EndBlock, msg.RootHash, params.MaxCheckpointLength, contractCaller, borChainTxConfirmations)
	if err != nil {
		logger.Error("Error validating checkpoint",
			"startBlock", msg.StartBlock,
			"endBlock", msg.EndBlock,
			"rootHash", common.Bytes2Hex(msg.RootHash),
			"error", err,
		)
	} else if validCheckpoint {
		logger.Info("Checkpoint validated successfully",
			"startBlock", msg.StartBlock,
			"endBlock", msg.EndBlock,
			"rootHash", common.Bytes2Hex(msg.RootHash),
		)
		// vote `yes` if checkpoint is valid
		return sidetxs.Vote_VOTE_YES
	}

	logger.Error(
		"rootHash is not valid",
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"rootHash", common.Bytes2Hex(msg.RootHash),
	)

	return sidetxs.Vote_VOTE_NO
}

// SideHandleMsgCheckpointAck handles side checkpoint-ack message
func (srv *sideMsgServer) SideHandleMsgCheckpointAck(ctx sdk.Context, sdkMsg sdk.Msg) sidetxs.Vote {
	var err error
	startTime := time.Now()
	defer recordCheckpointMetric(api.SideHandleMsgCheckpointAckMethod, api.SideType, startTime, &err)

	// logger
	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCpAck)
	if !ok {
		logger.Error("Type mismatch for MsgCpAck")
		return sidetxs.Vote_VOTE_NO
	}

	contractCaller := srv.IContractCaller

	chainParams, err := srv.ck.GetParams(ctx)
	if err != nil {
		logger.Error(hmTypes.ErrMsgErrorInGettingChainManagerParams, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	rootChainAddress := chainParams.ChainParams.RootChainAddress

	// get params
	params, err := srv.GetParams(ctx)
	if err != nil {
		logger.Error(hmTypes.ErrMsgFailedToFetchParams, hmTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	rootChainInstance, err := contractCaller.GetRootChainInstance(rootChainAddress)
	if err != nil {
		logger.Error("Unable to fetch rootChain contract instance",
			"eth address", rootChainAddress,
			"error", err,
		)

		return sidetxs.Vote_VOTE_NO
	}

	logger.Info("Validating checkpoint ack",
		"checkpointNumber", msg.Number,
		"proposer", msg.Proposer,
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"rootHash", common.Bytes2Hex(msg.RootHash),
	)

	root, start, end, _, proposer, err := contractCaller.GetHeaderInfo(msg.Number, rootChainInstance, params.ChildChainBlockInterval)
	if err != nil {
		logger.Error("Unable to fetch checkpoint from rootChain", "checkpointNumber", msg.Number, "error", err)
		return sidetxs.Vote_VOTE_NO
	}

	// validate start and end blocks
	if msg.StartBlock >= msg.EndBlock {
		logger.Error("End block should be greater than start block in checkpoint side handler",
			"startBlock", msg.StartBlock,
			"endBlock", msg.EndBlock,
			"rootHash", common.Bytes2Hex(msg.RootHash),
		)
		return sidetxs.Vote_VOTE_NO
	}

	// validate checkpoint number
	lastCheckpoint, err := srv.GetLastCheckpoint(ctx)
	if err != nil && !errors.Is(err, types.ErrNoCheckpointFound) {
		logger.Error("Unable to get last checkpoint in checkpoint side handler", "error", err)
		return sidetxs.Vote_VOTE_NO
	}

	expectedId := uint64(1)

	if err == nil {
		expectedId = lastCheckpoint.Id + 1
	}

	if msg.Number != expectedId {
		logger.Error("Ack number in checkpoint side handler is not sequential",
			"lastCheckpointNumber", expectedId,
			"checkpointNumber", msg.Number,
		)
		return sidetxs.Vote_VOTE_NO

	}

	// validate buffered checkpoint
	bufCheckpoint, err := srv.GetCheckpointFromBuffer(ctx)
	if err != nil {
		logger.Error("Unable to get buffered checkpoint in checkpoint side handler", "error", err)
		return sidetxs.Vote_VOTE_NO
	}

	if IsBufferedCheckpointZero(bufCheckpoint) {
		logger.Debug("No checkpoint in buffer, cannot process ack in checkpoint side handler")
		return sidetxs.Vote_VOTE_NO
	}

	if msg.StartBlock != bufCheckpoint.StartBlock {
		logger.Error("Invalid start block during side handler checkpoint ack", "startExpected", bufCheckpoint.StartBlock, "startReceived", msg.StartBlock)
		return sidetxs.Vote_VOTE_NO
	}

	// return an error if start and end match, but contract root hash doesn't match
	if msg.EndBlock == bufCheckpoint.EndBlock &&
		!bytes.Equal(msg.RootHash, bufCheckpoint.RootHash) {
		logger.Error("Invalid ack in checkpoint side handler",
			"startExpected", bufCheckpoint.StartBlock,
			"startReceived", msg.StartBlock,
			"endExpected", bufCheckpoint.EndBlock,
			"endReceived", msg.StartBlock,
			"rootExpected", common.Bytes2Hex(bufCheckpoint.RootHash),
			"rootReceived", common.Bytes2Hex(msg.RootHash),
		)
		return sidetxs.Vote_VOTE_NO
	}

	// check if message data matches with contract data
	if msg.StartBlock != start ||
		msg.EndBlock != end ||
		strings.Compare(util.FormatAddress(msg.Proposer), util.FormatAddress(proposer)) != 0 ||
		!bytes.Equal(msg.RootHash, root.Bytes()) {
		logger.Error("invalid message as it doesn't match with contract state",
			"checkpointNumber", msg.Number,
			"message start block", msg.StartBlock,
			"rootChain checkpoint start block", start,
			"message end block", msg.EndBlock,
			"rootChain checkpoint end block", end,
			"message proposer", msg.Proposer,
			"rootChain checkpoint proposer", proposer,
			"message root hash", common.Bytes2Hex(msg.RootHash),
			"rootChain checkpoint root hash", root,
			"error", err,
		)

		return sidetxs.Vote_VOTE_NO
	}

	logger.Info("Checkpoint ack validated successfully",
		"checkpointNumber", msg.Number,
		"proposer", msg.Proposer,
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"rootHash", common.Bytes2Hex(msg.RootHash),
	)

	return sidetxs.Vote_VOTE_YES
}

// PostHandleMsgCheckpoint handles the checkpoint msg
func (srv *sideMsgServer) PostHandleMsgCheckpoint(ctx sdk.Context, sdkMsg sdk.Msg, sideTxResult sidetxs.Vote) error {
	var err error
	startTime := time.Now()
	defer recordCheckpointMetric(api.PostHandleMsgCheckpointMethod, api.PostType, startTime, &err)

	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCheckpoint)
	if !ok {
		err := errors.New(helper.ErrTypeMismatch("MsgCheckpoint"))
		logger.Error(err.Error())
		return err
	}

	// Skip handler if stakeUpdate is not approved
	if !helper.IsSideTxApproved(sideTxResult) {
		logger.Debug(helper.ErrSkippingMsg("Checkpoint"))
		return errors.New(hmTypes.ErrMsgSideTxRejected)
	}

	// fetch the last checkpoint from the store
	lastCheckpoint, err := srv.GetLastCheckpoint(ctx)
	if err == nil {
		// check if the new checkpoint's start block starts from the current tip
		if lastCheckpoint.EndBlock+1 != msg.StartBlock {
			logger.Error("Checkpoint not in continuity",
				"currentTip", lastCheckpoint.EndBlock,
				"startBlock", msg.StartBlock)

			return errors.New(hmTypes.ErrMsgBlocksNotInContinuity)
		}
	} else if errors.Is(err, types.ErrNoCheckpointFound) && msg.StartBlock != 0 {
		logger.Error("First checkpoint to start from block 0", hmTypes.LogKeyError, err)
		return err
	}

	doExist, err := srv.HasCheckpointInBuffer(ctx)
	if err != nil {
		logger.Error("Error in checking the existence of checkpoint in buffer", hmTypes.LogKeyError, err)
		return err
	}

	checkpointBuffer, err := srv.GetCheckpointFromBuffer(ctx)
	if err != nil {
		logger.Error("Error in getting checkpoint from buffer", hmTypes.LogKeyError, err)
		return err
	}

	if doExist && !IsBufferedCheckpointZero(checkpointBuffer) {
		logger.Debug("Checkpoint already exists in buffer")

		// get checkpoint buffer time from params
		params, err := srv.GetParams(ctx)
		if err != nil {
			logger.Error("Checkpoint params not found", hmTypes.LogKeyError, err)
			return err
		}

		expiryTime := checkpointBuffer.Timestamp + uint64(params.CheckpointBufferTime.Seconds())

		logger.Error(fmt.Sprintf("Checkpoint already exists in buffer, ack expected, expires at %s", strconv.FormatUint(expiryTime, 10)))

		return errors.New("checkpoint already exists in buffer")
	}

	timeStamp := uint64(ctx.BlockTime().Unix())

	// add checkpoint to buffer with root hash and account hash
	if err = srv.SetCheckpointBuffer(ctx, types.Checkpoint{
		Id:         lastCheckpoint.Id + 1,
		StartBlock: msg.StartBlock,
		EndBlock:   msg.EndBlock,
		RootHash:   msg.RootHash,
		Proposer:   msg.Proposer,
		BorChainId: msg.BorChainId,
		Timestamp:  timeStamp,
	}); err != nil {
		logger.Error("Failed to set checkpoint buffer", "Error", err)
		return err
	}

	logger.Info("New checkpoint is stored in buffer",
		"checkpointId", lastCheckpoint.Id+1,
		"proposer", msg.Proposer,
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"rootHash", common.Bytes2Hex(msg.RootHash),
	)

	// TX bytes
	txBytes := ctx.TxBytes()

	// Emit event for checkpoints
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCheckpoint,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),    // module name
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, common.Bytes2Hex(txBytes)),   // tx hash
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()), // result
			sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyRootHash, common.Bytes2Hex(msg.RootHash)),
			sdk.NewAttribute(types.AttributeKeyAccountHash, common.Bytes2Hex(msg.AccountRootHash)),
		),
	})

	return nil
}

// PostHandleMsgCheckpointAck handles checkpoint-ack
func (srv *sideMsgServer) PostHandleMsgCheckpointAck(ctx sdk.Context, sdkMsg sdk.Msg, sideTxResult sidetxs.Vote) error {
	var err error
	startTime := time.Now()
	defer recordCheckpointMetric(api.PostHandleMsgCheckpointAckMethod, api.PostType, startTime, &err)

	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCpAck)
	if !ok {
		err := errors.New(helper.ErrTypeMismatch("MsgCpAck"))
		logger.Error(err.Error())
		return err
	}

	// skip handler if ACK is not approved
	if !helper.IsSideTxApproved(sideTxResult) {
		logger.Debug(helper.ErrSkippingMsg("CheckpointAck"))
		return errors.New(hmTypes.ErrMsgSideTxRejected)
	}

	// get the last checkpoint from the buffer
	checkpointObj, err := srv.GetCheckpointFromBuffer(ctx)
	if err != nil {
		logger.Error("Fnable to get checkpoint buffer", hmTypes.LogKeyError, err)
		return err
	}

	if IsBufferedCheckpointZero(checkpointObj) {
		logger.Debug("No checkpoint in buffer, cannot process checkpoint ack in postHandler")
		return errors.New("no checkpoint in buffer, cannot process checkpoint ack in postHandler")
	}

	// invalid start block
	if msg.StartBlock != checkpointObj.StartBlock {
		logger.Error("Invalid start block during postHandler checkpoint ack", "startExpected", checkpointObj.StartBlock, "startReceived", msg.StartBlock)
		return errors.New("invalid start block during postHandler checkpoint ack")
	}

	// return an error if start and end match, but contract root hash doesn't match
	if msg.EndBlock == checkpointObj.EndBlock && !bytes.Equal(msg.RootHash, checkpointObj.RootHash) {
		logger.Error("Invalid ack",
			"startExpected", checkpointObj.StartBlock,
			"startReceived", msg.StartBlock,
			"endExpected", checkpointObj.EndBlock,
			"endReceived", msg.EndBlock,
			"rootExpected", common.Bytes2Hex(checkpointObj.RootHash),
			"rootReceived", common.Bytes2Hex(msg.RootHash),
		)

		return errors.New("invalid ACK")
	}

	// adjust checkpoint data if the latest checkpoint is already submitted
	if checkpointObj.EndBlock != msg.EndBlock {
		logger.Info("Adjusting endBlock to one already submitted on chain", "endBlock", checkpointObj.EndBlock, "adjustedEndBlock", msg.EndBlock)
		checkpointObj.EndBlock = msg.EndBlock
		checkpointObj.RootHash = msg.RootHash
		checkpointObj.Proposer = msg.Proposer
	}

	// add checkpoint to store
	checkpointObj.Id = msg.Number
	if err = srv.AddCheckpoint(ctx, checkpointObj); err != nil {
		logger.Error("Error while adding checkpoint into store", "checkpointNumber", msg.Number)
		return err
	}

	logger.Info("Checkpoint added to the store",
		"checkpointNumber", msg.Number,
		"proposer", checkpointObj.Proposer,
		"startBlock", checkpointObj.StartBlock,
		"endBlock", checkpointObj.EndBlock,
		"rootHash", common.Bytes2Hex(checkpointObj.RootHash),
	)

	// flush buffer
	err = srv.FlushCheckpointBuffer(ctx)
	if err != nil {
		logger.Error("Error while flushing buffer", hmTypes.LogKeyError, err)
		return err
	}

	logger.Info("Checkpoint buffer flushed after receiving checkpoint ack", "checkpointNumber", msg.Number)

	// update ack count module
	err = srv.IncrementAckCount(ctx)
	if err != nil {
		logger.Error("Error while updating the ack count", "err", err)
		return err
	}

	// increment accum (selects new proposer)
	err = srv.stakeKeeper.IncrementAccum(ctx, 1)
	if err != nil {
		logger.Error("Error while incrementing accum", "err", err)
		return err
	}

	// get the new proposer from validators set
	vs, err := srv.stakeKeeper.GetValidatorSet(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "error in fetching the validator set")
	}

	newProposer := vs.GetProposer()
	// should never happen
	if newProposer == nil {
		logger.Error("No proposer available (empty validator set!) during postHandler ack message",
			"oldProposer", msg.From,
		)
		return errorsmod.Wrap(err, "no proposer available (empty validator set!) during postHandler ack message")
	}
	// log old and new proposer
	newProposerAddr := util.FormatAddress(newProposer.Signer)
	oldProposerAddr := util.FormatAddress(msg.From)
	logger.Info(
		"New proposer selected for checkpoint ack message",
		"oldProposer", oldProposerAddr,
		"newProposer", newProposerAddr,
	)

	txBytes := ctx.TxBytes()

	// Emit event for checkpoints
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCheckpointAck,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),    // module name
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, common.Bytes2Hex(txBytes)),   // tx hash
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()), // result
			sdk.NewAttribute(types.AttributeKeyHeaderIndex, strconv.FormatUint(msg.Number, 10)),
		),
	})

	return nil
}

// recordCheckpointMetric records metrics for side and post-handlers.
func recordCheckpointMetric(method string, apiType string, start time.Time, err *error) {
	success := *err == nil
	api.RecordAPICallWithStart(api.CheckpointSubsystem, method, apiType, success, start)
}

func IsBufferedCheckpointZero(cp types.Checkpoint) bool {
	return cp.Id == 0 &&
		cp.Proposer == "" &&
		cp.StartBlock == 0 &&
		cp.EndBlock == 0 &&
		len(cp.RootHash) == 0 &&
		cp.BorChainId == "" &&
		cp.Timestamp == 0
}
