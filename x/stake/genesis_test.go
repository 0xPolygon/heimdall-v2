package stake_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	addrCodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	cmKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	"github.com/0xPolygon/heimdall-v2/x/stake"
	stakeKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"
	stakeTestUtil "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestWriteValidatorsKeepsOnlyCurrentValidators(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	ctx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test")).Ctx
	encCfg := moduletestutil.MakeTestEncodingConfig()
	ctrl := gomock.NewController(t)

	cmk := cmKeeper.NewKeeper(encCfg.Codec, storeService, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	keeper := stakeKeeper.NewKeeper(encCfg.Codec, storeService, stakeTestUtil.NewMockBankKeeper(ctrl), cmk, addrCodec.NewHexCodec(), &mocks.IContractCaller{})
	keeper.SetCheckpointKeeper(stakeTestUtil.NewMockCheckpointKeeper(ctrl))

	// the current epoch is ackCount+1
	const ackCount = 10
	add := func(id, startEpoch, endEpoch uint64, power int64, jailed bool) string {
		pub := secp256k1.GenPrivKey().PubKey()
		v, err := types.NewValidator(id, startEpoch, endEpoch, 1, power, pub, pub.Address().String())
		require.NoError(t, err)
		v.Jailed = jailed
		require.NoError(t, keeper.AddValidator(ctx, *v))
		return v.Signer
	}
	want := []string{
		add(1, 0, 0, 100, false),
		add(2, ackCount+1, 0, 100, false), // starts this epoch
		add(3, 0, ackCount+2, 100, false), // ends next epoch
	}
	add(4, ackCount+2, 0, 100, false) // starts next epoch
	add(5, 0, ackCount+1, 100, false) // ends this epoch
	add(6, 0, 0, 0, false)            // pre-rotation record
	add(7, 0, 0, 100, true)           // jailed

	vals, err := stake.WriteValidators(ctx, &keeper, ackCount)
	require.NoError(t, err)
	got := make([]string, 0, len(vals))
	for _, v := range vals {
		got = append(got, v.Name)
	}
	require.ElementsMatch(t, want, got)
}
