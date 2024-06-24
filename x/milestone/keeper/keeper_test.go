package keeper_test

import (
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	hmModule "github.com/0xPolygon/heimdall-v2/module"
	milestoneKeeper "github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakekeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	addrCodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var (
	PKs = simtestutil.CreateTestPubKeys(500)
)

type KeeperTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	milestoneKeeper *milestoneKeeper.Keeper
	stakeKeeper     *stakekeeper.Keeper
	contractCaller  *mocks.IContractCaller
	queryClient     milestoneTypes.QueryClient
	msgServer       milestoneTypes.MsgServer
	sideMsgCfg      hmModule.SideTxConfigurator
}

func (s *KeeperTestSuite) Run(testname string, fn func()) {
	fn()
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(milestoneTypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	s.contractCaller = &mocks.IContractCaller{}

	stakekeeper := stakekeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		nil,
		addrCodec.NewHexCodec(),
		s.contractCaller,
	)

	keeper := milestoneKeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		*stakekeeper,
		s.contractCaller,
	)

	s.ctx = ctx
	s.milestoneKeeper = keeper
	s.stakeKeeper = stakekeeper

	milestoneGenesis := types.DefaultGenesisState()

	keeper.InitGenesis(ctx, milestoneGenesis)

	milestoneTypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	milestoneTypes.RegisterQueryServer(queryHelper, milestoneKeeper.Querier{Keeper: keeper})
	s.queryClient = milestoneTypes.NewQueryClient(queryHelper)
	s.msgServer = milestoneKeeper.NewMsgServerImpl(keeper)

	s.sideMsgCfg = hmModule.NewSideTxConfigurator()
	types.RegisterSideMsgServer(s.sideMsgCfg, milestoneKeeper.NewSideMsgServerImpl(keeper))
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestAddMilestone() {
	ctx, keeper := s.ctx, s.milestoneKeeper
	require := s.Require()

	startBlock := uint64(0)
	endBlock := uint64(63)
	hash := hmTypes.HexToHeimdallHash("123")
	proposerAddress := common.HexToAddress("123").String()
	timestamp := uint64(time.Now().Unix())
	borChainId := "1234"
	milestoneID := "0000"

	milestone := types.CreateMilestone(
		startBlock,
		endBlock,
		hash,
		proposerAddress,
		borChainId,
		milestoneID,
		timestamp,
	)

	err := keeper.AddMilestone(ctx, milestone)
	require.NoError(err)

	result, err := keeper.GetLastMilestone(ctx)
	require.NoError(err)
	require.Equal(startBlock, result.StartBlock)
	require.Equal(endBlock, result.EndBlock)
	require.Equal(hash, result.Hash)
	require.Equal(borChainId, result.BorChainID)
	require.Equal(proposerAddress, result.Proposer)
	require.Equal(timestamp, result.TimeStamp)

	result, err = keeper.GetMilestoneByNumber(ctx, 1)
	require.NoError(err)
	require.Equal(startBlock, result.StartBlock)
	require.Equal(endBlock, result.EndBlock)
	require.Equal(hash, result.Hash)
	require.Equal(borChainId, result.BorChainID)
	require.Equal(proposerAddress, result.Proposer)
	require.Equal(timestamp, result.TimeStamp)

	result, err = keeper.GetMilestoneByNumber(ctx, 2)
	require.Nil(result)
	require.Equal(err, types.ErrNoMilestoneFound)
}

func (s *KeeperTestSuite) TestGetCount() {
	ctx, keeper := s.ctx, s.milestoneKeeper
	require := s.Require()

	result := keeper.GetMilestoneCount(ctx)
	require.Equal(uint64(0), result)

	startBlock := uint64(0)
	endBlock := uint64(63)
	hash := hmTypes.HexToHeimdallHash("123")
	proposerAddress := common.HexToAddress("123").String()
	timestamp := uint64(time.Now().Unix())
	borChainId := "1234"
	milestoneID := "0000"

	milestone := types.CreateMilestone(
		startBlock,
		endBlock,
		hash,
		proposerAddress,
		borChainId,
		milestoneID,
		timestamp,
	)
	err := keeper.AddMilestone(ctx, milestone)
	require.NoError(err)

	result = keeper.GetMilestoneCount(ctx)
	require.Equal(uint64(1), result)
}

func (s *KeeperTestSuite) TestGetNoAckMilestone() {
	ctx, keeper := s.ctx, s.milestoneKeeper
	require := s.Require()

	result := keeper.GetMilestoneCount(ctx)
	require.Equal(uint64(0), result)

	milestoneID := "0000"

	keeper.SetNoAckMilestone(ctx, milestoneID)

	val := keeper.GetNoAckMilestone(ctx, "0000")
	require.True(val)

	val = keeper.GetNoAckMilestone(ctx, "00001")
	require.False(val)

	val = keeper.GetNoAckMilestone(ctx, "")
	require.False(val)

	milestoneID = "0001"
	keeper.SetNoAckMilestone(ctx, milestoneID)

	val = keeper.GetNoAckMilestone(ctx, "0001")
	require.True(val)

	val = keeper.GetNoAckMilestone(ctx, "0000")
	require.True(val)
}

func (s *KeeperTestSuite) TestLastNoAckMilestone() {
	ctx, keeper := s.ctx, s.milestoneKeeper
	require := s.Require()

	result := keeper.GetMilestoneCount(ctx)
	require.Equal(uint64(0), result)

	milestoneID := "0000"

	val := keeper.GetLastNoAckMilestone(ctx)
	require.NotEqual(val, milestoneID)

	keeper.SetNoAckMilestone(ctx, milestoneID)

	val = keeper.GetLastNoAckMilestone(ctx)
	require.Equal(val, milestoneID)

	milestoneID = "0001"

	keeper.SetNoAckMilestone(ctx, milestoneID)

	val = keeper.GetLastNoAckMilestone(ctx)
	require.Equal(val, milestoneID)
}

func (s *KeeperTestSuite) TestGetMilestoneTimout() {
	ctx, keeper := s.ctx, s.milestoneKeeper
	require := s.Require()

	val := keeper.GetLastMilestoneTimeout(ctx)
	require.Zero(val)

	keeper.SetLastMilestoneTimeout(ctx, uint64(21))

	val = keeper.GetLastMilestoneTimeout(ctx)
	require.Equal(uint64(21), val)
}
