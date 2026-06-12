package app

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	milestoneAbci "github.com/0xPolygon/heimdall-v2/x/milestone/abci"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

const (
	psSpanStart   = uint64(90)
	psSpanEnd     = uint64(190)
	psPendingHead = uint64(150)
)

// singleBlockPendingProp builds a one-block pending proposition whose head is block n.
func singleBlockPendingProp(n uint64, hashSeed byte) *milestoneTypes.MilestoneProposition {
	h := make([]byte, 32)
	for i := range h {
		h[i] = hashSeed
	}
	return &milestoneTypes.MilestoneProposition{
		StartBlockNumber: n,
		BlockHashes:      [][]byte{h},
		BlockTds:         []uint64{1000 + n},
	}
}

// seedSpan installs the committed span [psSpanStart, psSpanEnd] with producer validators[0]
// and returns the validators and the all-supporters set.
func seedSpan(t *testing.T, app *HeimdallApp, ctx sdk.Context) ([]*stakeTypes.Validator, map[uint64]struct{}) {
	t.Helper()
	validators := app.StakeKeeper.GetAllValidators(ctx)

	valSlice := make([]*stakeTypes.Validator, len(validators))
	selected := make([]stakeTypes.Validator, len(validators))
	supporters := make(map[uint64]struct{}, len(validators))
	for i, v := range validators {
		valSlice[i] = v
		selected[i] = *v
		supporters[v.ValId] = struct{}{}
	}

	span := borTypes.Span{
		Id:                1,
		StartBlock:        psSpanStart,
		EndBlock:          psSpanEnd,
		BorChainId:        "1",
		ValidatorSet:      stakeTypes.ValidatorSet{Validators: valSlice},
		SelectedProducers: selected,
	}
	require.NoError(t, app.BorKeeper.AddNewSpan(ctx, &span))
	return validators, supporters
}

// seedProducerSelection sets producer votes and params so a non-current producer is selectable.
func seedProducerSelection(t *testing.T, app *HeimdallApp, ctx sdk.Context, validators []*stakeTypes.Validator) {
	t.Helper()
	for _, val := range validators {
		var votes []uint64
		for _, other := range validators {
			if other.ValId != validators[0].ValId {
				votes = append(votes, other.ValId)
			}
		}
		votes = append(votes, validators[0].ValId)
		require.NoError(t, app.BorKeeper.SetProducerVotes(ctx, val.ValId, borTypes.ProducerVotes{Votes: votes}))
	}

	params, err := app.BorKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.ProducerCount = 3
	params.SpanDuration = 100
	require.NoError(t, app.BorKeeper.SetParams(ctx, params))
}

