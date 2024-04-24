package keeper_test

import (
	"math/big"
	"math/rand"
	"strconv"
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/app"
	// TODO HV2: enable when module is merged
	// mod "github.com/0xPolygon/heimdall-v2/module"
	"github.com/0xPolygon/heimdall-v2/types"
	topupKeeper "github.com/0xPolygon/heimdall-v2/x/topup/keeper"
	"github.com/0xPolygon/heimdall-v2/x/topup/testutil"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

const (
	AccountHash = "0x000000000000000000000000000000000000dEaD"
	TxHash      = "0x000000000000000000000000000000000000000000000000000000000000dead"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx    sdk.Context
	keeper topupKeeper.Keeper

	msgServer topupTypes.MsgServer
	// TODO HV2: enable when module is merged
	// sideMsgCfg  mod.SideTxConfigurator
	queryClient topupTypes.QueryClient

	/* TODO HV2: enable when contractCaller and chainManager are implemented
	contractCaller mocks.IContractCaller
	chainParams    chainTypes.Params
	*/
}

// createTestApp returns context and app
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

	testCtx := cosmostestutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()
	bankKeeper := testutil.NewMockBankKeeper(ctrl)

	keeper := topupKeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		bankKeeper,
	)

	topupGenesis := topupTypes.DefaultGenesisState()
	keeper.InitGenesis(ctx, topupGenesis)
	topupTypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	topupTypes.RegisterQueryServer(queryHelper, topupKeeper.NewQueryServer(&keeper))

	suite.ctx = ctx
	suite.keeper = keeper
	suite.queryClient = topupTypes.NewQueryClient(queryHelper)
	suite.msgServer = topupKeeper.NewMsgServerImpl(&keeper)
	// TODO HV2: enable when module is merged
	// suite.sideMsgCfg = mod.NewSideTxConfigurator()

	// TODO HV2: replace nil with suite.sideMsgCfg when module is merged
	topupTypes.RegisterSideMsgServer(nil, topupKeeper.NewSideMsgServerImpl(&keeper))
}

func (suite *KeeperTestSuite) TestTopupSequenceSet() {
	ctx, tk, require := suite.ctx, suite.keeper, suite.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	topupSequence := strconv.Itoa(simulation.RandIntBetween(r1, 1000, 100000))
	err := tk.SetTopupSequence(ctx, topupSequence)
	require.Nil(err)

	actualResult, err := tk.HasTopupSequence(ctx, topupSequence)
	require.Nil(err)

	sequences, err := tk.GetAllTopupSequences(ctx)
	require.Nil(err)
	require.Equal(true, actualResult)
	require.Equal(len(sequences), 1)
	require.Equal(topupSequence, sequences[0])
}

func (suite *KeeperTestSuite) TestDividendAccount() {
	ctx, tk, require := suite.ctx, suite.keeper, suite.Require()

	dividendAccount := types.DividendAccount{
		User:      AccountHash,
		FeeAmount: big.NewInt(0).String(),
	}
	err := tk.SetDividendAccount(ctx, dividendAccount)
	require.NoError(err)

	ok, _ := tk.HasDividendAccount(ctx, dividendAccount.User)
	require.Equal(ok, true)

	dividendAccounts, err := tk.GetAllDividendAccounts(ctx)
	require.NoError(err)
	require.Equal(1, len(dividendAccounts))
	require.Equal(dividendAccount, dividendAccounts[0])
}

func (suite *KeeperTestSuite) TestAddFeeToDividendAccount() {
	ctx, tk, require := suite.ctx, suite.keeper, suite.Require()

	amount, _ := big.NewInt(0).SetString("0", 10)
	err := tk.AddFeeToDividendAccount(ctx, AccountHash, amount)
	require.NoError(err)

	dividendAccount, _ := tk.GetDividendAccount(ctx, AccountHash)
	actualResult, ok := big.NewInt(0).SetString(dividendAccount.FeeAmount, 10)
	require.Equal(ok, true)
	require.Equal(amount, actualResult)
}

func (suite *KeeperTestSuite) TestDividendAccountTree() {
	require := suite.Require()

	divAccounts := make([]types.DividendAccount, 5)
	for i := 0; i < len(divAccounts); i++ {
		divAccounts[i] = types.DividendAccount{
			User:      AccountHash,
			FeeAmount: big.NewInt(0).String(),
		}
	}

	/* TODO HV2: enable when checkpoint is implemented
	accountRoot, err := checkpointTypes.GetAccountRootHash(divAccounts)
	require.NotNil(accountRoot)
	require.NoError(err)

	accountProof, _, err := checkpointTypes.GetAccountProof(divAccounts, AccountHash)
	require.NotNil(accountProof)
	require.NoError(err)
	*/

	leafHash, err := testutil.CalculateDividendAccountHash(divAccounts[0])
	require.NotNil(leafHash)
	require.NoError(err)
}
