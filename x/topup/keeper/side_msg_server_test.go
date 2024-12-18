package keeper_test

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/testutil"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"
)

func (s *KeeperTestSuite) sideHandler(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
	cfg := s.sideMsgCfg
	return cfg.GetSideHandler(msg)(ctx, msg)
}

func (s *KeeperTestSuite) postHandler(ctx sdk.Context, msg sdk.Msg, vote sidetxs.Vote) {
	cfg := s.sideMsgCfg
	cfg.GetPostHandler(msg)(ctx, msg, vote)
}

func (s *KeeperTestSuite) TestSideHandleTopupTx() {
	var msg types.MsgTopupTx

	ctx, keeper, require, t, contractCaller, sideHandler := s.ctx, s.keeper, s.Require(), s.T(), &s.contractCaller, s.sideHandler

	keeper.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(6)

	_, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()

	t.Run("success", func(t *testing.T) {
		logIndex := uint64(10)
		blockNumber := uint64(599)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber),
		}
		hash := []byte(TxHash)

		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
		require.NoError(err)

		// topup msg
		msg = *types.NewMsgTopupTx(
			addr1.String(),
			addr2.String(),
			coins.AmountOf(authTypes.FeeToken),
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
			User: common.HexToAddress(addr2.String()),
			Fee:  coins.AmountOf(authTypes.FeeToken).BigInt(),
		}

		contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything).Return(txReceipt, nil)
		contractCaller.On("DecodeValidatorTopupFeesEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil)

		res := sideHandler(ctx, &msg)

		require.NotNil(res)
		require.Equal(res, sidetxs.Vote_VOTE_YES, "side tx handler should succeed")
		ok, err := keeper.HasTopupSequence(ctx, sequence.String())
		require.NoError(err)
		require.False(ok)
	})

	t.Run("no receipt", func(t *testing.T) {
		logIndex := uint64(10)
		blockNumber := uint64(599)
		hash := []byte(TxHash)

		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
		require.NoError(err)

		msg = *types.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)
		contractCaller.On("GetConfirmedTxReceipt", hash, chainmanagertypes.DefaultParams().MainChainTxConfirmations).Return(nil, nil)
		contractCaller.On("DecodeValidatorTopupFeesEvent", chainmanagertypes.DefaultParams().ChainParams.StateSenderAddress, nil, logIndex).Return(nil, nil)

		res := sideHandler(ctx, &msg)
		require.Equal(res, sidetxs.Vote_VOTE_NO, "side tx handler should fail")
	})

	t.Run("no log", func(t *testing.T) {
		logIndex := uint64(10)
		blockNumber := uint64(599)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber),
		}
		hash := []byte(TxHash)

		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
		require.NoError(err)

		msg = *types.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)
		contractCaller.On("GetConfirmedTxReceipt", hash, chainmanagertypes.DefaultParams().MainChainTxConfirmations).Return(txReceipt, nil)
		contractCaller.On("DecodeValidatorTopupFeesEvent", chainmanagertypes.DefaultParams().ChainParams.StateSenderAddress, txReceipt, logIndex).Return(nil, nil)

		res := sideHandler(ctx, &msg)
		require.Equal(res, sidetxs.Vote_VOTE_NO, "side tx handler should fail")
	})

	t.Run("block mismatch", func(t *testing.T) {
		logIndex := uint64(10)
		blockNumber := uint64(599)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber + 1),
		}
		hash := []byte(TxHash)

		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
		require.NoError(err)

		msg = *types.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)

		event := &stakinginfo.StakinginfoTopUpFee{
			User: common.Address(sdk.AccAddress(addr1.String())),
			Fee:  coins.AmountOf(authTypes.FeeToken).BigInt(),
		}

		contractCaller.On("GetConfirmedTxReceipt", hash, chainmanagertypes.DefaultParams().MainChainTxConfirmations).Return(txReceipt, nil)
		contractCaller.On("DecodeValidatorTopupFeesEvent", chainmanagertypes.DefaultParams().ChainParams.StateSenderAddress, txReceipt, logIndex).Return(event, nil)

		res := sideHandler(ctx, &msg)
		require.Equal(res, sidetxs.Vote_VOTE_NO, "side tx handler should fail")
	})

	t.Run("user mismatch", func(t *testing.T) {
		logIndex := uint64(10)
		blockNumber := uint64(599)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber),
		}
		hash := []byte(TxHash)

		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
		require.NoError(err)

		msg = *types.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)

		event := &stakinginfo.StakinginfoTopUpFee{
			User: common.Address(sdk.AccAddress(addr2.String())),
			Fee:  coins.AmountOf(authTypes.FeeToken).BigInt(),
		}

		contractCaller.On("GetConfirmedTxReceipt", hash, chainmanagertypes.DefaultParams().MainChainTxConfirmations).Return(txReceipt, nil)
		contractCaller.On("DecodeValidatorTopupFeesEvent", chainmanagertypes.DefaultParams().ChainParams.StateSenderAddress, txReceipt, logIndex).Return(event, nil)

		res := sideHandler(ctx, &msg)
		require.Equal(res, sidetxs.Vote_VOTE_NO, "side tx handler should fail")
	})

	t.Run("fee mismatch", func(t *testing.T) {
		logIndex := uint64(10)
		blockNumber := uint64(599)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber),
		}
		hash := []byte(TxHash)

		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
		require.NoError(err)

		msg = *types.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)

		// mock external call
		event := &stakinginfo.StakinginfoTopUpFee{
			User: common.Address(sdk.AccAddress(addr2.String())),
			Fee:  new(big.Int).SetUint64(1),
		}

		contractCaller.On("GetConfirmedTxReceipt", hash, chainmanagertypes.DefaultParams().MainChainTxConfirmations).Return(txReceipt, nil)
		contractCaller.On("DecodeValidatorTopupFeesEvent", chainmanagertypes.DefaultParams().ChainParams.StateSenderAddress, txReceipt, logIndex).Return(event, nil)

		res := sideHandler(ctx, &msg)
		require.Equal(res, sidetxs.Vote_VOTE_NO, "side tx handler should fail")
	})
}

