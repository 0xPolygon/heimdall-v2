package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/types"
)

func TestEventConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "AttributeKeyTxHash",
			constant: types.AttributeKeyTxHash,
			expected: "txhash",
		},
		{
			name:     "AttributeKeyTxLogIndex",
			constant: types.AttributeKeyTxLogIndex,
			expected: "tx-log-index",
		},
		{
			name:     "AttributeKeySideTxResult",
			constant: types.AttributeKeySideTxResult,
			expected: "side-tx-result",
		},
		{
			name:     "EventTypeFeeTransfer",
			constant: types.EventTypeFeeTransfer,
			expected: "fee-transfer",
		},
		{
			name:     "AttributeKeyProposer",
			constant: types.AttributeKeyProposer,
			expected: "proposer",
		},
		{
			name:     "AttributeKeyDenom",
			constant: types.AttributeKeyDenom,
			expected: "denom",
		},
		{
			name:     "AttributeKeyAmount",
			constant: types.AttributeKeyAmount,
			expected: "amount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.constant)
		})
	}
}

func TestEventConstantsAreNotEmpty(t *testing.T) {
	require.NotEmpty(t, types.AttributeKeyTxHash)
	require.NotEmpty(t, types.AttributeKeyTxLogIndex)
	require.NotEmpty(t, types.AttributeKeySideTxResult)
	require.NotEmpty(t, types.EventTypeFeeTransfer)
	require.NotEmpty(t, types.AttributeKeyProposer)
	require.NotEmpty(t, types.AttributeKeyDenom)
	require.NotEmpty(t, types.AttributeKeyAmount)
}

func TestEventConstantsUniqueness(t *testing.T) {
	// Verify that all attribute key constants are unique
	constants := []string{
		types.AttributeKeyTxHash,
		types.AttributeKeyTxLogIndex,
		types.AttributeKeySideTxResult,
		types.EventTypeFeeTransfer,
		types.AttributeKeyProposer,
		types.AttributeKeyDenom,
		types.AttributeKeyAmount,
	}

	seen := make(map[string]bool)
	for _, c := range constants {
		require.False(t, seen[c], "constant value %s is not unique", c)
		seen[c] = true
	}
}
