package app

import (
	"context"
	"errors"
	"math"
	"sort"
	"testing"

	corestore "cosmossdk.io/core/store"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	borKeeper "github.com/0xPolygon/heimdall-v2/x/bor/keeper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	milestoneKeeper "github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
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

// propHeadID is the 32-byte actual-head identity the pending-stall path tracks in production (the
// agreed LatestBlockHash). Tests use the proposition's last block hash as that identity.
func propHeadID(prop *milestoneTypes.MilestoneProposition) []byte {
	return prop.BlockHashes[len(prop.BlockHashes)-1]
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

	// The pending-stall rotation draws its candidate set from the latest active producers (the
	// 2/3-fed set), so seed it with all validators for selection to succeed.
	active := make(map[uint64]struct{}, len(validators))
	for _, v := range validators {
		active[v.ValId] = struct{}{}
	}
	require.NoError(t, app.BorKeeper.UpdateLatestActiveProducer(ctx, active))
}

type scriptedStoreService struct {
	store corestore.KVStore
}

func (s scriptedStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return s.store
}

type scriptedKVStore struct {
	values     map[string][]byte
	setCount   int
	failOnSet  map[int]error
	hasErr     error
	getErr     error
	deleteErr  error
	iterErr    error
	reverseErr error
}

func newScriptedKVStore() *scriptedKVStore {
	return &scriptedKVStore{
		values:    map[string][]byte{},
		failOnSet: map[int]error{},
	}
}

func (s *scriptedKVStore) Get(key []byte) ([]byte, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if value, ok := s.values[string(key)]; ok {
		return append([]byte(nil), value...), nil
	}
	return nil, nil
}

func (s *scriptedKVStore) Has(key []byte) (bool, error) {
	if s.hasErr != nil {
		return false, s.hasErr
	}
	_, ok := s.values[string(key)]
	return ok, nil
}

func (s *scriptedKVStore) Set(key, value []byte) error {
	s.setCount++
	if err, ok := s.failOnSet[s.setCount]; ok {
		return err
	}
	s.values[string(key)] = append([]byte(nil), value...)
	return nil
}

func (s *scriptedKVStore) Delete(key []byte) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	delete(s.values, string(key))
	return nil
}

func (s *scriptedKVStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	if s.iterErr != nil {
		return nil, s.iterErr
	}
	return newScriptedIterator(s.values, start, end, false), nil
}

func (s *scriptedKVStore) ReverseIterator(start, end []byte) (corestore.Iterator, error) {
	if s.reverseErr != nil {
		return nil, s.reverseErr
	}
	return newScriptedIterator(s.values, start, end, true), nil
}

type scriptedIterator struct {
	keys   [][]byte
	values [][]byte
	idx    int
	start  []byte
	end    []byte
}

func newScriptedIterator(values map[string][]byte, start, end []byte, reverse bool) *scriptedIterator {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if reverse {
		for i, j := 0, len(keys)-1; i < j; i, j = i+1, j-1 {
			keys[i], keys[j] = keys[j], keys[i]
		}
	}

	outKeys := make([][]byte, 0, len(keys))
	outVals := make([][]byte, 0, len(keys))
	for _, k := range keys {
		keyBytes := []byte(k)
		if len(start) > 0 && string(keyBytes) < string(start) {
			continue
		}
		if len(end) > 0 && string(keyBytes) >= string(end) {
			continue
		}
		outKeys = append(outKeys, keyBytes)
		outVals = append(outVals, append([]byte(nil), values[k]...))
	}

	return &scriptedIterator{keys: outKeys, values: outVals, start: start, end: end}
}

func (it *scriptedIterator) Domain() (start []byte, end []byte) { return it.start, it.end }
func (it *scriptedIterator) Valid() bool                        { return it.idx >= 0 && it.idx < len(it.keys) }
func (it *scriptedIterator) Next()                              { it.idx++ }
func (it *scriptedIterator) Key() []byte                        { return it.keys[it.idx] }
func (it *scriptedIterator) Value() []byte                      { return it.values[it.idx] }
func (it *scriptedIterator) Error() error                       { return nil }
func (it *scriptedIterator) Close() error                       { return nil }

func newPendingStallTestApp(t *testing.T, base *HeimdallApp, milestoneStore corestore.KVStore, borStore corestore.KVStore) *HeimdallApp {
	t.Helper()
	authAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	milestoneKeeper := milestoneKeeper.NewKeeper(
		base.appCodec,
		authAddr,
		scriptedStoreService{store: milestoneStore},
		base.caller,
	)
	borKeeper := borKeeper.NewKeeper(
		base.appCodec,
		scriptedStoreService{store: borStore},
		authAddr,
		base.ChainManagerKeeper,
		&base.StakeKeeper,
		&milestoneKeeper,
		base.caller,
	)

	base.MilestoneKeeper = milestoneKeeper
	base.BorKeeper = borKeeper
	return base
}

