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
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/types"
	heimdallTypes "github.com/0xPolygon/heimdall-v2/types"
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

	msgServer      topupTypes.MsgServer
	sideMsgCfg     sidetxs.SideTxConfigurator
	queryClient    topupTypes.QueryClient
	contractCaller mocks.IContractCaller
}

func TestKeeperTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) SetupTest() {
	key := storetypes.NewKVStoreKey(topupTypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)

	testCtx := cosmostestutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: cmttime.Now()})
	encCfg := moduletestutil.MakeTestEncodingConfig()

	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	bankKeeper := testutil.NewMockBankKeeper(ctrl)
	chainKeeper := testutil.NewMockChainKeeper(ctrl)

	s.contractCaller = mocks.IContractCaller{}

	keeper := topupKeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		bankKeeper,
		chainKeeper,
		&s.contractCaller,
	)

	topupGenesis := topupTypes.DefaultGenesisState()
	keeper.InitGenesis(ctx, topupGenesis)
	topupTypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	queryHelper := baseapp.NewQueryServerTestHelper(ctx, encCfg.InterfaceRegistry)
	topupTypes.RegisterQueryServer(queryHelper, topupKeeper.NewQueryServer(&keeper))

	s.ctx = ctx
	s.keeper = keeper
	s.queryClient = topupTypes.NewQueryClient(queryHelper)
	s.msgServer = topupKeeper.NewMsgServerImpl(&keeper)
	s.sideMsgCfg = sidetxs.NewSideTxConfigurator()

	topupTypes.RegisterSideMsgServer(s.sideMsgCfg, topupKeeper.NewSideMsgServerImpl(&keeper))
}

func (s *KeeperTestSuite) TestTopupSequenceSet() {
	ctx, tk, require := s.ctx, s.keeper, s.Require()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	topupSequence := strconv.Itoa(simulation.RandIntBetween(r, 1000, 100000))
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

func (s *KeeperTestSuite) TestDividendAccount() {
	ctx, tk, require := s.ctx, s.keeper, s.Require()

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

func (s *KeeperTestSuite) TestAddFeeToDividendAccount() {
	ctx, tk, require := s.ctx, s.keeper, s.Require()

	amount, _ := big.NewInt(0).SetString("0", 10)
	err := tk.AddFeeToDividendAccount(ctx, AccountHash, amount)
	require.NoError(err)

	dividendAccount, _ := tk.GetDividendAccount(ctx, AccountHash)
	actualResult, ok := big.NewInt(0).SetString(dividendAccount.FeeAmount, 10)
	require.Equal(ok, true)
	require.Equal(amount, actualResult)
}

func (s *KeeperTestSuite) TestDividendAccountTree() {
	require := s.Require()
	codec := address.NewHexCodec()

	divAccounts := make([]types.DividendAccount, 5)
	for i := 0; i < len(divAccounts); i++ {
		_, err := codec.StringToBytes(AccountHash)
		require.NoError(err)
		divAccounts[i] = types.DividendAccount{
			User:      AccountHash,
			FeeAmount: big.NewInt(0).String(),
		}
	}

	accountRoot, err := heimdallTypes.GetAccountRootHash(divAccounts)
	require.NotNil(accountRoot)
	require.NoError(err)

	accountProof, index, err := heimdallTypes.GetAccountProof(divAccounts, AccountHash)
	require.NotNil(accountProof)
	require.NotNil(index)
	require.NoError(err)

	leafHash := CalculateDividendAccountHash(divAccounts[0])
	require.NotNil(leafHash)
}

// CalculateDividendAccountHash is a helper function to hash the values of a DividendAccount
func CalculateDividendAccountHash(da types.DividendAccount) []byte {
	fee, _ := big.NewInt(0).SetString(da.FeeAmount, 10)
	divAccountHash := crypto.Keccak256(AppendBytes32([]byte(da.User), fee.Bytes()))

	return divAccountHash
}

func AppendBytes32(data ...[]byte) []byte {
	var result []byte

	for _, v := range data {
		paddedV := convertTo32(v)
		result = append(result, paddedV[:]...)
	}

	return result
}

func convertTo32(input []byte) (output [32]byte) {
	l := len(input)
	if l > 32 || l == 0 {
		return
	}
	copy(output[32-l:], input[:])

	return output
}
