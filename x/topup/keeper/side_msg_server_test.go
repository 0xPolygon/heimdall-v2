package keeper_test

import (
	"fmt"
	"math/big"
	"math/rand"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	hModule "github.com/0xPolygon/heimdall-v2/module"
	hTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func (suite *KeeperTestSuite) sideHandler(ctx sdk.Context, msg sdk.Msg) hModule.Vote {
	cfg := suite.sideMsgCfg
	return cfg.SideHandler(msg)(ctx, msg)
}

func (suite *KeeperTestSuite) postHandler(ctx sdk.Context, msg sdk.Msg, vote hModule.Vote) {
	cfg := suite.sideMsgCfg

	cfg.PostHandler(msg)(ctx, msg, vote)
}

func (suite *KeeperTestSuite) TestSideHandleTopupTx() {
	var msg types.MsgTopupTx

	heimdallApp, ctx := suite.app, suite.ctx
	chainParams := heimdallApp.ChainKeeper.GetParams(suite.ctx)

	_, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()

	testCases := []struct {
		msg       string
		malleate  func()
		expPass   bool
		expErrMsg string
		posttests func(res hModule.Vote)
	}{
		{
			"success",
			func() {
				suite.contractCaller = mocks.IContractCaller{}

				logIndex := uint64(10)
				blockNumber := uint64(599)
				txReceipt := &ethTypes.Receipt{
					BlockNumber: new(big.Int).SetUint64(blockNumber),
				}
				// TODO HV2: use the following line when implemented
				// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000000dead")
				txHash := "0x000000000000000000000000000000000000000000000000000000000000dead"
				hash := hTypes.TxHash{Hash: []byte(txHash)}

				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}

				// topup msg
				msg = *types.NewMsgTopupTx(
					addr1.String(),
					addr1.String(),
					coins.Amount,
					hash,
					logIndex,
					blockNumber,
				)

				// sequence id
				bn := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

				// mock external call
				event := &stakinginfo.StakinginfoTopUpFee{
					User: common.Address(sdk.AccAddress(addr1.String())),
					Fee:  coins.Amount.BigInt(),
				}
				suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
				suite.contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(event, nil)

			},
			true,
			"",
			func(res hModule.Vote) {
				blockNumber := uint64(599)
				bn := new(big.Int).SetUint64(blockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				suite.Require().Equal(res, hModule.Vote_VOTE_YES, "side tx handler should succeed")
				// there should be no stored event record
				ok, err := heimdallApp.TopupKeeper.HasTopupSequence(ctx, sequence.String())
				suite.Require().NoError(err)
				suite.Require().False(ok)
			},
		},
		{
			"no receipt",
			func() {
				suite.contractCaller = mocks.IContractCaller{}

				logIndex := uint64(10)
				blockNumber := uint64(599)
				// TODO HV2: use the following line when implemented
				// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000000dead")
				txHash := "0x000000000000000000000000000000000000000000000000000000000000dead"
				hash := hTypes.TxHash{Hash: []byte(txHash)}

				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}

				// topup msg
				msg = *types.NewMsgTopupTx(
					addr1.String(),
					addr1.String(),
					coins.Amount,
					hash,
					logIndex,
					blockNumber,
				)

				suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(nil, nil)
				suite.contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), nil, logIndex).Return(nil, nil)

			},
			true,
			"",
			func(res hModule.Vote) {
				suite.Require().Equal(res, hModule.Vote_VOTE_NO, "side tx handler should fail")
			},
		},
		{
			"no log",
			func() {
				suite.contractCaller = mocks.IContractCaller{}

				logIndex := uint64(10)
				blockNumber := uint64(599)
				txReceipt := &ethTypes.Receipt{
					BlockNumber: new(big.Int).SetUint64(blockNumber),
				}
				// TODO HV2: use the following line when implemented
				// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000000dead")
				txHash := "0x000000000000000000000000000000000000000000000000000000000000dead"
				hash := hTypes.TxHash{Hash: []byte(txHash)}

				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}

				// topup msg
				msg = *types.NewMsgTopupTx(
					addr1.String(),
					addr1.String(),
					coins.Amount,
					hash,
					logIndex,
					blockNumber,
				)

				suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
				suite.contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(nil, nil)

			},
			true,
			"",
			func(res hModule.Vote) {
				suite.Require().Equal(res, hModule.Vote_VOTE_NO, "side tx handler should fail")
			},
		},
		{
			"block mismatch",
			func() {
				suite.contractCaller = mocks.IContractCaller{}

				logIndex := uint64(10)
				blockNumber := uint64(599)
				txReceipt := &ethTypes.Receipt{
					BlockNumber: new(big.Int).SetUint64(blockNumber + 1),
				}
				// TODO HV2: use the following line when implemented
				// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000000dead")
				txHash := "0x000000000000000000000000000000000000000000000000000000000000dead"
				hash := hTypes.TxHash{Hash: []byte(txHash)}

				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}

				// topup msg
				msg = *types.NewMsgTopupTx(
					addr1.String(),
					addr1.String(),
					coins.Amount,
					hash,
					logIndex,
					blockNumber,
				)

				// mock external call
				event := &stakinginfo.StakinginfoTopUpFee{
					User: common.Address(sdk.AccAddress(addr1.String())),
					Fee:  coins.Amount.BigInt(),
				}

				suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
				suite.contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(event, nil)

			},
			true,
			"",
			func(res hModule.Vote) {
				suite.Require().Equal(res, hModule.Vote_VOTE_NO, "side tx handler should fail")
			},
		},
		{
			"user mismatch",
			func() {
				suite.contractCaller = mocks.IContractCaller{}

				logIndex := uint64(10)
				blockNumber := uint64(599)
				txReceipt := &ethTypes.Receipt{
					BlockNumber: new(big.Int).SetUint64(blockNumber),
				}
				// TODO HV2: use the following line when implemented
				// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000000dead")
				txHash := "0x000000000000000000000000000000000000000000000000000000000000dead"
				hash := hTypes.TxHash{Hash: []byte(txHash)}

				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}

				// topup msg
				msg = *types.NewMsgTopupTx(
					addr1.String(),
					addr1.String(),
					coins.Amount,
					hash,
					logIndex,
					blockNumber,
				)

				// mock external call
				event := &stakinginfo.StakinginfoTopUpFee{
					User: common.Address(sdk.AccAddress(addr2.String())),
					Fee:  coins.Amount.BigInt(),
				}

				suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
				suite.contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(event, nil)

			},
			true,
			"",
			func(res hModule.Vote) {
				suite.Require().Equal(res, hModule.Vote_VOTE_NO, "side tx handler should fail")
			},
		},
		{
			"fee mismatch",
			func() {
				suite.contractCaller = mocks.IContractCaller{}

				logIndex := uint64(10)
				blockNumber := uint64(599)
				txReceipt := &ethTypes.Receipt{
					BlockNumber: new(big.Int).SetUint64(blockNumber),
				}
				// TODO HV2: use the following line when implemented
				// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000000dead")
				txHash := "0x000000000000000000000000000000000000000000000000000000000000dead"
				hash := hTypes.TxHash{Hash: []byte(txHash)}

				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}

				// topup msg
				msg = *types.NewMsgTopupTx(
					addr1.String(),
					addr1.String(),
					coins.Amount,
					hash,
					logIndex,
					blockNumber,
				)

				// mock external call
				event := &stakinginfo.StakinginfoTopUpFee{
					User: common.Address(sdk.AccAddress(addr2.String())),
					Fee:  new(big.Int).SetUint64(1),
				}

				suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
				suite.contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(event, nil)

			},
			true,
			"",
			func(res hModule.Vote) {
				suite.Require().Equal(res, hModule.Vote_VOTE_NO, "side tx handler should fail")
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()

			tc.malleate()
			res := suite.sideHandler(ctx, &msg)

			tc.posttests(res)
		})
	}
}

