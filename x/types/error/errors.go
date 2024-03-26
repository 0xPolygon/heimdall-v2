package error

import (
	"os"

	errorsmod "cosmossdk.io/errors"
)

// RootCodespace is the codespace for all errors defined in this package
const RootCodespace = "sdk"

//Please use the code starting from 100 as less code are used
//in the cosmos-sdk

var (
	// ErrInvalidMsg is returned if the message is invalid
	ErrInvalidMsg = errorsmod.Register(RootCodespace, 101, "invalid message")

	// ErrOldTx is returned if the respective stateSync tx from L1 has already been processed
	ErrOldTx = errorsmod.Register(RootCodespace, 102, "old tx, laready processed")

	// ErrNoValidator is returned if the respective doesn't exist
	ErrNoValidator = errorsmod.Register(RootCodespace, 103, "no respective validator found")

	// ErrNoSignerChange returned when the new signer address is same as old one
	ErrNoSignerChange = errorsmod.Register(RootCodespace, 104, "new singer is same as old one")

	// ErrUnknownRequest is returned when the respective validator is already unbonded
	ErrValUnbonded = errorsmod.Register(RootCodespace, 105, "validator already unbonded")

	// ErrInvalidNonce is returned when the nonce is wrong
	ErrInvalidNonce = errorsmod.Register(RootCodespace, 106, "invalid nonce")

	// ErrCheckpointBufferFound is returned when checkpoint is not found in buffer
	ErrCheckpointBufferFound = errorsmod.Register(RootCodespace, 107, "Checkpoint not found in buffer")

	ErrNoCheckpointFound = errorsmod.Register(RootCodespace, 108, "Checkpoint not found in database")

	ErrCheckpointAlreadyExists = errorsmod.Register(RootCodespace, 109, "Checkpoint already exists")

	ErrOldCheckpoint = errorsmod.Register(RootCodespace, 110, "Checkpoint already received for given start and end block")

	ErrDisCountinuousCheckpoint = errorsmod.Register(RootCodespace, 111, "Checkpoint not in continuity")

	ErrBadBlockDetails = errorsmod.Register(RootCodespace, 112, "Checkpoint not found in buffer")

	ErrNoACK = errorsmod.Register(RootCodespace, 113, "No ack invalid")

	ErrBadAck = errorsmod.Register(RootCodespace, 114, "Ack not valid")

	ErrInvalidNoACK = errorsmod.Register(RootCodespace, 115, "Invalid no aCK -- Waiting for last checkpoint ACK")

	ErrInvalidNoACKProposer = errorsmod.Register(RootCodespace, 116, "Invalid No ACK Proposer")

	ErrTooManyNoACK = errorsmod.Register(RootCodespace, 117, "Too many no-acks")

	ErrCheckpointParams = errorsmod.Register(RootCodespace, 118, "checkpoint params not found")
)

type InvalidPermissionsError struct {
	File string
	Perm os.FileMode
	Err  error
}

func (e InvalidPermissionsError) detailed() (valid bool) {
	if e.File != "" && e.Perm != 0 {
		valid = true
	}

	return
}

func (e InvalidPermissionsError) Error() string {
	errMsg := "Invalid file permission"
	if e.detailed() {
		errMsg += " for file " + e.File + " should be " + e.Perm.String()
	}

	if e.Err != nil {
		errMsg += " \nerr: " + e.Err.Error()
	}

	return errMsg
}
