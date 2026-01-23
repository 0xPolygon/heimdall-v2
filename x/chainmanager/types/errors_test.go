package types_test

import (
	"testing"

	"cosmossdk.io/errors"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

func TestErrInvalidParams(t *testing.T) {
	require.NotNil(t, types.ErrInvalidParams)
	require.Contains(t, types.ErrInvalidParams.Error(), "invalid params")
}

func TestErrInvalidParams_IsError(t *testing.T) {
	var err error = types.ErrInvalidParams
	require.NotNil(t, err)
}

func TestErrInvalidParams_ErrorCode(t *testing.T) {
	// SDK errors have code 1
	require.Contains(t, types.ErrInvalidParams.Error(), "invalid params")
}

func TestErrInvalidParams_Module(t *testing.T) {
	// Error should be a valid error with a message
	require.NotEmpty(t, types.ErrInvalidParams.Error())
	require.Contains(t, types.ErrInvalidParams.Error(), "invalid")
}

func TestErrInvalidParams_CanBeWrapped(t *testing.T) {
	wrappedErr := errors.Wrap(types.ErrInvalidParams, "additional context")
	require.Error(t, wrappedErr)
	require.Contains(t, wrappedErr.Error(), "invalid params")
	require.Contains(t, wrappedErr.Error(), "additional context")
}

func TestErrInvalidParams_ErrorMessage(t *testing.T) {
	require.NotEmpty(t, types.ErrInvalidParams.Error())
}
