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
	mod "github.com/0xPolygon/heimdall-v2/module"
	hTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func (suite *KeeperTestSuite) sideHandler(ctx sdk.Context, msg sdk.Msg) mod.Vote {
	cfg := suite.sideMsgCfg
	return cfg.SideHandler(msg)(ctx, msg)
}

func (suite *KeeperTestSuite) postHandler(ctx sdk.Context, msg sdk.Msg, vote mod.Vote) {
	cfg := suite.sideMsgCfg
	cfg.PostHandler(msg)(ctx, msg, vote)
}

func (suite *KeeperTestSuite) TestSideHandleTopupTx() {
	var msg types.MsgTopupTx

	ctx, keeper, require := suite.ctx, suite.keeper, suite.Require()
	// TODO HV2: enable when contractCaller is implemented
	// contractCaller := suite.contractCaller

	// TODO HV2: enable when chainmanager is implemented
	// chainParams := heimdallApp.ChainKeeper.GetParams(suite.ctx)

	_, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()

	testCases := []struct {
		msg       string
		malleate  func()
		expPass   bool
		expErrMsg string
		posttests func(res mod.Vote)
	}{
		{
			"success",
			func() {
				// TODO HV2: enable when contractCaller is implemented
				// contractCaller = mocks.IContractCaller{}

				logIndex := uint64(10)
				blockNumber := uint64(599)
				// TODO HV2: replace _ with txReceipt when implemented
				_ = &ethTypes.Receipt{
					BlockNumber: new(big.Int).SetUint64(blockNumber),
				}
				hash := hTypes.TxHash{Hash: []byte(TxHash)}

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
				// TODO HV2: replace _ with event when contractCaller implemented
				_ = &stakinginfo.StakinginfoTopUpFee{
					User: common.Address(sdk.AccAddress(addr1.String())),
					Fee:  coins.Amount.BigInt(),
				}
				// TODO HV2: enable when contractCaller is implemented
				// contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
				// contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(event, nil)

			},
			true,
			"",
			func(res mod.Vote) {
				blockNumber := uint64(599)
				bn := new(big.Int).SetUint64(blockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				// TODO HV2: enable this when side_msg_server code is fully functional (atm mod.Vote_VOTE_NO is hardcoded due to missing code)
				// require.Equal(res, mod.Vote_VOTE_YES, "side tx handler should succeed")
				// there should be no stored event record
				ok, err := keeper.HasTopupSequence(ctx, sequence.String())
				require.NoError(err)
				require.False(ok)
			},
		},
		{
			"no receipt",
			func() {
				// contractCaller = mocks.IContractCaller{}

				logIndex := uint64(10)
				blockNumber := uint64(599)
				hash := hTypes.TxHash{Hash: []byte(TxHash)}

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
				// TODO HV2: enable when contractCaller is implemented
				// contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(nil, nil)
				// contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), nil, logIndex).Return(nil, nil)

			},
			true,
			"",
			func(res mod.Vote) {
				require.Equal(res, mod.Vote_VOTE_NO, "side tx handler should fail")
			},
		},
		{
			"no log",
			func() {
				// TODO HV2: enable when contractCaller is implemented
				// contractCaller = mocks.IContractCaller{}

				logIndex := uint64(10)
				blockNumber := uint64(599)
				// TODO HV2: replace _ with txReceipt when implemented
				_ = &ethTypes.Receipt{
					BlockNumber: new(big.Int).SetUint64(blockNumber),
				}
				hash := hTypes.TxHash{Hash: []byte(TxHash)}

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
				// TODO HV2: enable when contractCaller is implemented
				// contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
				// contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(nil, nil)

			},
			true,
			"",
			func(res mod.Vote) {
				require.Equal(res, mod.Vote_VOTE_NO, "side tx handler should fail")
			},
		},
		{
			"block mismatch",
			func() {
				// TODO HV2: enable when contractCaller is implemented
				// contractCaller = mocks.IContractCaller{}

				logIndex := uint64(10)
				blockNumber := uint64(599)
				// TODO HV2: replace _ with txReceipt when implemented
				_ = &ethTypes.Receipt{
					BlockNumber: new(big.Int).SetUint64(blockNumber + 1),
				}
				hash := hTypes.TxHash{Hash: []byte(TxHash)}

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

				// TODO HV2: replace _ with event when implemented
				_ = &stakinginfo.StakinginfoTopUpFee{
					User: common.Address(sdk.AccAddress(addr1.String())),
					Fee:  coins.Amount.BigInt(),
				}

				// TODO HV2: enable when contractCaller is implemented
				// contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
				// contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(event, nil)

			},
			true,
			"",
			func(res mod.Vote) {
				require.Equal(res, mod.Vote_VOTE_NO, "side tx handler should fail")
			},
		},
		{
			"user mismatch",
			func() {
				// TODO HV2: enable when contractCaller is implemented
				// contractCaller = mocks.IContractCaller{}

				logIndex := uint64(10)
				blockNumber := uint64(599)
				// TODO HV2: replace _ with txReceipt when implemented
				_ = &ethTypes.Receipt{
					BlockNumber: new(big.Int).SetUint64(blockNumber),
				}
				hash := hTypes.TxHash{Hash: []byte(TxHash)}

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

				// TODO HV2: replace _ with event when implemented
				_ = &stakinginfo.StakinginfoTopUpFee{
					User: common.Address(sdk.AccAddress(addr2.String())),
					Fee:  coins.Amount.BigInt(),
				}

				// TODO HV2: enable when contractCaller is implemented
				// contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
				// contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(event, nil)

			},
			true,
			"",
			func(res mod.Vote) {
				require.Equal(res, mod.Vote_VOTE_NO, "side tx handler should fail")
			},
		},
		{
			"fee mismatch",
			func() {
				// TODO HV2: enable when contractCaller is implemented
				// contractCaller = mocks.IContractCaller{}

				logIndex := uint64(10)
				blockNumber := uint64(599)
				// TODO HV2: replace _ with txReceipt when implemented
				_ = &ethTypes.Receipt{
					BlockNumber: new(big.Int).SetUint64(blockNumber),
				}
				hash := hTypes.TxHash{Hash: []byte(TxHash)}

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
				// TODO HV2: replace _ with event when implemented
				_ = &stakinginfo.StakinginfoTopUpFee{
					User: common.Address(sdk.AccAddress(addr2.String())),
					Fee:  new(big.Int).SetUint64(1),
				}

				// TODO HV2: enable when contractCaller is implemented
				// contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
				// contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(event, nil)

			},
			true,
			"",
			func(res mod.Vote) {
				require.Equal(res, mod.Vote_VOTE_NO, "side tx handler should fail")
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {

			tc.malleate()
			res := suite.sideHandler(ctx, &msg)

			tc.posttests(res)
		})
	}
}

func (suite *KeeperTestSuite) TestPostHandleTopupTx() {
	var msg types.MsgTopupTx

	ctx, require, keeper, accountKeeper := suite.ctx, suite.Require(), suite.keeper, suite.accountKeeper
	// TODO HV2: enable when contractCaller is implemented
	// contractCaller := suite.contractCaller

	_, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()
	_, _, addr3 := testdata.KeyTestPubAddr()

	logIndex := rand.Uint64()
	blockNumber := rand.Uint64()
	hash := hTypes.TxHash{Hash: []byte(TxHash)}

	testCases := []struct {
		msg       string
		malleate  func()
		expPass   bool
		expErrMsg string
		posttests func(res mod.Vote)
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
			func(res mod.Vote) {
				require.Equal(res, mod.Vote_VOTE_NO, "post tx handler should fail")
				// there should be no stored event record
				bn := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				ok, err := keeper.HasTopupSequence(ctx, sequence.String())
				require.NoError(err)
				require.False(ok)
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
			func(res mod.Vote) {
				// TODO HV2: enable this when side_msg_server code is all fully functional (atm mod.Vote_VOTE_NO is hardcoded due to missing code)
				// require.Equal(res, mod.Vote_VOTE_YES, "post tx handler should succeed")
				// there should be no stored event record
				bn := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				ok, err := keeper.HasTopupSequence(ctx, sequence.String())
				require.NoError(err)
				require.False(ok)

				// TODO HV2: enable/edit the following once the issue with expected calls on BankKeeper is solved

				// account coins should be empty
				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}
				acc1 := accountKeeper.GetAccount(ctx, addr1)
				require.NotNil(acc1)
				coins1 := keeper.BankKeeper.GetBalance(ctx, acc1.GetAddress(), authTypes.FeeToken)
				require.False(coins1.IsZero())
				require.True(coins1.Equal(coins))
			},
		},
		{
			"with proposer",
			func() {

				logIndex := rand.Uint64()
				blockNumber := rand.Uint64()
				// TODO HV2: use the following line when implemented?
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
			func(res mod.Vote) {
				// TODO HV2: enable this when side_msg_server code is all fully functional (atm mod.Vote_VOTE_NO is hardcoded due to missing code)
				// require.Equal(res, mod.Vote_VOTE_YES, "side tx handler should succeed")
				// there should be stored sequence
				// check if incoming tx is older
				bn := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
				ok, err := keeper.HasTopupSequence(ctx, sequence.String())
				require.NoError(err)
				require.True(ok)

				// account coins should not be empty
				acc2 := accountKeeper.GetAccount(ctx, addr2)
				require.NotNil(acc2)
				coins2 := keeper.BankKeeper.GetBalance(ctx, acc2.GetAddress(), authTypes.FeeToken)
				require.False(coins2.IsZero())
				acc3 := accountKeeper.GetAccount(ctx, addr3)
				require.NotNil(acc3)
				coins3 := keeper.BankKeeper.GetBalance(ctx, acc3.GetAddress(), authTypes.FeeToken)
				require.False(coins3.IsZero())

				// check coins = acc1.coins + acc2.coins
				// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
				base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
				amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
				coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}
				require.True(coins.Equal(coins3.Add(coins2)))
			},
		},
		{
			"replay",
			func() {

				logIndex := rand.Uint64()
				blockNumber := rand.Uint64()
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
			func(res mod.Vote) {
				// TODO HV2: enable this when side_msg_server code is all fully functional (atm mod.Vote_VOTE_NO is hardcoded due to missing code)
				// require.Equal(res, mod.Vote_VOTE_YES, "side tx handler should succeed")
				bn := new(big.Int).SetUint64(msg.BlockNumber)
				sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
				sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
				_, err := keeper.HasTopupSequence(ctx, sequence.String())
				require.NoError(err)
				// TODO HV2: enable this when side_msg_server code is all fully functional (atm mod.Vote_VOTE_NO is hardcoded due to missing code)
				// require.True(ok)
				replayRes := suite.sideHandler(ctx, &msg)
				require.Equal(replayRes, mod.Vote_VOTE_NO, "side tx handler should fail")
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {

			tc.malleate()
			res := suite.sideHandler(ctx, &msg)

			/* TODO HV2: can we generalize here with the following and reduce the load in posttests(res)?
			if tc.expPass {
				require.Equal(res, hmVote.VOTE_YES, "side tx handler should succeed")
			} else {
				require.Equal(res, hmVote.VOTE_NO, "side tx handler should fail")
			}
			*/

			tc.posttests(res)
		})
	}
}
