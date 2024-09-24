package keeper_test

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"

	hTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/topup/testutil"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func (s *KeeperTestSuite) TestCreateTopupTx() {
	msgServer, require, keeper, ctx, t := s.msgServer, s.Require(), s.keeper, s.ctx, s.T()

	var msg types.MsgTopupTx
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	hash := hTypes.TxHash{Hash: []byte(TxHash)}
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

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SpendableCoin(gomock.Any(), gomock.Any(), gomock.Any()).Return(sdk.NewCoin("matic", math.ZeroInt())).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SendCoinsFromAccountToModule(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().BurnCoins(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

		res, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.NoError(err)
		require.NotNil(res)
	})

	t.Run("fail with insufficient funds", func(t *testing.T) {
		msg = *types.NewMsgWithdrawFeeTx(addr.String(), math.ZeroInt())

		keeper.BankKeeper.(*testutil.MockBankKeeper).EXPECT().SpendableCoin(gomock.Any(), gomock.Any(), gomock.Any()).Return(sdk.NewCoin("matic", math.ZeroInt())).Times(1)

		_, err := msgServer.WithdrawFeeTx(ctx, &msg)
		require.Error(err)
		require.Contains(err.Error(), "insufficient funds")
	})
}
