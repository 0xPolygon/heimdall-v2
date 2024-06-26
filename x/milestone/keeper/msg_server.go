package keeper

import (
	"context"
	"math"
	"strconv"
	"time"

	errorsmod "cosmossdk.io/errors"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the milestone MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// Milestone handles milestone transactions
func (m msgServer) Milestone(ctx context.Context, msg *types.MsgMilestone) (*types.MsgMilestoneResponse, error) {
	logger := m.Logger(ctx)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	params, err := m.GetParams(ctx)
	if err != nil {
		logger.Error("error in fetching milestone parameter")
		return nil, errorsmod.Wrap(types.ErrMilestoneParams, "error in fetching milestone parameter")
	}

	milestoneLength := params.MinMilestoneLength

	// Get the milestone proposer
	validatorSet := m.sk.GetMilestoneValidatorSet(ctx)
	if validatorSet.Proposer == nil {
		logger.Error("no proposer in validator set", "msgProposer", msg.Proposer)
		return nil, errorsmod.Wrap(types.ErrProposerNotFound, "milestone proposer not found ")
	}

	// Validate proposer

	//check for the milestone proposer
	if msg.Proposer != validatorSet.Proposer.Signer {
		logger.Error(
			"invalid proposer in msg",
			"proposer", validatorSet.Proposer.Signer,
			"msgProposer", msg.Proposer,
		)

		return nil, errorsmod.Wrap(types.ErrProposerMismatch, "msg and expected milestone proposer mismatch")
	}

	if sdkCtx.BlockHeight()-m.GetMilestoneBlockNumber(ctx) < 2 {
		logger.Error(
			"previous milestone still in voting phase",
			"previousMilestoneBlock", m.GetMilestoneBlockNumber(ctx),
			"currentMilestoneBlock", sdkCtx.BlockHeight(),
		)

		return nil, errorsmod.Wrap(types.ErrPrevMilestoneInVoting, "previous milestone still in voting phase")
	}

	//Increment the priority in the milestone validator set
	m.sk.MilestoneIncrementAccum(ctx, 1)

	// Calculate the milestone length
	msgMilestoneLength := int64(msg.EndBlock) - int64(msg.StartBlock) + 1

	//check for the minimum length of milestone
	if msgMilestoneLength < int64(milestoneLength) {
		logger.Error("length of the milestone should be greater than configured minimum milestone length",
			"StartBlock", msg.StartBlock,
			"EndBlock", msg.EndBlock,
			"Minimum Milestone Length", milestoneLength,
		)

		return nil, errorsmod.Wrap(types.ErrMilestoneInvalid, "milestone's length is less than permitted minimum milestone length")
	}

	// fetch last stored milestone from store
	if lastMilestone, err := m.GetLastMilestone(ctx); err == nil {
		// make sure new milestone is in continuity
		if lastMilestone.EndBlock+1 != msg.StartBlock {
			logger.Error("milestone not in continuity ",
				"lastMilestoneEndBlock", lastMilestone.EndBlock,
				"receivedMsgStartBlock", msg.StartBlock,
			)

			return nil, errorsmod.Wrap(types.ErrMilestoneNotInContinuity, "milestone not in continuity")
		}
	} else if msg.StartBlock != uint64(0) {
		logger.Error("first milestone to start from", "block", 0, "milestone start block", msg.StartBlock, "error", err)
		return nil, errorsmod.Wrap(types.ErrMilestoneInvalid, "start block doesn't match with expected one")
	}

	if err = m.SetMilestoneBlockNumber(ctx, sdkCtx.BlockHeight()); err != nil {
		logger.Error("error in setting milestone block number", "error", err)
		return nil, errorsmod.Wrapf(err, "error in setting milestone block number")

	}

	// Emit event for milestone
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeMilestone,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyProposer, msg.Proposer),
			sdk.NewAttribute(types.AttributeKeyStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
			sdk.NewAttribute(types.AttributeKeyHash, msg.Hash.String()),
		),
	})

	return &types.MsgMilestoneResponse{}, nil

}

// MilestoneTimeout handles milestone timeout transaction
func (m msgServer) MilestoneTimeout(ctx context.Context, msg *types.MsgMilestoneTimeout) (*types.MsgMilestoneTimeoutResponse, error) {
	logger := m.Logger(ctx)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get current block time
	currentTime := sdkCtx.BlockTime()

	// Get buffer time from params
	params, err := m.GetParams(ctx)
	if err != nil {
		logger.Error("error in fetching milestone parameter")
		return nil, errorsmod.Wrap(types.ErrMilestoneParams, "error in fetching milestone parameter")
	}

	bufferTime := params.MilestoneBufferTime

	// Fetch last milestone from the store
	// TODO HV2 figure out how to handle this error
	lastMilestone, err := m.GetLastMilestone(ctx)
	if err != nil {
		logger.Error("didn't find the last milestone", "err", err)
		return nil, errorsmod.Wrap(types.ErrNoMilestoneFound, "could fetch last milestone")
	}

	lastMilestoneTime := time.Unix(int64(lastMilestone.TimeStamp), 0)

	// If last milestone happens before milestone buffer time, then throw an error
	if lastMilestoneTime.After(currentTime) || (currentTime.Sub(lastMilestoneTime) < bufferTime) {
		logger.Error("invalid milestone timeout msg", "lastMilestoneTime", lastMilestoneTime, "current time", currentTime,
			"buffer Time", bufferTime.String(),
		)

		return nil, errorsmod.Wrap(types.ErrInvalidMilestoneTimeout, "msg is invalid as it came before the buffer time")
	}

	// Check last no ack - prevents repetitive no-ack
	lastMilestoneTimeout := m.GetLastMilestoneTimeout(ctx)
	if lastMilestoneTimeout > uint64(math.MaxInt64) {
		// handle the error appropriately, for example by returning an error
		return nil, errorsmod.Wrap(types.ErrInvalidMilestoneTimeout, "lastMilestoneTimeout is too large")
	}
	lastMilestoneTimeoutTime := time.Unix(int64(lastMilestoneTimeout), 0)

	if lastMilestoneTimeoutTime.After(currentTime) || (currentTime.Sub(lastMilestoneTimeoutTime) < bufferTime) {
		logger.Debug("too many milestone timeout messages", "lastMilestoneTimeoutTime", lastMilestoneTimeoutTime, "current time", currentTime,
			"buffer Time", bufferTime.String())

		return nil, errorsmod.Wrap(types.ErrTooManyMilestoneTimeout, "too many milestone timeout messages")
	}

	// Set new last milestone-timeout
	newLastMilestoneTimeout := uint64(currentTime.Unix())
	if err = m.SetLastMilestoneTimeout(ctx, newLastMilestoneTimeout); err != nil {
		logger.Error("error in setting last milestone timeout", "error", err)
		return nil, errorsmod.Wrapf(err, "error in setting last milestone timeout")
	}
	logger.Debug("last milestone-timeout set", "lastMilestoneTimeout", newLastMilestoneTimeout)

	// Increment accum (selects new proposer)
	m.sk.MilestoneIncrementAccum(ctx, 1)

	// Get new proposer
	vs := m.sk.GetMilestoneValidatorSet(ctx)

	newProposer := vs.GetProposer()
	logger.Debug(
		"new milestone proposer selected",
		"signer", newProposer.Signer,
		"power", newProposer.VotingPower,
	)

	// add events
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeMilestone,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeyNewProposer, newProposer.Signer),
		),
	})

	return &types.MsgMilestoneTimeoutResponse{}, nil
}

// UpdateParams defines a method to update the params in x/milestone module.
func (m msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if m.authority != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", m.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	if err := m.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
