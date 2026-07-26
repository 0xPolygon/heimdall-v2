package app

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// When the producer candidate set collapses to the current producer, span creation
// fails. Below Kyoto that error is fatal in PreBlocker (the halt this fixes); at/after
// Kyoto it is tolerated so milestone commit and liveness continue.
func TestCheckAndAddFutureSpan_NonFatalWhenSpanCreationFails(t *testing.T) {
	origKyoto := helper.GetKyotoHeight()
	origRio := helper.GetRioHeight()
	t.Cleanup(func() { helper.SetKyotoHeight(origKyoto); helper.SetRioHeight(origRio) })
	setup := func(t *testing.T) (*HeimdallApp, sdk.Context, *milestoneTypes.MilestoneProposition, borTypes.Span, map[uint64]struct{}) {
		t.Helper()
		_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
		ctx = ctx.WithBlockHeight(10)

		validators := app.StakeKeeper.GetAllValidators(ctx)
		valSlice := make([]*stakeTypes.Validator, len(validators))
		selectedProducers := make([]stakeTypes.Validator, len(validators))
		for i := range validators {
			valSlice[i] = validators[i]
			selectedProducers[i] = *validators[i]
		}
		valSet := stakeTypes.ValidatorSet{Validators: valSlice}

		lastSpan := borTypes.Span{Id: 1, StartBlock: 100, EndBlock: 200, BorChainId: "1", ValidatorSet: valSet, SelectedProducers: selectedProducers}
		require.NoError(t, app.BorKeeper.AddNewSpan(ctx, &lastSpan))

		producerValID := selectedProducers[0].ValId
		supportingValidatorIDs := map[uint64]struct{}{}
		for _, v := range validators {
			if v.ValId != producerValID {
				supportingValidatorIDs[v.ValId] = struct{}{}
			}
		}

		helper.SetRioHeight(int64(lastSpan.EndBlock + 1))

		// Collapse the candidate set to exactly the current producer: every validator
		// ranks only producerValID, so excluding the current producer empties the set
		// and SelectProducer fails inside AddNewVeBlopSpan.
		params, err := app.BorKeeper.GetParams(ctx)
		require.NoError(t, err)
		params.ProducerCount = 1
		require.NoError(t, app.BorKeeper.SetParams(ctx, params))

		allIDs := map[uint64]struct{}{}
		for _, v := range validators {
			allIDs[v.ValId] = struct{}{}
			require.NoError(t, app.BorKeeper.SetProducerVotes(ctx, v.ValId, borTypes.ProducerVotes{Votes: []uint64{producerValID}}))
		}
		require.NoError(t, app.BorKeeper.UpdateValidatorPerformanceScore(ctx, allIDs, 1))

		milestone := &milestoneTypes.MilestoneProposition{StartBlockNumber: 150, BlockHashes: [][]byte{[]byte("hash1")}}
		return app, ctx, milestone, lastSpan, supportingValidatorIDs
	}

	t.Run("below kyoto, span-creation failure is fatal", func(t *testing.T) {
		helper.SetKyotoHeight(0)
		app, ctx, milestone, lastSpan, supporting := setup(t)
		require.Error(t, app.checkAndAddFutureSpan(ctx, milestone, lastSpan, supporting))
	})

	t.Run("at/after kyoto, span-creation failure is tolerated", func(t *testing.T) {
		helper.SetKyotoHeight(1)
		app, ctx, milestone, lastSpan, supporting := setup(t)
		require.NoError(t, app.checkAndAddFutureSpan(ctx, milestone, lastSpan, supporting))
		last, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, lastSpan.Id, last.Id, "no new span should be created when selection fails")
	})
}
