package keeper

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	util "github.com/0xPolygon/heimdall-v2/common/address"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
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

// PostTxHandler returns a post handler for "checkpoint" type messages.
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
	// logger
	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCheckpoint)
	if !ok {
		logger.Error("type mismatch for MsgCheckpoint")
		return sidetxs.Vote_VOTE_NO
	}

	contractCaller := srv.IContractCaller

	chainParams, err := srv.ck.GetParams(ctx)
	if err != nil {
		logger.Error("error in getting chain manager params", "error", err)
		return sidetxs.Vote_VOTE_NO
	}

	borChainTxConfirmations := chainParams.BorChainTxConfirmations

	// get params
	params, err := srv.GetParams(ctx)
	if err != nil {
		logger.Error("error in getting params", "error", err)
		return sidetxs.Vote_VOTE_NO
	}

	// validate checkpoint
	validCheckpoint, err := types.IsValidCheckpoint(msg.StartBlock, msg.EndBlock, msg.RootHash, params.MaxCheckpointLength, contractCaller, borChainTxConfirmations)
	if err != nil {
		logger.Error("error validating checkpoint",
			"startBlock", msg.StartBlock,
			"endBlock", msg.EndBlock,
			"rootHash", msg.RootHash,
			"error", err,
		)
	} else if validCheckpoint {
		// vote `yes` if checkpoint is valid
		return sidetxs.Vote_VOTE_YES
	}

	logger.Error(
		"rootHash is not valid",
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"rootHash", msg.RootHash,
	)

	return sidetxs.Vote_VOTE_NO
}

// SideHandleMsgCheckpointAck handles side checkpoint-ack message
func (srv *sideMsgServer) SideHandleMsgCheckpointAck(ctx sdk.Context, sdkMsg sdk.Msg) sidetxs.Vote {
	// logger
	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCpAck)
	if !ok {
		logger.Error("type mismatch for MsgCpAck")
		return sidetxs.Vote_VOTE_NO
	}

	contractCaller := srv.IContractCaller

	chainParams, err := srv.ck.GetParams(ctx)
	if err != nil {
		logger.Error("error in getting chain manager params", "error", err)
		return sidetxs.Vote_VOTE_NO
	}

	rootChainAddress := chainParams.ChainParams.RootChainAddress

	// get params
	params, err := srv.GetParams(ctx)
	if err != nil {
		logger.Error("error in getting params", "error", err)
		return sidetxs.Vote_VOTE_NO
	}

	rootChainInstance, err := contractCaller.GetRootChainInstance(rootChainAddress)
	if err != nil {
		logger.Error("unable to fetch rootChain contract instance",
			"eth address", rootChainAddress,
			"error", err,
		)

		return sidetxs.Vote_VOTE_NO
	}

	headerInfo, err := contractCaller.GetHeaderInfo(msg.Number, rootChainInstance, params.ChildChainBlockInterval)
	if err != nil {
		logger.Error("unable to fetch checkpoint from rootChain", "checkpointNumber", msg.Number, "error", err)
		return sidetxs.Vote_VOTE_NO
	}

	root, start, end, proposer := headerInfo.Root, headerInfo.Start, headerInfo.End, headerInfo.Proposer

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
			"message root hash", string(msg.RootHash),
			"rootChain checkpoint root hash", root,
			"error", err,
		)

		return sidetxs.Vote_VOTE_NO
	}

	return sidetxs.Vote_VOTE_YES
}

// PostHandleMsgCheckpoint handles the checkpoint msg
func (srv *sideMsgServer) PostHandleMsgCheckpoint(ctx sdk.Context, sdkMsg sdk.Msg, sideTxResult sidetxs.Vote) {
	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCheckpoint)
	if !ok {
		logger.Error("type mismatch for MsgCheckpoint")
		return
	}

	// Skip handler if stakeUpdate is not approved
	if sideTxResult != sidetxs.Vote_VOTE_YES {
		logger.Debug("skipping stake update since side-tx didn't get yes votes")
		return
	}

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

	doExist, err := srv.HasCheckpointInBuffer(ctx)
	if err != nil {
		logger.Error("error in checking the existence of checkpoint in buffer", "error", err)
		return
	}

	checkpointBuffer, err := srv.GetCheckpointFromBuffer(ctx)
	if err == nil && doExist {
		logger.Debug("checkpoint already exists in buffer")

		// get checkpoint buffer time from params
		params, err := srv.GetParams(ctx)
		if err != nil {
			logger.Error("checkpoint params not found", "error", err)
			return
		}

		expiryTime := checkpointBuffer.Timestamp + uint64(params.CheckpointBufferTime.Seconds())

		logger.Error(fmt.Sprintf("checkpoint already exists in buffer, ack expected, expires at %s", strconv.FormatUint(expiryTime, 10)))

		return
	}

	timeStamp := uint64(ctx.BlockTime().Unix())

	// add checkpoint to buffer with root hash and account hash
	if err = srv.SetCheckpointBuffer(ctx, types.Checkpoint{
		StartBlock: msg.StartBlock,
		EndBlock:   msg.EndBlock,
		RootHash:   msg.RootHash,
		Proposer:   msg.Proposer,
		BorChainId: msg.BorChainId,
		Timestamp:  timeStamp,
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
}

// PostHandleMsgCheckpointAck handles checkpoint-ack
func (srv *sideMsgServer) PostHandleMsgCheckpointAck(ctx sdk.Context, sdkMsg sdk.Msg, sideTxResult sidetxs.Vote) {
	logger := srv.Logger(ctx)

	msg, ok := sdkMsg.(*types.MsgCpAck)
	if !ok {
		logger.Error("type mismatch for MsgCpAck")
		return
	}

	// skip handler if stakeUpdate is not approved
	if sideTxResult != sidetxs.Vote_VOTE_YES {
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

	// return err if start and end matches but contract root hash doesn't match
	if msg.StartBlock == checkpointObj.StartBlock && msg.EndBlock == checkpointObj.EndBlock && !bytes.Equal(msg.RootHash, checkpointObj.RootHash) {
		logger.Error("invalid ACK",
			"startExpected", checkpointObj.StartBlock,
			"startReceived", msg.StartBlock,
			"endExpected", checkpointObj.EndBlock,
			"endReceived", msg.StartBlock,
			"rootExpected", string(checkpointObj.RootHash),
			"rootReceived", string(msg.RootHash),
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

	// add checkpoint to store
	if err = srv.AddCheckpoint(ctx, msg.Number, checkpointObj); err != nil {
		logger.Error("error while adding checkpoint into store", "checkpointNumber", msg.Number)
		return
	}

	logger.Debug("checkpoint added to store", "checkpointNumber", msg.Number)

	// flush buffer
	err = srv.FlushCheckpointBuffer(ctx)
	if err != nil {
		logger.Error("error while flushing buffer", "error", err)
		return
	}

	logger.Debug("checkpoint buffer flushed after receiving checkpoint ack")

	// update ack count in staking module
	err = srv.IncrementAckCount(ctx)
	if err != nil {
		logger.Error("error while updating the ack count", "err", err)
		return
	}

	// increment accum (selects new proposer)
	err = srv.stakeKeeper.IncrementAccum(ctx, 1)
	if err != nil {
		logger.Error("error while incrementing accum", "err", err)
		return
	}

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
}
