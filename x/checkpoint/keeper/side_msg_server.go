package keeper

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	hmModule "github.com/0xPolygon/heimdall-v2/module"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
)

type sideMsgServer struct {
	*Keeper
}

var (
	checkpointAdjust = sdk.MsgTypeURL(&types.MsgCheckpointAdjust{})
	checkpoint       = sdk.MsgTypeURL(&types.MsgCheckpoint{})
	checkpointAck    = sdk.MsgTypeURL(&types.MsgCheckpointAck{})
)

// NewSideMsgServerImpl returns an implementation of the checkpoint sideMsgServer interface
// for the provided Keeper.
func NewSideMsgServerImpl(keeper *Keeper) types.SideMsgServer {
	return &sideMsgServer{Keeper: keeper}
}

// SideTxHandler returns a side handler for "checkpoin" type messages.
func (srv *sideMsgServer) SideTxHandler(methodName string) hmModule.SideTxHandler {

	switch methodName {
	case checkpointAdjust:
		return srv.SideHandleCheckpointAdjust
	case checkpoint:
		return srv.SideHandleMsgCheckpoint
	case checkpointAck:
		return srv.SideHandleMsgCheckpointAck
	default:
		return nil
	}
}

// PostTxHandler returns a post handler for "checkpoint" type messages.
func (srv *sideMsgServer) PostTxHandler(methodName string) hmModule.PostTxHandler {

	switch methodName {
	case checkpointAdjust:
		return srv.PostHandleMsgCheckpointAdjust
	case checkpoint:
		return srv.PostHandleMsgCheckpoint
	case checkpointAck:
		return srv.PostHandleMsgCheckpointAck
	default:
		return nil
	}
}

// SideHandleCheckpointAdjust side msg for checkpoint adjust
func (srv *sideMsgServer) SideHandleCheckpointAdjust(ctx sdk.Context, sdkMsg sdk.Msg) (result hmModule.Vote) {
	// logger
	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCheckpointAdjust)
	if !ok {
		logger.Error("msg type mismatched")
		return hmModule.Vote_VOTE_NO
	}

	chainParams, err := srv.ck.GetParams(ctx)
	if err != nil {
		logger.Error("error in getting chain manager params", "error", err)
		return hmModule.Vote_VOTE_NO
	}

	rootChainAddress := chainParams.ChainParams.RootChainAddress

	params, err := srv.GetParams(ctx)
	if err != nil {
		logger.Error("error in getting params", "error", err)
		return hmModule.Vote_VOTE_NO
	}

	contractCaller := srv.IContractCaller

	checkpointBuffer, err := srv.GetCheckpointFromBuffer(ctx)
	if checkpointBuffer != nil {
		logger.Error("checkpoint already exists in buffer", "error", err)
		return hmModule.Vote_VOTE_NO
	}

	checkpointObj, err := srv.GetCheckpointByNumber(ctx, msg.HeaderIndex)
	if err != nil {
		logger.Error("unable to get checkpoint from db", "header index", msg.HeaderIndex, "error", err)
		return hmModule.Vote_VOTE_NO
	}

	rootChainInstance, err := contractCaller.GetRootChainInstance(rootChainAddress)
	if err != nil {
		logger.Error("nable to fetch rootchain contract instance", "eth address", rootChainAddress, "error", err)
		return hmModule.Vote_VOTE_NO
	}

	root, start, end, _, proposer, err := contractCaller.GetHeaderInfo(msg.HeaderIndex, rootChainInstance, params.ChildBlockInterval)
	if err != nil {
		logger.Error("unable to fetch checkpoint from rootchain", "checkpointNumber", msg.HeaderIndex, "error", err)
		return hmModule.Vote_VOTE_NO
	}

	if checkpointObj.EndBlock == end &&
		checkpointObj.StartBlock == start &&
		bytes.Equal(checkpointObj.RootHash.Bytes(), root.Bytes()) &&
		strings.EqualFold(checkpointObj.Proposer, proposer) {
		logger.Error("same checkpoint in db")
		return hmModule.Vote_VOTE_NO
	}

	if msg.EndBlock != end || msg.StartBlock != start || !bytes.Equal(msg.RootHash.Bytes(), root.Bytes()) || strings.ToLower(msg.Proposer) != strings.ToLower(proposer) {
		logger.Error("checkpoint fields fetched from ethereum does match with those in msg",
			"message start block", msg.StartBlock,
			"Rootchain Checkpoint start block", start,
			"message end block", msg.EndBlock,
			"Rootchain Checkpointt end block", end,
			"message proposer", msg.Proposer,
			"Rootchain Checkpoint proposer", proposer,
			"message root hash", msg.RootHash,
			"Rootchain Checkpoint root hash", root,
		)

		return hmModule.Vote_VOTE_NO
	}

	return hmModule.Vote_VOTE_YES
}