func TestCheckAndRotateOnPendingStall(t *testing.T) {
	t.Run("first observation sets tracking and does not rotate", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
		_, supporters := seedSpan(t, app, ctx)
		ctx = ctx.WithBlockHeight(1000)
		prop := singleBlockPendingProp(psPendingHead, 0xAA)

		require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop, supporters))

		last, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(1), last.Id, "no rotation on first observation")

		block, id, height, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
		require.NoError(t, err)
		require.Equal(t, psPendingHead, block)
		require.Equal(t, milestoneAbci.MilestonePropositionHeadID(prop), id)
		require.Equal(t, uint64(1000), height)
	})

	t.Run("identity flap at same head resets clock, no rotation", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
		_, supporters := seedSpan(t, app, ctx)
		require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, []byte("stale-identity"), 1))
		ctx = ctx.WithBlockHeight(1000) // well past any threshold
		prop := singleBlockPendingProp(psPendingHead, 0xBB)

		require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop, supporters))

		last, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(1), last.Id, "a flapping tip must not rotate")

		_, id, height, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
		require.NoError(t, err)
		require.Equal(t, milestoneAbci.MilestonePropositionHeadID(prop), id, "clock re-baselined to the new identity")
		require.Equal(t, uint64(1000), height)
	})

	t.Run("head advance resets clock, no rotation", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
		_, supporters := seedSpan(t, app, ctx)
		require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead-5, []byte("old"), 1))
		ctx = ctx.WithBlockHeight(1000)
		prop := singleBlockPendingProp(psPendingHead, 0xCC)

		require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop, supporters))

		last, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(1), last.Id, "advancing head must not rotate")

		block, _, height, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
		require.NoError(t, err)
		require.Equal(t, psPendingHead, block)
		require.Equal(t, uint64(1000), height)
	})

	t.Run("stall exactly at threshold does not rotate", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
		_, supporters := seedSpan(t, app, ctx)
		prop := singleBlockPendingProp(psPendingHead, 0xDD)
		trackedHeight := uint64(1000)
		require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, milestoneAbci.MilestonePropositionHeadID(prop), trackedHeight))
		threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
		ctx = ctx.WithBlockHeight(int64(trackedHeight) + threshold) // borStallDiff == threshold

		require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop, supporters))

		last, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(1), last.Id, "diff == threshold must not rotate (strict >)")
	})

	t.Run("stall beyond threshold rotates from N+1 and debounces", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
		validators, supporters := seedSpan(t, app, ctx)
		seedProducerSelection(t, app, ctx, validators)

		prop := singleBlockPendingProp(psPendingHead, 0xEE)
		trackedHeight := uint64(1000)
		require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, milestoneAbci.MilestonePropositionHeadID(prop), trackedHeight))
		threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
		blockHeight := int64(trackedHeight) + threshold + 1 // borStallDiff == threshold+1
		ctx = ctx.WithBlockHeight(blockHeight)

		currentProducer := validators[0].ValId
		require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop, supporters))

		last, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(2), last.Id, "a new span must be minted")
		require.Equal(t, psPendingHead+1, last.StartBlock, "new span starts at N+1 (no reorg)")
		require.GreaterOrEqual(t, last.EndBlock, psSpanEnd, "new span must cover the old runway")
		require.NotEqual(t, currentProducer, last.SelectedProducers[0].ValId, "stalled producer must be excluded")

		failed, err := app.BorKeeper.GetLatestFailedProducer(ctx)
		require.NoError(t, err)
		_, isFailed := failed[currentProducer]
		require.True(t, isFailed, "stalled producer added to failed set")

		buffer := helper.GetSpanRotationBuffer(ctx)
		lmb, err := app.MilestoneKeeper.GetLastMilestoneBlock(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(blockHeight)+buffer, lmb, "<1/3 rotation clock debounced")
		_, _, trackHeight, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(blockHeight)+buffer, trackHeight, "pending-stall clock debounced")
	})
}

