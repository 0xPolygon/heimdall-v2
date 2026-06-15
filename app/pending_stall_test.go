package app

import (
	"math"
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

// TestCheckAndRotateOnPendingStallReRotatesAwayFromInstalledProducer pins the re-rotation path: when
// the head stays stalled across two rotations, the producer installed by the first rotation (not the
// original one) must be the one excluded on the second. This guards the next-block-to-produce
// (pendingHead+1) lookup — keying off pendingHead would resolve the overlapping older span and keep
// re-selecting the just-installed producer, so the failed set would never grow past the first.
func TestCheckAndRotateOnPendingStallReRotatesAwayFromInstalledProducer(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 5)
	validators, supporters := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)
	prop := singleBlockPendingProp(psPendingHead, 0xEE)
	propID := milestoneAbci.MilestonePropositionHeadID(prop)

	origProducer := validators[0].ValId

	// First rotation: head stalled beyond threshold under the seeded span.
	trackedHeight := uint64(1000)
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, propID, trackedHeight))
	threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
	firstHeight := int64(trackedHeight) + threshold + 1
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx.WithBlockHeight(firstHeight), prop, supporters))

	firstSpan, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), firstSpan.Id, "first rotation mints span 2")
	installedProducer := firstSpan.SelectedProducers[0].ValId
	require.NotEqual(t, origProducer, installedProducer, "first rotation excludes the original producer")

	// The same head stays stalled. The clock was debounced to firstHeight+buffer; age past it again.
	buffer := helper.GetSpanRotationBuffer(ctx)
	secondHeight := firstHeight + int64(buffer) + threshold + 1
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx.WithBlockHeight(secondHeight), prop, supporters))

	secondSpan, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(3), secondSpan.Id, "second rotation mints span 3")
	require.NotEqual(t, installedProducer, secondSpan.SelectedProducers[0].ValId,
		"second rotation must exclude the producer the first rotation installed")

	failed, err := app.BorKeeper.GetLatestFailedProducer(ctx)
	require.NoError(t, err)
	_, origFailed := failed[origProducer]
	_, installedFailed := failed[installedProducer]
	require.True(t, origFailed, "original stalled producer stays in the failed set")
	require.True(t, installedFailed, "the just-installed producer is added to the failed set on re-rotation")
}

// TestCheckAndRotateOnPendingStallReRotatesWhenHeadDrops pins the lower-boundary re-rotation path:
// if the pending head drops below the span installed by the prior rotation, the producer lookup must
// still resolve that just-installed span rather than the older overlapping span.
func TestCheckAndRotateOnPendingStallReRotatesWhenHeadDrops(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 5)
	validators, supporters := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)
	prop := singleBlockPendingProp(psPendingHead, 0xEE)
	propID := milestoneAbci.MilestonePropositionHeadID(prop)

	origProducer := validators[0].ValId

	trackedHeight := uint64(1000)
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, propID, trackedHeight))
	threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
	firstHeight := int64(trackedHeight) + threshold + 1
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx.WithBlockHeight(firstHeight), prop, supporters))

	firstSpan, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), firstSpan.Id, "first rotation mints span 2")
	require.Equal(t, psPendingHead+1, firstSpan.StartBlock)
	installedProducer := firstSpan.SelectedProducers[0].ValId
	require.NotEqual(t, origProducer, installedProducer, "first rotation excludes the original producer")

	// The pending tally drops by one block after the first rotation. The first observation of the
	// dropped head resets the clock, then the same dropped head ages past the threshold and rotates.
	droppedHead := psPendingHead - 1
	droppedProp := singleBlockPendingProp(droppedHead, 0xEF)
	buffer := helper.GetSpanRotationBuffer(ctx)
	resetHeight := firstHeight + int64(buffer) + 1
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx.WithBlockHeight(resetHeight), droppedProp, supporters))

	afterResetSpan, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), afterResetSpan.Id, "head drop only resets the clock")

	secondThreshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(resetHeight))
	secondHeight := resetHeight + secondThreshold + 1
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx.WithBlockHeight(secondHeight), droppedProp, supporters))

	secondSpan, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(3), secondSpan.Id, "second rotation mints span 3")
	require.Equal(t, droppedHead+1, secondSpan.StartBlock, "second rotation starts at the dropped head's N+1")
	require.NotEqual(t, installedProducer, secondSpan.SelectedProducers[0].ValId,
		"second rotation must exclude the producer installed by the first rotation")

	failed, err := app.BorKeeper.GetLatestFailedProducer(ctx)
	require.NoError(t, err)
	_, origFailed := failed[origProducer]
	_, installedFailed := failed[installedProducer]
	require.True(t, origFailed, "original stalled producer stays in the failed set")
	require.True(t, installedFailed, "the just-installed producer is added to the failed set when the head drops")
}

// TestCheckAndRotateOnPendingStallSpanExhaustionBoundary covers report-002 span exhaustion: the
// pending head sits at the very last block of the current span (pendingHead == lastSpan.EndBlock) with
// no successor span minted yet. The next-block lookup (pendingHead+1) lies beyond every span, so the
// rotation must fall back to pendingHead's producer rather than erroring — an erroring producer lookup
// would return up to PreBlocker and halt the chain. Rotation must still succeed from pendingHead+1.
func TestCheckAndRotateOnPendingStallSpanExhaustionBoundary(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 5)
	validators, supporters := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)

	// Head at the span's final block, single span only (no lookahead).
	exhaustedHead := psSpanEnd
	prop := singleBlockPendingProp(exhaustedHead, 0x22)
	trackedHeight := uint64(1000)
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, exhaustedHead, milestoneAbci.MilestonePropositionHeadID(prop), trackedHeight))
	threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
	ctx = ctx.WithBlockHeight(int64(trackedHeight) + threshold + 1)

	currentProducer := validators[0].ValId
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop, supporters), "boundary must not error/halt")

	last, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), last.Id, "a new span must be minted at the span-exhaustion boundary")
	require.Equal(t, exhaustedHead+1, last.StartBlock, "new span starts at N+1")
	require.Greater(t, last.EndBlock, psSpanEnd, "new span must extend past the exhausted runway")
	require.NotEqual(t, currentProducer, last.SelectedProducers[0].ValId, "stalled producer excluded")

	failed, err := app.BorKeeper.GetLatestFailedProducer(ctx)
	require.NoError(t, err)
	_, isFailed := failed[currentProducer]
	require.True(t, isFailed, "boundary stalled producer added to failed set")
}

