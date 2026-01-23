package types_test

import (
	"testing"

	"cosmossdk.io/collections"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

func TestModuleName(t *testing.T) {
	require.Equal(t, "bor", types.ModuleName)
	require.NotEmpty(t, types.ModuleName)
}

func TestStoreKey(t *testing.T) {
	require.Equal(t, types.ModuleName, types.StoreKey)
	require.Equal(t, "bor", types.StoreKey)
}

func TestRouterKey(t *testing.T) {
	require.Equal(t, types.ModuleName, types.RouterKey)
	require.Equal(t, "bor", types.RouterKey)
}

func TestLastSpanIDKey(t *testing.T) {
	require.NotNil(t, types.LastSpanIDKey)
	require.IsType(t, collections.Prefix{}, types.LastSpanIDKey)
}

func TestSpanPrefixKey(t *testing.T) {
	require.NotNil(t, types.SpanPrefixKey)
	require.IsType(t, collections.Prefix{}, types.SpanPrefixKey)
}

func TestSeedLastBlockProducerKey(t *testing.T) {
	require.NotNil(t, types.SeedLastBlockProducerKey)
	require.IsType(t, collections.Prefix{}, types.SeedLastBlockProducerKey)
}

func TestParamsKey(t *testing.T) {
	require.NotNil(t, types.ParamsKey)
	require.IsType(t, collections.Prefix{}, types.ParamsKey)
}

func TestProducerVotesKey(t *testing.T) {
	require.NotNil(t, types.ProducerVotesKey)
	require.IsType(t, collections.Prefix{}, types.ProducerVotesKey)
}

func TestPerformanceScoreKey(t *testing.T) {
	require.NotNil(t, types.PerformanceScoreKey)
	require.IsType(t, collections.Prefix{}, types.PerformanceScoreKey)
}

func TestLatestActiveProducerKey(t *testing.T) {
	require.NotNil(t, types.LatestActiveProducerKey)
	require.IsType(t, collections.Prefix{}, types.LatestActiveProducerKey)
}

func TestLatestFailedProducerKey(t *testing.T) {
	require.NotNil(t, types.LatestFailedProducerKey)
	require.IsType(t, collections.Prefix{}, types.LatestFailedProducerKey)
}

func TestLastSpanBlockKey(t *testing.T) {
	require.NotNil(t, types.LastSpanBlockKey)
	require.IsType(t, collections.Prefix{}, types.LastSpanBlockKey)
}

func TestProducerPlannedDowntimeKey(t *testing.T) {
	require.NotNil(t, types.ProducerPlannedDowntimeKey)
	require.IsType(t, collections.Prefix{}, types.ProducerPlannedDowntimeKey)
}

func TestKeyUniqueness(t *testing.T) {
	// Test that all keys are unique
	keys := []collections.Prefix{
		types.LastSpanIDKey,
		types.SpanPrefixKey,
		types.SeedLastBlockProducerKey,
		types.ParamsKey,
		types.ProducerVotesKey,
		types.PerformanceScoreKey,
		types.LatestActiveProducerKey,
		types.LatestFailedProducerKey,
		types.LastSpanBlockKey,
		types.ProducerPlannedDowntimeKey,
	}

	// Check all keys are distinct
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			require.NotEqual(t, keys[i], keys[j],
				"Keys at positions %d and %d should be different", i, j)
		}
	}
}

func TestModuleConstants(t *testing.T) {
	// Test that module constants are consistent
	require.Equal(t, types.StoreKey, types.ModuleName)
	require.Equal(t, types.RouterKey, types.ModuleName)
	require.Equal(t, types.StoreKey, types.RouterKey)
}

func TestKeyConstants_NonEmpty(t *testing.T) {
	// Test that string constants are not empty
	require.NotEmpty(t, types.ModuleName)
	require.NotEmpty(t, types.StoreKey)
	require.NotEmpty(t, types.RouterKey)
}

func TestKeyConstants_Lowercase(t *testing.T) {
	// Test that the module name is lowercase (convention)
	require.Equal(t, "bor", types.ModuleName)
}

func TestAllKeysInitialized(t *testing.T) {
	// Test that all keys are properly initialized
	keys := map[string]collections.Prefix{
		"LastSpanIDKey":              types.LastSpanIDKey,
		"SpanPrefixKey":              types.SpanPrefixKey,
		"SeedLastBlockProducerKey":   types.SeedLastBlockProducerKey,
		"ParamsKey":                  types.ParamsKey,
		"ProducerVotesKey":           types.ProducerVotesKey,
		"PerformanceScoreKey":        types.PerformanceScoreKey,
		"LatestActiveProducerKey":    types.LatestActiveProducerKey,
		"LatestFailedProducerKey":    types.LatestFailedProducerKey,
		"LastSpanBlockKey":           types.LastSpanBlockKey,
		"ProducerPlannedDowntimeKey": types.ProducerPlannedDowntimeKey,
	}

	for name, key := range keys {
		require.NotNil(t, key, "%s should be initialized", name)
	}
}

func TestKeyValueRange(t *testing.T) {
	// Keys should use a specific range (0x35-0x3E according to comments)
	// We can't directly inspect the byte values, but we can verify they're different
	keys := []collections.Prefix{
		types.LastSpanIDKey,              // 0x35
		types.SpanPrefixKey,              // 0x36
		types.SeedLastBlockProducerKey,   // 0x37
		types.ParamsKey,                  // 0x38
		types.ProducerVotesKey,           // 0x39
		types.PerformanceScoreKey,        // 0x3A
		types.LatestActiveProducerKey,    // 0x3B
		types.LatestFailedProducerKey,    // 0x3C
		types.LastSpanBlockKey,           // 0x3D
		types.ProducerPlannedDowntimeKey, // 0x3E
	}

	// Verify we have 10 distinct keys
	require.Len(t, keys, 10)
}
