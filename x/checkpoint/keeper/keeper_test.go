package keeper_test

import (
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	dbTestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	hmModule "github.com/0xPolygon/heimdall-v2/module"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	checkpointKeeper "github.com/0xPolygon/heimdall-v2/x/checkpoint/keeper"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

const (
	AccountHash = "0x000000000000000000000000000000000000dEaD"
	BorChainID  = "1234"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx              sdk.Context
	checkpointKeeper *checkpointKeeper.Keeper
	stakeKeeper      *testutil.MockStakeKeeper
	contractCaller   *mocks.IContractCaller
	topupKeeper      *testutil.MockTopupKeeper
	cmKeeper         *testutil.MockChainManagerKeeper
	queryClient      checkpointTypes.QueryClient
	msgServer        checkpointTypes.MsgServer
	sideMsgCfg       hmModule.SideTxConfigurator
}

func (s *KeeperTestSuite) Run(testname string, fn func()) {
	fn()
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(checkpointTypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)

	testCtx := dbTestutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
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

	checkpointTypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	checkpointTypes.RegisterQueryServer(queryHelper, checkpointKeeper.NewQueryServer(keeper))
	s.queryClient = checkpointTypes.NewQueryClient(queryHelper)
	s.msgServer = checkpointKeeper.NewMsgServerImpl(keeper)

	s.sideMsgCfg = hmModule.NewSideTxConfigurator()
	types.RegisterSideMsgServer(s.sideMsgCfg, checkpointKeeper.NewSideMsgServerImpl(keeper))
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
func (s *KeeperTestSuite) TestAddCheckpoint() {
	ctx, keeper := s.ctx, s.checkpointKeeper
	require := s.Require()

	headerBlockNumber := uint64(2000)
	startBlock := uint64(0)
	endBlock := uint64(256)
	rootHash := hmTypes.HeimdallHash{testutil.RandomBytes()}
	proposerAddress := common.Address{}.String()
	timestamp := uint64(time.Now().Unix())
	borChainId := "1234"

	checkpoint := types.CreateCheckpoint(
		startBlock,
		endBlock,
		rootHash,
		proposerAddress,
		borChainId,
		timestamp,
	)
	err := keeper.AddCheckpoint(ctx, headerBlockNumber, checkpoint)
	require.NoError(err)

	result, err := keeper.GetCheckpointByNumber(ctx, headerBlockNumber)
	require.NoError(err)
	require.True(checkpoint.Equal(result))
}

func (s *KeeperTestSuite) TestFlushCheckpointBuffer() {
	ctx, keeper := s.ctx, s.checkpointKeeper
	require := s.Require()

	err := keeper.FlushCheckpointBuffer(ctx)
	require.Nil(err)

	buffer, err := keeper.GetCheckpointFromBuffer(ctx)
	require.Nil(err)
	require.Nil(buffer)
}