func (suite *KeeperTestSuite) TestPostHandleTopupTx() {
	var msg types.MsgTopupTx

	heimdallApp, ctx := suite.app, suite.ctx

	_, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()
	_, _, addr3 := testdata.KeyTestPubAddr()

	logIndex := rand.Uint64()
	blockNumber := rand.Uint64()
	// TODO HV2: use the following line when implemented
	// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000000dead")
	txHash := "0x000000000000000000000000000000000000000000000000000000000000dead"
	hash := hTypes.TxHash{Hash: []byte(txHash)}

	testCases := []struct {
		msg       string
		malleate  func()
		expPass   bool
		expErrMsg string
		posttests func(res hModule.Vote)
	}{
		{
			"no result",
			func() {

				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}

				// topup msg
				msg = *types.NewMsgTopupTx(
					addr1.String(),
					addr1.String(),
					coins.Amount,
					hash,
					logIndex,
					blockNumber,
				)

				// sequence id
				bn := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
			},
			true,
			"",
			func(res hModule.Vote) {
				suite.Require().Equal(res, hModule.Vote_VOTE_NO, "post tx handler should fail")
				// there should be no stored event record
				bn := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				ok, err := heimdallApp.TopupKeeper.HasTopupSequence(ctx, sequence.String())
				suite.Require().NoError(err)
				suite.Require().False(ok)
			},
		},
		{
			"yes result",
			func() {

				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}

				// topup msg
				msg = *types.NewMsgTopupTx(
					addr1.String(),
					addr1.String(),
					coins.Amount,
					hash,
					logIndex,
					blockNumber,
				)

				// sequence id
				bn := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
			},
			true,
			"",
			func(res hModule.Vote) {
				suite.Require().Equal(res, hModule.Vote_VOTE_YES, "post tx handler should succeed")
				// there should be no stored event record
				bn := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				ok, err := heimdallApp.TopupKeeper.HasTopupSequence(ctx, sequence.String())
				suite.Require().NoError(err)
				suite.Require().False(ok)
				// account coins should be empty
				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}
				acc1 := heimdallApp.AccountKeeper.GetAccount(ctx, addr1)
				suite.Require().NotNil(acc1)
				coins1 := heimdallApp.BankKeeper.GetBalance(ctx, acc1.GetAddress(), authTypes.FeeToken)
				suite.Require().False(coins1.IsZero())
				suite.Require().True(coins1.Equal(coins))
			},
		},
		{
			"with proposer",
			func() {

				logIndex := rand.Uint64()
				blockNumber := rand.Uint64()
				// TODO HV2: use the following line when implemented
				// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000001dead")
				txHash := "0x000000000000000000000000000000000000000000000000000000000001dead"
				hash := hTypes.TxHash{Hash: []byte(txHash)}

				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}

				// topup msg
				msg = *types.NewMsgTopupTx(
					addr2.String(),
					addr3.String(),
					coins.Amount,
					hash,
					logIndex,
					blockNumber,
				)

				// check if incoming tx is older
				bn := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

			},
			true,
			"",
			func(res hModule.Vote) {
				suite.Require().Equal(res, hModule.Vote_VOTE_YES, "side tx handler should succeed")
				// there should be stored sequence
				// check if incoming tx is older
				bn := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
				ok, err := heimdallApp.TopupKeeper.HasTopupSequence(ctx, sequence.String())
				suite.Require().NoError(err)
				suite.Require().True(ok)

				// account coins should not be empty
				acc2 := heimdallApp.AccountKeeper.GetAccount(ctx, addr2)
				suite.Require().NotNil(acc2)
				coins2 := heimdallApp.BankKeeper.GetBalance(ctx, acc2.GetAddress(), authTypes.FeeToken)
				suite.Require().False(coins2.IsZero())
				acc3 := heimdallApp.AccountKeeper.GetAccount(ctx, addr3)
				suite.Require().NotNil(acc3)
				coins3 := heimdallApp.BankKeeper.GetBalance(ctx, acc3.GetAddress(), authTypes.FeeToken)
				suite.Require().False(coins3.IsZero())

				// check coins = acc1.coins + acc2.coins
				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}
				suite.Require().True(coins.Equal(coins3.Add(coins2)))
			},
		},
		{
			"replay",
			func() {

				logIndex := rand.Uint64()
				blockNumber := rand.Uint64()
				// TODO HV2: use the following line when implemented
				// hash := hTypes.HexToHeimdallHash("0x000000000000000000000000000000000000000000000000000000000002dead")
				txHash := "0x000000000000000000000000000000000000000000000000000000000002dead"
				hash := hTypes.TxHash{Hash: []byte(txHash)}

				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}

				// topup msg
				msg = *types.NewMsgTopupTx(
					addr1.String(),
					addr1.String(),
					coins.Amount,
					hash,
					logIndex,
					blockNumber,
				)

			},
			true,
			"",
			func(res hModule.Vote) {
				suite.Require().Equal(res, hModule.Vote_VOTE_YES, "side tx handler should succeed")
				bn := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
				ok, err := heimdallApp.TopupKeeper.HasTopupSequence(ctx, sequence.String())
				suite.Require().NoError(err)
				suite.Require().True(ok)
				replayRes := suite.sideHandler(ctx, &msg)
				suite.Require().Equal(replayRes, hModule.Vote_VOTE_NO, "side tx handler should fail")
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()

			tc.malleate()
			res := suite.sideHandler(ctx, &msg)

			/* TODO HV2: can we generalize here with the following and reduce the load in posttests(res)?
			if tc.expPass {
				suite.Require().Equal(res, hmVote.VOTE_YES, "side tx handler should succeed")
			} else {
				suite.Require().Equal(res, hmVote.VOTE_NO, "side tx handler should fail")
			}
			*/

			tc.posttests(res)
		})
	}
}
