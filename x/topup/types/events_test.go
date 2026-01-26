package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/topup/types"
)

func TestAttributeValueCategory(t *testing.T) {
	t.Parallel()

	t.Run("attribute value category matches module name", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, types.ModuleName, types.AttributeValueCategory)
	})
}

func TestEventTypes(t *testing.T) {
	t.Parallel()

	t.Run("EventTypeTopup is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "topup", types.EventTypeTopup)
	})

	t.Run("EventTypeFeeWithdraw is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "fee-withdraw", types.EventTypeFeeWithdraw)
	})

	t.Run("EventTypeWithdraw is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "withdraw", types.EventTypeWithdraw)
	})

	t.Run("all event types are non-empty", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, types.EventTypeTopup)
		require.NotEmpty(t, types.EventTypeFeeWithdraw)
		require.NotEmpty(t, types.EventTypeWithdraw)
	})

	t.Run("all event types are unique", func(t *testing.T) {
		t.Parallel()

		eventTypes := []string{
			types.EventTypeTopup,
			types.EventTypeFeeWithdraw,
			types.EventTypeWithdraw,
		}

		seen := make(map[string]bool)
		for _, et := range eventTypes {
			require.False(t, seen[et], "duplicate event type: %s", et)
			seen[et] = true
		}
	})
}

func TestAttributeKeys(t *testing.T) {
	t.Parallel()

	t.Run("AttributeKeyRecipient is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "recipient", types.AttributeKeyRecipient)
	})

	t.Run("AttributeKeySender is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "sender", types.AttributeKeySender)
	})

	t.Run("AttributeKeyUser is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "user", types.AttributeKeyUser)
	})

	t.Run("AttributeKeyTopupAmount is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "topup-amount", types.AttributeKeyTopupAmount)
	})

	t.Run("AttributeKeyFeeWithdrawAmount is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "fee-withdraw-amount", types.AttributeKeyFeeWithdrawAmount)
	})

	t.Run("all attribute keys are non-empty", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, types.AttributeKeyRecipient)
		require.NotEmpty(t, types.AttributeKeySender)
		require.NotEmpty(t, types.AttributeKeyUser)
		require.NotEmpty(t, types.AttributeKeyTopupAmount)
		require.NotEmpty(t, types.AttributeKeyFeeWithdrawAmount)
	})

	t.Run("all attribute keys are unique", func(t *testing.T) {
		t.Parallel()

		attributeKeys := []string{
			types.AttributeKeyRecipient,
			types.AttributeKeySender,
			types.AttributeKeyUser,
			types.AttributeKeyTopupAmount,
			types.AttributeKeyFeeWithdrawAmount,
		}

		seen := make(map[string]bool)
		for _, key := range attributeKeys {
			require.False(t, seen[key], "duplicate attribute key: %s", key)
			seen[key] = true
		}
	})
}

func TestEventConstants(t *testing.T) {
	t.Parallel()

	t.Run("all constants are properly initialized", func(t *testing.T) {
		t.Parallel()

		constants := []struct {
			name  string
			value string
		}{
			{"AttributeValueCategory", types.AttributeValueCategory},
			{"EventTypeTopup", types.EventTypeTopup},
			{"EventTypeFeeWithdraw", types.EventTypeFeeWithdraw},
			{"EventTypeWithdraw", types.EventTypeWithdraw},
			{"AttributeKeyRecipient", types.AttributeKeyRecipient},
			{"AttributeKeySender", types.AttributeKeySender},
			{"AttributeKeyUser", types.AttributeKeyUser},
			{"AttributeKeyTopupAmount", types.AttributeKeyTopupAmount},
			{"AttributeKeyFeeWithdrawAmount", types.AttributeKeyFeeWithdrawAmount},
		}

		for _, c := range constants {
			require.NotEmpty(t, c.value, "%s should not be empty", c.name)
		}
	})
}
