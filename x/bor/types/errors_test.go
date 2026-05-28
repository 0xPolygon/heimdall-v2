package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

func TestErrInvalidChainID(t *testing.T) {
	require.NotNil(t, types.ErrInvalidChainID)
	require.Equal(t, "invalid bor chain id", types.ErrInvalidChainID.Error())
}

func TestErrInvalidSpan(t *testing.T) {
	require.NotNil(t, types.ErrInvalidSpan)
	require.Equal(t, "invalid span", types.ErrInvalidSpan.Error())
}

func TestErrInvalidLastBorSpanID(t *testing.T) {
	require.NotNil(t, types.ErrInvalidLastBorSpanID)
	require.Equal(t, "invalid last bor span id", types.ErrInvalidLastBorSpanID.Error())
}

func TestErrInvalidSeedLength(t *testing.T) {
	require.NotNil(t, types.ErrInvalidSeedLength)
	require.Equal(t, "invalid seed length", types.ErrInvalidSeedLength.Error())
}

func TestErrFailedToQueryBor(t *testing.T) {
	require.NotNil(t, types.ErrFailedToQueryBor)
	require.Equal(t, "failed to query bor", types.ErrFailedToQueryBor.Error())
}

func TestErrLatestMilestoneNotFound(t *testing.T) {
	require.NotNil(t, types.ErrLatestMilestoneNotFound)
	require.Equal(t, "latest milestone not found", types.ErrLatestMilestoneNotFound.Error())
}

func TestAllErrorsAreErrors(t *testing.T) {
	// Test that all error variables implement error interface
	var err error

	err = types.ErrInvalidChainID
	require.NotNil(t, err)

	err = types.ErrInvalidSpan
	require.NotNil(t, err)

	err = types.ErrInvalidLastBorSpanID
	require.NotNil(t, err)

	err = types.ErrInvalidSeedLength
	require.NotNil(t, err)

	err = types.ErrFailedToQueryBor
	require.NotNil(t, err)

	err = types.ErrLatestMilestoneNotFound
	require.NotNil(t, err)
}

func TestErrorUniqueness(t *testing.T) {
	// Test that all errors have unique messages
	errorMessages := []string{
		types.ErrInvalidChainID.Error(),
		types.ErrInvalidSpan.Error(),
		types.ErrInvalidLastBorSpanID.Error(),
		types.ErrInvalidSeedLength.Error(),
		types.ErrFailedToQueryBor.Error(),
		types.ErrLatestMilestoneNotFound.Error(),
	}

	// Check all error messages are distinct
	for i := 0; i < len(errorMessages); i++ {
		for j := i + 1; j < len(errorMessages); j++ {
			require.NotEqual(t, errorMessages[i], errorMessages[j],
				"Error messages at positions %d and %d should be different", i, j)
		}
	}
}

func TestErrorsCanBeWrapped(t *testing.T) {
	// Test that errors can be wrapped with additional context
	wrappedErr := errors.Join(types.ErrInvalidChainID, errors.New("additional context"))
	require.Error(t, wrappedErr)
	require.Contains(t, wrappedErr.Error(), "invalid bor chain id")
	require.Contains(t, wrappedErr.Error(), "additional context")
}

func TestErrorsCanBeCompared(t *testing.T) {
	// Test that errors can be compared using errors.Is
	err := types.ErrInvalidChainID
	require.True(t, errors.Is(err, types.ErrInvalidChainID))
	require.False(t, errors.Is(err, types.ErrInvalidSpan))
}

func TestErrorMessagesNotEmpty(t *testing.T) {
	// Test that all error messages are not empty
	require.NotEmpty(t, types.ErrInvalidChainID.Error())
	require.NotEmpty(t, types.ErrInvalidSpan.Error())
	require.NotEmpty(t, types.ErrInvalidLastBorSpanID.Error())
	require.NotEmpty(t, types.ErrInvalidSeedLength.Error())
	require.NotEmpty(t, types.ErrFailedToQueryBor.Error())
	require.NotEmpty(t, types.ErrLatestMilestoneNotFound.Error())
}

func TestErrorsAreDescriptive(t *testing.T) {
	// Test that error messages are descriptive
	errs := map[string]error{
		"chain id":  types.ErrInvalidChainID,
		"span":      types.ErrInvalidSpan,
		"bor":       types.ErrInvalidLastBorSpanID,
		"seed":      types.ErrInvalidSeedLength,
		"query":     types.ErrFailedToQueryBor,
		"milestone": types.ErrLatestMilestoneNotFound,
	}

	for keyword, err := range errs {
		require.Contains(t, err.Error(), keyword,
			"Error message should contain keyword: %s", keyword)
	}
}

func TestInvalidSpanErrors(t *testing.T) {
	// Group test for span-related errors
	require.Contains(t, types.ErrInvalidSpan.Error(), "span")
	require.Contains(t, types.ErrInvalidLastBorSpanID.Error(), "span")
}

func TestQueryErrorContainsKeyword(t *testing.T) {
	require.Contains(t, types.ErrFailedToQueryBor.Error(), "query")
	require.Contains(t, types.ErrFailedToQueryBor.Error(), "bor")
}

func TestMilestoneErrorContainsKeyword(t *testing.T) {
	require.Contains(t, types.ErrLatestMilestoneNotFound.Error(), "milestone")
	require.Contains(t, types.ErrLatestMilestoneNotFound.Error(), "not found")
}

func TestChainIDErrorFormat(t *testing.T) {
	// Test chain ID error has proper formatting
	require.Contains(t, types.ErrInvalidChainID.Error(), "invalid")
	require.Contains(t, types.ErrInvalidChainID.Error(), "chain")
	require.Contains(t, types.ErrInvalidChainID.Error(), "id")
}

func TestSeedLengthErrorFormat(t *testing.T) {
	// Test seed length error has proper formatting
	require.Contains(t, types.ErrInvalidSeedLength.Error(), "invalid")
	require.Contains(t, types.ErrInvalidSeedLength.Error(), "seed")
	require.Contains(t, types.ErrInvalidSeedLength.Error(), "length")
}

func TestErrorsAreLowercase(t *testing.T) {
	// Test that error messages start with lowercase (Go convention)
	errs := []error{
		types.ErrInvalidChainID,
		types.ErrInvalidSpan,
		types.ErrInvalidLastBorSpanID,
		types.ErrInvalidSeedLength,
		types.ErrFailedToQueryBor,
		types.ErrLatestMilestoneNotFound,
	}

	for _, err := range errs {
		msg := err.Error()
		require.NotEmpty(t, msg)
		// The first character should be lowercase
		firstChar := rune(msg[0])
		require.True(t, firstChar >= 'a' && firstChar <= 'z',
			"Error message should start with lowercase: %s", msg)
	}
}
