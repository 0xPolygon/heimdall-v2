package types_test

import (
	"testing"

	"cosmossdk.io/collections"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

func TestModuleName(t *testing.T) {
	require.Equal(t, "chainmanager", types.ModuleName)
	require.NotEmpty(t, types.ModuleName)
}

func TestStoreKey(t *testing.T) {
	require.Equal(t, types.ModuleName, types.StoreKey)
	require.Equal(t, "chainmanager", types.StoreKey)
}

func TestRouterKey(t *testing.T) {
	require.Equal(t, types.ModuleName, types.RouterKey)
	require.Equal(t, "chainmanager", types.RouterKey)
}

func TestParamsKey(t *testing.T) {
	require.NotNil(t, types.ParamsKey)
	require.IsType(t, collections.Prefix{}, types.ParamsKey)
}

func TestModuleConstants(t *testing.T) {
	// Test that module constants are consistent
	require.Equal(t, types.StoreKey, types.ModuleName)
	require.Equal(t, types.RouterKey, types.ModuleName)
	require.Equal(t, types.StoreKey, types.RouterKey)
}

func TestKeyConstants_NonEmpty(t *testing.T) {
	require.NotEmpty(t, types.ModuleName)
	require.NotEmpty(t, types.StoreKey)
	require.NotEmpty(t, types.RouterKey)
}

func TestModuleName_Lowercase(t *testing.T) {
	require.Equal(t, "chainmanager", types.ModuleName)
	// Verify it's all lowercase
	for _, char := range types.ModuleName {
		require.True(t, char >= 'a' && char <= 'z',
			"Module name should be lowercase")
	}
}

func TestParamsKey_Initialized(t *testing.T) {
	require.NotNil(t, types.ParamsKey)
}
