package testutil

import (
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// BankKeeper import the cosmos-sdk/x/bank keeper for test purposes
type BankKeeper interface {
	bankkeeper.Keeper
}