func TestCheckAndRotateOnPendingStall(t *testing.T) {
	t.Run("first observation sets tracking and does not rotate", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
		seedSpan(t, app, ctx)
		ctx = ctx.WithBlockHeight(1000)
		prop := singleBlockPendingProp(psPendingHead, 0xAA)

		require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop.StartBlockNumber, propHeadID(prop)))

		last, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(1), last.Id, "no rotation on first observation")

		block, id, height, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
		require.NoError(t, err)
		require.Equal(t, psPendingHead, block)
		require.Equal(t, propHeadID(prop), id)
		require.Equal(t, uint64(1000), height)
	})

	t.Run("identity flap at same head resets clock, no rotation", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
		seedSpan(t, app, ctx)
		require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, []byte("stale-identity"), 1))
		ctx = ctx.WithBlockHeight(1000) // well past any threshold
		prop := singleBlockPendingProp(psPendingHead, 0xBB)

		require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop.StartBlockNumber, propHeadID(prop)))

		last, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(1), last.Id, "a flapping tip must not rotate")

		_, id, height, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
		require.NoError(t, err)
		require.Equal(t, propHeadID(prop), id, "clock re-baselined to the new identity")
		require.Equal(t, uint64(1000), height)
	})

	t.Run("head advance resets clock, no rotation", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
		seedSpan(t, app, ctx)
		require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead-5, []byte("old"), 1))
		ctx = ctx.WithBlockHeight(1000)
		prop := singleBlockPendingProp(psPendingHead, 0xCC)

		require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop.StartBlockNumber, propHeadID(prop)))

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
		seedSpan(t, app, ctx)
		prop := singleBlockPendingProp(psPendingHead, 0xDD)
		trackedHeight := uint64(1000)
		require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, propHeadID(prop), trackedHeight))
		threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
		ctx = ctx.WithBlockHeight(int64(trackedHeight) + threshold) // borStallDiff == threshold

		require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop.StartBlockNumber, propHeadID(prop)))

		last, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(1), last.Id, "diff == threshold must not rotate (strict >)")
	})

	t.Run("stall beyond threshold rotates from N+1 and debounces", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
		validators, _ := seedSpan(t, app, ctx)
		seedProducerSelection(t, app, ctx, validators)

		prop := singleBlockPendingProp(psPendingHead, 0xEE)
		trackedHeight := uint64(1000)
		require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, propHeadID(prop), trackedHeight))
		threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
		blockHeight := int64(trackedHeight) + threshold + 1 // borStallDiff == threshold+1
		ctx = ctx.WithBlockHeight(blockHeight)

		currentProducer := validators[0].ValId
		require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop.StartBlockNumber, propHeadID(prop)))

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

func TestCheckAndRotateOnPendingStallErrorsOnTrackingStoreRead(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	seedSpan(t, app, ctx)

	milestoneStore := newScriptedKVStore()
	milestoneStore.hasErr = errors.New("tracking read failed")
	app = newPendingStallTestApp(t, app, milestoneStore, newScriptedKVStore())

	prop := singleBlockPendingProp(psPendingHead, 0xAB)
	err := app.checkAndRotateOnPendingStall(ctx.WithBlockHeight(1000), prop.StartBlockNumber, propHeadID(prop))
	require.Error(t, err)
	require.Contains(t, err.Error(), "tracking read failed")
}

func TestCheckAndRotateOnPendingStallErrorsOnTrackingWrite(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	seedSpan(t, app, ctx)

	milestoneStore := newScriptedKVStore()
	milestoneStore.failOnSet[1] = errors.New("tracking write failed")
	app = newPendingStallTestApp(t, app, milestoneStore, newScriptedKVStore())

	prop := singleBlockPendingProp(psPendingHead, 0xAC)
	err := app.checkAndRotateOnPendingStall(ctx.WithBlockHeight(1000), prop.StartBlockNumber, propHeadID(prop))
	require.Error(t, err)
	require.Contains(t, err.Error(), "tracking write failed")
}

