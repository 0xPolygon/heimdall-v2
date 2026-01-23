package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestSortValidatorByAddress(t *testing.T) {
	t.Parallel()

	t.Run("sorts validators by signer address ascending", func(t *testing.T) {
		t.Parallel()

		validators := []staketypes.Validator{
			{Signer: "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"},
			{Signer: "0x0000000000000000000000000000000000000001"},
			{Signer: "0x5555555555555555555555555555555555555555"},
		}

		sorted := types.SortValidatorByAddress(validators)

		require.Len(t, sorted, 3)
		require.Equal(t, "0x0000000000000000000000000000000000000001", sorted[0].Signer)
		require.Equal(t, "0x5555555555555555555555555555555555555555", sorted[1].Signer)
		require.Equal(t, "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", sorted[2].Signer)
	})

	t.Run("handles empty validator list", func(t *testing.T) {
		t.Parallel()

		var validators []staketypes.Validator
		sorted := types.SortValidatorByAddress(validators)

		require.Empty(t, sorted)
	})

	t.Run("handles single validator", func(t *testing.T) {
		t.Parallel()

		validators := []staketypes.Validator{
			{Signer: "0x1234567890123456789012345678901234567890"},
		}

		sorted := types.SortValidatorByAddress(validators)

		require.Len(t, sorted, 1)
		require.Equal(t, "0x1234567890123456789012345678901234567890", sorted[0].Signer)
	})

	t.Run("maintains sort order for already sorted validators", func(t *testing.T) {
		t.Parallel()

		validators := []staketypes.Validator{
			{Signer: "0x0000000000000000000000000000000000000001"},
			{Signer: "0x0000000000000000000000000000000000000002"},
			{Signer: "0x0000000000000000000000000000000000000003"},
		}

		sorted := types.SortValidatorByAddress(validators)

		require.Len(t, sorted, 3)
		require.Equal(t, "0x0000000000000000000000000000000000000001", sorted[0].Signer)
		require.Equal(t, "0x0000000000000000000000000000000000000002", sorted[1].Signer)
		require.Equal(t, "0x0000000000000000000000000000000000000003", sorted[2].Signer)
	})
}

func TestSortSpansById(t *testing.T) {
	t.Parallel()

	t.Run("sorts spans by ID ascending", func(t *testing.T) {
		t.Parallel()

		spans := []types.Span{
			{Id: 100, StartBlock: 10000, EndBlock: 20000},
			{Id: 1, StartBlock: 1000, EndBlock: 2000},
			{Id: 50, StartBlock: 5000, EndBlock: 6000},
		}

		types.SortSpansById(spans)

		require.Len(t, spans, 3)
		require.Equal(t, uint64(1), spans[0].Id)
		require.Equal(t, uint64(50), spans[1].Id)
		require.Equal(t, uint64(100), spans[2].Id)
	})

	t.Run("handles empty span list", func(t *testing.T) {
		t.Parallel()

		var spans []types.Span
		types.SortSpansById(spans)

		require.Empty(t, spans)
	})

	t.Run("handles single span", func(t *testing.T) {
		t.Parallel()

		spans := []types.Span{
			{Id: 42, StartBlock: 1000, EndBlock: 2000},
		}

		types.SortSpansById(spans)

		require.Len(t, spans, 1)
		require.Equal(t, uint64(42), spans[0].Id)
	})

	t.Run("maintains sort order for already sorted spans", func(t *testing.T) {
		t.Parallel()

		spans := []types.Span{
			{Id: 1, StartBlock: 1000, EndBlock: 2000},
			{Id: 2, StartBlock: 2000, EndBlock: 3000},
			{Id: 3, StartBlock: 3000, EndBlock: 4000},
		}

		types.SortSpansById(spans)

		require.Len(t, spans, 3)
		require.Equal(t, uint64(1), spans[0].Id)
		require.Equal(t, uint64(2), spans[1].Id)
		require.Equal(t, uint64(3), spans[2].Id)
	})
}

