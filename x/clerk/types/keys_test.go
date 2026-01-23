package types_test

import (
	"testing"

	"cosmossdk.io/collections"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func TestModuleName(t *testing.T) {
	require.Equal(t, "clerk", types.ModuleName)
}

func TestStoreKey(t *testing.T) {
	require.Equal(t, types.ModuleName, types.StoreKey)
}

func TestRouterKey(t *testing.T) {
	require.Equal(t, types.ModuleName, types.RouterKey)
}

func TestRecordsWithIDKeyPrefix(t *testing.T) {
	require.NotNil(t, types.RecordsWithIDKeyPrefix)
	require.IsType(t, collections.Prefix{}, types.RecordsWithIDKeyPrefix)
}

func TestRecordsWithTimeKeyPrefix(t *testing.T) {
	require.NotNil(t, types.RecordsWithTimeKeyPrefix)
	require.IsType(t, collections.Prefix{}, types.RecordsWithTimeKeyPrefix)
}

func TestRecordSequencesKeyPrefix(t *testing.T) {
	require.NotNil(t, types.RecordSequencesKeyPrefix)
	require.IsType(t, collections.Prefix{}, types.RecordSequencesKeyPrefix)
}

func TestDefaultValue(t *testing.T) {
	require.NotNil(t, types.DefaultValue)
	require.Equal(t, []byte{0x01}, types.DefaultValue)
	require.Len(t, types.DefaultValue, 1)
}

func TestKeyUniqueness(t *testing.T) {
	keys := []collections.Prefix{
		types.RecordsWithIDKeyPrefix,
		types.RecordsWithTimeKeyPrefix,
		types.RecordSequencesKeyPrefix,
	}

	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			require.NotEqual(t, keys[i], keys[j])
		}
	}
}

func TestModuleConstants(t *testing.T) {
	require.Equal(t, types.StoreKey, types.ModuleName)
	require.Equal(t, types.RouterKey, types.ModuleName)
}
