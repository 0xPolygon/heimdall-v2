package types

import (
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// BankKeeper import the cosmos-sdk/x/bank keeper
type BankKeeper interface {
	bankkeeper.Keeper
}
