package keeper

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
	util "github.com/0xPolygon/heimdall-v2/common/address"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
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

	minMilestoneLength := params.MinMilestoneLength

	// Get the milestone proposer
	validatorSet, err := m.stakeKeeper.GetMilestoneValidatorSet(ctx)
	if err != nil {
		logger.Error("error in fetching milestone validator set", "error", err)
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, "error in fetching milestone validator set")
	}
	if validatorSet.Proposer == nil {
		logger.Error("no proposer in validator set", "msgProposer", msg.Proposer)
		return nil, errorsmod.Wrap(types.ErrProposerNotFound, "")
	}

	msgProposer := util.FormatAddress(msg.Proposer)
	valProposer := util.FormatAddress(validatorSet.Proposer.Signer)

	// check for the milestone proposer
	if strings.Compare(msgProposer, valProposer) != 0 {
		logger.Error(
			"invalid proposer in msg",
			"proposer", validatorSet.Proposer.Signer,
			"msgProposer", msg.Proposer,
		)

		return nil, errorsmod.Wrap(types.ErrProposerMismatch, "msg and expected milestone proposer mismatch")
	}

	mBlockNumber, err := m.GetMilestoneBlockNumber(ctx)
	if err != nil {
		logger.Error("error in fetching milestone block number", "error", err)
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, "error in fetching milestone block number")
	}

	// To prevent the new milestone to go into voting if the previous milestone
	// is still in process stage
	if sdkCtx.BlockHeight()-mBlockNumber < 2 {
		logger.Error(
			"previous milestone still in voting phase",
			"previousMilestoneBlock", mBlockNumber,
			"currentMilestoneBlock", sdkCtx.BlockHeight(),
		)

		return nil, errorsmod.Wrap(types.ErrPrevMilestoneInVoting, "")
	}

	// increment the priority in the milestone validator set
	m.stakeKeeper.MilestoneIncrementAccum(ctx, 1)

	// Calculate the milestone length
	msgMilestoneLength := int64(msg.EndBlock) - int64(msg.StartBlock) + 1

	// check for the minimum length of milestone
	if msgMilestoneLength < int64(minMilestoneLength) {
		logger.Error("length of the milestone should be greater than configured minimum milestone length",
			"StartBlock", msg.StartBlock,
			"EndBlock", msg.EndBlock,
			"Minimum Milestone Length", minMilestoneLength,
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
	} else if msg.StartBlock != types.StartBlock {
		logger.Error("first milestone to start from", "block", types.StartBlock, "milestone start block", msg.StartBlock, "error", err)
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
			sdk.NewAttribute(types.AttributeKeyHash, common.Bytes2Hex(msg.Hash)),
		),
	})

	return &types.MsgMilestoneResponse{}, nil
}

// MilestoneTimeout handles milestone timeout transaction
func (m msgServer) MilestoneTimeout(ctx context.Context, _ *types.MsgMilestoneTimeout) (*types.MsgMilestoneTimeoutResponse, error) {
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

	lastMilestone, err := m.GetLastMilestone(ctx)
	if err != nil {
		logger.Error("didn't find the last milestone", "err", err)
		return nil, errorsmod.Wrap(types.ErrNoMilestoneFound, "")
	}

	lastMilestoneTime := time.Unix(int64(lastMilestone.Timestamp), 0)

	// If last milestone happens before milestone buffer time, then throw an error
	if lastMilestoneTime.After(currentTime) || (currentTime.Sub(lastMilestoneTime) < bufferTime) {
		logger.Error("invalid milestone timeout msg", "lastMilestoneTime", lastMilestoneTime, "current time", currentTime,
			"buffer Time", bufferTime.String(),
		)

		return nil, errorsmod.Wrap(types.ErrInvalidMilestoneTimeout, "msg is invalid as it came before the buffer time")
	}

	lastMilestoneTimeout, err := m.GetLastMilestoneTimeout(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "error while fetching last milestone timeout")
	}

	if lastMilestoneTimeout > uint64(math.MaxInt64) {
		return nil, errorsmod.Wrap(types.ErrInvalidMilestoneTimeout, "lastMilestoneTimeout is too large")
	}

	lastMilestoneTimeoutTime := time.Unix(int64(lastMilestoneTimeout), 0)

	if lastMilestoneTimeoutTime.After(currentTime) || (currentTime.Sub(lastMilestoneTimeoutTime) < bufferTime) {
		logger.Debug("too many milestone timeout messages", "lastMilestoneTimeoutTime", lastMilestoneTimeoutTime, "current time", currentTime,
			"buffer Time", bufferTime.String())

		return nil, errorsmod.Wrap(types.ErrTooManyMilestoneTimeout, "too many milestone timeout messages")
	}

	// set new last milestone-timeout
	newLastMilestoneTimeout := uint64(currentTime.Unix())
	if err = m.SetLastMilestoneTimeout(ctx, newLastMilestoneTimeout); err != nil {
		logger.Error("error in setting last milestone timeout", "error", err)
		return nil, errorsmod.Wrapf(err, "error in setting last milestone timeout")
	}
	logger.Debug("last milestone-timeout set", "lastMilestoneTimeout", newLastMilestoneTimeout)

	// Increment accum (selects new proposer)
	m.stakeKeeper.MilestoneIncrementAccum(ctx, 1)

	// Get new proposer
	vs, err := m.stakeKeeper.GetMilestoneValidatorSet(ctx)
	if err != nil {
		logger.Error("error in fetching milestone validator set", "error", err)
		return nil, errorsmod.Wrap(types.ErrInvalidMsg, "error in fetching milestone validator set")
	}

	newProposer := vs.GetProposer()
	logger.Debug(
		"new milestone proposer selected",
		"signer", newProposer.Signer,
		"power", newProposer.VotingPower,
	)

	// add events
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeMilestoneTimeout,
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
