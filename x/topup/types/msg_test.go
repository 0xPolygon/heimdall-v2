package types_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func TestNewMsgTopupTx(t *testing.T) {
	t.Parallel()

	t.Run("creates message with all fields", func(t *testing.T) {
		t.Parallel()

		proposer := "0x1234567890123456789012345678901234567890"
		user := "0x1234567890123456789012345678901234567891"
		fee := sdkmath.NewInt(100)
		txHash := common.HexToHash("0xabcd").Bytes()
		index := uint64(5)
		blockNumber := uint64(1000)

		msg := types.NewMsgTopupTx(proposer, user, fee, txHash, index, blockNumber)

		require.NotNil(t, msg)
		require.Contains(t, msg.Proposer, "1234567890")
		require.Equal(t, user, msg.User)
		require.Equal(t, fee, msg.Fee)
		require.Equal(t, txHash, msg.TxHash)
		require.Equal(t, index, msg.LogIndex)
		require.Equal(t, blockNumber, msg.BlockNumber)
	})
}

func TestMsgTopupTx_Type(t *testing.T) {
	t.Parallel()

	t.Run("returns correct type", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgTopupTx{}
		require.Equal(t, types.EventTypeTopup, msg.Type())
	})
}

func TestMsgTopupTx_ValidateBasic(t *testing.T) {
	t.Parallel()

	t.Run("validates correct message", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgTopupTx{
			Proposer:    "0x1234567890123456789012345678901234567890",
			User:        "0x1234567890123456789012345678901234567891",
			Fee:         sdkmath.NewInt(100),
			TxHash:      make([]byte, 32),
			LogIndex:    1,
			BlockNumber: 100,
		}

		err := msg.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("rejects negative fee", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgTopupTx{
			Proposer:    "0x1234567890123456789012345678901234567890",
			User:        "0x1234567890123456789012345678901234567891",
			Fee:         sdkmath.NewInt(-100),
			TxHash:      make([]byte, 32),
			LogIndex:    1,
			BlockNumber: 100,
		}

		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "fee cannot be negative")
	})

	t.Run("rejects invalid proposer", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgTopupTx{
			Proposer:    "invalid",
			User:        "0x1234567890123456789012345678901234567891",
			Fee:         sdkmath.NewInt(100),
			TxHash:      make([]byte, 32),
			LogIndex:    1,
			BlockNumber: 100,
		}

		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid proposer")
	})

	t.Run("rejects invalid user", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgTopupTx{
			Proposer:    "0x1234567890123456789012345678901234567890",
			User:        "not-an-address",
			Fee:         sdkmath.NewInt(100),
			TxHash:      make([]byte, 32),
			LogIndex:    1,
			BlockNumber: 100,
		}

		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid user")
	})

	t.Run("rejects invalid tx hash length", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgTopupTx{
			Proposer:    "0x1234567890123456789012345678901234567890",
			User:        "0x1234567890123456789012345678901234567891",
			Fee:         sdkmath.NewInt(100),
			TxHash:      []byte{0x01, 0x02},
			LogIndex:    1,
			BlockNumber: 100,
		}

		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid tx hash")
	})

	t.Run("accepts zero fee", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgTopupTx{
			Proposer:    "0x1234567890123456789012345678901234567890",
			User:        "0x1234567890123456789012345678901234567891",
			Fee:         sdkmath.ZeroInt(),
			TxHash:      make([]byte, 32),
			LogIndex:    1,
			BlockNumber: 100,
		}

		err := msg.ValidateBasic()
		require.NoError(t, err)
	})
}

func TestNewMsgWithdrawFeeTx(t *testing.T) {
	t.Parallel()

	t.Run("creates message with all fields", func(t *testing.T) {
		t.Parallel()

		proposer := "0x1234567890123456789012345678901234567890"
		amount := sdkmath.NewInt(500)

		msg := types.NewMsgWithdrawFeeTx(proposer, amount)

		require.NotNil(t, msg)
		require.Contains(t, msg.Proposer, "1234567890")
		require.Equal(t, amount, msg.Amount)
	})
}

func TestMsgWithdrawFeeTx_Type(t *testing.T) {
	t.Parallel()

	t.Run("returns correct type", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgWithdrawFeeTx{}
		require.Equal(t, types.EventTypeWithdraw, msg.Type())
	})
}

func TestMsgWithdrawFeeTx_ValidateBasic(t *testing.T) {
	t.Parallel()

	t.Run("validates correct message", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgWithdrawFeeTx{
			Proposer: "0x1234567890123456789012345678901234567890",
			Amount:   sdkmath.NewInt(200),
		}

		err := msg.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("rejects negative amount", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgWithdrawFeeTx{
			Proposer: "0x1234567890123456789012345678901234567890",
			Amount:   sdkmath.NewInt(-200),
		}

		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "amount cannot be negative")
	})

	t.Run("rejects invalid proposer", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgWithdrawFeeTx{
			Proposer: "invalid-address",
			Amount:   sdkmath.NewInt(200),
		}

		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid proposer")
	})

	t.Run("accepts zero amount", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgWithdrawFeeTx{
			Proposer: "0x1234567890123456789012345678901234567890",
			Amount:   sdkmath.ZeroInt(),
		}

		err := msg.ValidateBasic()
		require.NoError(t, err)
	})
}
