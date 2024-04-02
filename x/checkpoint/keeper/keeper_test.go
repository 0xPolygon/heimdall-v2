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
	cmKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	checkpointKeeper "github.com/0xPolygon/heimdall-v2/x/checkpoint/keeper"
	testUtil "github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakekeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	cmTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
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

	ctx                sdk.Context
	checkpointKeeper   *checkpointKeeper.Keeper
	stakeKeeper        *stakekeeper.Keeper
	contractCaller     *mocks.IContractCaller
	moduleCommunicator *testUtil.ModuleCommunicatorMock
	cmKeeper           *cmKeeper.Keeper
	queryClient        checkpointTypes.QueryClient
	msgServer          checkpointTypes.MsgServer
	sideMsgCfg         hmModule.SideTxConfigurator
}

func (s *KeeperTestSuite) Run(testname string, fn func()) {
	fn()
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(checkpointTypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)

	cmkey := storetypes.NewKVStoreKey(cmTypes.StoreKey)
	cmStoreService := runtime.NewKVStoreService(cmkey)

	stakekey := storetypes.NewKVStoreKey(stakeTypes.StoreKey)
	stakeStoreService := runtime.NewKVStoreService(stakekey)

	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	s.contractCaller = &mocks.IContractCaller{}

	moduleCommunicator := testUtil.ModuleCommunicatorMock{}

	cmKeeper := cmKeeper.NewKeeper(encCfg.Codec, cmStoreService)

	stakekeeper := stakekeeper.NewKeeper(
		encCfg.Codec,
		stakeStoreService,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		&cmKeeper,
		addrCodec.NewHexCodec(),
		s.contractCaller,
	)

	keeper := checkpointKeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		*stakekeeper,
		cmKeeper,
		moduleCommunicator,
		s.contractCaller,
	)

	s.ctx = ctx
	s.checkpointKeeper = keeper
	s.stakeKeeper = stakekeeper
	s.cmKeeper = &cmKeeper
	s.moduleCommunicator = &moduleCommunicator

	checkpointGenesis := types.DefaultGenesisState()

	keeper.InitGenesis(ctx, checkpointGenesis)

	checkpointTypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	checkpointTypes.RegisterQueryServer(queryHelper, checkpointKeeper.Querier{Keeper: keeper})
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
	rootHash := hmTypes.HexToHeimdallHash("123")
	proposerAddress := common.Address{}.String()
	timestamp := uint64(time.Now().Unix())
	borChainId := "1234"

	Checkpoint := types.CreateBlock(
		startBlock,
		endBlock,
		rootHash,
		proposerAddress,
		borChainId,
		timestamp,
	)
	err := keeper.AddCheckpoint(ctx, headerBlockNumber, Checkpoint)
	require.NoError(err)

	result, err := keeper.GetCheckpointByNumber(ctx, headerBlockNumber)
	require.NoError(err)
	require.Equal(startBlock, result.StartBlock)
	require.Equal(endBlock, result.EndBlock)
	require.Equal(rootHash, result.RootHash)
	require.Equal(borChainId, result.BorChainID)
	require.Equal(proposerAddress, result.Proposer)
	require.Equal(timestamp, result.TimeStamp)
}

/*
func (s *KeeperTestSuite) TestGetCheckpointList() {
	ctx, keeper := s.ctx, s.checkpointKeeper
	require := s.Require()

	count := 5

	startBlock := uint64(0)
	endBlock := uint64(0)

	for i := 0; i < count; i++ {
		headerBlockNumber := uint64(i) + 1

		startBlock = startBlock + endBlock
		endBlock = endBlock + uint64(255)
		rootHash := hmTypes.HexToHeimdallHash("123")
		proposerAddress := common.Address{}.String()
		timestamp := uint64(time.Now().Unix()) + uint64(i)
		borChainId := "1234"

		Checkpoint := hmTypes.CreateBlock(
			startBlock,
			endBlock,
			rootHash,
			proposerAddress,
			borChainId,
			timestamp,
		)

		err := keeper.AddCheckpoint(ctx, headerBlockNumber, Checkpoint)
		require.NoError(err)

		keeper.UpdateACKCount(ctx)
	}

	result, err := keeper.GetCheckpointList(ctx, uint64(1), uint64(20))
	require.NoError(err)
	require.LessOrEqual(count, len(result))
}

*/

func (s *KeeperTestSuite) TestHasStoreValue() {
	ctx, keeper := s.ctx, s.checkpointKeeper
	require := s.Require()
	key := checkpointTypes.ACKCountKey
	result := keeper.HasStoreValue(ctx, key)
	require.True(result)
}

func (s *KeeperTestSuite) TestFlushCheckpointBuffer() {
	ctx, keeper := s.ctx, s.checkpointKeeper

	require := s.Require()
	key := checkpointTypes.BufferCheckpointKey

	keeper.FlushCheckpointBuffer(ctx)

	result := keeper.HasStoreValue(ctx, key)
	require.False(result)
}
