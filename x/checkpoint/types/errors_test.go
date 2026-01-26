package types_test

import (
	"testing"

	"cosmossdk.io/errors"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func TestErrInvalidMsg(t *testing.T) {
	require.NotNil(t, types.ErrInvalidMsg)
	require.Contains(t, types.ErrInvalidMsg.Error(), "invalid message")
}

func TestErrNoCheckpointFound(t *testing.T) {
	require.NotNil(t, types.ErrNoCheckpointFound)
	require.Contains(t, types.ErrNoCheckpointFound.Error(), "checkpoint not found")
}

func TestErrDiscontinuousCheckpoint(t *testing.T) {
	require.NotNil(t, types.ErrDiscontinuousCheckpoint)
	require.Contains(t, types.ErrDiscontinuousCheckpoint.Error(), "not in continuity")
}

func TestErrBadBlockDetails(t *testing.T) {
	require.NotNil(t, types.ErrBadBlockDetails)
	require.Contains(t, types.ErrBadBlockDetails.Error(), "buffer")
}

func TestErrNoAck(t *testing.T) {
	require.NotNil(t, types.ErrNoAck)
	require.Contains(t, types.ErrNoAck.Error(), "no-ack")
}

func TestErrBadAck(t *testing.T) {
	require.NotNil(t, types.ErrBadAck)
	require.Contains(t, types.ErrBadAck.Error(), "ack")
	require.Contains(t, types.ErrBadAck.Error(), "not valid")
}

func TestErrInvalidNoAck(t *testing.T) {
	require.NotNil(t, types.ErrInvalidNoAck)
	require.Contains(t, types.ErrInvalidNoAck.Error(), "no-ack")
}

func TestErrInvalidNoAckProposer(t *testing.T) {
	require.NotNil(t, types.ErrInvalidNoAckProposer)
	require.Contains(t, types.ErrInvalidNoAckProposer.Error(), "proposer")
}

func TestErrTooManyNoAck(t *testing.T) {
	require.NotNil(t, types.ErrTooManyNoAck)
	require.Contains(t, types.ErrTooManyNoAck.Error(), "too many")
}

func TestErrCheckpointParams(t *testing.T) {
	require.NotNil(t, types.ErrCheckpointParams)
	require.Contains(t, types.ErrCheckpointParams.Error(), "params")
}

func TestErrBufferFlush(t *testing.T) {
	require.NotNil(t, types.ErrBufferFlush)
	require.Contains(t, types.ErrBufferFlush.Error(), "buffer")
}

func TestErrAccountHash(t *testing.T) {
	require.NotNil(t, types.ErrAccountHash)
	require.Contains(t, types.ErrAccountHash.Error(), "account")
}

func TestErrAlreadyExists(t *testing.T) {
	require.NotNil(t, types.ErrAlreadyExists)
	require.Contains(t, types.ErrAlreadyExists.Error(), "already exists")
}

func TestAllErrorsAreSDKErrors(t *testing.T) {
	// All should implement error interface
	var err error

	err = types.ErrInvalidMsg
	require.NotNil(t, err)

	err = types.ErrNoCheckpointFound
	require.NotNil(t, err)

	err = types.ErrDiscontinuousCheckpoint
	require.NotNil(t, err)

	err = types.ErrBadBlockDetails
	require.NotNil(t, err)

	err = types.ErrNoAck
	require.NotNil(t, err)

	err = types.ErrBadAck
	require.NotNil(t, err)

	err = types.ErrInvalidNoAck
	require.NotNil(t, err)

	err = types.ErrInvalidNoAckProposer
	require.NotNil(t, err)

	err = types.ErrTooManyNoAck
	require.NotNil(t, err)

	err = types.ErrCheckpointParams
	require.NotNil(t, err)

	err = types.ErrBufferFlush
	require.NotNil(t, err)

	err = types.ErrAccountHash
	require.NotNil(t, err)

	err = types.ErrAlreadyExists
	require.NotNil(t, err)
}

func TestErrorCodes_Unique(t *testing.T) {
	// Error codes should be unique within the module
	codes := map[uint32]error{
		2:  types.ErrInvalidMsg,
		4:  types.ErrNoCheckpointFound,
		7:  types.ErrDiscontinuousCheckpoint,
		8:  types.ErrBadBlockDetails,
		9:  types.ErrNoAck,
		10: types.ErrBadAck,
		11: types.ErrInvalidNoAck,
		12: types.ErrInvalidNoAckProposer,
		13: types.ErrTooManyNoAck,
		14: types.ErrCheckpointParams,
		15: types.ErrBufferFlush,
		16: types.ErrAccountHash,
		17: types.ErrAlreadyExists,
	}

	require.Len(t, codes, 13)
}

func TestErrorsCanBeWrapped(t *testing.T) {
	wrappedErr := errors.Wrap(types.ErrInvalidMsg, "additional context")
	require.Error(t, wrappedErr)
	require.Contains(t, wrappedErr.Error(), "invalid message")
	require.Contains(t, wrappedErr.Error(), "additional context")
}

func TestCheckpointErrors_ContainKeywords(t *testing.T) {
	// Checkpoint-related errors
	require.Contains(t, types.ErrNoCheckpointFound.Error(), "checkpoint")
	require.Contains(t, types.ErrDiscontinuousCheckpoint.Error(), "checkpoint")
	require.Contains(t, types.ErrCheckpointParams.Error(), "checkpoint")
	require.Contains(t, types.ErrAlreadyExists.Error(), "checkpoint")
}

func TestAckErrors_ContainKeywords(t *testing.T) {
	// Ack-related errors
	require.Contains(t, types.ErrNoAck.Error(), "ack")
	require.Contains(t, types.ErrBadAck.Error(), "ack")
	require.Contains(t, types.ErrInvalidNoAck.Error(), "ack")
	require.Contains(t, types.ErrInvalidNoAckProposer.Error(), "ack")
	require.Contains(t, types.ErrTooManyNoAck.Error(), "ack")
}

func TestErrorMessages_NotEmpty(t *testing.T) {
	errs := []error{
		types.ErrInvalidMsg,
		types.ErrNoCheckpointFound,
		types.ErrDiscontinuousCheckpoint,
		types.ErrBadBlockDetails,
		types.ErrNoAck,
		types.ErrBadAck,
		types.ErrInvalidNoAck,
		types.ErrInvalidNoAckProposer,
		types.ErrTooManyNoAck,
		types.ErrCheckpointParams,
		types.ErrBufferFlush,
		types.ErrAccountHash,
		types.ErrAlreadyExists,
	}

	for _, err := range errs {
		require.NotEmpty(t, err.Error())
	}
}
