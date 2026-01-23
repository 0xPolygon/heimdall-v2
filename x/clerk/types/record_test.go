package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func TestNewEventRecord(t *testing.T) {
	t.Parallel()

	t.Run("creates event record with all fields", func(t *testing.T) {
		t.Parallel()

		txHash := "0xabcdef1234567890"
		logIndex := uint64(5)
		id := uint64(123)
		contract := "0x1234567890123456789012345678901234567890"
		data := []byte("test_event_data")
		chainID := "137"
		recordTime := time.Now()

		record := types.NewEventRecord(txHash, logIndex, id, contract, data, chainID, recordTime)

		require.Equal(t, txHash, record.TxHash)
		require.Equal(t, logIndex, record.LogIndex)
		require.Equal(t, id, record.Id)
		require.Equal(t, contract, record.Contract)
		require.Equal(t, data, record.Data)
		require.Equal(t, chainID, record.BorChainId)
		require.Equal(t, recordTime, record.RecordTime)
	})

	t.Run("creates event record with zero values", func(t *testing.T) {
		t.Parallel()

		record := types.NewEventRecord("", 0, 0, "", nil, "", time.Time{})

		require.Empty(t, record.TxHash)
		require.Equal(t, uint64(0), record.LogIndex)
		require.Equal(t, uint64(0), record.Id)
		require.Empty(t, record.Contract)
		require.Nil(t, record.Data)
		require.Empty(t, record.BorChainId)
		require.Equal(t, time.Time{}, record.RecordTime)
	})

	t.Run("creates event record with empty data", func(t *testing.T) {
		t.Parallel()

		record := types.NewEventRecord(
			"0xabcdef",
			1,
			5,
			"0x1234567890123456789012345678901234567890",
			[]byte{},
			"137",
			time.Now(),
		)

		require.Equal(t, "0xabcdef", record.TxHash)
		require.Equal(t, uint64(1), record.LogIndex)
		require.Equal(t, uint64(5), record.Id)
		require.Empty(t, record.Data)
	})

	t.Run("creates event record with large data", func(t *testing.T) {
		t.Parallel()

		largeData := make([]byte, 10000)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		record := types.NewEventRecord(
			"0xabcdef",
			1,
			5,
			"0x1234567890123456789012345678901234567890",
			largeData,
			"137",
			time.Now(),
		)

		require.Equal(t, largeData, record.Data)
		require.Len(t, record.Data, 10000)
	})

	t.Run("creates event record with different chain IDs", func(t *testing.T) {
		t.Parallel()

		chainIDs := []string{"1", "137", "80001", "42161"}

		for _, chainID := range chainIDs {
			record := types.NewEventRecord(
				"0xabcdef",
				1,
				5,
				"0x1234567890123456789012345678901234567890",
				[]byte("data"),
				chainID,
				time.Now(),
			)

			require.Equal(t, chainID, record.BorChainId)
		}
	})

	t.Run("creates multiple event records with different IDs", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		records := make([]types.EventRecord, 10)

		for i := range records {
			records[i] = types.NewEventRecord(
				"0xabcdef",
				uint64(i),
				uint64(i+100),
				"0x1234567890123456789012345678901234567890",
				[]byte("data"),
				"137",
				now,
			)
		}

		for i, record := range records {
			require.Equal(t, uint64(i), record.LogIndex)
			require.Equal(t, uint64(i+100), record.Id)
		}
	})

	t.Run("preserves timestamp accurately", func(t *testing.T) {
		t.Parallel()

		specificTime := time.Date(2024, 1, 15, 12, 30, 45, 123456789, time.UTC)

		record := types.NewEventRecord(
			"0xabcdef",
			1,
			5,
			"0x1234567890123456789012345678901234567890",
			[]byte("data"),
			"137",
			specificTime,
		)

		require.Equal(t, specificTime, record.RecordTime)
		require.Equal(t, 2024, record.RecordTime.Year())
		require.Equal(t, time.January, record.RecordTime.Month())
		require.Equal(t, 15, record.RecordTime.Day())
	})
}