// SideHandleMsgCheckpoint handles checkpoint message
func (srv *sideMsgServer) SideHandleMsgCheckpoint(ctx sdk.Context, sdkMsg sdk.Msg) (result hmModule.Vote) {
	// logger
	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCheckpoint)
	if !ok {
		logger.Error("msg type mismatched")
		return hmModule.Vote_VOTE_NO
	}

	contractCaller := srv.IContractCaller

	chainParams, err := srv.ck.GetParams(ctx)
	if err != nil {
		logger.Error("error in getting chain manager params", "error", err)
		return hmModule.Vote_VOTE_NO
	}

	maticTxConfirmations := chainParams.BorChainTxConfirmations

	// get params
	params, err := srv.GetParams(ctx)
	if err != nil {
		logger.Error("error in getting params", "error", err)
		return hmModule.Vote_VOTE_NO
	}

	// validate checkpoint
	validCheckpoint, err := types.ValidateCheckpoint(msg.StartBlock, msg.EndBlock, msg.RootHash, params.MaxCheckpointLength, contractCaller, maticTxConfirmations)
	if err != nil {
		logger.Error("error validating checkpoint",
			"startBlock", msg.StartBlock,
			"endBlock", msg.EndBlock,
			"rootHash", msg.RootHash,
			"error", err,
		)
	} else if validCheckpoint {
		// vote `yes` if checkpoint is valid
		return hmModule.Vote_VOTE_YES
	}

	logger.Error(
		"rootHash is not valid",
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"rootHash", msg.RootHash,
	)

	return hmModule.Vote_VOTE_NO
}

// SideHandleMsgCheckpointAck handles side checkpoint-ack message
func (srv *sideMsgServer) SideHandleMsgCheckpointAck(ctx sdk.Context, sdkMsg sdk.Msg) hmModule.Vote {
	// logger
	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCheckpointAck)
	if !ok {
		logger.Error("msg type mismatched")
		return hmModule.Vote_VOTE_NO
	}

	contractCaller := srv.IContractCaller

	chainParams, err := srv.ck.GetParams(ctx)
	if err != nil {
		logger.Error("error in getting chain manager params", "error", err)
		return hmModule.Vote_VOTE_NO
	}

	rootChainAddress := chainParams.ChainParams.RootChainAddress

	// get params
	params, err := srv.GetParams(ctx)
	if err != nil {
		logger.Error("error in getting params", "error", err)
		return hmModule.Vote_VOTE_NO
	}

	rootChainInstance, err := contractCaller.GetRootChainInstance(rootChainAddress)
	if err != nil {
		logger.Error("unable to fetch rootchain contract instance",
			"eth address", rootChainAddress,
			"error", err,
		)

		return hmModule.Vote_VOTE_NO
	}

	root, start, end, _, proposer, err := contractCaller.GetHeaderInfo(msg.Number, rootChainInstance, params.ChildBlockInterval)
	if err != nil {
		logger.Error("unable to fetch checkpoint from rootchain", "checkpointNumber", msg.Number, "error", err)
		return hmModule.Vote_VOTE_NO
	}

	// check if message data matches with contract data
	if msg.StartBlock != start ||
		msg.EndBlock != end ||
		strings.ToLower(msg.Proposer) != strings.ToLower(proposer) ||
		!bytes.Equal(msg.RootHash.Bytes(), root.Bytes()) {
		logger.Error("invalid message as it doesn't match with contract state",
			"checkpointNumber", msg.Number,
			"message start block", msg.StartBlock,
			"Rootchain Checkpoint start block", start,
			"message end block", msg.EndBlock,
			"Rootchain Checkpointt end block", end,
			"message proposer", msg.Proposer,
			"Rootchain Checkpoint proposer", proposer,
			"message root hash", msg.RootHash,
			"Rootchain Checkpoint root hash", root,
			"error", err,
		)

		return hmModule.Vote_VOTE_NO
	}

	// say `yes`
	return hmModule.Vote_VOTE_YES
}

/*
	Post Handlers - update the state of the tx
**/

