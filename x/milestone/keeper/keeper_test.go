package keeper_test

import (
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	hmModule "github.com/0xPolygon/heimdall-v2/module"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	milestoneKeeper "github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	"github.com/0xPolygon/heimdall-v2/x/milestone/testutil"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	cosmosTestutil "github.com/cosmos/cosmos-sdk/testutil"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	milestoneKeeper *milestoneKeeper.Keeper
	stakeKeeper     *testutil.MockStakeKeeper
	contractCaller  *mocks.IContractCaller
	queryClient     milestoneTypes.QueryClient
	msgServer       milestoneTypes.MsgServer
	sideMsgCfg      hmModule.SideTxConfigurator
}

func (s *KeeperTestSuite) Run(_ string, fn func()) {
	fn()
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(milestoneTypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := cosmosTestutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	s.ctx = ctx
	s.contractCaller = &mocks.IContractCaller{}
	s.stakeKeeper = testutil.NewMockStakeKeeper(ctrl)

	keeper := milestoneKeeper.NewKeeper(
		encCfg.Codec,
		"authority",
		storeService,
		s.stakeKeeper,
		s.contractCaller,
	)

	s.milestoneKeeper = &keeper

	milestoneGenesis := types.DefaultGenesisState()

	keeper.InitGenesis(ctx, milestoneGenesis)

	milestoneTypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	milestoneTypes.RegisterQueryServer(queryHelper, milestoneKeeper.NewQueryServer(&keeper))
	s.queryClient = milestoneTypes.NewQueryClient(queryHelper)
	s.msgServer = milestoneKeeper.NewMsgServerImpl(&keeper)

	s.sideMsgCfg = hmModule.NewSideTxConfigurator()
	types.RegisterSideMsgServer(s.sideMsgCfg, milestoneKeeper.NewSideMsgServerImpl(&keeper))
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestAddMilestone() {
	ctx, require, keeper := s.ctx, s.Require(), s.milestoneKeeper

	startBlock := uint64(0)
	endBlock := uint64(63)
	hash := hmTypes.HeimdallHash{Hash: testutil.RandomBytes()}
	proposerAddress := secp256k1.GenPrivKey().PubKey().Address().String()
	timestamp := uint64(time.Now().Unix())
	milestoneID := "0000"

	milestone := testutil.CreateMilestone(
		startBlock,
		endBlock,
		hash,
		proposerAddress,
		BorChainId,
		milestoneID,
		timestamp,
	)

	err := keeper.AddMilestone(ctx, milestone)
	require.NoError(err)

	result, err := keeper.GetLastMilestone(ctx)
	require.NoError(err)
	require.True(milestone.Equal(result))

	result, err = keeper.GetMilestoneByNumber(ctx, 1)
	require.NoError(err)
	require.True(milestone.Equal(result))

	result, err = keeper.GetMilestoneByNumber(ctx, 2)
	require.Nil(result)
	require.Error(err)
}

func (s *KeeperTestSuite) TestGetMilestoneCount() {
	ctx, require, keeper := s.ctx, s.Require(), s.milestoneKeeper

	result, err := keeper.GetMilestoneCount(ctx)
	require.NoError(err)
	require.Equal(uint64(0), result)

	startBlock := uint64(0)
	endBlock := uint64(63)
	hash := hmTypes.HeimdallHash{Hash: testutil.RandomBytes()}
	proposerAddress := secp256k1.GenPrivKey().PubKey().Address().String()
	timestamp := uint64(time.Now().Unix())
	milestoneID := "0000"

	milestone := testutil.CreateMilestone(
		startBlock,
		endBlock,
		hash,
		proposerAddress,
		BorChainId,
		milestoneID,
		timestamp,
	)
	err = keeper.AddMilestone(ctx, milestone)
	require.NoError(err)

	result, err = keeper.GetMilestoneCount(ctx)
	require.NoError(err)
	require.Equal(uint64(1), result)
}

func (s *KeeperTestSuite) TestGetNoAckMilestone() {
	ctx, require, keeper := s.ctx, s.Require(), s.milestoneKeeper

	result, err := keeper.GetMilestoneCount(ctx)
	require.NoError(err)
	require.Equal(uint64(0), result)

	milestoneID := "0000"

	err = keeper.SetNoAckMilestone(ctx, milestoneID)
	require.NoError(err)

	val, err := keeper.HasNoAckMilestone(ctx, milestoneID)
	require.NoError(err)
	require.True(val)

	val, err = keeper.HasNoAckMilestone(ctx, "00001")
	require.NoError(err)
	require.False(val)

	val, err = keeper.HasNoAckMilestone(ctx, "")
	require.NoError(err)
	require.False(val)

	milestoneID = "0001"
	err = keeper.SetNoAckMilestone(ctx, milestoneID)
	require.NoError(err)

	val, err = keeper.HasNoAckMilestone(ctx, "0001")
	require.NoError(err)
	require.True(val)

	val, err = keeper.HasNoAckMilestone(ctx, milestoneID)
	require.NoError(err)
	require.True(val)
}

func (s *KeeperTestSuite) TestLastNoAckMilestone() {
	ctx, require, keeper := s.ctx, s.Require(), s.milestoneKeeper

	result, err := keeper.GetMilestoneCount(ctx)
	require.NoError(err)
	require.Equal(uint64(0), result)

	milestoneID := "0000"

	val, err := keeper.GetLastNoAckMilestone(ctx)
	require.Error(err)

	err = keeper.SetNoAckMilestone(ctx, milestoneID)
	require.NoError(err)

	val, err = keeper.GetLastNoAckMilestone(ctx)
	require.NoError(err)
	require.Equal(val, milestoneID)

	milestoneID = "0001"

	err = keeper.SetNoAckMilestone(ctx, milestoneID)
	require.NoError(err)

	val, err = keeper.GetLastNoAckMilestone(ctx)
	require.NoError(err)
	require.Equal(val, milestoneID)
}

func (s *KeeperTestSuite) TestGetMilestoneTimout() {
	ctx, require, keeper := s.ctx, s.Require(), s.milestoneKeeper

	val, err := keeper.GetLastMilestoneTimeout(ctx)
	require.NoError(err)
	require.Zero(val)

	err = keeper.SetLastMilestoneTimeout(ctx, uint64(21))
	require.NoError(err)

	val, err = keeper.GetLastMilestoneTimeout(ctx)
	require.NoError(err)
	require.Equal(uint64(21), val)
}
