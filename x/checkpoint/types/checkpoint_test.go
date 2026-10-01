package types_test

import (
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func TestCreateCheckpoint(t *testing.T) {
	t.Parallel()

	t.Run("creates valid checkpoint from parameters", func(t *testing.T) {
		t.Parallel()

		id := uint64(1)
		proposer := "0x1234567890123456789012345678901234567890"
		startBlock := uint64(1000)
		endBlock := uint64(2000)
		rootHash := common.HexToHash("0xabcdef").Bytes()
		borChainId := "137"
		timestamp := uint64(1234567890)

		checkpoint := types.CreateCheckpoint(id, startBlock, endBlock, rootHash, proposer, borChainId, timestamp)

		require.Equal(t, id, checkpoint.Id)
		require.Equal(t, proposer, checkpoint.Proposer)
		require.Equal(t, startBlock, checkpoint.StartBlock)
		require.Equal(t, endBlock, checkpoint.EndBlock)
		require.Equal(t, rootHash, checkpoint.RootHash)
		require.Equal(t, borChainId, checkpoint.BorChainId)
		require.Equal(t, timestamp, checkpoint.Timestamp)
	})

	t.Run("creates checkpoint with empty root hash", func(t *testing.T) {
		t.Parallel()

		id := uint64(2)
		proposer := "0x1234567890123456789012345678901234567890"
		var rootHash []byte
		borChainId := "137"
		timestamp := uint64(1234567890)

		checkpoint := types.CreateCheckpoint(id, 1000, 2000, rootHash, proposer, borChainId, timestamp)

		require.Equal(t, id, checkpoint.Id)
		require.Empty(t, checkpoint.RootHash)
	})

	t.Run("creates checkpoint with zero timestamp", func(t *testing.T) {
		t.Parallel()

		id := uint64(3)
		proposer := "0x1234567890123456789012345678901234567890"
		rootHash := common.HexToHash("0xabcdef").Bytes()
		borChainId := "137"
		timestamp := uint64(0)

		checkpoint := types.CreateCheckpoint(id, 1000, 2000, rootHash, proposer, borChainId, timestamp)

		require.Equal(t, uint64(0), checkpoint.Timestamp)
	})

	t.Run("creates checkpoint with different bor chain ids", func(t *testing.T) {
		t.Parallel()

		chainIds := []string{"137", "80001", "1", "999"}
		rootHash := common.HexToHash("0xabcdef").Bytes()
		proposer := "0x1234567890123456789012345678901234567890"

		for i, chainId := range chainIds {
			checkpoint := types.CreateCheckpoint(uint64(i), 1000, 2000, rootHash, proposer, chainId, 123456)
			require.Equal(t, chainId, checkpoint.BorChainId)
		}
	})
}

func TestSortCheckpoints(t *testing.T) {
	t.Parallel()

	t.Run("sorts checkpoints by timestamp ascending", func(t *testing.T) {
		t.Parallel()

		proposer := "0x1234567890123456789012345678901234567890"
		rootHash := common.HexToHash("0xabcdef").Bytes()
		borChainId := "137"

		// Create checkpoints out of order by timestamp
		cp1 := types.CreateCheckpoint(1, 1000, 2000, rootHash, proposer, borChainId, 3000)
		cp2 := types.CreateCheckpoint(2, 3000, 4000, rootHash, proposer, borChainId, 1000)
		cp3 := types.CreateCheckpoint(3, 5000, 6000, rootHash, proposer, borChainId, 2000)

		checkpoints := []types.Checkpoint{cp1, cp2, cp3}

		sorted := types.SortCheckpoints(checkpoints)

		require.Equal(t, uint64(1000), sorted[0].Timestamp)
		require.Equal(t, uint64(2000), sorted[1].Timestamp)
		require.Equal(t, uint64(3000), sorted[2].Timestamp)
	})

	t.Run("handles empty checkpoint list", func(t *testing.T) {
		t.Parallel()

		var checkpoints []types.Checkpoint

		sorted := types.SortCheckpoints(checkpoints)

		require.Empty(t, sorted)
	})

	t.Run("handles single checkpoint", func(t *testing.T) {
		t.Parallel()

		proposer := "0x1234567890123456789012345678901234567890"
		rootHash := common.HexToHash("0xabcdef").Bytes()
		borChainId := "137"

		cp := types.CreateCheckpoint(1, 1000, 2000, rootHash, proposer, borChainId, 1234)
		checkpoints := []types.Checkpoint{cp}

		sorted := types.SortCheckpoints(checkpoints)

		require.Len(t, sorted, 1)
		require.Equal(t, uint64(1234), sorted[0].Timestamp)
	})

	t.Run("maintains order for already sorted checkpoints", func(t *testing.T) {
		t.Parallel()

		proposer := "0x1234567890123456789012345678901234567890"
		rootHash := common.HexToHash("0xabcdef").Bytes()
		borChainId := "137"

		cp1 := types.CreateCheckpoint(1, 1000, 2000, rootHash, proposer, borChainId, 1000)
		cp2 := types.CreateCheckpoint(2, 3000, 4000, rootHash, proposer, borChainId, 2000)
		cp3 := types.CreateCheckpoint(3, 5000, 6000, rootHash, proposer, borChainId, 3000)

		checkpoints := []types.Checkpoint{cp1, cp2, cp3}

		sorted := types.SortCheckpoints(checkpoints)

		require.Equal(t, uint64(1000), sorted[0].Timestamp)
		require.Equal(t, uint64(2000), sorted[1].Timestamp)
		require.Equal(t, uint64(3000), sorted[2].Timestamp)
	})

	t.Run("sorts checkpoints with same timestamp", func(t *testing.T) {
		t.Parallel()

		proposer := "0x1234567890123456789012345678901234567890"
		rootHash := common.HexToHash("0xabcdef").Bytes()
		borChainId := "137"
		timestamp := uint64(1234)

		cp1 := types.CreateCheckpoint(1, 1000, 2000, rootHash, proposer, borChainId, timestamp)
		cp2 := types.CreateCheckpoint(2, 3000, 4000, rootHash, proposer, borChainId, timestamp)

		checkpoints := []types.Checkpoint{cp1, cp2}

		sorted := types.SortCheckpoints(checkpoints)

		require.Len(t, sorted, 2)
		// Order should be preserved for the same timestamp
		require.Equal(t, timestamp, sorted[0].Timestamp)
		require.Equal(t, timestamp, sorted[1].Timestamp)
	})
}

func TestValidateCheckpointLength(t *testing.T) {
	t.Parallel()

	const (
		avg = uint64(256)
		max = uint64(8192)
	)

	tests := []struct {
		name       string
		start, end uint64
		avg, max   uint64
		wantErr    string
	}{
		{name: "single avg window from zero", start: 0, end: avg - 1, avg: avg, max: max},
		{name: "single avg window", start: 1000, end: 1000 + avg - 1, avg: avg, max: max},
		{name: "multiple of avg", start: 1000, end: 1000 + 4*avg - 1, avg: avg, max: max},
		{name: "exactly max", start: 1000, end: 1000 + max - 1, avg: avg, max: max},
		{name: "max not a multiple of avg", start: 1000, end: 1000 + 300 - 1, avg: avg, max: 300},
		{name: "avg equals max", start: 1000, end: 1000 + avg - 1, avg: avg, max: avg},

		{name: "one block", start: 1000, end: 1000, avg: avg, max: max, wantErr: "not a multiple"},
		{name: "one over avg", start: 1000, end: 1000 + avg, avg: avg, max: max, wantErr: "not a multiple"},
		{name: "one under avg", start: 1000, end: 1000 + avg - 2, avg: avg, max: max, wantErr: "not a multiple"},
		{name: "one over max", start: 1000, end: 1000 + max, avg: avg, max: max, wantErr: "window 1000-9192 is longer than the maximum 8192 blocks"},
		{name: "aligned but over max", start: 0, end: 64*avg - 1, avg: avg, max: max, wantErr: "longer than the maximum"},
		{name: "end before start", start: 1000, end: 999, avg: avg, max: max, wantErr: "precedes start block"},
		{name: "full uint64 range", start: 0, end: math.MaxUint64, avg: avg, max: max, wantErr: "longer than the maximum"},
		{name: "span one short of wrapping", start: 1, end: math.MaxUint64, avg: avg, max: max, wantErr: "longer than the maximum"},
		{name: "zero avg", start: 1000, end: 1000 + avg - 1, avg: 0, max: max, wantErr: "avg length is zero"},
		{name: "avg equals max, one block", start: 1000, end: 1000, avg: avg, max: avg, wantErr: "not a multiple"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := types.ValidateCheckpointLength(tt.start, tt.end, tt.avg, tt.max)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, types.ErrInvalidCheckpointLength)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// The windows the bridge derives from params are the ones the chain has to keep
// accepting: a whole number of avg blocks, or the max cap.
func TestValidateCheckpointLengthAcceptsBridgeWindows(t *testing.T) {
	t.Parallel()

	// Params validation only requires max >= avg, so avg == max is a legal configuration.
	params := []struct{ avg, max uint64 }{
		{256, 8192},
		{8, 1024},
		{256, 1024},
		{256, 300},
		{256, 256},
		{2, 2},
	}

	start := uint64(93_000_000)
	for _, p := range params {
		for available := uint64(1); available <= 40*p.avg; available++ {
			expectedDiff := available - available%p.avg
			if expectedDiff > 0 {
				expectedDiff--
			}
			if expectedDiff > p.max {
				expectedDiff = p.max - 1
			}
			end := start + expectedDiff
			if start >= end {
				continue // the bridge doesn't broadcast this one
			}
			require.NoError(t, types.ValidateCheckpointLength(start, end, p.avg, p.max),
				"avg=%d max=%d available=%d length=%d", p.avg, p.max, available, end-start+1)
		}
	}
}
