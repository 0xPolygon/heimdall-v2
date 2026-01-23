package types_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

func TestEventTypeMilestone(t *testing.T) {
	t.Parallel()

	t.Run("event type is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "milestone", types.EventTypeMilestone)
	})

	t.Run("event type is not empty", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, types.EventTypeMilestone)
	})
}

func TestNewMilestoneEvent(t *testing.T) {
	t.Parallel()

	t.Run("creates event with all attributes", func(t *testing.T) {
		t.Parallel()

		milestone := types.Milestone{
			Proposer:        "0x1234567890123456789012345678901234567890",
			Hash:            []byte{0x01, 0x02, 0x03, 0x04},
			StartBlock:      100,
			EndBlock:        200,
			BorChainId:      "137",
			MilestoneId:     "milestone-1",
			Timestamp:       1234567890,
			TotalDifficulty: 999,
		}
		milestoneNumber := uint64(42)

		event := types.NewMilestoneEvent(milestone, milestoneNumber)

		require.Equal(t, types.EventTypeMilestone, event.Type)
		require.Len(t, event.Attributes, 9)

		// Check all attributes are present
		attrs := make(map[string]string)
		for _, attr := range event.Attributes {
			attrs[attr.Key] = attr.Value
		}

		require.Equal(t, milestone.Proposer, attrs["proposer"])
		require.Equal(t, hex.EncodeToString(milestone.Hash), attrs["hash"])
		require.Equal(t, "100", attrs["start_block"])
		require.Equal(t, "200", attrs["end_block"])
		require.Equal(t, milestone.BorChainId, attrs["bor_chain_id"])
		require.Equal(t, milestone.MilestoneId, attrs["milestone_id"])
		require.Equal(t, "1234567890", attrs["timestamp"])
		require.Equal(t, "42", attrs["number"])
		require.Equal(t, "999", attrs["total_difficulty"])
	})

	t.Run("creates event with zero values", func(t *testing.T) {
		t.Parallel()

		milestone := types.Milestone{
			Proposer:        "",
			Hash:            []byte{},
			StartBlock:      0,
			EndBlock:        0,
			BorChainId:      "",
			MilestoneId:     "",
			Timestamp:       0,
			TotalDifficulty: 0,
		}
		milestoneNumber := uint64(0)

		event := types.NewMilestoneEvent(milestone, milestoneNumber)

		require.Equal(t, types.EventTypeMilestone, event.Type)
		require.Len(t, event.Attributes, 9)
	})

	t.Run("creates event with large values", func(t *testing.T) {
		t.Parallel()

		milestone := types.Milestone{
			Proposer:        "0xffffffffffffffffffffffffffffffffffffffff",
			Hash:            make([]byte, 32),
			StartBlock:      ^uint64(0) - 1,
			EndBlock:        ^uint64(0),
			BorChainId:      "999999",
			MilestoneId:     "milestone-999",
			Timestamp:       ^uint64(0),
			TotalDifficulty: ^uint64(0),
		}
		milestoneNumber := ^uint64(0)

		event := types.NewMilestoneEvent(milestone, milestoneNumber)

		require.Equal(t, types.EventTypeMilestone, event.Type)
		require.Len(t, event.Attributes, 9)
	})
}
