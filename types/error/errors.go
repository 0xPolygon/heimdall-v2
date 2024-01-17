package error

import (
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
)
