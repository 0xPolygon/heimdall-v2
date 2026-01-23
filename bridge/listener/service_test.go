package listener

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListenerService_Constants(t *testing.T) {
	t.Parallel()

	t.Run("verifies listener service constants", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "listener", listenerServiceStr)
		require.Equal(t, "rootchain", rootChainListenerStr)
		require.Equal(t, "heimdall", heimdallListenerStr)
		require.Equal(t, "borchain", borChainListenerStr)
	})

	t.Run("all listener names are unique", func(t *testing.T) {
		t.Parallel()

		names := []string{
			rootChainListenerStr,
			heimdallListenerStr,
			borChainListenerStr,
		}

		seen := make(map[string]bool)
		for _, name := range names {
			require.False(t, seen[name], "duplicate listener name: %s", name)
			seen[name] = true
		}

		require.Len(t, seen, 3)
	})
}

func TestListenerInterface_Methods(t *testing.T) {
	t.Parallel()

	t.Run("verifies listener interface implementation", func(t *testing.T) {
		t.Parallel()

		// Test that our listeners implement the Listener interface
		var _ Listener = (*RootChainListener)(nil)
		var _ Listener = (*BorChainListener)(nil)
		var _ Listener = (*HeimdallListener)(nil)
	})
}
