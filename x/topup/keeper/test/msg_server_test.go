package test

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

	var msg types.MsgTopupTx

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	// TODO HV2: use the following line when implemented
	// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000000dead")
	txHash := "0x000000000000000000000000000000000000000000000000000000000000dead"
	hash := hTypes.TxHash{Hash: []byte(txHash)}
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
				msg = types.MsgTopupTx{
					Proposer:    addr.String(),
					User:        addr.String(),
					Fee:         fee,
					TxHash:      hash,
					LogIndex:    logIndex,
					BlockNumber: blockNumber,
				}
			},
			true,
			"",
			func() {
			},
		},
		{
			"old tx",
			func() {
				msg = types.MsgTopupTx{
					Proposer:    addr.String(),
					User:        addr.String(),
					Fee:         fee,
					TxHash:      hash,
					LogIndex:    logIndex,
					BlockNumber: blockNumber,
				}
				blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(blockNumber, big.NewInt(types.DefaultLogIndexUnit))
				sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
				err := suite.app.TopupKeeper.SetTopupSequence(suite.ctx, sequence.String())
				suite.Require().NoError(err)
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
			_, err := msgServer.CreateTopupTx(suite.ctx, &msg)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
			}

			tc.posttests()
		})
	}
}

func (suite *KeeperTestSuite) TestWithdrawFeeTx() {
	msgServer := suite.msgServer

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
				msg = types.MsgWithdrawFeeTx{
					Proposer: addr.String(),
					Amount:   math.ZeroInt(),
				}
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
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amount := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amount)}}

				// fund account from module
				account := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr)
				// TODO HV2: is this the right way to set coins for account? Will the topup module have funds?
				err := suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, account.GetAddress(), coins)
				suite.Require().NoError(err)
				suite.app.AccountKeeper.SetAccount(suite.ctx, account)
				// check coins are set
				suite.Require().True(suite.app.BankKeeper.GetBalance(suite.ctx, account.GetAddress(), authTypes.FeeToken).Amount.GT(math.ZeroInt()))
			},
			true,
			"",
			func() {
				// check zero balance for account
				account := suite.app.AccountKeeper.GetAccount(suite.ctx, addr)
				suite.Require().True(suite.app.BankKeeper.GetBalance(suite.ctx, account.GetAddress(), authTypes.FeeToken).IsZero())
			},
		},
		{
			"success partial amount",
			func() {
				// TODO HV2: replace the following lines with `coins := simulation.RandomFeeCoins()` when simulation types are implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amount := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amount)}}

				// fund account from module
				account := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr)
				// TODO HV2: is this the right way to set coins for account? Will the topup module have funds?
				err := suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, account.GetAddress(), coins)
				suite.Require().NoError(err)
				suite.app.AccountKeeper.SetAccount(suite.ctx, account)
				// check coins are set
				suite.Require().True(suite.app.BankKeeper.GetBalance(suite.ctx, account.GetAddress(), authTypes.FeeToken).Amount.GT(math.ZeroInt()))

				amt, _ := math.NewIntFromString("2")
				coins = coins.Sub(sdk.Coin{Denom: authTypes.FeeToken, Amount: amt})
				msg = types.MsgWithdrawFeeTx{
					Proposer: addr.String(),
					Amount:   coins.AmountOf(authTypes.FeeToken),
				}
			},
			true,
			"",
			func() {
				amt, _ := math.NewIntFromString("2")
				account := suite.app.AccountKeeper.GetAccount(suite.ctx, addr)
				suite.Require().True(suite.app.BankKeeper.GetBalance(suite.ctx, account.GetAddress(), authTypes.FeeToken).Amount.Equal(amt))
			},
		},
		{
			"amount not enough",
			func() {
				// TODO HV2: replace the following lines with `coins := simulation.RandomFeeCoins()` when simulation types are implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amount := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amount)}}

				// fund account from module
				account := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr)
				// TODO HV2: is this the right way to set coins for account? Will the topup module have funds?
				err := suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, account.GetAddress(), coins)
				suite.Require().NoError(err)
				suite.app.AccountKeeper.SetAccount(suite.ctx, account)
				// check coins are set
				suite.Require().True(suite.app.BankKeeper.GetBalance(suite.ctx, account.GetAddress(), authTypes.FeeToken).Amount.GT(math.ZeroInt()))

				amt, _ := math.NewIntFromString("1")
				coins = coins.Add(sdk.Coin{Denom: authTypes.FeeToken, Amount: amt})
				msg = types.MsgWithdrawFeeTx{
					Proposer: addr.String(),
					Amount:   coins.AmountOf(authTypes.FeeToken),
				}
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
			_, err := msgServer.WithdrawFeeTx(suite.ctx, &msg)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expErrMsg)
			}

			tc.posttests()
		})
	}
}