// TestRotateSpanFromPendingHeadBeyondSpanEnd guards against an unbounded pendingHead. The aggregated
// pending proposition's StartBlockNumber is not bounded against chain state, so a >=1/3 byzantine slice
// could push pendingHead far past lastSpan.EndBlock. An honest producer never advances past its span
// end, so such a head is not a real stall: the rotation must bail (no new span, no error → no PreBlocker
// halt) rather than spin the runway loop or error the producer lookup. The MaxUint64 case also pins the
// loop-overflow guard — without the clamp that loop never terminates.
func TestRotateSpanFromPendingHeadBeyondSpanEnd(t *testing.T) {
	cases := []struct {
		name        string
		pendingHead uint64
	}{
		{"modest overshoot", psSpanEnd + 5},
		{"max uint64 (loop-overflow guard)", math.MaxUint64},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 5)
			validators, supporters := seedSpan(t, app, ctx)
			seedProducerSelection(t, app, ctx, validators)
			ctx = ctx.WithBlockHeight(2000)
			prop := singleBlockPendingProp(tc.pendingHead, 0x33)

			require.NoError(t, app.rotateSpanFromPendingHead(ctx, tc.pendingHead, milestoneAbci.MilestonePropositionHeadID(prop), supporters),
				"a head beyond the span end must not error/halt")

			last, err := app.BorKeeper.GetLastSpan(ctx)
			require.NoError(t, err)
			require.Equal(t, uint64(1), last.Id, "no rotation when pendingHead is beyond the span end")
		})
	}
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
	producerAddr := common.HexToAddress(validators[0].Signer)
	mockCaller.On("GetBorChainBlockAuthor", mock.Anything, mock.Anything).
		Return(&producerAddr, nil)
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

// TestCheckAndRotateCurrentSpanDebouncesPendingStallClock pins the cross-path debounce: after the
// sibling rotation path (checkAndRotateCurrentSpan) installs a fresh producer, a pending milestone
// reappearing at the same head must not immediately re-rotate it. Without advancing the pending-stall
// clock here, the stale pre-rotation baseline would trip rotateSpanFromPendingHead one block later,
// rotating out a producer that has had no chance to extend the head.
func TestCheckAndRotateCurrentSpanDebouncesPendingStallClock(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() {
		helper.SetSpanRotationOnStallHeight(origFork)
		helper.SetRioHeight(0)
	})
	helper.SetSpanRotationOnStallHeight(1) // enable the fork

	validators, supporters := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)

	lastMilestone := milestoneTypes.Milestone{EndBlock: 100, BorChainId: "1"}
	require.NoError(t, app.MilestoneKeeper.AddMilestone(ctx, lastMilestone))
	lastMilestoneBlock := uint64(50)
	require.NoError(t, app.MilestoneKeeper.SetLastMilestoneBlock(ctx, lastMilestoneBlock))

	active := make(map[uint64]struct{}, len(validators))
	for _, v := range validators {
		active[v.ValId] = struct{}{}
	}
	require.NoError(t, app.BorKeeper.UpdateLatestActiveProducer(ctx, active))

	mockCaller := new(helpermocks.IContractCaller)
	producerAddr := common.HexToAddress(validators[0].Signer)
	mockCaller.On("GetBorChainBlockAuthor", mock.Anything, mock.Anything).Return(&producerAddr, nil)
	app.BorKeeper.SetContractCaller(mockCaller)

	helper.SetRioHeight(int64(lastMilestone.EndBlock + 1)) // IsRio(101) == true

	// diff > ChangeProducerThreshold so the sibling path rotates.
	ctx = ctx.WithBlockHeight(int64(lastMilestoneBlock) + helper.GetChangeProducerThreshold(ctx) + 1)
	currentHeight := uint64(ctx.BlockHeight())

	// Seed a pending-stall clock aged well past the threshold against a head inside the rotated span.
	pendingHead := uint64(150)
	prop := singleBlockPendingProp(pendingHead, 0x07)
	pendingHeadID := milestoneAbci.MilestonePropositionHeadID(prop)
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, pendingHead, pendingHeadID, 1))

	require.NoError(t, app.checkAndRotateCurrentSpan(ctx))

	rotated, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), rotated.Id, "sibling path must rotate (diff > threshold, IsRio)")

	// The pending-stall clock must have been debounced to the post-buffer height, head/id preserved.
	block, id, height, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
	require.NoError(t, err)
	require.Equal(t, pendingHead, block, "tracked head preserved")
	require.Equal(t, pendingHeadID, id, "tracked identity preserved")
	require.Equal(t, currentHeight+helper.GetSpanRotationBuffer(ctx), height, "pending-stall clock debounced past the buffer")

	// A pending milestone reappearing at the same head in the same block must not re-rotate.
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop, supporters))
	afterPending, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, rotated.Id, afterPending.Id, "just-installed producer must keep the buffer window; no premature pending-stall re-rotation")
}
