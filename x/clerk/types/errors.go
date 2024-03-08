package types

import (
	"cosmossdk.io/errors"
)

var (
	ErrEventRecordAlreadySynced = errors.Register(ModuleName, 5400, "Event record already synced")
	ErrEventRecordInvalid       = errors.Register(ModuleName, 5401, "Event record is invalid")
	ErrEventUpdate              = errors.Register(ModuleName, 5402, "Event record update error")
	ErrSizeExceed               = errors.Register(ModuleName, 5403, "Data size exceed")
)