// TestCheckAndRotateOnPendingStallReRotatesAwayFromInstalledProducer pins the re-rotation path: when
// the head stays stalled across two rotations, the producer installed by the first rotation (not the
// original one) must be the one excluded on the second. This guards the next-block-to-produce
// (pendingHead+1) lookup — keying off pendingHead would resolve the overlapping older span and keep
// re-selecting the just-installed producer, so the failed set would never grow past the first.
func TestCheckAndRotateOnPendingStallReRotatesAwayFromInstalledProducer(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 5)
	validators, _ := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)
	prop := singleBlockPendingProp(psPendingHead, 0xEE)
	propID := propHeadID(prop)

	origProducer := validators[0].ValId

	// First rotation: head stalled beyond threshold under the seeded span.
	trackedHeight := uint64(1000)
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, propID, trackedHeight))
	threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
	firstHeight := int64(trackedHeight) + threshold + 1
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx.WithBlockHeight(firstHeight), prop.StartBlockNumber, propHeadID(prop)))

	firstSpan, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), firstSpan.Id, "first rotation mints span 2")
	installedProducer := firstSpan.SelectedProducers[0].ValId
	require.NotEqual(t, origProducer, installedProducer, "first rotation excludes the original producer")

	// The same head stays stalled. The clock was debounced to firstHeight+buffer; age past it again.
	buffer := helper.GetSpanRotationBuffer(ctx)
	secondHeight := firstHeight + int64(buffer) + threshold + 1
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx.WithBlockHeight(secondHeight), prop.StartBlockNumber, propHeadID(prop)))

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
// if the pending head drops below the span installed by the prior rotation, the stalled producer is
// the owner of droppedHead+1 in the older span, not the future producer installed for the old head.
func TestCheckAndRotateOnPendingStallReRotatesWhenHeadDrops(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 5)
	validators, _ := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)
	prop := singleBlockPendingProp(psPendingHead, 0xEE)
	propID := propHeadID(prop)

	origProducer := validators[0].ValId

	trackedHeight := uint64(1000)
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, propID, trackedHeight))
	threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
	firstHeight := int64(trackedHeight) + threshold + 1
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx.WithBlockHeight(firstHeight), prop.StartBlockNumber, propHeadID(prop)))

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
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx.WithBlockHeight(resetHeight), droppedProp.StartBlockNumber, propHeadID(droppedProp)))

	afterResetSpan, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), afterResetSpan.Id, "head drop only resets the clock")

	secondThreshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(resetHeight))
	secondHeight := resetHeight + secondThreshold + 1
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx.WithBlockHeight(secondHeight), droppedProp.StartBlockNumber, propHeadID(droppedProp)))

	secondSpan, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(3), secondSpan.Id, "second rotation mints span 3")
	require.Equal(t, droppedHead+1, secondSpan.StartBlock, "second rotation starts at the dropped head's N+1")
	require.NotEqual(t, origProducer, secondSpan.SelectedProducers[0].ValId,
		"second rotation must exclude the producer owning droppedHead+1")

	failed, err := app.BorKeeper.GetLatestFailedProducer(ctx)
	require.NoError(t, err)
	_, origFailed := failed[origProducer]
	_, installedFailed := failed[installedProducer]
	require.True(t, origFailed, "producer owning droppedHead+1 stays in the failed set")
	require.False(t, installedFailed, "future producer is not failed for an older-span next block")
}

// TestCheckAndRotateOnPendingStallUsesNextBlockOwnerBeforeFutureLastSpan guards the case where
// checkAndAddFutureSpan has already scheduled a non-overlapping future span, but the pending bor head
// is still inside the current span. The pending-stall rotation must exclude the producer that owns
// pendingHead+1, not the producer of the already-scheduled future span.
func TestCheckAndRotateOnPendingStallUsesNextBlockOwnerBeforeFutureLastSpan(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 5)
	validators, _ := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)

	baseSpan, err := app.BorKeeper.GetSpan(ctx, 1)
	require.NoError(t, err)
	origProducer := validators[0].ValId
	futureProducer := validators[1].ValId
	futureSpan := borTypes.Span{
		Id:                2,
		StartBlock:        psSpanEnd + 1,
		EndBlock:          psSpanEnd + 100,
		BorChainId:        baseSpan.BorChainId,
		ValidatorSet:      baseSpan.ValidatorSet,
		SelectedProducers: []stakeTypes.Validator{*validators[1]},
	}
	require.NoError(t, app.BorKeeper.AddNewSpan(ctx, &futureSpan))

	prop := singleBlockPendingProp(psPendingHead, 0x77)
	trackedHeight := uint64(1000)
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, propHeadID(prop), trackedHeight))
	threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx.WithBlockHeight(int64(trackedHeight)+threshold+1), prop.StartBlockNumber, propHeadID(prop)))

	last, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(3), last.Id, "pending-stall rotation mints a span after the future span")
	require.Equal(t, psPendingHead+1, last.StartBlock, "new span starts after the stalled pending head")
	require.Equal(t, futureSpan.EndBlock, last.EndBlock, "new span covers the scheduled runway")
	require.NotEqual(t, origProducer, last.SelectedProducers[0].ValId, "stalled current-range producer excluded")

	failed, err := app.BorKeeper.GetLatestFailedProducer(ctx)
	require.NoError(t, err)
	_, origFailed := failed[origProducer]
	_, futureFailed := failed[futureProducer]
	require.True(t, origFailed, "current-range producer is recorded as failed")
	require.False(t, futureFailed, "future scheduled producer is not recorded as failed")
}

// TestCheckAndRotateOnPendingStallSpanExhaustionBoundary covers report-002 span exhaustion: the
// pending head sits at the very last block of the current span (pendingHead == lastSpan.EndBlock) with
// no successor span minted yet. The next-block lookup (pendingHead+1) lies beyond every span, so the
// rotation must fall back to pendingHead's producer rather than erroring — an erroring producer lookup
// would return up to PreBlocker and halt the chain. Rotation must still succeed from pendingHead+1.
func TestCheckAndRotateOnPendingStallSpanExhaustionBoundary(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 5)
	validators, _ := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)

	// Head at the span's final block, single span only (no lookahead).
	exhaustedHead := psSpanEnd
	prop := singleBlockPendingProp(exhaustedHead, 0x22)
	trackedHeight := uint64(1000)
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, exhaustedHead, propHeadID(prop), trackedHeight))
	threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
	ctx = ctx.WithBlockHeight(int64(trackedHeight) + threshold + 1)

	currentProducer := validators[0].ValId
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop.StartBlockNumber, propHeadID(prop)), "boundary must not error/halt")

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
			validators, _ := seedSpan(t, app, ctx)
			seedProducerSelection(t, app, ctx, validators)
			ctx = ctx.WithBlockHeight(2000)
			prop := singleBlockPendingProp(tc.pendingHead, 0x33)

			require.NoError(t, app.rotateSpanFromPendingHead(ctx, tc.pendingHead, propHeadID(prop)),
				"a head beyond the span end must not error/halt")

			last, err := app.BorKeeper.GetLastSpan(ctx)
			require.NoError(t, err)
			require.Equal(t, uint64(1), last.Id, "no rotation when pendingHead is beyond the span end")
		})
	}
}

