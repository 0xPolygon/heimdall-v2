package keeper_test

import (
	"io"
	"math/big"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	testutil3 "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	keeper2 "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/app"
	mod "github.com/0xPolygon/heimdall-v2/module"
	"github.com/0xPolygon/heimdall-v2/types"
	topupKeeper "github.com/0xPolygon/heimdall-v2/x/topup/keeper"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

type KeeperTestSuite struct {
	suite.Suite
	ctx           sdk.Context
	keeper        topupKeeper.Keeper
	accountKeeper bankTypes.AccountKeeper
	bankKeeper    bankkeeper.Keeper
	chainID       string
	msgServer     topupTypes.MsgServer
	queryClient   topupTypes.QueryClient
	sideMsgServer topupTypes.SideMsgServer
	sideMsgCfg    mod.SideTxConfigurator

	/* TODO HV2: enable when contractCaller and chainManager are implemented
	contractCaller mocks.IContractCaller
	chainParams    chainTypes.Params
	*/
}

// createTestApp returns context and app on topup keeper
func createTestApp(t *testing.T, isCheckTx bool) (*app.HeimdallApp, sdk.Context) {
	heimdallApp, _, _ := app.SetupApp(t, 4)
	ctx := heimdallApp.BaseApp.NewContext(isCheckTx)

	return heimdallApp, ctx
}

func TestKeeperTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(topupTypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := testutil3.MakeTestEncodingConfig()
	authority := authtypes.NewModuleAddress("gov").String()
	logger := log.NewLogger(io.Discard)

	accountKeeper := keeper2.NewAccountKeeper(
		encCfg.Codec,
		storeService,
		authtypes.ProtoBaseAccount,
		nil,
		address.NewHexCodec(),
		authority,
	)

	bankKeeper := bankkeeper.NewBaseKeeper(encCfg.Codec, storeService, accountKeeper, nil, authority, logger)

	keeper := topupKeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		bankKeeper,
	)

	suite.ctx = ctx
	suite.keeper = keeper

	topupGenesis := topupTypes.DefaultGenesisState()
	keeper.InitGenesis(ctx, topupGenesis)

	topupTypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	topupTypes.RegisterQueryServer(queryHelper, topupKeeper.NewQueryServer(&keeper))

	suite.queryClient = topupTypes.NewQueryClient(queryHelper)
	suite.msgServer = topupKeeper.NewMsgServerImpl(&keeper)
	suite.sideMsgCfg = mod.NewSideTxConfigurator()
	topupTypes.RegisterSideMsgServer(suite.sideMsgCfg, topupKeeper.NewSideMsgServerImpl(&keeper))
}

func (suite *KeeperTestSuite) TestTopupSequenceSet() {
	t, ctx, tk := suite.T(), suite.ctx, suite.keeper
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	topupSequence := strconv.Itoa(simulation.RandIntBetween(r1, 1000, 100000))
	_ = tk.SetTopupSequence(ctx, topupSequence)
	actualResult, _ := tk.HasTopupSequence(ctx, topupSequence)

	require.Equal(t, true, actualResult)
}

func (suite *KeeperTestSuite) TestDividendAccount() {
	t, ctx, tk := suite.T(), suite.ctx, suite.keeper

	dividendAccount := types.DividendAccount{
		User:      "0x000000000000000000000000000000000000dEaD",
		FeeAmount: big.NewInt(0).String(),
	}
	err := tk.SetDividendAccount(ctx, dividendAccount)
	require.NoError(t, err)

	ok, _ := tk.HasDividendAccount(ctx, dividendAccount.User)
	require.Equal(t, ok, true)
}

func (suite *KeeperTestSuite) TestAddFeeToDividendAccount() {
	t, ctx, tk := suite.T(), suite.ctx, suite.keeper

	addr := "0x000000000000000000000000000000000000dEaD"
	amount, _ := big.NewInt(0).SetString("0", 10)
	err := tk.AddFeeToDividendAccount(ctx, addr, amount)
	require.NoError(t, err)

	dividendAccount, _ := tk.GetDividendAccount(ctx, addr)
	actualResult, ok := big.NewInt(0).SetString(dividendAccount.FeeAmount, 10)
	require.Equal(t, ok, true)
	require.Equal(t, amount, actualResult)
}

func (suite *KeeperTestSuite) TestDividendAccountTree() {
	t := suite.T()

	divAccounts := make([]types.DividendAccount, 5)
	for i := 0; i < len(divAccounts); i++ {
		divAccounts[i] = types.DividendAccount{
			User:      "0x000000000000000000000000000000000000dEaD",
			FeeAmount: big.NewInt(0).String(),
		}
	}

	/* TODO HV2: enable when checkpoint is implemented
	accountRoot, err := checkpointTypes.GetAccountRootHash(divAccounts)
	require.NotNil(t, accountRoot)
	require.NoError(t, err)

	accountProof, _, err := checkpointTypes.GetAccountProof(divAccounts, "0x000000000000000000000000000000000000dEaD")
	require.NotNil(t, accountProof)
	require.NoError(t, err)
	*/

	leafHash, err := CalculateDividendAccountHash(divAccounts[0])
	require.NotNil(t, leafHash)
	require.NoError(t, err)
}

// CalculateDividendAccountHash hashes the values of a DividendAccount
func CalculateDividendAccountHash(da types.DividendAccount) ([]byte, error) {
	fee, _ := big.NewInt(0).SetString(da.FeeAmount, 10)
	divAccountHash := crypto.Keccak256(topupTypes.AppendBytes32(
		[]byte(da.User),
		fee.Bytes(),
	))

	return divAccountHash, nil
}
