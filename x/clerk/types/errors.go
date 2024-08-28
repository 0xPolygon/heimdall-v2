package types

import (
	"cosmossdk.io/errors"
)

var (
	ErrEventRecordAlreadySynced = errors.Register(ModuleName, 5400, "Event record already synced")
	ErrSizeExceed               = errors.Register(ModuleName, 5401, "Data size exceed")
	ErrEmptyTxHash              = errors.Register(ModuleName, 5402, "Empty tx hash")
)
