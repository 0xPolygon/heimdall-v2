package keeper_test

import (
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	addrCodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	hmModule "github.com/0xPolygon/heimdall-v2/module"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	cmKeeper "github.com/0xPolygon/heimdall-v2/x/chainmanager/keeper"
	cmTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	checkpointKeeper "github.com/0xPolygon/heimdall-v2/x/checkpoint/keeper"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	testUtil "github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakekeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"
)

const (
	AccountHash = "0x000000000000000000000000000000000000dEaD"
	BorChainID  = "1234"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx              sdk.Context
	checkpointKeeper *checkpointKeeper.Keeper
	stakeKeeper      *stakekeeper.Keeper
	contractCaller   *mocks.IContractCaller
	topupKeeper      *testutil.MockTopupKeeper
	cmKeeper         *cmKeeper.Keeper
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

	testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	s.contractCaller = &mocks.IContractCaller{}

	moduleCommunicator := testUtil.ModuleCommunicatorMock{}

	chainManagerKeeper := cmKeeper.NewKeeper(encCfg.Codec, storeService)

	cmParams := cmTypes.DefaultParams()
	chainManagerKeeper.SetParams(ctx, cmParams)

	stakeKeeper := stakekeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		nil,
		&chainManagerKeeper,
		addrCodec.NewHexCodec(),
		s.contractCaller,
	)

	keeper := checkpointKeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		*stakeKeeper,
		chainManagerKeeper,
		moduleCommunicator,
		s.contractCaller,
	)

	s.ctx = ctx
	s.checkpointKeeper = keeper
	s.stakeKeeper = stakeKeeper
	s.cmKeeper = &chainManagerKeeper
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
