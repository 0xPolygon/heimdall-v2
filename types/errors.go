package types

import "cosmossdk.io/errors"

func ErrInvalidBorChainID(ModuleName string) error {
	return errors.Register(ModuleName, 3506, "Event record already synced")
}

func ErrOldTx(ModuleName string) error {
	return errors.Register(ModuleName, 1401, "Old txhash not allowed")
}
