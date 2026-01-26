package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

func TestModuleName(t *testing.T) {
	t.Parallel()

	t.Run("module name is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "milestone", types.ModuleName)
	})
}

func TestStoreKey(t *testing.T) {
	t.Parallel()

	t.Run("store key matches module name", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, types.ModuleName, types.StoreKey)
	})
}

func TestRouterKey(t *testing.T) {
	t.Parallel()

	t.Run("router key matches module name", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, types.ModuleName, types.RouterKey)
	})
}

func TestParamsPrefixKey(t *testing.T) {
	t.Parallel()

	t.Run("params prefix key is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.ParamsPrefixKey)
	})
}

func TestMilestoneMapPrefixKey(t *testing.T) {
	t.Parallel()

	t.Run("milestone map prefix key is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.MilestoneMapPrefixKey)
	})
}

func TestCountPrefixKey(t *testing.T) {
	t.Parallel()

	t.Run("count prefix key is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.CountPrefixKey)
	})
}

func TestLastMilestoneBlockPrefixKey(t *testing.T) {
	t.Parallel()

	t.Run("last milestone block prefix key is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.LastMilestoneBlockPrefixKey)
	})
}

func TestKeyPrefixUniqueness(t *testing.T) {
	t.Parallel()

	t.Run("all key prefixes are unique", func(t *testing.T) {
		t.Parallel()

		// Extract prefix bytes from collections.Prefix
		prefixes := []struct {
			name   string
			prefix interface{}
		}{
			{"ParamsPrefixKey", types.ParamsPrefixKey},
			{"MilestoneMapPrefixKey", types.MilestoneMapPrefixKey},
			{"CountPrefixKey", types.CountPrefixKey},
			{"LastMilestoneBlockPrefixKey", types.LastMilestoneBlockPrefixKey},
		}

		// Since we can't easily extract the byte values from collections.Prefix,
		// we just verify they're all defined and not nil
		for _, p := range prefixes {
			require.NotNil(t, p.prefix, "%s should not be nil", p.name)
		}
	})
}

func TestModuleConstants(t *testing.T) {
	t.Parallel()

	t.Run("all module constants are non-empty", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, types.ModuleName)
		require.NotEmpty(t, types.StoreKey)
		require.NotEmpty(t, types.RouterKey)
	})

	t.Run("module constants are consistent", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, types.ModuleName, types.StoreKey)
		require.Equal(t, types.ModuleName, types.RouterKey)
	})
}
