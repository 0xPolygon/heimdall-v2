package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	hmModule "github.com/0xPolygon/heimdall-v2/module"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

type sideMsgServer struct {
	*Keeper
}

var (
	milestoneMsgTypeURL = sdk.MsgTypeURL(&types.MsgMilestone{})
)

// NewSideMsgServerImpl returns an implementation of the milestone MsgServer interface
// for the provided Keeper.
func NewSideMsgServerImpl(keeper *Keeper) types.SideMsgServer {
	return &sideMsgServer{Keeper: keeper}
}

// SideTxHandler returns a side handler for milestone type messages.
func (srv *sideMsgServer) SideTxHandler(methodName string) hmModule.SideTxHandler {

	switch methodName {
	case milestoneMsgTypeURL:
		return srv.SideHandleMilestone
	default:
		return nil
	}
}

// PostTxHandler returns a side handler for milestone type messages.
func (srv *sideMsgServer) PostTxHandler(methodName string) hmModule.PostTxHandler {

	switch methodName {
	case milestoneMsgTypeURL:
		return srv.PostHandleMsgMilestone
	default:
		return nil
	}
}

// SideHandleMilestone handles the side msg for milestones
func (srv *sideMsgServer) SideHandleMilestone(ctx sdk.Context, msgI sdk.Msg) (result hmModule.Vote) {
	logger := srv.Logger(ctx)

	msg, ok := msgI.(*types.MsgMilestone)
	if !ok {
		logger.Error("type mismatch for MsgMilestone")
		return hmModule.Vote_VOTE_NO
	}

	params, err := srv.GetParams(ctx)
	if err != nil {
		logger.Error("error in getting params", "error", err)
		return hmModule.Vote_VOTE_NO
	}

	minMilestoneLength := params.MinMilestoneLength
	borChainMilestoneTxConfirmations := params.MilestoneTxConfirmations

	contractCaller := srv.IContractCaller

	// Get the milestone count
	count, err := srv.GetMilestoneCount(ctx)
	if err != nil {
		logger.Error("error in fetching milestone count", "error", err)
		return hmModule.Vote_VOTE_NO
	}

	lastMilestone, err := srv.GetLastMilestone(ctx)

	if count != uint64(0) && err != nil {
		logger.Error("error while receiving the last milestone in the side handler", "err", err)
		return hmModule.Vote_VOTE_NO
	}

	if count != uint64(0) && lastMilestone != nil && msg.StartBlock != lastMilestone.EndBlock+1 {
		logger.Error("milestone is not in continuity to last stored milestone",
			"startBlock", msg.StartBlock,
			"endBlock", msg.EndBlock,
			"hash", msg.Hash,
			"milestoneId", msg.MilestoneID,
			"error", err,
		)

		return hmModule.Vote_VOTE_NO
	}

	validMilestone, err := ValidateMilestone(msg.StartBlock, msg.EndBlock, msg.Hash, msg.MilestoneID, contractCaller, minMilestoneLength, borChainMilestoneTxConfirmations)
	if err != nil {
		logger.Error("error validating milestone",
			"startBlock", msg.StartBlock,
			"endBlock", msg.EndBlock,
			"hash", msg.Hash,
			"milestoneId", msg.MilestoneID,
			"error", err,
		)
	} else if validMilestone {
		return hmModule.Vote_VOTE_YES
	}

	logger.Error(
		"milestone is not valid",
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"hash", msg.Hash,
		"milestoneId", msg.MilestoneID,
		"err", err,
	)

	return hmModule.Vote_VOTE_NO
}

// PostHandleMsgMilestone handles the post side tx for a milestone
func (srv *sideMsgServer) PostHandleMsgMilestone(ctx sdk.Context, msgI sdk.Msg, sideTxResult hmModule.Vote) {
	logger := srv.Logger(ctx)

	msg, ok := msgI.(*types.MsgMilestone)
	if !ok {
		logger.Error("type mismatch for MsgMilestone")
		return
	}

	if sideTxResult != hmModule.Vote_VOTE_YES {
		srv.SetNoAckMilestone(ctx, msg.MilestoneID)
		logger.Debug("skipping new validator-join since side-tx didn't get yes votes")
		return
	}

	timeStamp := uint64(ctx.BlockTime().Unix())

	if lastMilestone, err := srv.GetLastMilestone(ctx); err == nil {
		// make sure new milestone is after tip
		if lastMilestone.EndBlock > msg.StartBlock {
			logger.Error("milestone already exists",
				"currentTip", lastMilestone.EndBlock,
				"startBlock", msg.StartBlock,
			)

			srv.SetNoAckMilestone(ctx, msg.MilestoneID)

			return
		}

		// check if new milestone's start block start from current tip
		if lastMilestone.EndBlock+1 != msg.StartBlock {
			logger.Error("milestone not in continuity",
				"currentTip", lastMilestone.EndBlock,
				"startBlock", msg.StartBlock)

			srv.SetNoAckMilestone(ctx, msg.MilestoneID)

			return
		}
	} else if msg.StartBlock != uint64(0) {
		logger.Error("first milestone to start from", "block", 0, "Error", err)

		srv.SetNoAckMilestone(ctx, msg.MilestoneID)

		return
	}

	// Add the milestone to the store
	if err := srv.AddMilestone(ctx, types.Milestone{
		StartBlock:  msg.StartBlock,
		EndBlock:    msg.EndBlock,
		Hash:        msg.Hash,
		Proposer:    msg.Proposer,
		BorChainID:  msg.BorChainID,
		MilestoneID: msg.MilestoneID,
		TimeStamp:   timeStamp,
	}); err != nil {
		srv.SetNoAckMilestone(ctx, msg.MilestoneID)
		logger.Error("failed to set milestone ", "Error", err)
	}

	// TX bytes
	txBytes := ctx.TxBytes()
	hash := hmTypes.TxHash{Hash: txBytes}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeMilestone,
			sdk.NewAttribute(sdk.AttributeKeyAction, msg.Type()),
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, hash.String()),
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),
			sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyHash, msg.Hash.String()),
			sdk.NewAttribute(types.AttributeKeyMilestoneID, msg.MilestoneID),
		),
	})
}
