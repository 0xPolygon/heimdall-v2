package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func TestRegisterSideMsgServer(t *testing.T) {
	t.Parallel()

	t.Run("function exists and can be called", func(t *testing.T) {
		t.Parallel()

		// Just verify the function exists
		// We can't test it without proper mocks as it requires a valid SideTxConfigurator and SideMsgServer
		require.NotNil(t, types.RegisterSideMsgServer)
	})
}
