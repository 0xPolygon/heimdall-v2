package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrInvalidMsg              = errorsmod.Register(ModuleName, 2, "invalid message")
	ErrCheckpointBufferFound   = errorsmod.Register(ModuleName, 3, "checkpoint not found in buffer")
	ErrNoCheckpointFound       = errorsmod.Register(ModuleName, 4, "checkpoint not found")
	ErrCheckpointAlreadyExists = errorsmod.Register(ModuleName, 5, "checkpoint already exists")
	ErrOldCheckpoint           = errorsmod.Register(ModuleName, 6, "checkpoint already received for given start and end blocks")
	ErrDiscontinuousCheckpoint = errorsmod.Register(ModuleName, 7, "checkpoint is not in continuity")
	ErrBadBlockDetails         = errorsmod.Register(ModuleName, 8, "checkpoint not found in buffer")
	ErrNoAck                   = errorsmod.Register(ModuleName, 9, "no-ack invalid")
	ErrBadAck                  = errorsmod.Register(ModuleName, 10, "checkpoint ack is not valid")
	ErrInvalidNoAck            = errorsmod.Register(ModuleName, 11, "invalid no-ack, waiting for the last checkpoint ack")
	ErrInvalidNoAckProposer    = errorsmod.Register(ModuleName, 12, "invalid no-ack proposer")
	ErrTooManyNoAck            = errorsmod.Register(ModuleName, 13, "too many no-acks")
	ErrCheckpointParams        = errorsmod.Register(ModuleName, 14, "checkpoint params not found")
	ErrBufferFlush             = errorsmod.Register(ModuleName, 15, "flushing buffer failed")
	ErrAccountHash             = errorsmod.Register(ModuleName, 16, "error while fetching account root hash")
)
