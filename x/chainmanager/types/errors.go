package types

import "cosmossdk.io/errors"

// x/chainmanager module sentinel errors
var (
	ErrInvalidParams = errors.Register(ModuleName, 1, "invalid params")
)
