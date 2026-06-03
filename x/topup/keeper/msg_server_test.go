package keeper_test

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/topup/testutil"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func (s *KeeperTestSuite) TestCreateTopupTx() {
	msgServer, require, keeper, ctx, t := s.msgServer, s.Require(), s.keeper, s.ctx, s.T()

	var msg types.MsgTopupTx
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	hash := []byte(TxHash)
	logIndex := r.Uint64()
	blockNumber := r.Uint64()

	_, _, addr := testdata.KeyTestPubAddr()
	fee := math.NewInt(100000000000000000)

	t.Run("success", func(t *testing.T) {
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().IsSendEnabledDenom(gomock.Any(), gomock.Any()).Return(true).Times(1)

		msg = *types.NewMsgTopupTx(addr.String(), addr.String(), fee, hash, logIndex, blockNumber)

		res, err := msgServer.HandleTopupTx(ctx, &msg)
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

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().IsSendEnabledDenom(gomock.Any(), gomock.Any()).Return(true).Times(1)

		_, err = msgServer.HandleTopupTx(ctx, &msg)
		require.Error(err)
		require.Contains(err.Error(), "already exists")
	})
}

func (s *KeeperTestSuite) TestWithdrawFeeTx() {
	msgServer, require, keeper, ctx, t := s.msgServer, s.Require(), s.keeper, s.ctx, s.T()

	var msg types.MsgWithdrawFeeTx

	_, _, addr := testdata.KeyTestPubAddr()

	t.Run("success full amount", func(t *testing.T) {
		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
		require.NoError(err)

		msg = *types.NewMsgWithdrawFeeTx(addr.String(), math.ZeroInt())

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SpendableCoin(gomock.Any(), gomock.Any(), gomock.Any()).Return(coins[0]).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().BurnCoins(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		res, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.NoError(err)
		require.NotNil(res)
	})

	t.Run("success partial amount", func(t *testing.T) {
		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
		require.NoError(err)

		amt, _ := math.NewIntFromString("2")
		coins = coins.Sub(sdk.Coin{Denom: authTypes.FeeToken, Amount: amt})
		msg = *types.NewMsgWithdrawFeeTx(addr.String(), coins.AmountOf(authTypes.FeeToken))

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SpendableCoin(gomock.Any(), gomock.Any(), gomock.Any()).Return(sdk.NewCoin(authTypes.FeeToken, math.ZeroInt())).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().BurnCoins(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		res, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.NoError(err)
		require.NotNil(res)
	})

	t.Run("fail with negative amount", func(t *testing.T) {
		msg = *types.NewMsgWithdrawFeeTx(addr.String(), math.NewInt(-1))

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SpendableCoin(gomock.Any(), gomock.Any(), gomock.Any()).Return(sdk.NewCoin(authTypes.FeeToken, math.ZeroInt())).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().BurnCoins(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		res, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.Error(err)
		require.Nil(res)
	})

	t.Run("fail with insufficient funds", func(t *testing.T) {
		msg = *types.NewMsgWithdrawFeeTx(addr.String(), math.ZeroInt())

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SpendableCoin(gomock.Any(), gomock.Any(), gomock.Any()).Return(sdk.NewCoin(authTypes.FeeToken, math.ZeroInt())).Times(1)

		_, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.Error(err)
		require.Contains(err.Error(), "insufficient funds")
	})
}

func (s *KeeperTestSuite) TestWithdrawFeeTxValidatorGate() {
	msgServer, require, keeper, t := s.msgServer, s.Require(), s.keeper, s.T()

	const gateHeight int64 = 100

	_, _, addr := testdata.KeyTestPubAddr()
	mockStake := keeper.StakeKeeper.(*testutil.MockStakeKeeper)
	mockBank := keeper.BankKeeper.(*testutil.MockBankKeeper)

	// Restore the disabled-gate default after this test so other tests aren't affected.
	prev := helper.GetFeeWithdrawValidatorGateHeight()
	helper.SetFeeWithdrawValidatorGateHeight(gateHeight)
	defer helper.SetFeeWithdrawValidatorGateHeight(prev)

	expectBankSuccess := func() {
		coins, err := simulation.RandomFees(rand.New(rand.NewSource(time.Now().UnixNano())), s.ctx, sdk.Coins{sdk.NewCoin(authTypes.FeeToken, math.NewInt(1000000000000000000))})
		require.NoError(err)
		mockBank.EXPECT().SpendableCoin(gomock.Any(), gomock.Any(), gomock.Any()).Return(coins[0]).Times(1)
		mockBank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockBank.EXPECT().BurnCoins(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
	}

	t.Run("pre-gate non-validator passes (H-1)", func(t *testing.T) {
		ctx := s.ctx.WithBlockHeight(gateHeight - 1)
		msg := *types.NewMsgWithdrawFeeTx(addr.String(), math.ZeroInt())
		expectBankSuccess()

		res, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.NoError(err)
		require.NotNil(res)
	})

	// The real stake keeper surfaces "address not in validators store" as a
	// wrapped collections.ErrNotFound, not (false, nil). Mock matches production.
	notFound := fmt.Errorf("error while fetching the validator from the store %w", collections.ErrNotFound)

	t.Run("at-gate non-validator rejected (H)", func(t *testing.T) {
		ctx := s.ctx.WithBlockHeight(gateHeight)
		msg := *types.NewMsgWithdrawFeeTx(addr.String(), math.ZeroInt())
		mockStake.EXPECT().IsCurrentValidatorByAddress(gomock.Any(), gomock.Any()).Return(false, notFound).Times(1)

		res, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.Error(err)
		require.Nil(res)
		require.Contains(err.Error(), "unauthorized")
		require.Contains(err.Error(), "restricted to current validators")
	})

	t.Run("post-gate non-validator rejected (H+1)", func(t *testing.T) {
		ctx := s.ctx.WithBlockHeight(gateHeight + 1)
		msg := *types.NewMsgWithdrawFeeTx(addr.String(), math.ZeroInt())
		mockStake.EXPECT().IsCurrentValidatorByAddress(gomock.Any(), gomock.Any()).Return(false, notFound).Times(1)

		res, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.Error(err)
		require.Nil(res)
		require.Contains(err.Error(), "unauthorized")
		require.Contains(err.Error(), "restricted to current validators")
	})

	t.Run("post-gate (false, nil) still rejected as unauthorized", func(t *testing.T) {
		ctx := s.ctx.WithBlockHeight(gateHeight + 1)
		msg := *types.NewMsgWithdrawFeeTx(addr.String(), math.ZeroInt())
		mockStake.EXPECT().IsCurrentValidatorByAddress(gomock.Any(), gomock.Any()).Return(false, nil).Times(1)

		res, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.Error(err)
		require.Nil(res)
		require.Contains(err.Error(), "unauthorized")
	})

	t.Run("post-gate validator passes through", func(t *testing.T) {
		ctx := s.ctx.WithBlockHeight(gateHeight + 1)
		msg := *types.NewMsgWithdrawFeeTx(addr.String(), math.ZeroInt())
		mockStake.EXPECT().IsCurrentValidatorByAddress(gomock.Any(), gomock.Any()).Return(true, nil).Times(1)
		expectBankSuccess()

		res, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.NoError(err)
		require.NotNil(res)
	})

	t.Run("post-gate stake lookup error propagates", func(t *testing.T) {
		ctx := s.ctx.WithBlockHeight(gateHeight + 1)
		msg := *types.NewMsgWithdrawFeeTx(addr.String(), math.ZeroInt())
		mockStake.EXPECT().IsCurrentValidatorByAddress(gomock.Any(), gomock.Any()).Return(false, errors.New("lookup failure")).Times(1)

		res, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.Error(err)
		require.Nil(res)
		require.Contains(err.Error(), "validator lookup failed")
		require.NotContains(err.Error(), "restricted to current validators")
	})
}
