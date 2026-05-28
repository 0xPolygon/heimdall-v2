package keeper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/errors"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"

	util "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics/api"
	heimdallTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the bor MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (srv msgServer) ProposeSpan(ctx context.Context, msg *types.MsgProposeSpan) (*types.MsgProposeSpanResponse, error) {
	var err error
	start := time.Now()
	defer recordBorTransactionMetric(api.ProposeSpanMethod, start, &err)

	logger := srv.Logger(ctx)

	logger.Debug(helper.LogValidatingExternalCall("MsgProposeSpan"),
		"proposer", msg.Proposer,
		"spanId", msg.SpanId,
		"startBlock", msg.StartBlock,
		"endBlock", msg.EndBlock,
		"seed", msg.Seed,
	)

	_, err = sdk.ValAddressFromHex(msg.Proposer)
	if err != nil {
		logger.Error(heimdallTypes.ErrMsgInvalidProposerAddress, "error", err)
		return nil, errors.Wrapf(err, heimdallTypes.ErrMsgInvalidProposerAddress)
	}

	// verify chain id
	chainParams, err := srv.ck.GetParams(ctx)
	if err != nil {
		logger.Error(heimdallTypes.ErrMsgFailedToGetChainParams, "error", err)
		return nil, errors.Wrapf(err, heimdallTypes.ErrMsgFailedToGetChainParams)
	}

	if !helper.ValidateChainID(msg.ChainId, chainParams.ChainParams.BorChainId, logger, "bor") {
		return nil, types.ErrInvalidChainID
	}

	// verify seed length
	if len(msg.Seed) != common.HashLength {
		logger.Error("Invalid seed length", "expected", common.HashLength, "got", len(msg.Seed))
		return nil, types.ErrInvalidSeedLength
	}

	lastSpan, err := srv.GetLastSpan(ctx)
	if err != nil {
		logger.Error("Unable to fetch last span", "Error", err)
		return nil, errors.Wrapf(err, "unable to fetch last span")
	}

	// Validate span continuity
	if lastSpan.Id+1 != msg.SpanId || msg.StartBlock != lastSpan.EndBlock+1 || msg.EndBlock <= msg.StartBlock {
		logger.Error(heimdallTypes.ErrMsgBlocksNotInContinuity,
			"lastSpanId", lastSpan.Id,
			"spanId", msg.SpanId,
			"lastSpanStartBlock", lastSpan.StartBlock,
			"lastSpanEndBlock", lastSpan.EndBlock,
			"spanStartBlock", msg.StartBlock,
			"spanEndBlock", msg.EndBlock,
		)

		return nil, types.ErrInvalidSpan
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// add events
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeProposeSpan,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(types.AttributeKeySpanID, strconv.FormatUint(msg.SpanId, 10)),
			sdk.NewAttribute(types.AttributeKeySpanStartBlock, strconv.FormatUint(msg.StartBlock, 10)),
			sdk.NewAttribute(types.AttributeKeySpanEndBlock, strconv.FormatUint(msg.EndBlock, 10)),
		),
	})

	logger.Debug("Emitted ProposeSpan event")
	return &types.MsgProposeSpanResponse{}, nil
}

// UpdateParams defines a method to update the params in x/bor module.
func (srv msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	var err error
	start := time.Now()
	defer recordBorTransactionMetric(api.BorUpdateParamsMethod, start, &err)

	if srv.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", srv.authority, msg.Authority)
	}

	if err := msg.Params.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := srv.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (srv msgServer) VoteProducers(ctx context.Context, msg *types.MsgVoteProducers) (*types.MsgVoteProducersResponse, error) {
	var err error
	start := time.Now()
	defer recordBorTransactionMetric(api.VoteProducersMethod, start, &err)

	// Validate veBlop phase
	if err := srv.CanVoteProducers(ctx); err != nil {
		return nil, err
	}

	voter, err := sdk.AccAddressFromHex(msg.Voter)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid voter address")
	}

	validator, err := srv.sk.GetValidatorFromValID(ctx, msg.VoterId)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid voter id")
	}

	pk := secp256k1.PubKey(validator.PubKey)

	if util.FormatAddress(voter.String()) != util.FormatAddress(pk.Address().String()) {
		return nil, fmt.Errorf("voter address %s does not match validator address %s under validator id %d", voter.String(), pk.Address().String(), msg.VoterId)
	}

	// Check if there are any duplicate votes in the msg.Votes
	seen := make(map[uint64]bool)
	for _, vote := range msg.Votes.Votes {
		if seen[vote] {
			return nil, fmt.Errorf("duplicate vote for validator id %d", vote)
		}
		seen[vote] = true
	}

	err = srv.SetProducerVotes(ctx, msg.VoterId, msg.Votes)
	if err != nil {
		return nil, err
	}

	return &types.MsgVoteProducersResponse{}, nil
}

