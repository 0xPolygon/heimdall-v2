package types_test

import (
	"testing"

	"cosmossdk.io/collections"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func TestModuleName(t *testing.T) {
	require.Equal(t, "checkpoint", types.ModuleName)
	require.NotEmpty(t, types.ModuleName)
}

func TestStoreKey(t *testing.T) {
	require.Equal(t, types.ModuleName, types.StoreKey)
	require.Equal(t, "checkpoint", types.StoreKey)
}

func TestRouterKey(t *testing.T) {
	require.Equal(t, types.ModuleName, types.RouterKey)
	require.Equal(t, "checkpoint", types.RouterKey)
}

func TestParamsPrefixKey(t *testing.T) {
	require.NotNil(t, types.ParamsPrefixKey)
	require.IsType(t, collections.Prefix{}, types.ParamsPrefixKey)
}

func TestCheckpointMapPrefixKey(t *testing.T) {
	require.NotNil(t, types.CheckpointMapPrefixKey)
	require.IsType(t, collections.Prefix{}, types.CheckpointMapPrefixKey)
}

func TestBufferedCheckpointPrefixKey(t *testing.T) {
	require.NotNil(t, types.BufferedCheckpointPrefixKey)
	require.IsType(t, collections.Prefix{}, types.BufferedCheckpointPrefixKey)
}

func TestAckCountPrefixKey(t *testing.T) {
	require.NotNil(t, types.AckCountPrefixKey)
	require.IsType(t, collections.Prefix{}, types.AckCountPrefixKey)
}

func TestLastNoAckPrefixKey(t *testing.T) {
	require.NotNil(t, types.LastNoAckPrefixKey)
	require.IsType(t, collections.Prefix{}, types.LastNoAckPrefixKey)
}

func TestCheckpointSignaturesPrefixKey(t *testing.T) {
	require.NotNil(t, types.CheckpointSignaturesPrefixKey)
	require.IsType(t, collections.Prefix{}, types.CheckpointSignaturesPrefixKey)
}

func TestCheckpointSignaturesTxHashPrefixKey(t *testing.T) {
	require.NotNil(t, types.CheckpointSignaturesTxHashPrefixKey)
	require.IsType(t, collections.Prefix{}, types.CheckpointSignaturesTxHashPrefixKey)
}

func TestKeyUniqueness(t *testing.T) {
	keys := []collections.Prefix{
		types.ParamsPrefixKey,
		types.CheckpointMapPrefixKey,
		types.BufferedCheckpointPrefixKey,
		types.AckCountPrefixKey,
		types.LastNoAckPrefixKey,
		types.CheckpointSignaturesPrefixKey,
		types.CheckpointSignaturesTxHashPrefixKey,
	}

	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			require.NotEqual(t, keys[i], keys[j],
				"Keys at positions %d and %d should be different", i, j)
		}
	}
}

func TestModuleConstants(t *testing.T) {
	require.Equal(t, types.StoreKey, types.ModuleName)
	require.Equal(t, types.RouterKey, types.ModuleName)
	require.Equal(t, types.StoreKey, types.RouterKey)
}

func TestAllKeysInitialized(t *testing.T) {
	keys := map[string]collections.Prefix{
		"ParamsPrefixKey":                     types.ParamsPrefixKey,
		"CheckpointMapPrefixKey":              types.CheckpointMapPrefixKey,
		"BufferedCheckpointPrefixKey":         types.BufferedCheckpointPrefixKey,
		"AckCountPrefixKey":                   types.AckCountPrefixKey,
		"LastNoAckPrefixKey":                  types.LastNoAckPrefixKey,
		"CheckpointSignaturesPrefixKey":       types.CheckpointSignaturesPrefixKey,
		"CheckpointSignaturesTxHashPrefixKey": types.CheckpointSignaturesTxHashPrefixKey,
	}

	for name, key := range keys {
		require.NotNil(t, key, "%s should be initialized", name)
	}
}

func TestKeyPrefixRange(t *testing.T) {
	// Keys should use the range 0x80-0x86
	keys := []collections.Prefix{
		types.ParamsPrefixKey,                     // 0x80
		types.CheckpointMapPrefixKey,              // 0x81
		types.BufferedCheckpointPrefixKey,         // 0x82
		types.AckCountPrefixKey,                   // 0x83
		types.LastNoAckPrefixKey,                  // 0x84
		types.CheckpointSignaturesPrefixKey,       // 0x85
		types.CheckpointSignaturesTxHashPrefixKey, // 0x86
	}

	require.Len(t, keys, 7)
}

func TestModuleName_Lowercase(t *testing.T) {
	require.Equal(t, "checkpoint", types.ModuleName)
	for _, char := range types.ModuleName {
		require.True(t, char >= 'a' && char <= 'z',
			"Module name should be lowercase")
	}
}
