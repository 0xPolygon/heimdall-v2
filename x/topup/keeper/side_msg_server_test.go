package keeper_test

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"

	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/testutil"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func (s *KeeperTestSuite) sideHandler(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
	cfg := s.sideMsgCfg
	return cfg.GetSideHandler(msg)(ctx, msg)
}

func (s *KeeperTestSuite) postHandler(ctx sdk.Context, msg sdk.Msg, vote sidetxs.Vote) {
	cfg := s.sideMsgCfg
	_ = cfg.GetPostHandler(msg)(ctx, msg, vote)
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

func (s *KeeperTestSuite) TestPostHandleTopupTx() {
	ctx, require, keeper, postHandler, t := s.ctx, s.Require(), s.keeper, s.postHandler, s.T()

	var msg types.MsgTopupTx

	_, _, addr1 := testdata.KeyTestPubAddr()
	_, _, addr2 := testdata.KeyTestPubAddr()
	_, _, addr3 := testdata.KeyTestPubAddr()

	logIndex := rand.Uint64()
	blockNumber := rand.Uint64()
	hash := []byte(TxHash)

	var balances map[string]sdk.Coins

	keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().GetBalance(gomock.Any(), gomock.Any(), authTypes.FeeToken).
		DoAndReturn(func(_ sdk.Context, addr sdk.AccAddress, _ string) sdk.Coin {
			balances := balances[addr.String()]
			for _, balance := range balances {
				if balance.Denom == authTypes.FeeToken {
					return balance
				}
			}
			return sdk.NewCoin(authTypes.FeeToken, math.NewInt(0))
		}).AnyTimes()

	t.Run("no result", func(t *testing.T) {
		balances = make(map[string]sdk.Coins)
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
		balances = make(map[string]sdk.Coins)
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
		topupAmount := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: msg.Fee}}

		s.trackMockBalances(addr1, addr1, topupAmount, balances)

		postHandler(ctx, &msg, sidetxs.Vote_VOTE_YES)

		ok, err := keeper.HasTopupSequence(ctx, sequence.String())
		require.NoError(err)
		require.True(ok)

		bal := keeper.BankKeeper.GetBalance(ctx, addr1, authTypes.FeeToken)
		require.Equal(topupAmount.AmountOf(authTypes.FeeToken), bal.Amount)

	})

	t.Run("yes result with proposer", func(t *testing.T) {
		balances = make(map[string]sdk.Coins)
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

		bn := new(big.Int).SetUint64(msg.BlockNumber)
		sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
		sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
		topupAmount := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: msg.Fee}}

		s.trackMockBalances(addr2, addr3, topupAmount, balances)

		postHandler(ctx, &msg, sidetxs.Vote_VOTE_YES)

		// there should be a stored sequence
		ok, err := keeper.HasTopupSequence(ctx, sequence.String())
		require.NoError(err)
		require.True(ok)

		expBal1, isNeg := topupAmount.SafeSub(ante.DefaultFeeWantedPerTx...)
		require.False(isNeg)
		bal1 := keeper.BankKeeper.GetBalance(ctx, addr3, authTypes.FeeToken)
		require.Equal(expBal1.AmountOf(authTypes.FeeToken), bal1.Amount)

		bal2 := keeper.BankKeeper.GetBalance(ctx, addr2, authTypes.FeeToken)
		require.Equal(ante.DefaultFeeWantedPerTx.AmountOf(authTypes.FeeToken), bal2.Amount)
	})

	t.Run("replay", func(t *testing.T) {
		balances = make(map[string]sdk.Coins)
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

		bn := new(big.Int).SetUint64(msg.BlockNumber)
		sequence := new(big.Int).Mul(bn, big.NewInt(types.DefaultLogIndexUnit))
		sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
		topupAmount := sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: msg.Fee}}

		s.trackMockBalances(addr1, addr1, topupAmount, balances)

		postHandler(ctx, &msg, sidetxs.Vote_VOTE_YES)

		// there should be a stored sequence
		ok, err := keeper.HasTopupSequence(ctx, sequence.String())
		require.NoError(err)
		require.True(ok)

		bal := keeper.BankKeeper.GetBalance(ctx, addr1, authTypes.FeeToken)
		require.Equal(topupAmount.AmountOf(authTypes.FeeToken), bal.Amount)

		// replay
		postHandler(ctx, &msg, sidetxs.Vote_VOTE_YES)

		bal = keeper.BankKeeper.GetBalance(ctx, addr1, authTypes.FeeToken)
		require.Equal(topupAmount.AmountOf(authTypes.FeeToken), bal.Amount)
	})
}

// trackMockBalances tracks the balances of the proposer and recipient of the topup tx
func (s *KeeperTestSuite) trackMockBalances(proposerAddr, recipientAddr sdk.AccAddress, topupAmount sdk.Coins, balances map[string]sdk.Coins) {
	s.T().Helper()
	s.keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().MintCoins(gomock.Any(), types.ModuleName, topupAmount).
		DoAndReturn(func(_ context.Context, moduleName string, amt sdk.Coins) error {
			balances[moduleName] = balances[moduleName].Add(amt...)
			return nil
		}).
		Times(1)

	s.keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, recipientAddr, topupAmount).
		DoAndReturn(func(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
			newBalance, isNeg := balances[senderModule].SafeSub(amt...)
			if isNeg {
				return fmt.Errorf("not enough balance")
			}
			balances[senderModule] = newBalance
			balances[recipientAddr.String()] = balances[recipientAddr.String()].Add(amt...)
			return nil
		}).
		Times(1)

	s.keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoins(gomock.Any(), recipientAddr, proposerAddr, ante.DefaultFeeWantedPerTx).
		DoAndReturn(func(_ context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
			newBalance, isNeg := balances[fromAddr.String()].SafeSub(amt...)
			if isNeg {
				return fmt.Errorf("not enough balance")
			}
			balances[fromAddr.String()] = newBalance
			balances[toAddr.String()] = balances[toAddr.String()].Add(amt...)
			return nil
		}).
		Times(1)
}