// TestCheckAndRotateOnPendingStallErrorsWhenLastSpanPointerIsInvalid covers the error branch inside
// rotateSpanFromPendingHead: if the latest-span pointer is corrupted, the pending-stall rotation must
// fail immediately instead of minting a new span from incomplete state.
func TestCheckAndRotateOnPendingStallErrorsWhenLastSpanPointerIsInvalid(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	validators, _ := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)

	prop := singleBlockPendingProp(psPendingHead, 0x44)
	trackedHeight := uint64(1000)
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, propHeadID(prop), trackedHeight))
	threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
	ctx = ctx.WithBlockHeight(int64(trackedHeight) + threshold + 1)

	// Zeroing the latest-span pointer makes GetLastSpan fail inside rotateSpanFromPendingHead.
	require.NoError(t, app.BorKeeper.UpdateLastSpan(ctx, 0))

	err := app.checkAndRotateOnPendingStall(ctx, prop.StartBlockNumber, propHeadID(prop))
	require.Error(t, err)
	require.Contains(t, err.Error(), "span not found")
}

// TestRotateSpanFromPendingHeadErrorsOnMissingParams covers the GetParams failure path inside the
// pending-stall rotation helper by removing the raw params entry after seeding the span state.
func TestRotateSpanFromPendingHeadErrorsOnMissingParams(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	validators, _ := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)

	store := ctx.KVStore(app.keys[borTypes.StoreKey])
	store.Delete(borTypes.ParamsKey.Bytes())

	prop := singleBlockPendingProp(psPendingHead, 0x45)
	err := app.rotateSpanFromPendingHead(ctx.WithBlockHeight(5000), prop.StartBlockNumber, propHeadID(prop))
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestRecordPendingStallRotationErrorsOnMilestoneWrite(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	seedSpan(t, app, ctx)

	milestoneStore := newScriptedKVStore()
	milestoneStore.failOnSet[1] = errors.New("last milestone block failed")
	app = newPendingStallTestApp(t, app, milestoneStore, newScriptedKVStore())

	err := app.recordPendingStallRotation(ctx.WithBlockHeight(5000), psPendingHead, []byte("pending"), psSpanEnd, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "last milestone block failed")
}

func TestRecordPendingStallRotationErrorsOnTrackingWrite(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	seedSpan(t, app, ctx)

	milestoneStore := newScriptedKVStore()
	milestoneStore.failOnSet[2] = errors.New("tracking update failed")
	app = newPendingStallTestApp(t, app, milestoneStore, newScriptedKVStore())

	err := app.recordPendingStallRotation(ctx.WithBlockHeight(5000), psPendingHead, []byte("pending"), psSpanEnd, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tracking update failed")
}

func TestRecordPendingStallRotationErrorsOnFailedProducerWrite(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	seedSpan(t, app, ctx)

	milestoneStore := newScriptedKVStore()
	borStore := newScriptedKVStore()
	borStore.failOnSet[1] = errors.New("failed producer write failed")
	app = newPendingStallTestApp(t, app, milestoneStore, borStore)

	err := app.recordPendingStallRotation(ctx.WithBlockHeight(5000), psPendingHead, []byte("pending"), psSpanEnd, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed producer write failed")
}

func TestDebouncePendingStallClockErrorsOnTrackingWrite(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	seedSpan(t, app, ctx)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(1)

	milestoneStore := newScriptedKVStore()
	app = newPendingStallTestApp(t, app, milestoneStore, newScriptedKVStore())
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, []byte("tracked"), 100))
	milestoneStore.failOnSet[4] = errors.New("debounce tracking write failed")

	err := app.debouncePendingStallClock(ctx.WithBlockHeight(9000), 9999)
	require.Error(t, err)
	require.Contains(t, err.Error(), "debounce tracking write failed")
}