// TestPreBlockerPendingStallRotatesWhenForkEnabled drives the full PreBlocker dispatch with the
// hardfork ON: a 40%-band pending milestone whose head has already been static beyond the stall
// threshold must rotate. The companion TestPreBlockerSpanRotationWithMinorityMilestone covers the
// fork-OFF case (no rotation), so together they pin the dispatch branch in both directions.
func TestPreBlockerPendingStallRotatesWhenForkEnabled(t *testing.T) {
	_, app, ctx, validatorPrivKeys := SetupAppWithABCICtxAndValidators(t, 10)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() {
		helper.SetSpanRotationOnStallHeight(origFork)
		helper.SetRioHeight(0)
	})
	helper.SetSpanRotationOnStallHeight(1) // enable the fork

	ctx = ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Abci: &cmtproto.ABCIParams{VoteExtensionsEnableHeight: 1},
	})

	milestone := milestoneTypes.Milestone{
		MilestoneId: "1",
		StartBlock:  0,
		EndBlock:    100,
		Hash:        common.HexToHash("0x1234").Bytes(),
	}
	require.NoError(t, app.MilestoneKeeper.AddMilestone(ctx, milestone))
	require.NoError(t, app.MilestoneKeeper.SetLastMilestoneBlock(ctx, milestone.EndBlock))

	span := &borTypes.Span{
		Id:                1,
		StartBlock:        1,
		EndBlock:          200,
		ValidatorSet:      stakeTypes.ValidatorSet{Validators: validators, Proposer: validators[0]},
		SelectedProducers: []stakeTypes.Validator{*validators[0]},
		BorChainId:        "test",
	}
	require.NoError(t, app.BorKeeper.AddNewSpan(ctx, span))
	seedProducerSelection(t, app, ctx, validators)

	mockCaller := new(helpermocks.IContractCaller)
	mockCaller.On("GetBorChainBlockAuthor", mock.Anything, mock.Anything).
		Return(new(common.HexToAddress(validators[0].Signer)), nil)
	app.BorKeeper.SetContractCaller(mockCaller)

	threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(milestone.EndBlock)))
	blockHeight := int64(milestone.EndBlock) + threshold + 1
	ctx = ctx.WithBlockHeight(blockHeight)
	helper.SetRioHeight(int64(milestone.EndBlock + 1))

	// Pre-age the stall clock against the exact head the partial-support helper proposes
	// (StartBlockNumber = EndBlock+1, hash 0x5678, td 1), so this block trips the rotation.
	expectedProp := &milestoneTypes.MilestoneProposition{
		StartBlockNumber: milestone.EndBlock + 1,
		BlockHashes:      [][]byte{common.HexToHash("0x5678").Bytes()},
		BlockTds:         []uint64{1},
	}
	pendingHead := milestone.EndBlock + 1
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, pendingHead,
		milestoneAbci.MilestonePropositionHeadID(expectedProp), milestone.EndBlock))

	voteExtensions := createVoteExtensionsWithPartialSupport(t, validators, validatorPrivKeys, &milestone, 40, blockHeight-1)
	extCommit := &abci.ExtendedCommitInfo{Round: 0, Votes: voteExtensions}
	extCommitBytes, err := extCommit.Marshal()
	require.NoError(t, err)

	req := &abci.RequestFinalizeBlock{
		Height:          blockHeight,
		Txs:             [][]byte{extCommitBytes, []byte("dummy-tx")},
		ProposerAddress: common.FromHex(validators[0].Signer),
	}

	_, err = app.PreBlocker(ctx, req)
	require.NoError(t, err)

	last, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), last.Id, "fork ON + stalled pending head must rotate via the PreBlocker dispatch")
	require.Equal(t, pendingHead+1, last.StartBlock, "new span starts at N+1")

	// PreBlocker must run to completion after the dispatch (the block proposer is set at the very
	// end); this pins the dispatch's error handling against a swallow/early-return on the happy path.
	_, proposerSet := app.AccountKeeper.GetBlockProposer(ctx)
	require.True(t, proposerSet, "PreBlocker must complete, not early-return after the pending-stall dispatch")
}

// TestCheckAndRotateOnPendingStallEmptyProposition pins the early return for an empty pending
// proposition: it must be a no-op (no rotation, stall clock untouched).
func TestCheckAndRotateOnPendingStallEmptyProposition(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	_, supporters := seedSpan(t, app, ctx)
	ctx = ctx.WithBlockHeight(1000)

	require.NoError(t, app.checkAndRotateOnPendingStall(ctx, &milestoneTypes.MilestoneProposition{}, supporters))

	last, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), last.Id, "empty proposition must not rotate")

	_, _, height, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
	require.NoError(t, err)
	require.Zero(t, height, "empty proposition must not touch the stall clock")
}

// TestRotateSpanFromPendingHeadNoSelectableProducer covers the path where every candidate is
// excluded (the stalled producer plus a full failed set): selection fails, so the rotation logs
// and returns nil rather than erroring (consensus must not halt), and no new span is minted.
func TestRotateSpanFromPendingHeadNoSelectableProducer(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	validators, supporters := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)
	for _, v := range validators {
		require.NoError(t, app.BorKeeper.AddLatestFailedProducer(ctx, v.ValId))
	}
	ctx = ctx.WithBlockHeight(2000)
	prop := singleBlockPendingProp(psPendingHead, 0x11)

	require.NoError(t, app.rotateSpanFromPendingHead(ctx, psPendingHead, milestoneAbci.MilestonePropositionHeadID(prop), supporters))

	last, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), last.Id, "no new span when no producer is selectable")
}
