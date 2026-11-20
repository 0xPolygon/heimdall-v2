package keeper

import (
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostest "github.com/cosmos/cosmos-sdk/testutil"
	moduletest "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

// Replacement needs Purley, an expired buffer and params.
func TestBufferReplacement(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ctx := cosmostest.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test")).Ctx
	encoding := moduletest.MakeTestEncodingConfig()
	keeper := NewKeeper(encoding.Codec, runtime.NewKVStoreService(key), authtypes.NewModuleAddress(govtypes.ModuleName).String(), nil, nil, nil, nil)
	params := types.DefaultParams()
	params.CheckpointBufferTime = time.Second
	require.NoError(t, keeper.SetParams(ctx, params))
	buffer := types.Checkpoint{Id: 1, Timestamp: 10}
	require.Error(t, keeper.checkBufferReplacement(ctx, buffer, true, false, 12))
	require.NoError(t, keeper.checkBufferReplacement(ctx, buffer, true, true, 12))
	require.Error(t, keeper.checkBufferReplacement(ctx, buffer, true, true, 10))
	params.CheckpointBufferTime = 0
	require.NoError(t, keeper.SetParams(ctx, params))
	require.Error(t, keeper.checkBufferReplacement(ctx, buffer, true, true, 10))
	require.NoError(t, keeper.params.Remove(ctx))
	require.Error(t, keeper.checkBufferReplacement(ctx, buffer, true, true, 12))
	ctx.KVStore(key).Set(types.BufferedCheckpointPrefixKey, []byte{0xff})
	require.Error(t, keeper.checkReplaceableBuffer(ctx))
}

func TestBufferInstallHeight(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ctx := cosmostest.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test")).Ctx.WithBlockHeight(10)
	encoding := moduletest.MakeTestEncodingConfig()
	keeper := NewKeeper(encoding.Codec, runtime.NewKVStoreService(key), authtypes.NewModuleAddress(govtypes.ModuleName).String(), nil, nil, nil, nil)
	params := types.DefaultParams()
	params.CheckpointBufferTime = time.Second
	require.NoError(t, keeper.SetParams(ctx, params))

	orig := helper.GetPurleyHeight()
	t.Cleanup(func() { helper.SetPurleyHeight(orig) })
	helper.SetPurleyHeight(11)

	buffer := types.Checkpoint{Id: 1, Timestamp: 10}
	require.NoError(t, keeper.SetCheckpointBuffer(ctx, buffer))
	has, err := keeper.bufferHeight.Has(ctx)
	require.NoError(t, err)
	require.False(t, has)

	require.NoError(t, keeper.SetCheckpointBuffer(ctx.WithBlockHeight(11), buffer))
	installedAt, err := keeper.bufferHeight.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(11), installedAt)
	require.Error(t, keeper.checkVotedBufferReplacement(ctx, buffer, true, 11, 12))
	require.NoError(t, keeper.checkVotedBufferReplacement(ctx, buffer, true, 12, 12))

	ctx.KVStore(key).Set(types.BufferedCheckpointHeightPrefixKey, []byte{0xff})
	require.Error(t, keeper.checkVotedBufferReplacement(ctx, buffer, true, 12, 12))
}