// TestRotateSpanFromPendingHeadErrorsOnCorruptSpanLookup covers the defensive fallback branch where
// the last span is present but corrupt, so the producer lookup on both the pending head and the
// fallback start block fails.
func TestRotateSpanFromPendingHeadErrorsOnCorruptSpanLookup(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	seedProducerSelection(t, app, ctx, validators)

	corruptSpan := borTypes.Span{
		Id:                2,
		StartBlock:        1000,
		EndBlock:          900,
		BorChainId:        "1",
		ValidatorSet:      stakeTypes.ValidatorSet{Validators: validators},
		SelectedProducers: []stakeTypes.Validator{*validators[0]},
	}
	require.NoError(t, app.BorKeeper.AddNewRawSpan(ctx, &corruptSpan))
	require.NoError(t, app.BorKeeper.UpdateLastSpan(ctx, corruptSpan.Id))

	prop := singleBlockPendingProp(psPendingHead, 0x46)
	err := app.rotateSpanFromPendingHead(ctx.WithBlockHeight(5000), prop.StartBlockNumber, propHeadID(prop))
	require.Error(t, err)
	require.Contains(t, err.Error(), "span not found")
}

// TestRotateSpanFromPendingHeadErrorsOnCorruptActiveProducerSet covers the active-producer iterator
// error branch by writing a malformed raw key into the producer-set collection.
func TestRotateSpanFromPendingHeadErrorsOnCorruptActiveProducerSet(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	validators, _ := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)

	store := ctx.KVStore(app.keys[borTypes.StoreKey])
	store.Set(append(borTypes.LatestActiveProducerKey.Bytes(), 0x80), []byte{0x01})

	prop := singleBlockPendingProp(psPendingHead, 0x47)
	err := app.rotateSpanFromPendingHead(ctx.WithBlockHeight(5000), prop.StartBlockNumber, propHeadID(prop))
	require.Error(t, err)
}

// TestPendingStallExcludedProducersErrorsOnCorruptFailedProducerSet covers the failed-producer
// iterator error branch by corrupting the raw collection entry the helper reads.
func TestPendingStallExcludedProducersErrorsOnCorruptFailedProducerSet(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	validators, _ := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)

	store := ctx.KVStore(app.keys[borTypes.StoreKey])
	store.Set(append(borTypes.LatestFailedProducerKey.Bytes(), 0x80), []byte{0x01})

	prop := singleBlockPendingProp(psPendingHead, 0x48)
	err := app.rotateSpanFromPendingHead(ctx.WithBlockHeight(5000), prop.StartBlockNumber, propHeadID(prop))
	require.Error(t, err)
}

// TestDebouncePendingStallClockErrorsOnCorruptTracking covers the pending-tracking decode error path
// by planting an invalid raw value under the tracking prefix before invoking the debounce helper.
func TestDebouncePendingStallClockErrorsOnCorruptTracking(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	seedSpan(t, app, ctx)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(1)

	store := ctx.KVStore(app.keys[milestoneTypes.StoreKey])
	store.Set(milestoneTypes.PendingBorBlockPrefixKey.Bytes(), []byte{0x80})

	err := app.debouncePendingStallClock(ctx.WithBlockHeight(9000), 9999)
	require.Error(t, err)
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

	// Pre-age the stall clock against the exact actual head the partial-support helper reports
	// (LatestBlockNumber = EndBlock+1, LatestBlockHash 0x5678), so this block trips the rotation. The
	// tracked identity is now the actual-head hash, not the proposition head-ID.
	pendingHead := milestone.EndBlock + 1
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, pendingHead,
		common.HexToHash("0x5678").Bytes(), milestone.EndBlock))

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

// TestRotateSpanFromPendingHeadNoSelectableProducer covers the path where every candidate is
// excluded (the stalled producer plus a full failed set): selection fails, so the rotation logs
// and returns nil rather than erroring (consensus must not halt), and no new span is minted.
func TestRotateSpanFromPendingHeadNoSelectableProducer(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	validators, _ := seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)
	for _, v := range validators {
		require.NoError(t, app.BorKeeper.AddLatestFailedProducer(ctx, v.ValId))
	}
	ctx = ctx.WithBlockHeight(2000)
	prop := singleBlockPendingProp(psPendingHead, 0x11)

	require.NoError(t, app.rotateSpanFromPendingHead(ctx, psPendingHead, propHeadID(prop)))

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

	validators, _ := seedSpan(t, app, ctx)
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
	pendingHeadID := propHeadID(prop)
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
	require.NoError(t, app.checkAndRotateOnPendingStall(ctx, prop.StartBlockNumber, propHeadID(prop)))
	afterPending, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, rotated.Id, afterPending.Id, "just-installed producer must keep the buffer window; no premature pending-stall re-rotation")
}

// fill32 returns a 32-byte slice filled with seed.
func fill32(seed byte) []byte {
	h := make([]byte, 32)
	for i := range h {
		h[i] = seed
	}
	return h
}

