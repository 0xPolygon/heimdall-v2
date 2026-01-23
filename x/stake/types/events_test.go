package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestEventTypeValidatorJoin(t *testing.T) {
	t.Parallel()

	t.Run("event type is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "validator-join", types.EventTypeValidatorJoin)
	})
}

func TestEventTypeSignerUpdate(t *testing.T) {
	t.Parallel()

	t.Run("event type is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "signer-update", types.EventTypeSignerUpdate)
	})
}

func TestEventTypeStakeUpdate(t *testing.T) {
	t.Parallel()

	t.Run("event type is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "stake-update", types.EventTypeStakeUpdate)
	})
}

func TestEventTypeValidatorExit(t *testing.T) {
	t.Parallel()

	t.Run("event type is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "validator-exit", types.EventTypeValidatorExit)
	})
}

func TestAttributeKeySigner(t *testing.T) {
	t.Parallel()

	t.Run("attribute key is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "signer", types.AttributeKeySigner)
	})
}

func TestAttributeKeyValidatorID(t *testing.T) {
	t.Parallel()

	t.Run("attribute key is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "validator-id", types.AttributeKeyValidatorID)
	})
}

func TestAttributeKeyValidatorNonce(t *testing.T) {
	t.Parallel()

	t.Run("attribute key is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "validator-nonce", types.AttributeKeyValidatorNonce)
	})
}

func TestAttributeValueCategory(t *testing.T) {
	t.Parallel()

	t.Run("category matches module name", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, types.ModuleName, types.AttributeValueCategory)
	})
}

func TestAllEventTypes_ContainValidPrefix(t *testing.T) {
	t.Parallel()

	t.Run("all event types contain validator prefix", func(t *testing.T) {
		t.Parallel()

		require.Contains(t, types.EventTypeValidatorJoin, "validator")
		require.Contains(t, types.EventTypeValidatorExit, "validator")
	})

	t.Run("all event types contain action suffix", func(t *testing.T) {
		t.Parallel()

		require.Contains(t, types.EventTypeValidatorJoin, "join")
		require.Contains(t, types.EventTypeSignerUpdate, "update")
		require.Contains(t, types.EventTypeStakeUpdate, "update")
		require.Contains(t, types.EventTypeValidatorExit, "exit")
	})
}

func TestAllAttributeKeys_Uniqueness(t *testing.T) {
	t.Parallel()

	t.Run("attribute keys are unique", func(t *testing.T) {
		t.Parallel()

		keys := []string{
			types.AttributeKeySigner,
			types.AttributeKeyValidatorID,
			types.AttributeKeyValidatorNonce,
		}

		seen := make(map[string]bool)
		for _, key := range keys {
			require.False(t, seen[key], "duplicate attribute key: %s", key)
			seen[key] = true
		}
	})
}

func TestAllEventConstants_NotEmpty(t *testing.T) {
	t.Parallel()

	t.Run("all event constants are non-empty", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, types.EventTypeValidatorJoin)
		require.NotEmpty(t, types.EventTypeSignerUpdate)
		require.NotEmpty(t, types.EventTypeStakeUpdate)
		require.NotEmpty(t, types.EventTypeValidatorExit)
		require.NotEmpty(t, types.AttributeKeySigner)
		require.NotEmpty(t, types.AttributeKeyValidatorID)
		require.NotEmpty(t, types.AttributeKeyValidatorNonce)
		require.NotEmpty(t, types.AttributeValueCategory)
	})
}

func TestEventTypes_NoSpaces(t *testing.T) {
	t.Parallel()

	t.Run("event types use hyphens not spaces", func(t *testing.T) {
		t.Parallel()

		require.NotContains(t, types.EventTypeValidatorJoin, " ")
		require.NotContains(t, types.EventTypeSignerUpdate, " ")
		require.NotContains(t, types.EventTypeStakeUpdate, " ")
		require.NotContains(t, types.EventTypeValidatorExit, " ")
	})
}