func (srv msgServer) SetProducerDowntime(ctx context.Context, msg *types.MsgSetProducerDowntime) (*types.MsgSetProducerDowntimeResponse, error) {
	var err error
	start := time.Now()
	defer recordBorTransactionMetric(api.ProducerDowntimeMethod, start, &err)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if err := srv.CanSetProducerDowntime(sdkCtx); err != nil {
		return nil, err
	}

	if msg.DowntimeRange.StartBlock >= msg.DowntimeRange.EndBlock {
		return nil, fmt.Errorf("start block must be less than end block")
	}

	if msg.DowntimeRange.EndBlock-msg.DowntimeRange.StartBlock < types.PlannedDowntimeMinRange {
		return nil, fmt.Errorf("time range must be at least %d blocks. start block %d, end block %d",
			types.PlannedDowntimeMinRange,
			msg.DowntimeRange.StartBlock,
			msg.DowntimeRange.EndBlock,
		)
	}

	if msg.DowntimeRange.EndBlock-msg.DowntimeRange.StartBlock > types.PlannedDowntimeMaxRange {
		return nil, fmt.Errorf("time range must be at most %d blocks. start block %d, end block %d",
			types.PlannedDowntimeMaxRange,
			msg.DowntimeRange.StartBlock,
			msg.DowntimeRange.EndBlock,
		)
	}

	producerId := uint64(0)
	validators := srv.sk.GetSpanEligibleValidators(ctx)
	found := false
	for _, v := range validators {
		if util.FormatAddress(v.Signer) == util.FormatAddress(msg.Producer) {
			producerId = v.ValId
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("producer with address %s not found in the current validator set", msg.Producer)
	}

	candidates, err := srv.CalculateProducerSet(ctx, helper.GetProducerSetLimit(sdkCtx))
	if err != nil {
		return nil, fmt.Errorf("failed to calculate producer set: %w", err)
	}

	if len(candidates) == 0 {
		candidates = helper.GetFallbackProducerVotes()
	}

	isProducer := false
	for _, c := range candidates {
		if c == producerId {
			isProducer = true
			break
		}
	}

	if !isProducer {
		return nil, fmt.Errorf("producer with address %s and id %d is not in the current producer set", msg.Producer, producerId)
	}

	// Reject non-default target before the fork height.
	if msg.TargetProducerId != types.RoundRobinDefault && sdkCtx.BlockHeight() < helper.GetV080HardforkHeight() {
		return nil, fmt.Errorf("target producer override is not enabled until height %d", helper.GetV080HardforkHeight())
	}

	// Validate target producer if it is not the round-robin default and the block height is greater than the target producer override height.
	if msg.TargetProducerId != types.RoundRobinDefault && sdkCtx.BlockHeight() >= helper.GetV080HardforkHeight() {
		if msg.TargetProducerId == producerId {
			return nil, fmt.Errorf("target producer cannot be the same as the current producer")
		}
		targetInCandidates := false
		for _, c := range candidates {
			if c == msg.TargetProducerId {
				targetInCandidates = true
				break
			}
		}
		if !targetInCandidates {
			return nil, fmt.Errorf("target producer id %d is not in the current producer set", msg.TargetProducerId)
		}
	}

	return &types.MsgSetProducerDowntimeResponse{}, nil
}

func recordBorTransactionMetric(method string, start time.Time, err *error) {
	success := *err == nil
	api.RecordAPICallWithStart(api.BorSubsystem, method, api.TxType, success, start)
}
