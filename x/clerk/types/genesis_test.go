package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func TestDefaultGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("returns default genesis state", func(t *testing.T) {
		t.Parallel()

		gs := types.DefaultGenesisState()

		require.NotNil(t, gs)
		require.NotNil(t, gs.EventRecords)
		require.Empty(t, gs.EventRecords)
		require.Nil(t, gs.RecordSequences)
	})

	t.Run("default genesis state is valid", func(t *testing.T) {
		t.Parallel()

		gs := types.DefaultGenesisState()

		err := types.Validate(*gs)
		require.NoError(t, err)
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("validates correct genesis state", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			EventRecords:    []types.EventRecord{},
			RecordSequences: []string{"sequence1", "sequence2"},
		}

		err := types.Validate(gs)
		require.NoError(t, err)
	})

	t.Run("validates genesis state with event records", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			EventRecords: []types.EventRecord{
				types.NewEventRecord(
					"0xabcdef",
					1,
					5,
					"0x1234567890123456789012345678901234567890",
					[]byte("test_data"),
					"137",
					time.Now(),
				),
			},
			RecordSequences: []string{"sequence1"},
		}

		err := types.Validate(gs)
		require.NoError(t, err)
	})

	t.Run("rejects empty sequence", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			EventRecords:    []types.EventRecord{},
			RecordSequences: []string{""},
		}

		err := types.Validate(gs)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid sequence")
	})

	t.Run("rejects sequence with empty string in middle", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			EventRecords:    []types.EventRecord{},
			RecordSequences: []string{"seq1", "", "seq3"},
		}

		err := types.Validate(gs)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid sequence")
	})

	t.Run("validates empty record sequences", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			EventRecords:    []types.EventRecord{},
			RecordSequences: nil,
		}

		err := types.Validate(gs)
		require.NoError(t, err)
	})

	t.Run("validates genesis with multiple event records", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		gs := types.GenesisState{
			EventRecords: []types.EventRecord{
				types.NewEventRecord("0xabcdef", 1, 5, "0x1234567890123456789012345678901234567890", []byte("data1"), "137", now),
				types.NewEventRecord("0x123456", 2, 6, "0x0987654321098765432109876543210987654321", []byte("data2"), "80001", now),
			},
			RecordSequences: []string{"seq1", "seq2"},
		}

		err := types.Validate(gs)
		require.NoError(t, err)
	})
}