// TestHandlePendingMilestoneRotatesFromActualHead pins the POS-3629 fix: the rotation keys on the
// >1/3-agreed actual bor head reported in vote extensions, not the (capped) milestone proposition
// tail. The proposition passed here heads at psSpanStart, but the validators agree the actual head
// is psPendingHead, so the new span must start at psPendingHead+1 — the blocks between the
// proposition tail and the actual head are preserved, not reorged.
func TestHandlePendingMilestoneRotatesFromActualHead(t *testing.T) {
	_, app, ctx, privKeys := SetupAppWithABCICtxAndValidators(t, 5)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(1)

	valSet := stakeTypes.NewValidatorSet(validators)
	minVP := valSet.GetTotalVotingPower()/3 + 1

	actualHash := fill32(0x5A)
	extVotes := actualHeadExtVotes(t, validators, privKeys, psPendingHead, actualHash, len(validators))

	// Pre-age the stall clock against the agreed actual head so this block trips the rotation.
	trackedHeight := uint64(1000)
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, actualHash, trackedHeight))
	threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
	ctx = ctx.WithBlockHeight(int64(trackedHeight) + threshold + 1)

	shortProp := singleBlockPendingProp(psSpanStart, 0x01) // proposition tail far below the actual head
	require.NoError(t, app.handlePendingMilestone(ctx, shortProp, valSet, extVotes, minVP))

	last, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), last.Id, "a stalled actual head must rotate")
	require.Equal(t, psPendingHead+1, last.StartBlock,
		"rotation starts at the >1/3-agreed actual head + 1, not the truncated proposition tail")
}

// TestHandlePendingMilestoneSkipsWithoutActualHeadAgreement pins the no-fallback rule: when no actual
// head clears >1/3 (here, no vote carries the latest-head fields), the rotation is skipped and the
// stall clock is left untouched — we never fall back to rotating from the truncated proposition tail.
func TestHandlePendingMilestoneSkipsWithoutActualHeadAgreement(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	seedSpan(t, app, ctx)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(1)

	ctx = ctx.WithBlockHeight(5000) // well past any threshold, were a head agreed
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, fill32(0x07), 1))

	valSet := stakeTypes.NewValidatorSet(validators)
	prop := singleBlockPendingProp(psPendingHead, 0x07)

	// No vote extensions carry an actual head → tally finds nothing → skip.
	require.NoError(t, app.handlePendingMilestone(ctx, prop, valSet, nil, 1))

	last, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), last.Id, "no rotation without a >1/3-agreed actual head")

	block, _, height, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
	require.NoError(t, err)
	require.Equal(t, psPendingHead, block, "tracking head left untouched")
	require.Equal(t, uint64(1), height, "stall clock not reset on a no-agreement block")
}

// TestHandlePendingMilestonePreForkSkipsRotation covers the fork-off branch of
// handlePendingMilestone: before the span-rotation-on-stall hardfork, the
// pending milestone path must remain a pure log-and-return no-op.
func TestHandlePendingMilestonePreForkSkipsRotation(t *testing.T) {
	_, app, ctx, privKeys := SetupAppWithABCICtxAndValidators(t, 3)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	seedSpan(t, app, ctx)
	seedProducerSelection(t, app, ctx, validators)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(0)

	valSet := stakeTypes.NewValidatorSet(validators)
	minVP := valSet.GetTotalVotingPower()/3 + 1
	extVotes := actualHeadExtVotes(t, validators, privKeys, psPendingHead, fill32(0x5A), len(validators))

	ctx = ctx.WithBlockHeight(5000)
	require.NoError(t, app.handlePendingMilestone(ctx, singleBlockPendingProp(psPendingHead, 0x01), valSet, extVotes, minVP))

	last, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), last.Id, "fork-off pending milestone path must not rotate")

	block, _, height, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
	require.NoError(t, err)
	require.Zero(t, block, "fork-off path must not write pending stall tracking")
	require.Zero(t, height, "fork-off path must not start the pending stall clock")
}

// TestDebouncePendingStallClockPreForkNoop covers the helper debounce's fork-off
// guard: if the hardfork is not active, the pending-stall tracking must be left
// unchanged so pre-fork blocks do not write new state.
func TestDebouncePendingStallClockPreForkNoop(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	seedSpan(t, app, ctx)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(0)

	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, fill32(0x7A), 777))
	require.NoError(t, app.debouncePendingStallClock(ctx.WithBlockHeight(9000), 9999))

	block, id, height, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
	require.NoError(t, err)
	require.Equal(t, psPendingHead, block)
	require.Equal(t, fill32(0x7A), id)
	require.Equal(t, uint64(777), height)
}

// TestDebouncePendingStallClockNoTrackingNoop covers the other debounce branch:
// under the fork, an unset pending-stall clock must remain a no-op rather than
// writing a synthetic baseline.
func TestDebouncePendingStallClockNoTrackingNoop(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	seedSpan(t, app, ctx)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(1)

	require.NoError(t, app.debouncePendingStallClock(ctx.WithBlockHeight(9000), 9999))

	block, _, height, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
	require.NoError(t, err)
	require.Zero(t, block)
	require.Zero(t, height)
}

// TestDebouncePendingStallClockForkActiveUpdatesTracking covers the active fork path of the
// debounce helper: when the pending-stall clock is already running, the helper must preserve the
// tracked head and identity while advancing only the height baseline.
func TestDebouncePendingStallClockForkActiveUpdatesTracking(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	seedSpan(t, app, ctx)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(1)

	prop := singleBlockPendingProp(psPendingHead, 0x7B)
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, psPendingHead, propHeadID(prop), 777))
	require.NoError(t, app.debouncePendingStallClock(ctx.WithBlockHeight(9000), 9999))

	block, id, height, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
	require.NoError(t, err)
	require.Equal(t, psPendingHead, block)
	require.Equal(t, propHeadID(prop), id)
	require.Equal(t, uint64(9999), height)
}

