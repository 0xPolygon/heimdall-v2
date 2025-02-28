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
	cosmosTestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	util "github.com/0xPolygon/heimdall-v2/common/address"
	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	milestoneKeeper "github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	"github.com/0xPolygon/heimdall-v2/x/milestone/testutil"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

const (
	TestMilestoneID  = "17ce48fe-0a18-41a8-ab7e-59d8002f027b - 0x901a64406d97a3fa9b87b320cbeb86b3c62328f5"
	TestMilestoneID2 = "18ce48fe-0a18-41a8-ab7e-59d8002f027b - 0x801a64406d97a3fa9b87b320cbeb86b3c62328f6"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	milestoneKeeper *milestoneKeeper.Keeper
	stakeKeeper     *testutil.MockStakeKeeper
	contractCaller  *mocks.IContractCaller
	queryClient     milestoneTypes.QueryClient
	sideMsgCfg      sidetxs.SideTxConfigurator
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
		storeService,
		s.contractCaller,
	)

	s.milestoneKeeper = &keeper

	milestoneGenesis := types.DefaultGenesisState()

	keeper.InitGenesis(ctx, milestoneGenesis)

	milestoneTypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	milestoneTypes.RegisterQueryServer(queryHelper, milestoneKeeper.NewQueryServer(&keeper))
	s.queryClient = milestoneTypes.NewQueryClient(queryHelper)

	s.sideMsgCfg = sidetxs.NewSideTxConfigurator()
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestAddMilestone() {
	ctx, require, keeper := s.ctx, s.Require(), s.milestoneKeeper

	startBlock := uint64(0)
	endBlock := uint64(63)
	hash := testutil.RandomBytes()
	proposerAddress := util.FormatAddress(secp256k1.GenPrivKey().PubKey().Address().String())
	timestamp := uint64(time.Now().Unix())
	milestoneID := TestMilestoneID
	borChainId := "1234"

	milestone := testutil.CreateMilestone(
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
	hash := testutil.RandomBytes()
	proposerAddress := secp256k1.GenPrivKey().PubKey().Address().String()
	timestamp := uint64(time.Now().Unix())
	milestoneID := TestMilestoneID
	borChainId := "1234"

	milestone := testutil.CreateMilestone(
		startBlock,
		endBlock,
		hash,
		proposerAddress,
		borChainId,
		milestoneID,
		timestamp,
	)
	err = keeper.AddMilestone(ctx, milestone)
	require.NoError(err)

	result, err = keeper.GetMilestoneCount(ctx)
	require.NoError(err)
	require.Equal(uint64(1), result)
}
