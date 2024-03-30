package keeper

import (
	"strconv"

	hmModule "github.com/0xPolygon/heimdall-v2/module"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type sideMsgServer struct {
	*Keeper
}

var (
	milestone = sdk.MsgTypeURL(&types.MsgMilestone{})
)

// NewMsgServerImpl returns an implementation of the staking MsgServer interface
// for the provided Keeper.
func NewSideMsgServerImpl(keeper *Keeper) types.SideMsgServer {
	return &sideMsgServer{Keeper: keeper}
}

// SideTxHandler returns a side handler for "milestone" type messages.
func (srv *sideMsgServer) SideTxHandler(methodName string) hmModule.SideTxHandler {

	switch methodName {
	case milestone:
		return srv.SideHandleMilestone
	default:
		return nil
	}
}

// PostTxHandler returns a side handler for "milestone" type messages.
func (srv *sideMsgServer) PostTxHandler(methodName string) hmModule.PostTxHandler {

	switch methodName {
	case milestone:
		return srv.PostHandleMsgMilestone
	default:
		return nil
	}
}

// SideHandleMsgValidatorJoin side msg validator join
func (k *sideMsgServer) SideHandleMilestone(ctx sdk.Context, _msg sdk.Msg) (result hmModule.Vote) {
	// logger
	logger := k.Logger(ctx)

	msg, ok := _msg.(*types.MsgMilestone)
	if !ok {
		logger.Error("msg type mismatched")
		return hmModule.Vote_VOTE_NO
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		logger.Error("Error in getting params", "error", err)
		return hmModule.Vote_VOTE_NO
	}

	milestoneLength := params.MinMilestoneLength
	maticchainMilestoneTxConfirmations := params.MilestoneTxConfirmations

	contractCaller := k.IContractCaller

	//Get the milestone count
	count := k.GetMilestoneCount(ctx)
	lastMilestone, err := k.GetLastMilestone(ctx)

	if count != uint64(0) && err != nil {
		logger.Error("Error while receiving the last milestone in the side handler")
		return hmModule.Vote_VOTE_NO
	}

	if count != uint64(0) && msg.StartBlock != lastMilestone.EndBlock+1 {
		logger.Error("Milestone is not in continuity to last stored milestone",
			"startBlock", msg.StartBlock,
			"endBlock", msg.EndBlock,
			"hash", msg.Hash,
			"milestoneId", msg.MilestoneID,
			"error", err,
		)

		return hmModule.Vote_VOTE_NO
	}

	//Validating the milestone
	validMilestone, err := ValidateMilestone(msg.StartBlock, msg.EndBlock, msg.Hash, msg.MilestoneID, contractCaller, milestoneLength, maticchainMilestoneTxConfirmations)
	if err != nil {
		logger.Error("Error validating milestone",
			"startBlock", msg.StartBlock,
			"endBlock", msg.EndBlock,
			"hash", msg.Hash,
			"milestoneId", msg.MilestoneID,
			"error", err,
		)
	} else if validMilestone {
		// vote `yes` if milestone is valid
		return hmModule.Vote_VOTE_YES
	}

	logger.Error(
		"Hash is not valid",
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"hash", msg.Hash,
		"milestoneId", msg.MilestoneID,
	)

	return hmModule.Vote_VOTE_NO
}

// PostHandleMsgValidatorJoin msg validator join
func (k *sideMsgServer) PostHandleMsgMilestone(ctx sdk.Context, _msg sdk.Msg, sideTxResult hmModule.Vote) {
	logger := k.Logger(ctx)

	msg, ok := _msg.(*types.MsgMilestone)
	if !ok {
		logger.Error("msg type mismatched")
		return
	}

	// Skip handler if validator join is not approved
	if sideTxResult != hmModule.Vote_VOTE_YES {
		k.SetNoAckMilestone(ctx, msg.MilestoneID)
		logger.Debug("Skipping new validator-join since side-tx didn't get yes votes")
		return
	}

	timeStamp := uint64(ctx.BlockTime().Unix())

	//Get the latest stored milestone from store
	if lastMilestone, err := k.GetLastMilestone(ctx); err == nil { // fetch last milestone from store
		// make sure new milestoen is after tip
		if lastMilestone.EndBlock > msg.StartBlock {
			logger.Error(" Milestone already exists",
				"currentTip", lastMilestone.EndBlock,
				"startBlock", msg.StartBlock,
			)

			k.SetNoAckMilestone(ctx, msg.MilestoneID)

			return
		}

		// check if new milestone's start block start from current tip
		if lastMilestone.EndBlock+1 != msg.StartBlock {
			logger.Error("milestone not in countinuity",
				"currentTip", lastMilestone.EndBlock,
				"startBlock", msg.StartBlock)

			k.SetNoAckMilestone(ctx, msg.MilestoneID)

			return
		}
	} else if msg.StartBlock != uint64(0) {
		logger.Error("First milestone to start from", "block", 0, "Error", err)

		k.SetNoAckMilestone(ctx, msg.MilestoneID)

		return
	}

	//Add the milestone to the store
	if err := k.AddMilestone(ctx, types.Milestone{ // Save milestone to db
		StartBlock:  msg.StartBlock, //Add milestone to store with root hash
		EndBlock:    msg.EndBlock,
		Hash:        msg.Hash,
		Proposer:    msg.Proposer,
		BorChainID:  msg.BorChainID,
		MilestoneID: msg.MilestoneID,
		TimeStamp:   timeStamp,
	}); err != nil {
		k.SetNoAckMilestone(ctx, msg.MilestoneID)
		logger.Error("Failed to set milestone ", "Error", err)
	}

	// TX bytes
	txBytes := ctx.TxBytes()
	hash := hmTypes.TxHash{txBytes}.Bytes()

	// Emit event for milestone
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeMilestone,
			sdk.NewAttribute(sdk.AttributeKeyAction, msg.Type()),                                  // action
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),                // module name
			sdk.NewAttribute(hmTypes.AttributeKeyTxHash, hmTypes.BytesToHeimdallHash(hash).Hex()), // tx hash
			sdk.NewAttribute(hmTypes.AttributeKeySideTxResult, sideTxResult.String()),             // result
			sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyHash, msg.Hash.String()),
			sdk.NewAttribute(types.AttributeKeyMilestoneID, msg.MilestoneID),
		),
	})

	return
}