// TestHandlePendingMilestoneMissingLastSpanErrors pins the error-return branch:
// when the keeper cannot resolve the last span, the pending milestone path must
// fail fast instead of trying to rotate from a nonexistent runway.
func TestHandlePendingMilestoneMissingLastSpanErrors(t *testing.T) {
	_, app, ctx, privKeys := SetupAppWithABCICtxAndValidators(t, 3)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(1)

	valSet := stakeTypes.NewValidatorSet(validators)
	minVP := valSet.GetTotalVotingPower()/3 + 1
	extVotes := actualHeadExtVotes(t, validators, privKeys, psPendingHead, fill32(0x5A), len(validators))

	ctx = ctx.WithBlockHeight(5000)
	err := app.handlePendingMilestone(ctx, singleBlockPendingProp(psPendingHead, 0x01), valSet, extVotes, minVP)
	require.Error(t, err)
}

// TestHandlePendingMilestoneErrorsOnMalformedActualHeadVote covers the error path from the actual-head
// tally helper: malformed vote-extension bytes must fail fast instead of being treated as an empty
// pending-stall signal.
func TestHandlePendingMilestoneErrorsOnMalformedActualHeadVote(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 3)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	seedSpan(t, app, ctx)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(1)

	valSet := stakeTypes.NewValidatorSet(validators)
	minVP := valSet.GetTotalVotingPower()/3 + 1
	badVote := abci.ExtendedVoteInfo{
		BlockIdFlag:   cmtproto.BlockIDFlagCommit,
		VoteExtension: []byte{0xFF, 0x00, 0x01},
		Validator:     abci.Validator{Address: common.HexToAddress(validators[0].Signer).Bytes()},
	}

	err := app.handlePendingMilestone(ctx.WithBlockHeight(5000), singleBlockPendingProp(psPendingHead, 0x01), valSet, []abci.ExtendedVoteInfo{badVote}, minVP)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error while unmarshalling vote extension")
}

// TestHandlePendingMilestoneDropsOutOfRangeActualHead pins the byzantine-poison guard (POS-3629): a
// >1/3 slice agreeing on a head beyond the last span's end (the scheduled runway — no honest producer
// can advance past it) must be filtered by the maxBlock bound, so it is never written into the stall
// tracking and never triggers a rotation. Without the bound the fabricated head would be tracked on
// first observation.
func TestHandlePendingMilestoneDropsOutOfRangeActualHead(t *testing.T) {
	_, app, ctx, privKeys := SetupAppWithABCICtxAndValidators(t, 5)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	seedSpan(t, app, ctx) // span [psSpanStart, psSpanEnd]
	seedProducerSelection(t, app, ctx, validators)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(1)

	valSet := stakeTypes.NewValidatorSet(validators)
	minVP := valSet.GetTotalVotingPower()/3 + 1

	// All validators agree on a head far beyond the last span's end — fabricated, since no honest
	// producer can advance past the scheduled runway. The bound must drop it from the tally.
	outOfRange := psSpanEnd + 10_000
	extVotes := actualHeadExtVotes(t, validators, privKeys, outOfRange, fill32(0x5A), len(validators))

	ctx = ctx.WithBlockHeight(5000)
	require.NoError(t, app.handlePendingMilestone(ctx, singleBlockPendingProp(psPendingHead, 0x01), valSet, extVotes, minVP))

	last, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), last.Id, "an out-of-range agreed head must not trigger a rotation")

	block, _, height, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
	require.NoError(t, err)
	require.Zero(t, height, "an out-of-range head must never be written into the stall tracking")
	require.Zero(t, block, "tracking head must stay unset")
}

// TestRotateSpanFromPendingHeadFallsBackToLastSpanStart covers the defensive fallback
// in producer lookup: if the next block after the pending head is not owned by any
// span, the code must fall back to the last span's start block instead of failing.
func TestRotateSpanFromPendingHeadFallsBackToLastSpanStart(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCICtxAndValidators(t, 5)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Build only a future-scheduled span, so pendingHead+1 has no owner and the
	// fallback path must resolve the current producer from the last span's start.
	futureSpan := borTypes.Span{
		Id:                2,
		StartBlock:        1000,
		EndBlock:          1100,
		BorChainId:        "1",
		ValidatorSet:      stakeTypes.ValidatorSet{Validators: validators},
		SelectedProducers: []stakeTypes.Validator{*validators[1]},
	}
	require.NoError(t, app.BorKeeper.AddNewSpan(ctx, &futureSpan))
	seedProducerSelection(t, app, ctx, validators)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(1)

	prop := singleBlockPendingProp(150, 0x66)
	require.NoError(t, app.rotateSpanFromPendingHead(ctx.WithBlockHeight(5000), prop.StartBlockNumber, propHeadID(prop)))

	last, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(3), last.Id, "fallback path must still mint a span")
	require.Equal(t, uint64(151), last.StartBlock, "rotation must still start at pendingHead+1")
}

