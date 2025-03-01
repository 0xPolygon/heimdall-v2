package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	// ErrInvalidMsg is returned if the message is invalid
	ErrInvalidMsg = errorsmod.Register(ModuleName, 2, "invalid message")

	// ErrOldTx is returned if the respective stake related tx from L1 has already been processed
	ErrOldTx = errorsmod.Register(ModuleName, 3, "old tx, already processed")

	// ErrNoValidator is returned if the respective validator doesn't exist
	ErrNoValidator = errorsmod.Register(ModuleName, 4, "no respective validator found")

	// ErrNoSignerChange returned when the new signer address is same as old one
	ErrNoSignerChange = errorsmod.Register(ModuleName, 5, "new singer is same as old one")

	// ErrValUnbonded is returned when the respective validator is already unbonded
	ErrValUnbonded = errorsmod.Register(ModuleName, 6, "validator already unbonded")

	// ErrInvalidNonce is returned when the nonce is wrong
	ErrInvalidNonce = errorsmod.Register(ModuleName, 7, "invalid nonce")
)
