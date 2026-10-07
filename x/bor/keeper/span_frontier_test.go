package keeper

import (
	"context"
	"errors"
	"testing"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

var (
	errSpanIterator = errors.New("span iterator unavailable")
	errSpanSet      = errors.New("span store unavailable")
	errSpanHas      = errors.New("span availability unavailable")
)

type failingSpanStoreService struct {
	store corestore.KVStore
}

func (s failingSpanStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return failingSpanStore{KVStore: s.store}
}

type failingSpanStore struct {
	corestore.KVStore
}

func (failingSpanStore) Iterator([]byte, []byte) (corestore.Iterator, error) {
	return nil, errSpanIterator
}

type failingSpanSetStoreService struct{}

func (failingSpanSetStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return failingSpanSetStore{}
}

type failingSpanSetStore struct {
	corestore.KVStore
}

func (failingSpanSetStore) Set([]byte, []byte) error {
	return errSpanSet
}

type failingSpanHasStoreService struct {
	store corestore.KVStore
}

func (s failingSpanHasStoreService) OpenKVStore(context.Context) corestore.KVStore {
	return failingSpanHasStore{KVStore: s.store}
}

type failingSpanHasStore struct {
	corestore.KVStore
}

func (failingSpanHasStore) Has([]byte) (bool, error) {
	return false, errSpanHas
}

func TestSpanEndFrontierReusesCachedValue(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_frontier"))
	encCfg := moduletestutil.MakeTestEncodingConfig()
	keeper := NewKeeper(
		encCfg.Codec,
		runtime.NewKVStoreService(key),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		nil,
		nil,
		nil,
	)
	require.NoError(t, keeper.AddNewSpan(testCtx.Ctx, &types.Span{Id: 0, EndBlock: 100}))
	require.NoError(t, keeper.AddNewSpan(testCtx.Ctx, &types.Span{Id: 1, EndBlock: 100}))

	maxEnd, err := keeper.maxSpanEndBlock(testCtx.Ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), maxEnd)
	require.True(t, keeper.spanFrontier.ready)

	require.NoError(t, keeper.spans.Remove(testCtx.Ctx, 0))
	maxEnd, err = keeper.maxSpanEndBlock(testCtx.Ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), maxEnd)
	keeper.setSpanEndFrontier(50)
	require.Equal(t, uint64(100), keeper.spanFrontier.max)

	_, err = keeper.SpanByBlockNumber(testCtx.Ctx, 101)
	require.ErrorContains(t, err, "span not found for block 101")
}

func TestWarmSpanEndFrontierLeavesEmptyStateLazy(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_frontier_empty"))
	encCfg := moduletestutil.MakeTestEncodingConfig()
	keeper := NewKeeper(
		encCfg.Codec,
		runtime.NewKVStoreService(key),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		nil,
		nil,
		nil,
	)

	require.NoError(t, keeper.WarmSpanEndFrontier(testCtx.Ctx))
	require.False(t, keeper.spanFrontier.ready)

	// Model state sync restoring the KV store directly after app construction.
	restored := types.Span{Id: 7, StartBlock: 1, EndBlock: 100}
	require.NoError(t, keeper.spans.Set(testCtx.Ctx, restored.Id, restored))
	require.NoError(t, keeper.latestSpan.Set(testCtx.Ctx, restored.Id))

	span, err := keeper.SpanByBlockNumber(testCtx.Ctx, 50)
	require.NoError(t, err)
	require.Equal(t, restored.Id, span.Id)
	require.True(t, keeper.spanFrontier.ready)
	require.Equal(t, restored.EndBlock, keeper.spanFrontier.max)
}

func TestWarmSpanEndFrontierPropagatesLastSpanCheckFailure(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_frontier_has_error"))
	encCfg := moduletestutil.MakeTestEncodingConfig()
	baseStore := runtime.NewKVStoreService(key).OpenKVStore(testCtx.Ctx)
	keeper := NewKeeper(
		encCfg.Codec,
		failingSpanHasStoreService{store: baseStore},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		nil,
		nil,
		nil,
	)

	require.ErrorIs(t, keeper.WarmSpanEndFrontier(testCtx.Ctx), errSpanHas)
	require.False(t, keeper.spanFrontier.ready)
}

