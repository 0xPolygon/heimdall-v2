package keeper_test

import (
	"math/big"
	"math/rand"
	"strconv"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/app"
	"github.com/0xPolygon/heimdall-v2/types"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

type KeeperTestSuite struct {
	suite.Suite
	ctx           sdk.Context
	app           *app.HeimdallApp
	queryClient   topupTypes.QueryClient
	msgServer     topupTypes.MsgServer
	sideMsgServer topupTypes.SideMsgServer
	/* TODO HV2: enable when SideTxConfigurator, helper, contractCaller and chainManager are implemented
	sideMsgCfg    SideTxConfigurator
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

func (suite *KeeperTestSuite) SetupTest() {
	suite.app, suite.ctx = createTestApp(suite.T(), false)
}

func TestKeeperTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestTopupSequenceSet() {
	t, heimdallApp, ctx := suite.T(), suite.app, suite.ctx
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	topupSequence := strconv.Itoa(simulation.RandIntBetween(r1, 1000, 100000))
	_ = heimdallApp.TopupKeeper.SetTopupSequence(ctx, topupSequence)
	actualResult, _ := heimdallApp.TopupKeeper.HasTopupSequence(ctx, topupSequence)

	require.Equal(t, true, actualResult)
}

func (suite *KeeperTestSuite) TestDividendAccount() {
	t, heimdallApp, ctx := suite.T(), suite.app, suite.ctx

	dividendAccount := types.DividendAccount{
		User:      "0x000000000000000000000000000000000000dEaD",
		FeeAmount: big.NewInt(0).String(),
	}
	err := heimdallApp.TopupKeeper.SetDividendAccount(ctx, dividendAccount)
	require.NoError(t, err)

	ok, _ := heimdallApp.TopupKeeper.HasDividendAccount(ctx, dividendAccount.User)
	require.Equal(t, ok, true)
}

func (suite *KeeperTestSuite) TestAddFeeToDividendAccount() {
	t, heimdallApp, ctx := suite.T(), suite.app, suite.ctx
	address := "0x000000000000000000000000000000000000dEaD"
	amount, _ := big.NewInt(0).SetString("0", 10)
	err := heimdallApp.TopupKeeper.AddFeeToDividendAccount(ctx, address, amount)
	require.NoError(t, err)

	dividentAccount, _ := heimdallApp.TopupKeeper.GetDividendAccount(ctx, address)
	actualResult, ok := big.NewInt(0).SetString(dividentAccount.FeeAmount, 10)
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
