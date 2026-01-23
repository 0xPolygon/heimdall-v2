package types_test

import (
	"testing"

	"cosmossdk.io/errors"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func TestErrEventRecordAlreadySynced(t *testing.T) {
	require.NotNil(t, types.ErrEventRecordAlreadySynced)
	require.Contains(t, types.ErrEventRecordAlreadySynced.Error(), "already synced")
}

func TestErrSizeExceed(t *testing.T) {
	require.NotNil(t, types.ErrSizeExceed)
	require.Contains(t, types.ErrSizeExceed.Error(), "size exceed")
}

func TestErrInvalidTxHash(t *testing.T) {
	require.NotNil(t, types.ErrInvalidTxHash)
	require.Contains(t, types.ErrInvalidTxHash.Error(), "tx hash")
}

func TestAllErrorsAreErrors(t *testing.T) {
	var err error

	err = types.ErrEventRecordAlreadySynced
	require.NotNil(t, err)

	err = types.ErrSizeExceed
	require.NotNil(t, err)

	err = types.ErrInvalidTxHash
	require.NotNil(t, err)
}

func TestErrorsCanBeWrapped(t *testing.T) {
	wrappedErr := errors.Wrap(types.ErrSizeExceed, "additional context")
	require.Error(t, wrappedErr)
	require.Contains(t, wrappedErr.Error(), "size exceed")
	require.Contains(t, wrappedErr.Error(), "additional context")
}

func TestErrorCodes(t *testing.T) {
	// Errors should have descriptive messages
	require.Contains(t, types.ErrEventRecordAlreadySynced.Error(), "synced")
	require.Contains(t, types.ErrSizeExceed.Error(), "exceed")
	require.Contains(t, types.ErrInvalidTxHash.Error(), "Invalid")
}

func TestErrorMessages_NotEmpty(t *testing.T) {
	require.NotEmpty(t, types.ErrEventRecordAlreadySynced.Error())
	require.NotEmpty(t, types.ErrSizeExceed.Error())
	require.NotEmpty(t, types.ErrInvalidTxHash.Error())
}
