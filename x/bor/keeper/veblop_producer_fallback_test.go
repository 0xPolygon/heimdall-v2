package keeper

import (
	"errors"
	"slices"
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

func valSetOfPowers(powers map[uint64]int64) staketypes.ValidatorSet {
	ids := make([]uint64, 0, len(powers))
	for id := range powers {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	vals := make([]*staketypes.Validator, 0, len(ids))
	for _, id := range ids {
		vals = append(vals, &staketypes.Validator{ValId: id, VotingPower: powers[id]})
	}
	return staketypes.ValidatorSet{Validators: vals}
}

func TestEligibleProducerFallback(t *testing.T) {
	t.Run("prefers supporters that are current positive-power validators", func(t *testing.T) {
		k, ctx, sk := newFallbackKeeper(t)
		sk.EXPECT().GetValidatorSet(gomock.Any()).Return(valSetOf(1, 2, 3), nil)
		got := k.eligibleProducerFallback(ctx, 1, map[uint64]struct{}{1: {}, 2: {}, 3: {}}, nil)
		require.Equal(t, []uint64{2, 3}, got)
	})

	t.Run("drops supporters absent from the current validator set", func(t *testing.T) {
		k, ctx, sk := newFallbackKeeper(t)
		// Supporter 7 exited and is gone from the current set; it must not be selected.
		sk.EXPECT().GetValidatorSet(gomock.Any()).Return(valSetOf(1, 2, 3), nil)
		got := k.eligibleProducerFallback(ctx, 1, map[uint64]struct{}{2: {}, 7: {}}, nil)
		require.Equal(t, []uint64{2}, got)
	})

	t.Run("drops supporters with zero voting power", func(t *testing.T) {
		k, ctx, sk := newFallbackKeeper(t)
		// Supporter 3 is still present but exited (power 0); it must not be selected.
		sk.EXPECT().GetValidatorSet(gomock.Any()).Return(valSetOfPowers(map[uint64]int64{1: 10, 2: 10, 3: 0}), nil)
		got := k.eligibleProducerFallback(ctx, 1, map[uint64]struct{}{2: {}, 3: {}}, nil)
		require.Equal(t, []uint64{2}, got)
	})

	t.Run("falls back to the validator set when no supporter is eligible", func(t *testing.T) {
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

	t.Run("excludes zero-power validators from the validator-set fallback", func(t *testing.T) {
		k, ctx, sk := newFallbackKeeper(t)
		sk.EXPECT().GetValidatorSet(gomock.Any()).Return(valSetOfPowers(map[uint64]int64{1: 10, 2: 0, 3: 10}), nil)
		got := k.eligibleProducerFallback(ctx, 1, map[uint64]struct{}{}, nil)
		require.Equal(t, []uint64{3}, got)
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

// An exited validator can remain in the elected candidate set (voters still rank it) and in the
// supporter set, without the candidate set ever collapsing to the fallback. Post-Ithaca the
// selection pipeline must drop it (zero power, absent from the current set) and pick a real
// producer, keeping the chain live rather than freezing a span Bor cannot build a snapshot from.
// Exercised across the exact activation boundary (F-1, F, F+1) at a configured Ithaca height: the
// gate flips behavior only from block F onward.
func TestSelectNextSpanProducerSkipsExitedElectedCandidate(t *testing.T) {
	k, ctx, sk := newFallbackKeeper(t)

	oldZurich, oldIthaca := helper.GetZurichHardforkHeight(), helper.GetIthacaHeight()
	t.Cleanup(func() {
		helper.SetZurichHardforkHeight(oldZurich)
		helper.SetIthacaHeight(oldIthaca)
	})
	helper.SetZurichHardforkHeight(1)

	const forkHeight int64 = 100
	helper.SetIthacaHeight(forkHeight)

	// Validator 2 has exited: still present in the set but with zero power. 1 and 3 are active.
	sk.EXPECT().GetValidatorSet(gomock.Any()).Return(valSetOfPowers(map[uint64]int64{1: 10, 2: 0, 3: 10}), nil).AnyTimes()

	// Active voters 1 and 3 rank the exited validator 2 first, then 3, so the elected set is [2, 3].
	require.NoError(t, k.SetProducerVotes(ctx, 1, types.ProducerVotes{Votes: []uint64{2, 3}}))
	require.NoError(t, k.SetProducerVotes(ctx, 3, types.ProducerVotes{Votes: []uint64{2, 3}}))

	active := map[uint64]struct{}{2: {}, 3: {}}

	tests := []struct {
		name   string
		height int64
		want   uint64
	}{
		// F-1: gate off, the exited validator is elected, active, and selected — the latent bug.
		{name: "one block before Ithaca selects the exited validator", height: forkHeight - 1, want: 2},
		// F and F+1: gate on, the exited validator is filtered out and a positive-power validator wins.
		{name: "at Ithaca skips the exited validator", height: forkHeight, want: 3},
		{name: "after Ithaca skips the exited validator", height: forkHeight + 1, want: 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cctx := ctx.WithBlockHeight(tc.height)
			limit := helper.GetProducerSetLimit(cctx)

			candidates, err := k.CalculateProducerSet(cctx, limit)
			require.NoError(t, err)
			require.Equal(t, []uint64{2, 3}, candidates)

			got, err := k.SelectNextSpanProducer(cctx, 1, active, limit, 240, 383, types.RoundRobinDefault, nil, false)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// Within a single block an approved exit zeroes a validator's individual record while the
// current-set snapshot (GetValidatorSet) is not refreshed until the stake module's PreBlock. A
// later span-creating post-handler then sees a stale-positive snapshot but a zero-power resolved
// record. AddNewVeBlopSpan must validate the resolved record it actually serializes, not the
// snapshot by id, so it refuses to freeze the zero-power producer that would crash Bor.
func TestAddNewVeBlopSpanRejectsStaleZeroPowerProducer(t *testing.T) {
	k, ctx, sk := newFallbackKeeper(t)

	oldZurich, oldIthaca := helper.GetZurichHardforkHeight(), helper.GetIthacaHeight()
	t.Cleanup(func() {
		helper.SetZurichHardforkHeight(oldZurich)
		helper.SetIthacaHeight(oldIthaca)
	})
	helper.SetZurichHardforkHeight(1)
	helper.SetIthacaHeight(1)
	ctx = ctx.WithBlockHeight(10)

	require.NoError(t, k.AddNewSpan(ctx, &types.Span{
		Id: 1, StartBlock: 100, EndBlock: 199, BorChainId: "1",
		SelectedProducers: []staketypes.Validator{{ValId: 1, VotingPower: 10}},
	}))

	// Snapshot still lists validator 2 with positive power (exit not yet applied to the set), but the
	// resolved individual record already reports zero power.
	sk.EXPECT().GetValidatorSet(gomock.Any()).Return(valSetOfPowers(map[uint64]int64{1: 10, 2: 10}), nil).AnyTimes()
	sk.EXPECT().GetValidatorFromValID(gomock.Any(), uint64(2)).Return(staketypes.Validator{ValId: 2, VotingPower: 0}, nil).AnyTimes()

	// Votes elect [2]; current producer 1 with active {2} selects 2.
	require.NoError(t, k.SetProducerVotes(ctx, 1, types.ProducerVotes{Votes: []uint64{2}}))
	require.NoError(t, k.SetProducerVotes(ctx, 2, types.ProducerVotes{Votes: []uint64{2}}))

	active := map[uint64]struct{}{2: {}}
	err := k.AddNewVeBlopSpan(ctx, 1, 200, 300, "1", active, 5000, types.RoundRobinDefault, nil, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-positive voting power")

	// No span with the stale zero-power producer was frozen.
	last, err := k.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), last.Id)
}

// A signer change maps a validator id to a new, positive-power signer while the current-set
// snapshot may still carry the old signer. Bor's VEBLOP snapshot consumes only SelectedProducers,
// so the new signer is the correct one to seal with; span creation must succeed and freeze the new
// signer. This pins that the guard checks the resolved record's power, not signer equality against
// the (possibly stale) snapshot — a signer-equality check would wrongly reject this valid case.
func TestAddNewVeBlopSpanAcceptsSignerChangedProducer(t *testing.T) {
	k, ctx, sk := newFallbackKeeper(t)

	oldZurich, oldIthaca := helper.GetZurichHardforkHeight(), helper.GetIthacaHeight()
	t.Cleanup(func() {
		helper.SetZurichHardforkHeight(oldZurich)
		helper.SetIthacaHeight(oldIthaca)
	})
	helper.SetZurichHardforkHeight(1)
	helper.SetIthacaHeight(1)
	ctx = ctx.WithBlockHeight(10)

	const oldSigner = "0x0000000000000000000000000000000000000aa1"
	const newSigner = "0x0000000000000000000000000000000000000bb2"

	require.NoError(t, k.AddNewSpan(ctx, &types.Span{
		Id: 1, StartBlock: 100, EndBlock: 199, BorChainId: "1",
		SelectedProducers: []staketypes.Validator{{ValId: 1, VotingPower: 10}},
	}))

	// Snapshot still lists validator 2 under its OLD signer; the resolved individual record maps the
	// same id to a NEW, positive-power signer.
	snapshot := staketypes.ValidatorSet{Validators: []*staketypes.Validator{
		{ValId: 1, VotingPower: 10},
		{ValId: 2, VotingPower: 10, Signer: oldSigner},
	}}
	sk.EXPECT().GetValidatorSet(gomock.Any()).Return(snapshot, nil).AnyTimes()
	sk.EXPECT().GetValidatorFromValID(gomock.Any(), uint64(2)).Return(staketypes.Validator{ValId: 2, VotingPower: 10, Signer: newSigner}, nil).AnyTimes()

	require.NoError(t, k.SetProducerVotes(ctx, 1, types.ProducerVotes{Votes: []uint64{2}}))
	require.NoError(t, k.SetProducerVotes(ctx, 2, types.ProducerVotes{Votes: []uint64{2}}))

	active := map[uint64]struct{}{2: {}}
	require.NoError(t, k.AddNewVeBlopSpan(ctx, 1, 200, 300, "1", active, 5000, types.RoundRobinDefault, nil, false))

	last, err := k.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), last.Id)
	require.Len(t, last.SelectedProducers, 1)
	require.Equal(t, uint64(2), last.SelectedProducers[0].ValId)
	require.Equal(t, newSigner, last.SelectedProducers[0].Signer)
	require.Positive(t, last.SelectedProducers[0].VotingPower)
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
