package keeper_test

import (
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmosTestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	checkpointKeeper "github.com/0xPolygon/heimdall-v2/x/checkpoint/keeper"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

const (
	AccountHash    = "0x000000000000000000000000000000000000dEaD"
	TestBorChainID = "1234"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx              sdk.Context
	checkpointKeeper *checkpointKeeper.Keeper
	stakeKeeper      *testutil.MockStakeKeeper
	contractCaller   *mocks.IContractCaller
	topupKeeper      *testutil.MockTopupKeeper
	cmKeeper         *testutil.MockChainManagerKeeper
	queryClient      types.QueryClient
	msgServer        types.MsgServer
	sideMsgCfg       sidetxs.SideTxConfigurator
}

func (s *KeeperTestSuite) Run(_ string, fn func()) {
	fn()
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)

	testCtx := cosmosTestutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	s.contractCaller = &mocks.IContractCaller{}

	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	s.ctx = ctx
	s.cmKeeper = testutil.NewMockChainManagerKeeper(ctrl)
	s.stakeKeeper = testutil.NewMockStakeKeeper(ctrl)
	s.topupKeeper = testutil.NewMockTopupKeeper(ctrl)

	keeper := checkpointKeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		s.stakeKeeper,
		s.cmKeeper,
		s.topupKeeper,
		s.contractCaller,
	)

	checkpointGenesis := types.DefaultGenesisState()

	keeper.InitGenesis(ctx, checkpointGenesis)

	s.checkpointKeeper = &keeper

	types.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	types.RegisterQueryServer(queryHelper, checkpointKeeper.NewQueryServer(&keeper))
	s.queryClient = types.NewQueryClient(queryHelper)
	s.msgServer = checkpointKeeper.NewMsgServerImpl(&keeper)

	s.sideMsgCfg = sidetxs.NewSideTxConfigurator()
	types.RegisterSideMsgServer(s.sideMsgCfg, checkpointKeeper.NewSideMsgServerImpl(&keeper))
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestAddAndGetCheckpoints() {
	ctx, require, keeper := s.ctx, s.Require(), s.checkpointKeeper

	cpNumber := uint64(2000)
	startBlock := uint64(0)
	endBlock := uint64(256)
	rootHash := testutil.RandomBytes()
	proposerAddress := common.Address{}.String()
	timestamp := uint64(time.Now().Unix())

	checkpoint := types.CreateCheckpoint(
		cpNumber,
		startBlock,
		endBlock,
		rootHash,
		proposerAddress,
		TestBorChainID,
		timestamp,
	)
	err := keeper.AddCheckpoint(ctx, checkpoint)
	require.NoError(err)

	cpNumber2 := uint64(2001)
	startBlock2 := uint64(257)
	endBlock2 := uint64(513)
	rootHash2 := testutil.RandomBytes()
	proposerAddress2 := common.Address{}.String()
	timestamp2 := uint64(time.Now().Unix())

	checkpoint2 := types.CreateCheckpoint(
		cpNumber2,
		startBlock2,
		endBlock2,
		rootHash2,
		proposerAddress2,
		TestBorChainID,
		timestamp2,
	)
	err = keeper.AddCheckpoint(ctx, checkpoint2)
	require.NoError(err)

	result, err := keeper.GetCheckpointByNumber(ctx, cpNumber)
	require.NoError(err)
	require.True(checkpoint.Equal(result))

	result2, err := keeper.GetCheckpointByNumber(ctx, cpNumber2)
	require.NoError(err)
	require.True(checkpoint2.Equal(result2))

	checkpoints, err := keeper.GetCheckpoints(ctx)
	require.NoError(err)
	require.Equal(2, len(checkpoints))
}

func (s *KeeperTestSuite) TestFlushCheckpointBuffer() {
	ctx, require, keeper := s.ctx, s.Require(), s.checkpointKeeper

	err := keeper.FlushCheckpointBuffer(ctx)
	require.Nil(err)

	res, err := keeper.HasCheckpointInBuffer(ctx)
	require.NoError(err)
	require.False(res)

	resp, err := keeper.GetCheckpointFromBuffer(ctx)
	require.NoError(err)
	require.Equal(types.Checkpoint{}, resp)
}
