package keeper_test

import (
	mod "github.com/0xPolygon/heimdall-v2/module"
	"math/big"
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"

	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	hTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/testutil" //nolint:typecheck
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func (suite *KeeperTestSuite) sideHandler(ctx sdk.Context, msg sdk.Msg) mod.Vote {
	cfg := suite.sideMsgCfg
	return cfg.GetSideHandler(msg)(ctx, msg)
}

func (suite *KeeperTestSuite) postHandler(ctx sdk.Context, msg sdk.Msg, vote mod.Vote) {
	cfg := suite.sideMsgCfg
	cfg.GetPostHandler(msg)(ctx, msg, vote)
}

// TODO HV2: possibly refactor these cases into subtests to remove redundant setup code

func (suite *KeeperTestSuite) TestSideHandleTopupTx() {
	var msg types.MsgTopupTx

	ctx, keeper, require, t := suite.ctx, suite.keeper, suite.Require(), suite.T()

	// TODO HV2: enable when chainmanager is implemented
	// chainParams := keeper.ChainKeeper.GetParams(suite.ctx)

	_, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()

	t.Run("success", func(t *testing.T) {
		// TODO HV2: enable when contractCaller is implemented
		// contractCaller = mocks.IContractCaller{}

		logIndex := uint64(10)
		blockNumber := uint64(599)
		// TODO HV2: replace `_` with `txReceipt` when implemented
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
			addr2.String(),
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
		// TODO HV2: replace `_` with `event` when contractCaller implemented
		_ = &stakinginfo.StakinginfoTopUpFee{
			User: common.Address(sdk.AccAddress(addr1.String())),
			Fee:  coins.Amount.BigInt(),
		}
		// TODO HV2: enable when contractCaller is implemented
		// contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
		// contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(event, nil)

		res := suite.sideHandler(ctx, &msg)

		require.NotNil(res)
		// TODO HV2: enable this when side_msg_server code is fully functional (atm mod.Vote_VOTE_NO is hardcoded due to commented code)
		// require.Equal(res, mod.Vote_VOTE_YES, "side tx handler should succeed")
		ok, err := keeper.HasTopupSequence(ctx, sequence.String())
		require.NoError(err)
		require.False(ok)
	})

	t.Run("no receipt", func(t *testing.T) {
		// TODO HV2: enable contractCaller when implemented
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

		res := suite.sideHandler(ctx, &msg)
		require.Equal(res, mod.Vote_VOTE_NO, "side tx handler should fail")
	})

	t.Run("no log", func(t *testing.T) {
		// TODO HV2: enable when contractCaller is implemented
		// contractCaller = mocks.IContractCaller{}

		logIndex := uint64(10)
		blockNumber := uint64(599)
		// TODO HV2: replace `_` with `txReceipt` when implemented
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

		res := suite.sideHandler(ctx, &msg)
		require.Equal(res, mod.Vote_VOTE_NO, "side tx handler should fail")

	})

	t.Run("block mismatch", func(t *testing.T) {
		// TODO HV2: enable when contractCaller is implemented
		// contractCaller = mocks.IContractCaller{}

		logIndex := uint64(10)
		blockNumber := uint64(599)
		// TODO HV2: replace `_` with `txReceipt` when implemented
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

		// TODO HV2: replace `_` with `event` when implemented
		_ = &stakinginfo.StakinginfoTopUpFee{
			User: common.Address(sdk.AccAddress(addr1.String())),
			Fee:  coins.Amount.BigInt(),
		}

		// TODO HV2: enable when contractCaller is implemented
		// contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
		// contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(event, nil)

		res := suite.sideHandler(ctx, &msg)
		require.Equal(res, mod.Vote_VOTE_NO, "side tx handler should fail")
	})

	t.Run("user mismatch", func(t *testing.T) {
		// TODO HV2: enable when contractCaller is implemented
		// contractCaller = mocks.IContractCaller{}

		logIndex := uint64(10)
		blockNumber := uint64(599)
		// TODO HV2: replace `_` with `txReceipt` when implemented
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

		// TODO HV2: replace `_` with `event` when implemented
		_ = &stakinginfo.StakinginfoTopUpFee{
			User: common.Address(sdk.AccAddress(addr2.String())),
			Fee:  coins.Amount.BigInt(),
		}

		// TODO HV2: enable when contractCaller is implemented
		// contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
		// contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(event, nil)

		res := suite.sideHandler(ctx, &msg)
		require.Equal(res, mod.Vote_VOTE_NO, "side tx handler should fail")
	})

	t.Run("fee mismatch", func(t *testing.T) {
		// TODO HV2: enable when contractCaller is implemented
		// contractCaller = mocks.IContractCaller{}

		logIndex := uint64(10)
		blockNumber := uint64(599)
		// TODO HV2: replace `_` with `txReceipt` when implemented
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
		// TODO HV2: replace `_` with `event` when implemented
		_ = &stakinginfo.StakinginfoTopUpFee{
			User: common.Address(sdk.AccAddress(addr2.String())),
			Fee:  new(big.Int).SetUint64(1),
		}

		// TODO HV2: enable when contractCaller is implemented
		// contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txReceipt, nil)
		// contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StateSenderAddress.EthAddress(), txReceipt, logIndex).Return(event, nil)

		res := suite.sideHandler(ctx, &msg)
		require.Equal(res, mod.Vote_VOTE_NO, "side tx handler should fail")
	})
}

/* TODO HV2: we need to implement checks about account balances for `TestPostHandleTopupTx`
   This was done in heimdall-v1 by using a real app setup (no mocks).
   Hence, either we do that when app test setup is fixed,
   or we achieve something similar with mocked balances tracking
*/

func (suite *KeeperTestSuite) TestPostHandleTopupTx() {
	var msg types.MsgTopupTx

	ctx, require, keeper, t := suite.ctx, suite.Require(), suite.keeper, suite.T()
	// TODO HV2: enable when contractCaller is implemented
	// contractCaller := suite.contractCaller

	_, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()
	_, _, addr3 := testdata.KeyTestPubAddr()

	logIndex := rand.Uint64()
	blockNumber := rand.Uint64()
	hash := hTypes.TxHash{Hash: []byte(TxHash)}

	t.Run("no result", func(t *testing.T) {
		// TODO HV2: replace the following with simulation.RandomFeeCoins() when implemented
		base, _ := big.NewInt(0).SetString("1000000000000000000", 10)
		amt := big.NewInt(0).Mul(big.NewInt(0).SetInt64(int64(rand.Intn(1000000))), base)
		coins := sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(amt)}

		// topup msg
		msg = *types.NewMsgTopupTx(
			addr1.String(),
			addr2.String(),
			coins.Amount,
			hash,
			logIndex,
			blockNumber,
		)

		// sequence id
		bn := new(big.Int).SetUint64(msg.BlockNumber)
		sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
		sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

		suite.postHandler(ctx, &msg, mod.Vote_VOTE_NO)
		ok, err := keeper.HasTopupSequence(ctx, sequence.String())
		require.NoError(err)
		require.False(ok)
	})

	t.Run("yes result", func(t *testing.T) {
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

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoins(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		suite.postHandler(ctx, &msg, mod.Vote_VOTE_YES)

		ok, err := keeper.HasTopupSequence(ctx, sequence.String())
		require.NoError(err)
		require.True(ok)
	})

	t.Run("yes result with proposer", func(t *testing.T) {
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

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoins(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		suite.postHandler(ctx, &msg, mod.Vote_VOTE_YES)

		// there should be stored sequence
		ok, err := keeper.HasTopupSequence(ctx, sequence.String())
		require.NoError(err)
		require.True(ok)
	})

	t.Run("replay", func(t *testing.T) {
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

		// check if incoming tx is older
		bn := new(big.Int).SetUint64(msg.BlockNumber)
		sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
		sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoins(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		suite.postHandler(ctx, &msg, mod.Vote_VOTE_YES)

		// there should be a stored sequence
		_, err := keeper.HasTopupSequence(ctx, sequence.String())
		require.NoError(err)
		// TODO HV2: enable this when side_msg_server code is all fully functional (atm mod.Vote_VOTE_NO is hardcoded due to commented code)
		// require.True(ok)

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoins(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		// replay
		suite.postHandler(ctx, &msg, mod.Vote_VOTE_YES)
	})
}
