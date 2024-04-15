package keeper_test

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	hTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func (suite *KeeperTestSuite) TestCreateTopupTx() {
	msgServer, require, keeper, ctx, t := suite.msgServer, suite.Require(), suite.keeper, suite.ctx, suite.T()

	var msg types.MsgTopupTx

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	hash := hTypes.TxHash{Hash: []byte(TxHash)}
	logIndex := r1.Uint64()
	blockNumber := r1.Uint64()

	_, _, addr := testdata.KeyTestPubAddr()
	fee := math.NewInt(100000000000000000)

	t.Run("success", func(t *testing.T) {
		msg = *types.NewMsgTopupTx(addr.String(), addr.String(), fee, hash, logIndex, blockNumber)
		res, err := msgServer.CreateTopupTx(ctx, &msg)
		require.NoError(err)
		require.NotNil(res)
	})

	t.Run("old tx", func(t *testing.T) {
		msg = *types.NewMsgTopupTx(addr.String(), addr.String(), fee, hash, logIndex, blockNumber)
		blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
		sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
		sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
		err := keeper.SetTopupSequence(ctx, sequence.String())
		require.NoError(err)
		_, err = msgServer.CreateTopupTx(ctx, &msg)
		require.Error(err)
		require.Contains(err.Error(), "already exists")
	})
}

func (suite *KeeperTestSuite) TestWithdrawFeeTx() {
	msgServer, require, keeper, accountKeeper, ctx, t := suite.msgServer, suite.Require(), suite.keeper, suite.accountKeeper, suite.ctx, suite.T()

	var msg types.MsgWithdrawFeeTx

	_, _, addr := testdata.KeyTestPubAddr()

	t.Run("fail with no fee coins", func(t *testing.T) {
		msg = *types.NewMsgWithdrawFeeTx(addr.String(), math.ZeroInt())
		_, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.Error(err)
		require.Contains(err.Error(), "insufficient funds")
	})

	t.Run("success full amount", func(t *testing.T) {
		// TODO HV2: fix this test

		// TODO HV2: replace the following lines with `coins := simulation.RandomFeeCoins()` when simulation types are implemented
		base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
		amount := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
		coins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amount)}}
		msg = *types.NewMsgWithdrawFeeTx(addr.String(), math.ZeroInt())

		// fund account from module
		account := accountKeeper.NewAccountWithAddress(ctx, addr)
		// TODO HV2: is this the right way to set coins for account? Will the topup module have funds?
		err := keeper.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, account.GetAddress(), coins)
		require.NoError(err)
		accountKeeper.SetAccount(ctx, account)
		// check coins are set
		require.True(keeper.BankKeeper.GetBalance(ctx, account.GetAddress(), authTypes.FeeToken).Amount.GT(math.ZeroInt()))

		res, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.NoError(err)
		require.NotNil(res)

		// check zero balance for account
		acc := accountKeeper.GetAccount(ctx, addr)
		require.True(keeper.BankKeeper.GetBalance(ctx, acc.GetAddress(), authTypes.FeeToken).IsZero())
	})

	t.Run("success partial amount", func(t *testing.T) {
		// TODO HV2: fix this test

		// TODO HV2: replace the following lines with `coins := simulation.RandomFeeCoins()` when simulation types are implemented
		base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
		amount := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
		coins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amount)}}

		// TODO HV2: check `setupGovKeeper` and `trackMockBalances` in cosmos-sdk to check how to set/track balances for accounts

		// fund account from module
		account := accountKeeper.NewAccountWithAddress(ctx, addr)
		// TODO HV2: is this the right way to set coins for account? Will the topup module have funds?
		err := keeper.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, account.GetAddress(), coins)
		require.NoError(err)
		accountKeeper.SetAccount(ctx, account)
		// check coins are set
		require.True(keeper.BankKeeper.GetBalance(ctx, account.GetAddress(), authTypes.FeeToken).Amount.GT(math.ZeroInt()))

		amt, _ := math.NewIntFromString("2")
		coins = coins.Sub(sdk.Coin{Denom: authTypes.FeeToken, Amount: amt})
		msg = *types.NewMsgWithdrawFeeTx(addr.String(), coins.AmountOf(authTypes.FeeToken))

		res, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.NoError(err)
		require.NotNil(res)

		acc := accountKeeper.GetAccount(ctx, addr)
		require.True(keeper.BankKeeper.GetBalance(ctx, acc.GetAddress(), authTypes.FeeToken).Amount.Equal(amt))
	})

	t.Run("fail with not enough amount", func(t *testing.T) {
		// TODO HV2: replace the following lines with `coins := simulation.RandomFeeCoins()` when simulation types are implemented
		base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
		amount := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
		coins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amount)}}

		// fund account from module
		account := accountKeeper.NewAccountWithAddress(ctx, addr)
		// TODO HV2: is this the right way to set coins for account? Will the topup module have funds?
		err := keeper.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, account.GetAddress(), coins)
		require.NoError(err)
		accountKeeper.SetAccount(ctx, account)
		// check coins are set
		require.True(keeper.BankKeeper.GetBalance(ctx, account.GetAddress(), authTypes.FeeToken).Amount.GT(math.ZeroInt()))

		amt, _ := math.NewIntFromString("1")
		coins = coins.Add(sdk.Coin{Denom: authTypes.FeeToken, Amount: amt})
		msg = *types.NewMsgWithdrawFeeTx(addr.String(), coins.AmountOf(authTypes.FeeToken))

		_, err = msgServer.WithdrawFeeTx(ctx, &msg)
		require.Error(err)
		require.Contains(err.Error(), "not enough balance")
	})
}
