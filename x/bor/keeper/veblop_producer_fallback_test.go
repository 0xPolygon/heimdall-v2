package keeper

import (
	"errors"
	"testing"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	bortestutil "github.com/0xPolygon/heimdall-v2/x/bor/testutil"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func newFallbackKeeper(t *testing.T) (Keeper, sdk.Context, *bortestutil.MockStakeKeeper) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	ctrl := gomock.NewController(t)
	sk := bortestutil.NewMockStakeKeeper(ctrl)
	k := NewKeeper(
		encCfg.Codec,
		runtime.NewKVStoreService(key),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		bortestutil.NewMockChainManagerKeeper(ctrl),
		sk,
		bortestutil.NewMockMilestoneKeeper(ctrl),
		nil,
	)
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))
	return k, ctx, sk
}

func valSetOf(ids ...uint64) staketypes.ValidatorSet {
	vals := make([]*staketypes.Validator, 0, len(ids))
	for _, id := range ids {
		vals = append(vals, &staketypes.Validator{ValId: id, VotingPower: 10})
	}
	return staketypes.ValidatorSet{Validators: vals}
}

func TestEligibleProducerFallback(t *testing.T) {
	t.Run("prefers the active set and never reads the validator set", func(t *testing.T) {
		k, ctx, _ := newFallbackKeeper(t) // GetValidatorSet must not be called
		got := k.eligibleProducerFallback(ctx, 1, map[uint64]struct{}{1: {}, 2: {}, 3: {}}, nil)
		require.Equal(t, []uint64{2, 3}, got)
	})

	t.Run("falls back to the validator set when the active set is empty", func(t *testing.T) {
		k, ctx, sk := newFallbackKeeper(t)
		sk.EXPECT().GetValidatorSet(gomock.Any()).Return(valSetOf(1, 2, 3, 4), nil)
		got := k.eligibleProducerFallback(ctx, 2, map[uint64]struct{}{}, nil)
		require.Equal(t, []uint64{1, 3, 4}, got)
	})

	t.Run("honors the excluded set in the validator-set fallback", func(t *testing.T) {
		k, ctx, sk := newFallbackKeeper(t)
		sk.EXPECT().GetValidatorSet(gomock.Any()).Return(valSetOf(1, 2, 3, 4), nil)
		got := k.eligibleProducerFallback(ctx, 2, map[uint64]struct{}{}, map[uint64]struct{}{3: {}})
		require.Equal(t, []uint64{1, 4}, got)
	})

	t.Run("returns nil when the validator set cannot be read", func(t *testing.T) {
		k, ctx, sk := newFallbackKeeper(t)
		sk.EXPECT().GetValidatorSet(gomock.Any()).Return(staketypes.ValidatorSet{}, errors.New("boom"))
		got := k.eligibleProducerFallback(ctx, 1, map[uint64]struct{}{}, nil)
		require.Nil(t, got)
	})
}

// SelectNextSpanProducer must not hand SelectProducer an empty slice once the elected set
// has collapsed to the current producer, but only when the caller opts in (the future-span
// path). Pre-Ithaca it still errors (gate off); post-Ithaca without opt-in it still errors
// (non-fatal paths keep skip-and-retry); post-Ithaca with opt-in the fallback supplies a
// replacement from the active set.
func TestSelectNextSpanProducerEmptyCandidateFallback(t *testing.T) {
	k, ctx, sk := newFallbackKeeper(t)

	oldZurich, oldIthaca := helper.GetZurichHardforkHeight(), helper.GetIthacaHeight()
	t.Cleanup(func() {
		helper.SetZurichHardforkHeight(oldZurich)
		helper.SetIthacaHeight(oldIthaca)
	})
	helper.SetZurichHardforkHeight(1)

	vals := valSetOf(1, 2, 3, 4)
	sk.EXPECT().GetValidatorSet(gomock.Any()).Return(vals, nil).AnyTimes()
	// Every validator ranks only producer 1, so CalculateProducerSet collapses to [1].
	for _, v := range vals.Validators {
		require.NoError(t, k.SetProducerVotes(ctx, v.ValId, types.ProducerVotes{Votes: []uint64{1}}))
	}
	ctx = ctx.WithBlockHeight(10)
	limit := helper.GetProducerSetLimit(ctx)

	candidates, err := k.CalculateProducerSet(ctx, limit)
	require.NoError(t, err)
	require.Equal(t, []uint64{1}, candidates)

	// Current producer (1) is excluded from the active set, emptying the candidate list.
	active := map[uint64]struct{}{2: {}, 3: {}, 4: {}}

	helper.SetIthacaHeight(0) // gate off -> still errors even with opt-in
	_, err = k.SelectNextSpanProducer(ctx, 1, active, limit, 240, 383, types.RoundRobinDefault, nil, true)
	require.Error(t, err)

	helper.SetIthacaHeight(1)
	// Without opt-in the non-fatal paths keep their skip-and-retry behavior: still errors
	// when fallbackToActiveSet is false.
	_, err = k.SelectNextSpanProducer(ctx, 1, active, limit, 240, 383, types.RoundRobinDefault, nil, false)
	require.Error(t, err)

	// With opt-in (the future-span path) the fallback selects a non-current producer.
	got, err := k.SelectNextSpanProducer(ctx, 1, active, limit, 240, 383, types.RoundRobinDefault, nil, true)
	require.NoError(t, err)
	require.NotEqual(t, uint64(1), got)
	require.Contains(t, []uint64{2, 3, 4}, got)
}

func TestSortedEligibleProducers(t *testing.T) {
	set := func(ids ...uint64) map[uint64]struct{} {
		m := make(map[uint64]struct{}, len(ids))
		for _, id := range ids {
			m[id] = struct{}{}
		}
		return m
	}

	tests := []struct {
		name            string
		in              map[uint64]struct{}
		currentProducer uint64
		excluded        map[uint64]struct{}
		want            []uint64
	}{
		{
			name:            "sorts ascending and drops current",
			in:              set(5, 2, 4, 3),
			currentProducer: 4,
			want:            []uint64{2, 3, 5},
		},
		{
			name:            "drops excluded",
			in:              set(1, 2, 3, 4),
			currentProducer: 1,
			excluded:        set(3),
			want:            []uint64{2, 4},
		},
		{
			name:            "current and excluded combined empty the set",
			in:              set(1, 2),
			currentProducer: 1,
			excluded:        set(2),
			want:            []uint64{},
		},
		{
			name:            "empty input",
			in:              set(),
			currentProducer: 1,
			want:            []uint64{},
		},
		{
			name:            "nil excluded is a no-op",
			in:              set(9, 7, 8),
			currentProducer: 7,
			excluded:        nil,
			want:            []uint64{8, 9},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sortedEligibleProducers(tc.in, tc.currentProducer, tc.excluded)
			require.Equal(t, tc.want, got)
		})
	}
}