// PostHandleMsgCheckpointAdjust msg for checkpointAdjust
func (srv *sideMsgServer) PostHandleMsgCheckpointAdjust(ctx sdk.Context, sdkMsg sdk.Msg, sideTxResult hmModule.Vote) {
	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCheckpointAdjust)
	if !ok {
		logger.Error("msg type mismatched")
		return
	}

	// Skip handler if validator join is not approved
	if sideTxResult != hmModule.Vote_VOTE_YES {
		logger.Debug("skipping new validator-join since side-tx didn't get yes votes")
		return
	}

	checkpointBuffer, err := srv.GetCheckpointFromBuffer(ctx)
	if checkpointBuffer != nil {
		logger.Error("checkpoint buffer exists", "error", err)
		return
	}

	checkpointObj, err := srv.GetCheckpointByNumber(ctx, msg.HeaderIndex)
	if err != nil {
		logger.Error("unable to get checkpoint from db",
			"checkpoint number", msg.HeaderIndex,
			"error", err)

		return
	}

	if checkpointObj.EndBlock == msg.EndBlock && checkpointObj.StartBlock == msg.StartBlock && bytes.Equal(checkpointObj.RootHash.Bytes(), msg.RootHash.Bytes()) && strings.ToLower(checkpointObj.Proposer) == strings.ToLower(msg.Proposer) {
		logger.Error("same Checkpoint in db")
		return
	}

	logger.Info("Previous checkpoint details: EndBlock -", checkpointObj.EndBlock, ", RootHash -", msg.RootHash, " Proposer -", checkpointObj.Proposer)

	checkpointObj.EndBlock = msg.EndBlock
	checkpointObj.RootHash = hmTypes.BytesToHeimdallHash(msg.RootHash.Bytes())
	checkpointObj.Proposer = msg.Proposer

	logger.Info("new checkpoint details: endBlock", checkpointObj.EndBlock, ", rootHash :", msg.RootHash, " proposer :", checkpointObj.Proposer)

	//
	// Update checkpoint state
	//

	// Add checkpoint to store
	if err = srv.AddCheckpoint(ctx, msg.HeaderIndex, checkpointObj); err != nil {
		logger.Error("error while adding checkpoint into store", "checkpointNumber", msg.HeaderIndex)
		return
	}

	logger.Debug("checkpoint updated to store", "checkpointNumber", msg.HeaderIndex)

	// Emit event for checkpoints
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCheckpointAck,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),    // module name
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()), // result
			sdk.NewAttribute(types.AttributeKeyHeaderIndex, strconv.FormatUint(msg.HeaderIndex, 10)),
			sdk.NewAttribute(types.AttributeKeyStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyRootHash, msg.RootHash.String()),
		),
	})

	return
}

// PostHandleMsgCheckpoint handles the checkpoint msg
func (srv *sideMsgServer) PostHandleMsgCheckpoint(ctx sdk.Context, sdkMsg sdk.Msg, sideTxResult hmModule.Vote) {
	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCheckpoint)
	if !ok {
		logger.Error("msg type mismatched")
		return
	}

	// Skip handler if stakeUpdate is not approved
	if sideTxResult != hmModule.Vote_VOTE_YES {
		logger.Debug("skipping stake update since side-tx didn't get yes votes")
		return
	}

	//
	// Validate last checkpoint
	//

	// fetch last checkpoint from store
	if lastCheckpoint, err := srv.GetLastCheckpoint(ctx); err == nil {
		// make sure new checkpoint is after tip
		if lastCheckpoint.EndBlock > msg.StartBlock {
			logger.Error("checkpoint already exists",
				"currentTip", lastCheckpoint.EndBlock,
				"startBlock", msg.StartBlock,
			)

			return
		}

		// check if new checkpoint's start block start from current tip
		if lastCheckpoint.EndBlock+1 != msg.StartBlock {
			logger.Error("checkpoint not in continuity",
				"currentTip", lastCheckpoint.EndBlock,
				"startBlock", msg.StartBlock)

			return
		}
	} else if err.Error() == types.ErrNoCheckpointFound.Error() && msg.StartBlock != 0 {
		logger.Error("first checkpoint to start from block 0", "error", err)
		return
	}

	//
	// Save checkpoint to buffer store
	//

	checkpointBuffer, err := srv.GetCheckpointFromBuffer(ctx)
	if err == nil && checkpointBuffer != nil {
		logger.Debug("checkpoint already exists in buffer")

		// get checkpoint buffer time from params
		params, err := srv.GetParams(ctx)
		if err != nil {
			logger.Error("checkpoint params not found", "error", err)
		}

		expiryTime := checkpointBuffer.TimeStamp + uint64(params.CheckpointBufferTime.Seconds())

		logger.Error(fmt.Sprintf("checkpoint already exists in buffer, ack expected, expires at %s", strconv.FormatUint(expiryTime, 10)))

		return
	}

	timeStamp := uint64(ctx.BlockTime().Unix())

	// Add checkpoint to buffer with root hash and account hash
	if err = srv.SetCheckpointBuffer(ctx, types.Checkpoint{
		StartBlock: msg.StartBlock,
		EndBlock:   msg.EndBlock,
		RootHash:   msg.RootHash,
		Proposer:   msg.Proposer,
		BorChainID: msg.BorChainID,
		TimeStamp:  timeStamp,
	}); err != nil {
		logger.Error("failed to set checkpoint buffer", "Error", err)
	}

	logger.Debug("new checkpoint into buffer stored",
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"rootHash", msg.RootHash,
	)

	// TX bytes
	txBytes := ctx.TxBytes()
	hash := hmTypes.TxHash{txBytes}.Bytes()

	// Emit event for checkpoints
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCheckpoint,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),                // module name
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, hmTypes.BytesToHeimdallHash(hash).Hex()), // tx hash
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),             // result
			sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyRootHash, msg.RootHash.String()),
			sdk.NewAttribute(types.AttributeKeyAccountHash, msg.AccountRootHash.String()),
		),
	})
}

