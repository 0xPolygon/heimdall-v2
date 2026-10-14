package keeper_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	corestore "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
	addrCodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	cmKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	stakeKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"
	testUtil "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// failingStoreService fails writes under prefix once fail is set.
type failingStoreService struct {
	corestore.KVStoreService
	prefix []byte
	fail   *bool
}

func (s failingStoreService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return failingStore{KVStore: s.KVStoreService.OpenKVStore(ctx), prefix: s.prefix, fail: s.fail}
}

type failingStore struct {
	corestore.KVStore
	prefix []byte
	fail   *bool
}

func (s failingStore) Set(key, value []byte) error {
	if *s.fail && bytes.HasPrefix(key, s.prefix) {
		return errors.New("store write failed")
	}
	return s.KVStore.Set(key, value)
}

func newGenesisTestKeeper(t *testing.T, storeService corestore.KVStoreService, key *storetypes.KVStoreKey) (sdk.Context, stakeKeeper.Keeper) {
	t.Helper()
	ctx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test")).Ctx
	encCfg := moduletestutil.MakeTestEncodingConfig()
	ctrl := gomock.NewController(t)

	cmk := cmKeeper.NewKeeper(encCfg.Codec, storeService, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	keeper := stakeKeeper.NewKeeper(encCfg.Codec, storeService, testUtil.NewMockBankKeeper(ctrl), cmk, addrCodec.NewHexCodec(), &mocks.IContractCaller{})
	keeper.SetCheckpointKeeper(testUtil.NewMockCheckpointKeeper(ctrl))
	return ctx, keeper
}

func exportedGenesis(t *testing.T) *types.GenesisState {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	ctx, keeper := newGenesisTestKeeper(t, runtime.NewKVStoreService(key), key)

	validators := make([]*types.Validator, 0, 3)
	for id := uint64(1); id <= 3; id++ {
		pub := secp256k1.GenPrivKey().PubKey()
		v, err := types.NewValidator(id, 0, 0, 1, 10, pub, pub.Address().String())
		require.NoError(t, err)
		validators = append(validators, v)
	}
	keeper.InitGenesis(ctx, types.NewGenesisState(validators, *types.NewValidatorSet(validators), nil))

	exported := keeper.ExportGenesis(ctx)
	require.NotEmpty(t, exported.ValidatorSigners)
	return exported
}

func TestInitGenesisPanicsWhenTheValidatorStoreCannotBeWritten(t *testing.T) {
	exported := exportedGenesis(t)

	key := storetypes.NewKVStoreKey(types.StoreKey)
	fail := false
	ctx, keeper := newGenesisTestKeeper(t, failingStoreService{KVStoreService: runtime.NewKVStoreService(key), prefix: types.SignerKey, fail: &fail}, key)
	fail = true

	require.PanicsWithError(t,
		"error importing the validator store while initializing stake genesis: store write failed\nstore write failed\nstore write failed",
		func() { keeper.InitGenesis(ctx, exported) },
	)
}

func TestExportGenesisPanicsOnAnUnreadableSignerMap(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	ctx, keeper := newGenesisTestKeeper(t, storeService, key)
	keeper.InitGenesis(ctx, exportedGenesis(t))

	// a signer map key too short to decode as a validator ID
	require.NoError(t, storeService.OpenKVStore(ctx).Set(append(append([]byte{}, types.SignerKey...), 0x01), []byte("0x01")))

	require.PanicsWithError(t,
		"error exporting the validator signers: collections: encoding error: wanted at least 8, got: 1",
		func() { keeper.ExportGenesis(ctx) },
	)
}
