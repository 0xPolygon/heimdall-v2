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

	// ErrUnknownRequest is returned when the request is unknown
	ErrUnknownRequest = errorsmod.Register(RootCodespace, 105, "unknown request")

	// ErrInvalidNonce is returned when the nonce is wrong
	ErrInvalidNonce = errorsmod.Register(RootCodespace, 106, "invalid nonce")
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
