package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func TestModuleName(t *testing.T) {
	t.Parallel()

	t.Run("module name is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "topup", types.ModuleName)
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

func TestDefaultLogIndexUnit(t *testing.T) {
	t.Parallel()

	t.Run("default log index unit is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, 100000, types.DefaultLogIndexUnit)
	})

	t.Run("default log index unit is greater than zero", func(t *testing.T) {
		t.Parallel()

		require.Greater(t, types.DefaultLogIndexUnit, 0)
	})
}

func TestTopupSequencePrefixKey(t *testing.T) {
	t.Parallel()

	t.Run("topup sequence prefix key is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.TopupSequencePrefixKey)
	})
}

func TestDividendAccountMapKey(t *testing.T) {
	t.Parallel()

	t.Run("dividend account map key is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.DividendAccountMapKey)
	})
}

func TestKeyPrefixUniqueness(t *testing.T) {
	t.Parallel()

	t.Run("all key prefixes are defined", func(t *testing.T) {
		t.Parallel()

		prefixes := []struct {
			name   string
			prefix interface{}
		}{
			{"TopupSequencePrefixKey", types.TopupSequencePrefixKey},
			{"DividendAccountMapKey", types.DividendAccountMapKey},
		}

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
