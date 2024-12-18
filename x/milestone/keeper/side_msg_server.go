package keeper

import (
	"errors"
	"strconv"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

type sideMsgServer struct {
	*Keeper
}

var milestoneMsgTypeURL = sdk.MsgTypeURL(&types.MsgMilestone{})

// NewSideMsgServerImpl returns an implementation of the milestone MsgServer interface
// for the provided Keeper.
func NewSideMsgServerImpl(keeper *Keeper) sidetxs.SideMsgServer {
	return &sideMsgServer{Keeper: keeper}
}

// SideTxHandler returns a side handler for milestone type messages.
func (srv *sideMsgServer) SideTxHandler(methodName string) sidetxs.SideTxHandler {
	switch methodName {
	case milestoneMsgTypeURL:
		return srv.SideHandleMsgMilestone
	default:
		return nil
	}
}

// PostTxHandler returns a side handler for milestone type messages.
func (srv *sideMsgServer) PostTxHandler(methodName string) sidetxs.PostTxHandler {
	switch methodName {
	case milestoneMsgTypeURL:
		return srv.PostHandleMsgMilestone
	default:
		return nil
	}
}

// SideHandleMsgMilestone handles the side msg for milestones
func (srv *sideMsgServer) SideHandleMsgMilestone(ctx sdk.Context, msgI sdk.Msg) (result sidetxs.Vote) {
	logger := srv.Logger(ctx)

	msg, ok := msgI.(*types.MsgMilestone)
	if !ok {
		logger.Error("type mismatch for MsgMilestone")
		return sidetxs.Vote_VOTE_NO
	}

	params, err := srv.GetParams(ctx)
	if err != nil {
		logger.Error("error in getting params", "error", err)
		return sidetxs.Vote_VOTE_NO
	}

	minMilestoneLength := params.MinMilestoneLength
	borChainMilestoneTxConfirmations := params.MilestoneTxConfirmations

	contractCaller := srv.IContractCaller

	doExist, err := srv.HasMilestone(ctx)
	if err != nil {
		logger.Error("error in existence of last milestone", "error", err)
		return sidetxs.Vote_VOTE_NO
	}

	lastMilestone, err := srv.GetLastMilestone(ctx)
	if doExist && err != nil {
		logger.Error("error while receiving the last milestone in the side handler", "err", err)
		return sidetxs.Vote_VOTE_NO
	}

	if doExist && lastMilestone != nil && msg.StartBlock != lastMilestone.EndBlock+1 {
		logger.Error("milestone is not in continuity to last stored milestone",
			"startBlock", msg.StartBlock,
			"endBlock", msg.EndBlock,
			"hash", msg.Hash,
			"milestoneId", msg.MilestoneId,
			"error", err,
		)

		return sidetxs.Vote_VOTE_NO
	}

	if !doExist && msg.StartBlock != types.StartBlock {
		logger.Error("milestone's start block is not correct",
			"msg start block", msg.StartBlock,
			"expected start block", types.StartBlock,
		)

		return sidetxs.Vote_VOTE_NO

	}

	isValid, err := ValidateMilestone(msg.StartBlock, msg.EndBlock, msg.Hash, msg.MilestoneId, contractCaller, minMilestoneLength, borChainMilestoneTxConfirmations)
	if err != nil || !isValid {
		logger.Error("error validating milestone",
			"startBlock", msg.StartBlock,
			"endBlock", msg.EndBlock,
			"hash", msg.Hash,
			"milestoneId", msg.MilestoneId,
			"error", err,
		)
		return sidetxs.Vote_VOTE_NO
	}

	return sidetxs.Vote_VOTE_YES
}

// PostHandleMsgMilestone handles the post side tx for a milestone
func (srv *sideMsgServer) PostHandleMsgMilestone(ctx sdk.Context, msgI sdk.Msg, sideTxResult sidetxs.Vote) error {
	logger := srv.Logger(ctx)

	msg, ok := msgI.(*types.MsgMilestone)
	if !ok {
		err := errors.New("type mismatch for MsgMilestone")
		logger.Error(err.Error())
		return err
	}

	if sideTxResult != sidetxs.Vote_VOTE_YES {
		err := srv.SetNoAckMilestone(ctx, msg.MilestoneId)
		if err != nil {
			logger.Error("error while setting no-ack", "err", err)
			return err
		}
		logger.Debug("skipping milestone handler since side-tx didn't get yes votes")
		return errors.New("side-tx didn't get yes votes")
	}

	timeStamp := uint64(ctx.BlockTime().Unix())

	doExist, err := srv.HasMilestone(ctx)
	if err != nil {
		logger.Error("error while checking for the last milestone", "err", err)
		return err
	}

	lastMilestone, err := srv.GetLastMilestone(ctx)
	if doExist && err != nil {
		logger.Error("error while fetching  the last milestone", "err", err)
		return err
	}

	if doExist && lastMilestone == nil {
		logger.Error("last milestone shouldn't be nil")
		return errors.New("last milestone shouldn't be nil")
	}

	if doExist && (lastMilestone.EndBlock+1) != msg.StartBlock {
		logger.Error("milestone is not in continuity",
			"currentTip", lastMilestone.EndBlock,
			"startBlock", msg.StartBlock,
		)

		err = srv.SetNoAckMilestone(ctx, msg.MilestoneId)
		if err != nil {
			logger.Error("error while setting no-ack", "err", err)
			return err
		}

		return errors.New("milestone is not in continuity")
	}

	if !doExist && msg.StartBlock != types.StartBlock {
		logger.Error("first milestone to start from", "block", types.StartBlock, "Error", err)

		err = srv.SetNoAckMilestone(ctx, msg.MilestoneId)
		if err != nil {
			logger.Error("error while setting no-ack", "err", err)
			return err
		}

		return errors.New("first milestone to start from")
	}

	// Add the milestone to the store
	err = srv.AddMilestone(ctx, types.Milestone{
		StartBlock:  msg.StartBlock,
		EndBlock:    msg.EndBlock,
		Hash:        msg.Hash,
		Proposer:    msg.Proposer,
		BorChainId:  msg.BorChainId,
		MilestoneId: msg.MilestoneId,
		Timestamp:   timeStamp,
	})
	if err != nil {
		err = srv.SetNoAckMilestone(ctx, msg.MilestoneId)
		if err != nil {
			logger.Error("error while setting no-ack", "err", err)
		}
		logger.Error("failed to set milestone ", "Error", err)
	}

	// TX bytes
	txBytes := ctx.TxBytes()
	hash := txBytes

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeMilestone,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, common.Bytes2Hex(hash)),
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),
			sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyHash, common.Bytes2Hex(msg.Hash)),
			sdk.NewAttribute(types.AttributeKeyMilestoneID, msg.MilestoneId),
		),
	})

	return nil
}
