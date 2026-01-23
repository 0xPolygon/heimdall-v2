package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/types"
)

func TestDefaultLogIndexUnit(t *testing.T) {
	require.Equal(t, 100000, types.DefaultLogIndexUnit)
}

func TestDefaultLogIndexUnitIsPositive(t *testing.T) {
	require.Greater(t, types.DefaultLogIndexUnit, 0)
}

func TestDefaultLogIndexUnitValue(t *testing.T) {
	// Verify the specific value is as expected
	require.Equal(t, 100000, types.DefaultLogIndexUnit)

	// Verify it's a power of 10
	value := types.DefaultLogIndexUnit
	for value > 1 {
		if value%10 != 0 {
			t.Errorf("DefaultLogIndexUnit %d is not a power of 10", types.DefaultLogIndexUnit)
			break
		}
		value /= 10
	}
}

func TestDefaultLogIndexUnitMathOperations(t *testing.T) {
	// Test that the constant can be used in calculations
	unit := types.DefaultLogIndexUnit

	// Example: Calculate sequence ID
	txIndex := uint64(5)
	logIndex := uint64(10)
	sequenceID := txIndex*uint64(unit) + logIndex

	require.Equal(t, uint64(500010), sequenceID)

	// Verify reverse calculation
	extractedTxIndex := sequenceID / uint64(unit)
	extractedLogIndex := sequenceID % uint64(unit)

	require.Equal(t, txIndex, extractedTxIndex)
	require.Equal(t, logIndex, extractedLogIndex)
}

func TestDefaultLogIndexUnitRange(t *testing.T) {
	// Verify that the log index unit allows for sufficient log indices (0-99999)
	maxLogIndex := types.DefaultLogIndexUnit - 1
	require.Equal(t, 99999, maxLogIndex)

	// Verify this provides enough values for typical transaction logs
	require.GreaterOrEqual(t, maxLogIndex, 1000, "Should support at least 1000 logs per transaction")
}
