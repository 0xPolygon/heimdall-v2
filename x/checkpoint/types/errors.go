package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	// ErrInvalidMsg is returned if the message is invalid
	ErrInvalidMsg = errorsmod.Register(ModuleName, 2, "invalid message")

	// ErrOldTx is returned if the respective stateSync tx from L1 has already been processed
	ErrOldTx = errorsmod.Register(ModuleName, 3, "old tx, laready processed")

	// ErrInvalidNonce is returned when the nonce is wrong
	ErrInvalidNonce = errorsmod.Register(ModuleName, 4, "invalid nonce")

	// ErrCheckpointBufferFound is returned when checkpoint is not found in buffer
	ErrCheckpointBufferFound = errorsmod.Register(ModuleName, 5, "Checkpoint not found in buffer")

	ErrNoCheckpointFound = errorsmod.Register(ModuleName, 6, "Checkpoint not found in database")

	ErrCheckpointAlreadyExists = errorsmod.Register(ModuleName, 7, "Checkpoint already exists")

	ErrOldCheckpoint = errorsmod.Register(ModuleName, 8, "Checkpoint already received for given start and end block")

	ErrDisCountinuousCheckpoint = errorsmod.Register(ModuleName, 9, "Checkpoint not in continuity")

	ErrBadBlockDetails = errorsmod.Register(ModuleName, 10, "Checkpoint not found in buffer")

	ErrNoACK = errorsmod.Register(ModuleName, 11, "No ack invalid")

	ErrBadAck = errorsmod.Register(ModuleName, 12, "Ack not valid")

	ErrInvalidNoACK = errorsmod.Register(ModuleName, 13, "Invalid no aCK -- Waiting for last checkpoint ACK")

	ErrInvalidNoACKProposer = errorsmod.Register(ModuleName, 14, "Invalid No ACK Proposer")

	ErrTooManyNoACK = errorsmod.Register(ModuleName, 15, "Too many no-acks")

	ErrCheckpointParams = errorsmod.Register(ModuleName, 16, "checkpoint params not found")
)
