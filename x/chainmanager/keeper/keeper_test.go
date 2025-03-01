package keeper_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx         sdk.Context
	cmKeeper    keeper.Keeper
	queryClient types.QueryClient
	msgServer   types.MsgServer
}

func (s *KeeperTestSuite) SetupTest() {
	require := s.Require()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	cmKeeper := keeper.NewKeeper(encCfg.Codec, storeService, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	require.NoError(cmKeeper.SetParams(ctx, types.DefaultParams()))

	s.ctx = ctx
	s.cmKeeper = cmKeeper

	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	types.RegisterQueryServer(queryHelper, keeper.NewQueryServer(&cmKeeper))
	s.queryClient = types.NewQueryClient(queryHelper)
	s.msgServer = keeper.NewMsgServerImpl(cmKeeper)
}

func TestKeeperTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestParamsGetterSetter() {
	ctx, require, cmKeeper := s.ctx, s.Require(), s.cmKeeper

	expParams := types.DefaultParams()
	// check that the empty keeper loads the default
	resParams, err := cmKeeper.GetParams(ctx)
	require.NoError(err)
	require.Equal(expParams, resParams)

	expParams.BorChainTxConfirmations = 256
	expParams.MainChainTxConfirmations = 512
	expParams.ChainParams.BorChainId = "1337"
	expParams.ChainParams.HeimdallChainId = "heimdall-1337"
	require.NoError(cmKeeper.SetParams(ctx, expParams))
	resParams, err = cmKeeper.GetParams(ctx)
	require.NoError(err)
	require.True(expParams.Equal(resParams))
}