func TestWarmSpanEndFrontierPropagatesScanFailure(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_frontier_warm_error"))
	encCfg := moduletestutil.MakeTestEncodingConfig()
	baseStore := runtime.NewKVStoreService(key).OpenKVStore(testCtx.Ctx)
	keeper := NewKeeper(
		encCfg.Codec,
		failingSpanStoreService{store: baseStore},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		nil,
		nil,
		nil,
	)
	require.NoError(t, keeper.AddNewSpan(testCtx.Ctx, &types.Span{Id: 0, StartBlock: 1, EndBlock: 100}))

	require.ErrorIs(t, keeper.WarmSpanEndFrontier(testCtx.Ctx), errSpanIterator)
	require.False(t, keeper.spanFrontier.ready)
}

func TestInitGenesisInitializesSpanEndFrontierWithoutIterator(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_frontier_genesis"))
	encCfg := moduletestutil.MakeTestEncodingConfig()
	baseStore := runtime.NewKVStoreService(key).OpenKVStore(testCtx.Ctx)
	keeper := NewKeeper(
		encCfg.Codec,
		failingSpanStoreService{store: baseStore},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		nil,
		nil,
		nil,
	)

	keeper.InitGenesis(testCtx.Ctx, &types.GenesisState{
		Params: types.DefaultParams(),
		Spans: []types.Span{
			{Id: 1, StartBlock: 1, EndBlock: 500},
			{Id: 2, StartBlock: 101, EndBlock: 300},
		},
	})

	maxEnd, err := keeper.maxSpanEndBlock(testCtx.Ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(500), maxEnd)
}

func TestWarmSpanEndFrontierDoesNotChangeAppHash(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_frontier_hash"))
	encCfg := moduletestutil.MakeTestEncodingConfig()
	keeper := NewKeeper(
		encCfg.Codec,
		runtime.NewKVStoreService(key),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		nil,
		nil,
		nil,
	)
	require.NoError(t, keeper.AddNewSpan(testCtx.Ctx, &types.Span{Id: 0, StartBlock: 1, EndBlock: 100}))

	before := testCtx.CMS.Commit()
	require.NoError(t, keeper.WarmSpanEndFrontier(testCtx.Ctx))
	after := testCtx.CMS.Commit()

	require.Equal(t, before.Hash, after.Hash)
}

func TestSpanEndFrontierDoesNotCacheIteratorFailure(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	keeper := NewKeeper(
		encCfg.Codec,
		failingSpanStoreService{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		nil,
		nil,
		nil,
	)

	ctx := sdk.Context{}.WithLogger(log.NewNopLogger())
	_, err := keeper.maxSpanEndBlock(ctx)
	require.ErrorIs(t, err, errSpanIterator)
	require.False(t, keeper.spanFrontier.ready)
}

func TestSpanByBlockNumberPropagatesFrontierInitializationFailure(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_frontier_error"))
	encCfg := moduletestutil.MakeTestEncodingConfig()
	baseStore := runtime.NewKVStoreService(key).OpenKVStore(testCtx.Ctx)
	keeper := NewKeeper(
		encCfg.Codec,
		failingSpanStoreService{store: baseStore},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		nil,
		nil,
		nil,
	)
	require.NoError(t, keeper.AddNewSpan(testCtx.Ctx, &types.Span{Id: 0, EndBlock: 100}))

	_, err := keeper.SpanByBlockNumber(testCtx.Ctx, 50)
	require.ErrorIs(t, err, errSpanIterator)
}

func TestAddNewRawSpanDoesNotAdvanceFrontierWhenStoreFails(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	keeper := NewKeeper(
		encCfg.Codec,
		failingSpanSetStoreService{},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		nil,
		nil,
		nil,
	)
	keeper.spanFrontier.ready = true
	keeper.spanFrontier.max = 10

	err := keeper.AddNewRawSpan(context.Background(), &types.Span{Id: 1, EndBlock: 100})
	require.ErrorIs(t, err, errSpanSet)
	require.Equal(t, uint64(10), keeper.spanFrontier.max)
}
