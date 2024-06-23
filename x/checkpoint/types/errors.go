package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	// ErrInvalidMsg is returned if the message is invalid
	ErrInvalidMsg = errorsmod.Register(ModuleName, 2, "invalid message")

	// ErrOldTx is returned if the respective stateSync tx from L1 has already been processed
	ErrOldTx = errorsmod.Register(ModuleName, 3, "old tx, laready processed")

	// ErrCheckpointBufferFound is returned when checkpoint is not found in buffer
	ErrCheckpointBufferFound = errorsmod.Register(ModuleName, 4, "checkpoint not found in buffer")

	ErrNoCheckpointFound = errorsmod.Register(ModuleName, 5, "checkpoint not found in database")

	ErrCheckpointAlreadyExists = errorsmod.Register(ModuleName, 6, "checkpoint already exists")

	ErrOldCheckpoint = errorsmod.Register(ModuleName, 7, "checkpoint already received for given start and end block")

	ErrDisCountinuousCheckpoint = errorsmod.Register(ModuleName, 8, "checkpoint is not in continuity")

	ErrBadBlockDetails = errorsmod.Register(ModuleName, 9, "checkpoint not found in buffer")

	ErrNoACK = errorsmod.Register(ModuleName, 10, "no ack invalid")

	ErrBadAck = errorsmod.Register(ModuleName, 11, "checkpoint ack is not valid")

	ErrInvalidNoACK = errorsmod.Register(ModuleName, 12, "invalid no ack - waiting for the last checkpoint ack")

	ErrInvalidNoACKProposer = errorsmod.Register(ModuleName, 13, "invalid No ACK Proposer")

	ErrTooManyNoACK = errorsmod.Register(ModuleName, 14, "too many no-acks")

	ErrCheckpointParams = errorsmod.Register(ModuleName, 15, "checkpoint params not found")
)
