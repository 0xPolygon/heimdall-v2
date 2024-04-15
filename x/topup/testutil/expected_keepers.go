package testutil

import (
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// BankKeeper extends topups's actual expected BankKeeper with additional
// methods used in tests.
type BankKeeper interface {
	bankkeeper.Keeper
}