// TODO HV2: https://polygon.atlassian.net/browse/POS-2765

func (s *KeeperTestSuite) TestPostHandleTopupTx() {
	ctx, require, keeper, postHandler, t := s.ctx, s.Require(), s.keeper, s.postHandler, s.T()

	var msg types.MsgTopupTx

	_, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()
	_, _, addr3 := testdata.KeyTestPubAddr()

	logIndex := rand.Uint64()
	blockNumber := rand.Uint64()
	hash := []byte(TxHash)

	t.Run("no result", func(t *testing.T) {
		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
		require.NoError(err)

		msg = *types.NewMsgTopupTx(
			addr1.String(),
			addr2.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)

		bn := new(big.Int).SetUint64(msg.BlockNumber)
		sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
		sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

		postHandler(ctx, &msg, sidetxs.Vote_VOTE_NO)
		ok, err := keeper.HasTopupSequence(ctx, sequence.String())
		require.NoError(err)
		require.False(ok)
	})

	t.Run("yes result", func(t *testing.T) {
		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
		require.NoError(err)

		msg = *types.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)

		bn := new(big.Int).SetUint64(msg.BlockNumber)
		sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
		sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().MintCoins(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoins(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		postHandler(ctx, &msg, sidetxs.Vote_VOTE_YES)

		ok, err := keeper.HasTopupSequence(ctx, sequence.String())
		require.NoError(err)
		require.True(ok)
	})

	t.Run("yes result with proposer", func(t *testing.T) {
		logIndex = rand.Uint64()
		blockNumber = rand.Uint64()

		txHash := common.HexToHash("0x000000000000000000000000000000000000000000000000000000000001dead")
		hash := txHash.Bytes()

		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
		require.NoError(err)

		msg = *types.NewMsgTopupTx(
			addr2.String(),
			addr3.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)

		// check if incoming tx is older
		bn := new(big.Int).SetUint64(msg.BlockNumber)
		sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
		sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().MintCoins(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoins(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		postHandler(ctx, &msg, sidetxs.Vote_VOTE_YES)

		// there should be stored sequence
		ok, err := keeper.HasTopupSequence(ctx, sequence.String())
		require.NoError(err)
		require.True(ok)
	})

	t.Run("replay", func(t *testing.T) {
		logIndex = rand.Uint64()
		blockNumber = rand.Uint64()
		txHash := "0x000000000000000000000000000000000000000000000000000000000002dead"
		hash := []byte(txHash)

		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
		require.NoError(err)

		msg = *types.NewMsgTopupTx(
			addr1.String(),
			addr1.String(),
			coins.AmountOf(authTypes.FeeToken),
			hash,
			logIndex,
			blockNumber,
		)

		// check if incoming tx is older
		bn := new(big.Int).SetUint64(msg.BlockNumber)
		sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
		sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().MintCoins(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoins(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		postHandler(ctx, &msg, sidetxs.Vote_VOTE_YES)

		// there should be a stored sequence
		ok, err := keeper.HasTopupSequence(ctx, sequence.String())
		require.NoError(err)
		require.True(ok)

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoins(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		// replay
		postHandler(ctx, &msg, sidetxs.Vote_VOTE_YES)
	})
}
