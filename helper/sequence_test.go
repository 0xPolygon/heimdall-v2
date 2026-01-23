package helper

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"
)

// mockSequenceChecker is a mock implementation of SequenceChecker for testing
type mockSequenceChecker struct {
	sequences map[string]bool
}

func newMockSequenceChecker() *mockSequenceChecker {
	return &mockSequenceChecker{
		sequences: make(map[string]bool),
	}
}

func (m *mockSequenceChecker) HasSequence(_ context.Context, sequence string) bool {
	return m.sequences[sequence]
}

func (m *mockSequenceChecker) addSequence(sequence string) {
	m.sequences[sequence] = true
}

func TestCalculateSequence(t *testing.T) {
	tests := []struct {
		name        string
		blockNumber uint64
		logIndex    uint64
		expected    string
	}{
		{
			name:        "basic calculation",
			blockNumber: 100,
			logIndex:    5,
			expected:    "10000005", // 100 * 100000 + 5
		},
		{
			name:        "zero log index",
			blockNumber: 50,
			logIndex:    0,
			expected:    "5000000", // 50 * 100000 + 0
		},
		{
			name:        "large block number",
			blockNumber: 1000000,
			logIndex:    99999,
			expected:    "100000099999", // 1000000 * 100000 + 99999
		},
		{
			name:        "block 1 log 1",
			blockNumber: 1,
			logIndex:    1,
			expected:    "100001", // 1 * 100000 + 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSequence(tt.blockNumber, tt.logIndex)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckEventAlreadyProcessed(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()

	tests := []struct {
		name             string
		blockNumber      uint64
		logIndex         uint64
		existingSequence string
		moduleName       string
		expected         bool
	}{
		{
			name:             "event not processed",
			blockNumber:      100,
			logIndex:         5,
			existingSequence: "",
			moduleName:       "test",
			expected:         false,
		},
		{
			name:             "event already processed",
			blockNumber:      100,
			logIndex:         5,
			existingSequence: "10000005", // matches calculated sequence
			moduleName:       "test",
			expected:         true,
		},
		{
			name:             "different event exists",
			blockNumber:      100,
			logIndex:         5,
			existingSequence: "10000006", // different sequence
			moduleName:       "test",
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := newMockSequenceChecker()
			if tt.existingSequence != "" {
				checker.addSequence(tt.existingSequence)
			}

			result := CheckEventAlreadyProcessed(
				ctx,
				checker,
				tt.blockNumber,
				tt.logIndex,
				logger,
				tt.moduleName,
			)

			require.Equal(t, tt.expected, result)
		})
	}
}