// PostHandleMsgCheckpointAck handles checkpoint-ack
func (srv *sideMsgServer) PostHandleMsgCheckpointAck(ctx sdk.Context, sdkMsg sdk.Msg, sideTxResult hmModule.Vote) {
	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCheckpointAck)
	if !ok {
		logger.Error("msg type mismatched")
		return
	}

	// Skip handler if stakeUpdate is not approved
	if sideTxResult != hmModule.Vote_VOTE_YES {
		logger.Debug("skipping stake update since side-tx didn't get yes votes")
		return
	}

	// get last checkpoint from buffer
	checkpointObj, err := srv.GetCheckpointFromBuffer(ctx)
	if err != nil {
		logger.Error("unable to get checkpoint buffer", "error", err)
		return
	}

	// invalid start block
	if msg.StartBlock != checkpointObj.StartBlock {
		logger.Error("invalid start block", "startExpected", checkpointObj.StartBlock, "startReceived", msg.StartBlock)
		return
	}

	// return err if start and end matche but contract root hash doesn't match
	if msg.StartBlock == checkpointObj.StartBlock && msg.EndBlock == checkpointObj.EndBlock && !msg.RootHash.Equals(checkpointObj.RootHash) {
		logger.Error("invalid ACK",
			"startExpected", checkpointObj.StartBlock,
			"startReceived", msg.StartBlock,
			"endExpected", checkpointObj.EndBlock,
			"endReceived", msg.StartBlock,
			"rootExpected", checkpointObj.RootHash.String(),
			"rootReceived", msg.RootHash.String(),
		)

		return
	}

	// adjust checkpoint data if latest checkpoint is already submitted

	if checkpointObj.EndBlock != msg.EndBlock {
		logger.Info("adjusting endBlock to one already submitted on chain", "endBlock", checkpointObj.EndBlock, "adjustedEndBlock", msg.EndBlock)
		checkpointObj.EndBlock = msg.EndBlock
		checkpointObj.RootHash = msg.RootHash
		checkpointObj.Proposer = msg.Proposer
	}

	//
	// Update checkpoint state
	//

	// Add checkpoint to store
	if err = srv.AddCheckpoint(ctx, msg.Number, *checkpointObj); err != nil {
		logger.Error("error while adding checkpoint into store", "checkpointNumber", msg.Number)
		return
	}

	logger.Debug("checkpoint added to store", "checkpointNumber", msg.Number)

	// Flush buffer
	srv.FlushCheckpointBuffer(ctx)

	logger.Debug("checkpoint buffer flushed after receiving checkpoint ack")

	// Update ack count in staking module
	srv.UpdateACKCount(ctx)

	logger.Info("valid ack received", "currentACKCount", srv.GetACKCount(ctx)-1, "updatedACKCount", srv.GetACKCount(ctx))

	// Increment accum (selects new proposer)
	srv.sk.IncrementAccum(ctx, 1)

	// TX bytes
	txBytes := ctx.TxBytes()
	hash := hmTypes.TxHash{txBytes}.Bytes()

	// Emit event for checkpoints
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCheckpointAck,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),                // module name
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, hmTypes.BytesToHeimdallHash(hash).Hex()), // tx hash
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),             // result
			sdk.NewAttribute(types.AttributeKeyHeaderIndex, strconv.FormatUint(msg.Number, 10)),
		),
	})

	return
}