func TestIsBlockCloseToSpanEnd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		blockNumber uint64
		spanEnd     uint64
		want        bool
	}{
		{
			name:        "block exactly at span end",
			blockNumber: 1000,
			spanEnd:     1000,
			want:        true,
		},
		{
			name:        "block 50 blocks before span end",
			blockNumber: 950,
			spanEnd:     1000,
			want:        true,
		},
		{
			name:        "block 99 blocks before span end",
			blockNumber: 901,
			spanEnd:     1000,
			want:        true,
		},
		{
			name:        "block 100 blocks before span end (edge case - inclusive)",
			blockNumber: 900,
			spanEnd:     1000,
			want:        true, // 900 >= (1000-100) = 900, so it's within range
		},
		{
			name:        "block 101 blocks before span end",
			blockNumber: 899,
			spanEnd:     1000,
			want:        false,
		},
		{
			name:        "block after span end",
			blockNumber: 1001,
			spanEnd:     1000,
			want:        false,
		},
		{
			name:        "block far before span end",
			blockNumber: 500,
			spanEnd:     1000,
			want:        false,
		},
		{
			name:        "block at zero span end",
			blockNumber: 0,
			spanEnd:     0,
			want:        false, // Due to uint64 underflow: 0 >= (0-100) = 0 >= MAX_UINT64 is false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := types.IsBlockCloseToSpanEnd(tt.blockNumber, tt.spanEnd)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateBorCommittedSpans(t *testing.T) {
	t.Parallel()

	t.Run("generates single span when bor block is one span ahead", func(t *testing.T) {
		t.Parallel()

		latestBorUsedSpan := &types.Span{
			Id:         1,
			StartBlock: 0,
			EndBlock:   6400,
			BorChainId: "137",
			SelectedProducers: []staketypes.Validator{
				{ValId: 1, Signer: "0x1234"},
			},
			ValidatorSet: staketypes.ValidatorSet{
				Validators: []*staketypes.Validator{
					{ValId: 1},
				},
			},
		}

		latestBorBlock := uint64(8000) // Beyond the first span

		spans := types.GenerateBorCommittedSpans(latestBorBlock, latestBorUsedSpan)

		require.Len(t, spans, 1)
		require.Equal(t, uint64(2), spans[0].Id)
		require.Equal(t, uint64(6401), spans[0].StartBlock)
		require.Equal(t, uint64(12801), spans[0].EndBlock)
		require.Equal(t, "137", spans[0].BorChainId)
		require.Equal(t, latestBorUsedSpan.SelectedProducers, spans[0].SelectedProducers)
		require.Equal(t, latestBorUsedSpan.ValidatorSet, spans[0].ValidatorSet)
	})

	t.Run("generates multiple spans when bor block is multiple spans ahead", func(t *testing.T) {
		t.Parallel()

		latestBorUsedSpan := &types.Span{
			Id:         1,
			StartBlock: 0,
			EndBlock:   1000,
			BorChainId: "137",
		}

		latestBorBlock := uint64(3500) // 2+ spans ahead

		spans := types.GenerateBorCommittedSpans(latestBorBlock, latestBorUsedSpan)

		require.GreaterOrEqual(t, len(spans), 2)
		require.Equal(t, uint64(2), spans[0].Id)
		require.Equal(t, uint64(1001), spans[0].StartBlock)
	})

	// Note: GenerateBorCommittedSpans expects the latestBorBlock > latestBorUsedSpan.EndBlock
	// Testing with the latestBorBlock <= EndBlock would cause underflow in the function
	t.Run("returns empty slice when bor block equals span end", func(t *testing.T) {
		t.Parallel()

		latestBorUsedSpan := &types.Span{
			Id:         1,
			StartBlock: 0,
			EndBlock:   1000,
			BorChainId: "137",
		}

		latestBorBlock := uint64(1000)

		spans := types.GenerateBorCommittedSpans(latestBorBlock, latestBorUsedSpan)

		require.Empty(t, spans)
	})

	t.Run("preserves validator set and producers across generated spans", func(t *testing.T) {
		t.Parallel()

		latestBorUsedSpan := &types.Span{
			Id:         5,
			StartBlock: 1000,
			EndBlock:   2000,
			BorChainId: "80001",
			SelectedProducers: []staketypes.Validator{
				{ValId: 1, Signer: "0xaaa"},
				{ValId: 2, Signer: "0xbbb"},
			},
			ValidatorSet: staketypes.ValidatorSet{
				Validators: []*staketypes.Validator{
					{ValId: 1},
					{ValId: 2},
				},
			},
		}

		latestBorBlock := uint64(3500)

		spans := types.GenerateBorCommittedSpans(latestBorBlock, latestBorUsedSpan)

		require.NotEmpty(t, spans)
		for _, span := range spans {
			require.Equal(t, latestBorUsedSpan.BorChainId, span.BorChainId)
			require.Equal(t, latestBorUsedSpan.SelectedProducers, span.SelectedProducers)
			require.Equal(t, latestBorUsedSpan.ValidatorSet, span.ValidatorSet)
		}
	})
}

func TestCalcCurrentBorSpanId(t *testing.T) {
	t.Parallel()

	t.Run("returns span ID when bor block is within heimdall span", func(t *testing.T) {
		t.Parallel()

		heimdallSpan := &types.Span{
			Id:         10,
			StartBlock: 1000,
			EndBlock:   2000,
		}

		latestBorBlock := uint64(1500)

		spanID, err := types.CalcCurrentBorSpanId(latestBorBlock, heimdallSpan)

		require.NoError(t, err)
		require.Equal(t, uint64(10), spanID)
	})

	t.Run("returns span ID when bor block equals span end", func(t *testing.T) {
		t.Parallel()

		heimdallSpan := &types.Span{
			Id:         10,
			StartBlock: 1000,
			EndBlock:   2000,
		}

		latestBorBlock := uint64(2000)

		spanID, err := types.CalcCurrentBorSpanId(latestBorBlock, heimdallSpan)

		require.NoError(t, err)
		require.Equal(t, uint64(10), spanID)
	})

	t.Run("returns span ID when bor block equals span start", func(t *testing.T) {
		t.Parallel()

		heimdallSpan := &types.Span{
			Id:         10,
			StartBlock: 1000,
			EndBlock:   2000,
		}

		latestBorBlock := uint64(1000)

		spanID, err := types.CalcCurrentBorSpanId(latestBorBlock, heimdallSpan)

		require.NoError(t, err)
		require.Equal(t, uint64(10), spanID)
	})

	t.Run("calculates next span ID when bor block is beyond heimdall span", func(t *testing.T) {
		t.Parallel()

		heimdallSpan := &types.Span{
			Id:         10,
			StartBlock: 1000,
			EndBlock:   2000, // span length = 1001
		}

		latestBorBlock := uint64(2500) // halfway into the next span

		spanID, err := types.CalcCurrentBorSpanId(latestBorBlock, heimdallSpan)

		require.NoError(t, err)
		require.Equal(t, uint64(11), spanID)
	})

	t.Run("calculates span ID multiple spans ahead", func(t *testing.T) {
		t.Parallel()

		heimdallSpan := &types.Span{
			Id:         1,
			StartBlock: 0,
			EndBlock:   999, // span length = 1000
		}

		latestBorBlock := uint64(3500) // 3+ spans ahead

		spanID, err := types.CalcCurrentBorSpanId(latestBorBlock, heimdallSpan)

		require.NoError(t, err)
		require.Equal(t, uint64(4), spanID) // spans 1,2,3,4
	})

	t.Run("returns error when heimdall span is nil", func(t *testing.T) {
		t.Parallel()

		latestBorBlock := uint64(1000)

		spanID, err := types.CalcCurrentBorSpanId(latestBorBlock, nil)

		require.Error(t, err)
		require.Contains(t, err.Error(), "nil Heimdall span")
		require.Equal(t, uint64(0), spanID)
	})

	t.Run("returns error when heimdall span has invalid range", func(t *testing.T) {
		t.Parallel()

		heimdallSpan := &types.Span{
			Id:         10,
			StartBlock: 2000,
			EndBlock:   1000, // end < start
		}

		latestBorBlock := uint64(1500)

		spanID, err := types.CalcCurrentBorSpanId(latestBorBlock, heimdallSpan)

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid Heimdall span")
		require.Equal(t, uint64(0), spanID)
	})

	t.Run("returns error when bor block is before span start", func(t *testing.T) {
		t.Parallel()

		heimdallSpan := &types.Span{
			Id:         10,
			StartBlock: 1000,
			EndBlock:   2000,
		}

		latestBorBlock := uint64(500) // before span start

		spanID, err := types.CalcCurrentBorSpanId(latestBorBlock, heimdallSpan)

		require.Error(t, err)
		require.Contains(t, err.Error(), "latestBorBlock")
		require.Contains(t, err.Error(), "must be >= Heimdall span StartBlock")
		require.Equal(t, uint64(0), spanID)
	})

	t.Run("handles span starting at block zero", func(t *testing.T) {
		t.Parallel()

		heimdallSpan := &types.Span{
			Id:         0,
			StartBlock: 0,
			EndBlock:   1000,
		}

		latestBorBlock := uint64(500)

		spanID, err := types.CalcCurrentBorSpanId(latestBorBlock, heimdallSpan)

		require.NoError(t, err)
		require.Equal(t, uint64(0), spanID)
	})
}

func TestLogSpan(t *testing.T) {
	t.Parallel()

	t.Run("formats span with selected producers", func(t *testing.T) {
		t.Parallel()

		span := &types.Span{
			Id:         42,
			StartBlock: 1000,
			EndBlock:   2000,
			BorChainId: "137",
			SelectedProducers: []staketypes.Validator{
				{ValId: 1, Signer: "0xaaa"},
				{ValId: 2, Signer: "0xbbb"},
			},
			ValidatorSet: staketypes.ValidatorSet{
				Validators: []*staketypes.Validator{
					{ValId: 1},
					{ValId: 2},
				},
			},
		}

		logStr := span.LogSpan()

		require.Contains(t, logStr, "id=42")
		require.Contains(t, logStr, "startBlock=1000")
		require.Contains(t, logStr, "endBlock=2000")
		require.Contains(t, logStr, "borChainId=137")
		require.Contains(t, logStr, "validatorCount=2")
		require.Contains(t, logStr, "valID=1")
		require.Contains(t, logStr, "valID=2")
	})

	t.Run("formats span without selected producers", func(t *testing.T) {
		t.Parallel()

		span := &types.Span{
			Id:                42,
			StartBlock:        1000,
			EndBlock:          2000,
			BorChainId:        "137",
			SelectedProducers: []staketypes.Validator{},
			ValidatorSet: staketypes.ValidatorSet{
				Validators: []*staketypes.Validator{},
			},
		}

		logStr := span.LogSpan()

		require.Contains(t, logStr, "id=42")
		require.Contains(t, logStr, "validatorCount=0")
		require.Contains(t, logStr, "selectedProducers=[]")
	})

	t.Run("returns nil string for nil span", func(t *testing.T) {
		t.Parallel()

		var span *types.Span = nil

		logStr := span.LogSpan()

		require.Equal(t, "nil", logStr)
	})

	t.Run("handles span with zero values", func(t *testing.T) {
		t.Parallel()

		span := &types.Span{
			Id:         0,
			StartBlock: 0,
			EndBlock:   0,
			BorChainId: "",
		}

		logStr := span.LogSpan()

		require.Contains(t, logStr, "id=0")
		require.Contains(t, logStr, "startBlock=0")
		require.Contains(t, logStr, "endBlock=0")
		require.Contains(t, logStr, "borChainId=")
	})
}
