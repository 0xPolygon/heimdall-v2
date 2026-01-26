package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestErrInvalidMsg(t *testing.T) {
	t.Parallel()

	t.Run("error message is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.ErrInvalidMsg)
		require.Contains(t, types.ErrInvalidMsg.Error(), "invalid message")
	})

	t.Run("error can be compared", func(t *testing.T) {
		t.Parallel()

		testErr := types.ErrInvalidMsg
		require.Equal(t, types.ErrInvalidMsg, testErr)
	})
}

func TestErrNoValidator(t *testing.T) {
	t.Parallel()

	t.Run("error message is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.ErrNoValidator)
		require.Contains(t, types.ErrNoValidator.Error(), "no respective validator found")
	})
}

func TestErrNoSignerChange(t *testing.T) {
	t.Parallel()

	t.Run("error message is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.ErrNoSignerChange)
		require.Contains(t, types.ErrNoSignerChange.Error(), "new signer is same as old one")
	})
}

func TestErrValUnBonded(t *testing.T) {
	t.Parallel()

	t.Run("error message is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.ErrValUnBonded)
		require.Contains(t, types.ErrValUnBonded.Error(), "validator already unBonded")
	})
}

func TestAllErrorsAreErrors(t *testing.T) {
	t.Parallel()

	t.Run("all error variables implement error interface", func(t *testing.T) {
		t.Parallel()

		var err error
		err = types.ErrInvalidMsg
		require.NotNil(t, err)
		err = types.ErrNoValidator
		require.NotNil(t, err)
		err = types.ErrNoSignerChange
		require.NotNil(t, err)
		err = types.ErrValUnBonded
		require.NotNil(t, err)
	})
}

func TestErrorsCanBeWrapped(t *testing.T) {
	t.Parallel()

	t.Run("errors can be wrapped with additional context", func(t *testing.T) {
		t.Parallel()

		wrappedErr := types.ErrInvalidMsg.Wrap("additional context")
		require.NotNil(t, wrappedErr)
		require.Contains(t, wrappedErr.Error(), "invalid message")
		require.Contains(t, wrappedErr.Error(), "additional context")
	})
}

func TestErrorCodes(t *testing.T) {
	t.Parallel()

	t.Run("errors have unique codes", func(t *testing.T) {
		t.Parallel()

		codes := map[uint32]bool{}

		// Check that error codes are unique
		require.False(t, codes[types.ErrInvalidMsg.ABCICode()])
		codes[types.ErrInvalidMsg.ABCICode()] = true

		require.False(t, codes[types.ErrNoValidator.ABCICode()])
		codes[types.ErrNoValidator.ABCICode()] = true

		require.False(t, codes[types.ErrNoSignerChange.ABCICode()])
		codes[types.ErrNoSignerChange.ABCICode()] = true

		require.False(t, codes[types.ErrValUnBonded.ABCICode()])
		codes[types.ErrValUnBonded.ABCICode()] = true
	})
}

func TestErrorMessages_NotEmpty(t *testing.T) {
	t.Parallel()

	t.Run("all error messages are non-empty", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, types.ErrInvalidMsg.Error())
		require.NotEmpty(t, types.ErrNoValidator.Error())
		require.NotEmpty(t, types.ErrNoSignerChange.Error())
		require.NotEmpty(t, types.ErrValUnBonded.Error())
	})
}

func TestErrorComparison(t *testing.T) {
	t.Parallel()

	t.Run("errors can be compared with errors.Is", func(t *testing.T) {
		t.Parallel()

		wrappedErr := types.ErrInvalidMsg.Wrap("context")
		require.True(t, errors.Is(wrappedErr, types.ErrInvalidMsg))
		require.False(t, errors.Is(wrappedErr, types.ErrNoValidator))
	})
}
