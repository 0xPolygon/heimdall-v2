package keeper_test

import (
	"fmt"
	"math/big"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/app"
	mod "github.com/0xPolygon/heimdall-v2/module"
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

	msgServer   topupTypes.MsgServer
	sideMsgCfg  mod.SideTxConfigurator
	queryClient topupTypes.QueryClient

	accountKeeper *testutil.MockAccountKeeper

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
	testCtx := cosmostestutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()

	bankKeeper := testutil.NewMockBankKeeper(ctrl)
	accountKeeper := testutil.NewMockAccountKeeper(ctrl)

	// TODO HV2: fix this part

	balances := make(map[string]sdk.Coins)
	distAcct := authtypes.NewModuleAddress("distribution")
	balances[distAcct.String()] = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(0)))

	bankKeeper.EXPECT().IsSendEnabledDenom(gomock.Any(), gomock.Any()).Return(true).AnyTimes()
	bankKeeper.EXPECT().GetBalance(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ sdk.Context, addr sdk.AccAddress, _ string) sdk.Coin {
		balances := balances[addr.String()]
		for _, balance := range balances {
			if balance.Denom == sdk.DefaultBondDenom {
				return balance
			}
		}
		return sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(0))
	}).AnyTimes()
	acc := authtypes.NewBaseAccountWithAddress(sdk.AccAddress(AccountHash))
	accountKeeper.EXPECT().NewAccountWithAddress(gomock.Any(), gomock.Any()).Return(acc).AnyTimes()
	accountKeeper.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Return(acc).AnyTimes()
	bankKeeper.EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ sdk.Context, module string, rcpt sdk.AccAddress, coins sdk.Coins) error {
		balances[rcpt.String()] = balances[rcpt.String()].Add(coins...)
		return nil
	}).AnyTimes()
	accountKeeper.EXPECT().SetAccount(gomock.Any(), gomock.Any()).Return().AnyTimes()
	bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ sdk.Context, sender sdk.AccAddress, _ string, coins sdk.Coins) error {
		newBalance, negative := balances[sender.String()].SafeSub(coins...)
		if negative {
			return fmt.Errorf("not enough balance")
		}
		balances[sender.String()] = newBalance
		return nil
	}).AnyTimes()
	bankKeeper.EXPECT().BurnCoins(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

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
	suite.accountKeeper = accountKeeper
	suite.sideMsgCfg = mod.NewSideTxConfigurator()
	topupTypes.RegisterSideMsgServer(suite.sideMsgCfg, topupKeeper.NewSideMsgServerImpl(&keeper))
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

	leafHash, err := CalculateDividendAccountHash(divAccounts[0])
	require.NotNil(leafHash)
	require.NoError(err)
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