// seedFutureSpan adds a future-scheduled span 2 [psSpanEnd+1, psSpanEnd+6310] (producer valSlice[1]),
// making it the highest-ID span so GetLastSpan returns it while span 1 stays the active range.
func seedFutureSpan(t *testing.T, app *HeimdallApp, ctx sdk.Context, valSlice []*stakeTypes.Validator) {
	t.Helper()
	require.NoError(t, app.BorKeeper.AddNewSpan(ctx, &borTypes.Span{
		Id:                2,
		StartBlock:        psSpanEnd + 1,
		EndBlock:          psSpanEnd + 6310,
		BorChainId:        "1",
		ValidatorSet:      stakeTypes.ValidatorSet{Validators: valSlice},
		SelectedProducers: []stakeTypes.Validator{*valSlice[1]},
	}))
}

// TestHandlePendingMilestoneRecoversAcrossSpanBoundary pins that recovery still works when bor
// honestly crosses into an already-scheduled future span while milestones lag (POS-3629). The bound
// is the last span's end (the scheduled runway), so an honest head inside the future span is kept,
// not dropped — an earlier active-span-only bound would have denied this legitimate recovery.
func TestHandlePendingMilestoneRecoversAcrossSpanBoundary(t *testing.T) {
	_, app, ctx, privKeys := SetupAppWithABCICtxAndValidators(t, 5)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	valSlice, _ := seedSpan(t, app, ctx) // active span 1 [psSpanStart, psSpanEnd]
	seedProducerSelection(t, app, ctx, validators)
	seedFutureSpan(t, app, ctx, valSlice)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(1)

	valSet := stakeTypes.NewValidatorSet(validators)
	minVP := valSet.GetTotalVotingPower()/3 + 1

	// All validators honestly report a head inside the future span — bor crossed the boundary and the
	// new producer stalled there. The head is within the scheduled runway, so it must be honored.
	crossedHead := psSpanEnd + 60
	actualHash := fill32(0x5A)
	extVotes := actualHeadExtVotes(t, validators, privKeys, crossedHead, actualHash, len(validators))

	// Pre-age the clock against that head so this block trips the rotation.
	trackedHeight := uint64(1000)
	require.NoError(t, app.MilestoneKeeper.SetPendingBorBlockTracking(ctx, crossedHead, actualHash, trackedHeight))
	threshold := helper.GetBorStallThreshold(ctx.WithBlockHeight(int64(trackedHeight)))
	ctx = ctx.WithBlockHeight(int64(trackedHeight) + threshold + 1)

	require.NoError(t, app.handlePendingMilestone(ctx, singleBlockPendingProp(psPendingHead, 0x01), valSet, extVotes, minVP))

	last, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(3), last.Id, "a stalled head across the span boundary must still rotate")
	require.Equal(t, crossedHead+1, last.StartBlock, "rotation starts at the agreed cross-boundary head + 1")
}

// TestHandlePendingMilestoneByzantineMinorityCannotSteerHead pins the highest-voting-power tally
// (POS-3629): a >1/3 byzantine minority reporting a higher fabricated head (here inside a future span)
// cannot outvote the converged honest majority on the real head. The agreed head — and therefore the
// stall tracking — follows the honest majority, so the minority can neither steer rotation into the
// future producer nor reset the clock by rotating a fake head's hash. Highest-block-number selection
// would instead pick the minority's higher head.
func TestHandlePendingMilestoneByzantineMinorityCannotSteerHead(t *testing.T) {
	_, app, ctx, privKeys := SetupAppWithABCICtxAndValidators(t, 5)
	validators := app.StakeKeeper.GetAllValidators(ctx)
	valSlice, _ := seedSpan(t, app, ctx) // active span 1 [psSpanStart, psSpanEnd]
	seedProducerSelection(t, app, ctx, validators)
	seedFutureSpan(t, app, ctx, valSlice)

	origFork := helper.GetSpanRotationOnStallHeight()
	t.Cleanup(func() { helper.SetSpanRotationOnStallHeight(origFork) })
	helper.SetSpanRotationOnStallHeight(1)

	valSet := stakeTypes.NewValidatorSet(validators)
	minVP := valSet.GetTotalVotingPower()/3 + 1

	// 2 byzantine validators (200 VP, clears 1/3+1) report a higher fabricated head in the future span;
	// 3 honest validators (300 VP) report the real head in the active span. Greatest VP must win.
	byzantineHead := psSpanEnd + 100
	honestHead := psPendingHead
	extVotes := actualHeadExtVotesSplit(t, validators, privKeys, 2, byzantineHead, fill32(0xBB), honestHead, fill32(0x5A))

	ctx = ctx.WithBlockHeight(5000)
	require.NoError(t, app.handlePendingMilestone(ctx, singleBlockPendingProp(psPendingHead, 0x01), valSet, extVotes, minVP))

	block, id, _, err := app.MilestoneKeeper.GetPendingBorBlockTracking(ctx)
	require.NoError(t, err)
	require.Equal(t, honestHead, block, "the tracked head must be the honest majority's, not the byzantine minority's higher head")
	require.Equal(t, fill32(0x5A), id, "the tracked identity must be the honest majority's head hash")
}
