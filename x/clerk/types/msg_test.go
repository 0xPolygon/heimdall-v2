package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func TestNewMsgEventRecord(t *testing.T) {
	t.Parallel()

	t.Run("creates message with all fields", func(t *testing.T) {
		t.Parallel()

		contractAddr := make([]byte, 20)
		for i := range contractAddr {
			contractAddr[i] = byte(i)
		}

		msg := types.NewMsgEventRecord(
			"0x1234567890123456789012345678901234567890",
			"0xabcdef1234567890",
			5,
			1000,
			123,
			contractAddr,
			[]byte("test_data"),
			"137",
		)

		require.Equal(t, "0x1234567890123456789012345678901234567890", msg.From)
		require.Equal(t, "0xabcdef1234567890", msg.TxHash)
		require.Equal(t, uint64(5), msg.LogIndex)
		require.Equal(t, uint64(1000), msg.BlockNumber)
		require.Equal(t, uint64(123), msg.Id)
		require.NotEmpty(t, msg.ContractAddress)
		require.Equal(t, []byte("test_data"), msg.Data)
		require.Equal(t, "137", msg.ChainId)
	})

	t.Run("handles invalid contract address", func(t *testing.T) {
		t.Parallel()

		invalidAddr := []byte{0x00} // Invalid address

		msg := types.NewMsgEventRecord(
			"0x1234567890123456789012345678901234567890",
			"0xabcdef1234567890",
			5,
			1000,
			123,
			invalidAddr,
			[]byte("test_data"),
			"137",
		)

		// Should handle error and set the empty contract address
		require.NotNil(t, msg)
	})

	t.Run("formats from address", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgEventRecord(
			"1234567890123456789012345678901234567890", // Without 0x prefix
			"0xabcdef",
			1,
			1000,
			5,
			make([]byte, 20),
			[]byte("data"),
			"137",
		)

		require.Contains(t, msg.From, "0x")
	})
}

func TestMsgEventRecord_Route(t *testing.T) {
	t.Parallel()

	t.Run("returns correct route", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgEventRecord{}
		route := msg.Route()

		require.Equal(t, types.RouterKey, route)
		require.NotEmpty(t, route)
	})
}

func TestMsgEventRecord_Type(t *testing.T) {
	t.Parallel()

	t.Run("returns correct type", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgEventRecord{}
		msgType := msg.Type()

		require.Equal(t, "event-record", msgType)
	})
}

func TestMsgEventRecord_ValidateBasic(t *testing.T) {
	t.Parallel()

	validFrom := "0x1234567890123456789012345678901234567890"
	validContract := "0x0987654321098765432109876543210987654321"
	validTxHash := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	tests := []struct {
		name        string
		msg         types.MsgEventRecord
		shouldError bool
		errorCheck  func(*testing.T, error)
	}{
		{
			name: "valid message",
			msg: types.MsgEventRecord{
				From:            validFrom,
				TxHash:          validTxHash,
				LogIndex:        1,
				BlockNumber:     1000,
				Id:              5,
				ContractAddress: validContract,
				Data:            []byte("test_data"),
				ChainId:         "137",
			},
			shouldError: false,
		},
		{
			name: "invalid from address",
			msg: types.MsgEventRecord{
				From:            "invalid-address",
				TxHash:          validTxHash,
				ContractAddress: validContract,
			},
			shouldError: true,
			errorCheck: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "invalid address")
			},
		},
		{
			name: "empty from address",
			msg: types.MsgEventRecord{
				From:            "",
				TxHash:          validTxHash,
				ContractAddress: validContract,
			},
			shouldError: true,
			errorCheck: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "invalid address")
			},
		},
		{
			name: "invalid contract address",
			msg: types.MsgEventRecord{
				From:            validFrom,
				TxHash:          validTxHash,
				ContractAddress: "invalid-contract",
			},
			shouldError: true,
			errorCheck: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "invalid address")
			},
		},
		{
			name: "empty contract address",
			msg: types.MsgEventRecord{
				From:            validFrom,
				TxHash:          validTxHash,
				ContractAddress: "",
			},
			shouldError: true,
			errorCheck: func(t *testing.T, err error) {
				require.Contains(t, err.Error(), "invalid address")
			},
		},
		{
			name: "empty tx hash",
			msg: types.MsgEventRecord{
				From:            validFrom,
				TxHash:          "",
				ContractAddress: validContract,
				Data:            []byte("test"),
			},
			shouldError: true,
			errorCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, types.ErrInvalidTxHash)
			},
		},
		{
			name: "data size exceeds max",
			msg: types.MsgEventRecord{
				From:            validFrom,
				TxHash:          validTxHash,
				ContractAddress: validContract,
				Data:            []byte(strings.Repeat("a", 100000)), // Very large data
			},
			shouldError: true,
			errorCheck: func(t *testing.T, err error) {
				require.ErrorIs(t, err, types.ErrSizeExceed)
			},
		},
		{
			name: "zero log index is valid",
			msg: types.MsgEventRecord{
				From:            validFrom,
				TxHash:          validTxHash,
				LogIndex:        0,
				ContractAddress: validContract,
				Data:            []byte("test"),
				ChainId:         "137",
			},
			shouldError: false,
		},
		{
			name: "large log index is valid",
			msg: types.MsgEventRecord{
				From:            validFrom,
				TxHash:          validTxHash,
				LogIndex:        999999,
				ContractAddress: validContract,
				Data:            []byte("test"),
				ChainId:         "137",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.msg.ValidateBasic()
			if tt.shouldError {
				require.Error(t, err)
				if tt.errorCheck != nil {
					tt.errorCheck(t, err)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgEventRecord_GetTxHash(t *testing.T) {
	t.Parallel()

	t.Run("returns tx hash", func(t *testing.T) {
		t.Parallel()

		expectedHash := "0xabcdef1234567890"
		msg := types.MsgEventRecord{
			TxHash: expectedHash,
		}

		hash := msg.GetTxHash()
		require.Equal(t, expectedHash, hash)
	})

	t.Run("returns empty string for empty tx hash", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgEventRecord{
			TxHash: "",
		}

		hash := msg.GetTxHash()
		require.Equal(t, "", hash)
	})
}

func TestMsgEventRecord_GetLogIndex(t *testing.T) {
	t.Parallel()

	t.Run("returns log index", func(t *testing.T) {
		t.Parallel()

		expectedIndex := uint64(42)
		msg := types.MsgEventRecord{
			LogIndex: expectedIndex,
		}

		index := msg.GetLogIndex()
		require.Equal(t, expectedIndex, index)
	})

	t.Run("returns zero for zero log index", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgEventRecord{
			LogIndex: 0,
		}

		index := msg.GetLogIndex()
		require.Equal(t, uint64(0), index)
	})

	t.Run("returns large log index", func(t *testing.T) {
		t.Parallel()

		expectedIndex := uint64(999999999)
		msg := types.MsgEventRecord{
			LogIndex: expectedIndex,
		}

		index := msg.GetLogIndex()
		require.Equal(t, expectedIndex, index)
	})
}
