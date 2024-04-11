package keeper_test

import (
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	hTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func (suite *KeeperTestSuite) TestCreateTopupTx() {
	msgServer := suite.msgServer
	require := suite.Require()
	keeper := suite.keeper
	ctx := suite.ctx

	var msg types.MsgTopupTx

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	hash := hTypes.TxHash{Hash: []byte(TxHash)}
	logIndex := r1.Uint64()
	blockNumber := r1.Uint64()

	_, _, addr := testdata.KeyTestPubAddr()
	fee := math.NewInt(100000000000000000)

	testCases := []struct {
		msg       string
		malleate  func()
		expPass   bool
		expErrMsg string
		posttests func()
	}{
		{
			"success",
			func() {
				msg = *types.NewMsgTopupTx(addr.String(), addr.String(), fee, hash, logIndex, blockNumber)
			},
			true,
			"",
			func() {
			},
		},
		{
			"old tx",
			func() {
				msg = *types.NewMsgTopupTx(addr.String(), addr.String(), fee, hash, logIndex, blockNumber)
				blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
				sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
				err := keeper.SetTopupSequence(ctx, sequence.String())
				require.NoError(err)
			},
			false,
			"already exists",
			func() {
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()

			tc.malleate()
			_, err := msgServer.CreateTopupTx(ctx, &msg)

			if tc.expPass {
				require.NoError(err)
			} else {
				require.Error(err)
				require.Contains(err.Error(), tc.expErrMsg)
			}

			tc.posttests()
		})
	}
}

func (suite *KeeperTestSuite) TestWithdrawFeeTx() {
	msgServer := suite.msgServer
	ctx := suite.ctx
	require := suite.Require()

	var msg types.MsgWithdrawFeeTx

	_, _, addr := testdata.KeyTestPubAddr()

	testCases := []struct {
		msg       string
		malleate  func()
		expPass   bool
		expErrMsg string
		posttests func()
	}{
		{
			"fail with no fee coins",
			func() {
				msg = *types.NewMsgWithdrawFeeTx(addr.String(), math.ZeroInt())
			},
			false,
			"no balance to withdraw",
			func() {
			},
		},
		{
			"success full amount",
			func() {
				// TODO HV2: replace the following lines with `coins := simulation.RandomFeeCoins()` when simulation types are implemented
				//base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				//amount := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				//coins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amount)}}

				// TODO HV2: enable/edit the following once the issue with expected calls on BankKeeper is solved
				//  Also check `setupGovKeeper` and `trackMockBalances` in cosmos-sdk to check how to set/track balances for accounts

				// fund account from module
				//account := accountKeeper.NewAccountWithAddress(ctx, addr)
				// TODO HV2: is this the right way to set coins for account? Will the topup module have funds?
				//err := BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, account.GetAddress(), coins)
				//require.NoError(err)
				//accountKeeper.SetAccount(ctx, account)
				// check coins are set
				//require.True(BankKeeper.GetBalance(ctx, account.GetAddress(), authTypes.FeeToken).Amount.GT(math.ZeroInt()))
			},
			true,
			"",
			func() {
				// check zero balance for account
				//account := accountKeeper.GetAccount(ctx, addr)
				//require.True(BankKeeper.GetBalance(ctx, account.GetAddress(), authTypes.FeeToken).IsZero())
			},
		},
		{
			"success partial amount",
			func() {
				// TODO HV2: replace the following lines with `coins := simulation.RandomFeeCoins()` when simulation types are implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amount := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amount)}}

				// TODO HV2: enable/edit the following once the issue with expected calls on BankKeeper is solved
				//  Also check `setupGovKeeper` and `trackMockBalances` in cosmos-sdk to check how to set/track balances for accounts

				// fund account from module
				//account := accountKeeper.NewAccountWithAddress(ctx, addr)
				// TODO HV2: is this the right way to set coins for account? Will the topup module have funds?
				//err := BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, account.GetAddress(), coins)
				//require.NoError(err)
				//accountKeeper.SetAccount(ctx, account)
				// check coins are set
				//require.True(BankKeeper.GetBalance(ctx, account.GetAddress(), authTypes.FeeToken).Amount.GT(math.ZeroInt()))

				amt, _ := math.NewIntFromString("2")
				coins = coins.Sub(sdk.Coin{Denom: authTypes.FeeToken, Amount: amt})
				msg = *types.NewMsgWithdrawFeeTx(addr.String(), coins.AmountOf(authTypes.FeeToken))
			},
			true,
			"",
			func() {

				// TODO HV2: enable/edit the following once the issue with expected calls on BankKeeper is solved
				//  Also check `setupGovKeeper` and `trackMockBalances` in cosmos-sdk to check how to set/track balances for accounts

				//amt, _ := math.NewIntFromString("2")
				//account := accountKeeper.GetAccount(ctx, addr)
				//require.True(BankKeeper.GetBalance(ctx, account.GetAddress(), authTypes.FeeToken).Amount.Equal(amt))
			},
		},
		{
			"amount not enough",
			func() {
				// TODO HV2: replace the following lines with `coins := simulation.RandomFeeCoins()` when simulation types are implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amount := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amount)}}

				// TODO HV2: enable/edit the following once the issue with expected calls on BankKeeper is solved
				//  Also check `setupGovKeeper` and `trackMockBalances` in cosmos-sdk to check how to set/track balances for accounts

				// fund account from module
				//account := accountKeeper.NewAccountWithAddress(ctx, addr)
				// TODO HV2: is this the right way to set coins for account? Will the topup module have funds?
				//err := BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, account.GetAddress(), coins)
				//require.NoError(err)
				//accountKeeper.SetAccount(ctx, account)
				// check coins are set
				//require.True(BankKeeper.GetBalance(ctx, account.GetAddress(), authTypes.FeeToken).Amount.GT(math.ZeroInt()))

				amt, _ := math.NewIntFromString("1")
				coins = coins.Add(sdk.Coin{Denom: authTypes.FeeToken, Amount: amt})
				msg = *types.NewMsgWithdrawFeeTx(addr.String(), coins.AmountOf(authTypes.FeeToken))
			},
			false,
			"insufficient funds",
			func() {
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()

			tc.malleate()
			_, err := msgServer.WithdrawFeeTx(ctx, &msg)

			if tc.expPass {
				require.NoError(err)
			} else {
				require.Error(err)
				require.Contains(err.Error(), tc.expErrMsg)
			}

			tc.posttests()
		})
	}
}
