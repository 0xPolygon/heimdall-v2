package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

func TestErrNoMilestoneFound(t *testing.T) {
	t.Parallel()

	t.Run("error is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.ErrNoMilestoneFound)
	})

	t.Run("error message contains expected text", func(t *testing.T) {
		t.Parallel()

		require.Contains(t, types.ErrNoMilestoneFound.Error(), "milestone not found")
	})

	t.Run("error has correct code", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, uint32(3), types.ErrNoMilestoneFound.ABCICode())
	})

	t.Run("error wrapping works", func(t *testing.T) {
		t.Parallel()

		wrapped := types.ErrNoMilestoneFound.Wrap("additional context")
		require.Contains(t, wrapped.Error(), "milestone not found")
		require.Contains(t, wrapped.Error(), "additional context")
	})
}

func TestErrorCodes(t *testing.T) {
	t.Parallel()

	t.Run("all error codes are unique", func(t *testing.T) {
		t.Parallel()

		errors := []struct {
			name string
			err  error
		}{
			{"ErrNoMilestoneFound", types.ErrNoMilestoneFound},
		}

		seen := make(map[uint32]string)
		for _, e := range errors {
			code := types.ErrNoMilestoneFound.ABCICode()
			if existing, ok := seen[code]; ok {
				t.Errorf("duplicate error code %d for %s and %s", code, e.name, existing)
			}
			seen[code] = e.name
		}
	})
}
