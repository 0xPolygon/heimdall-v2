package testutil

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
)

// In case of changes to this file, regenerate the mocks with
//  `mockgen -source=x/topup/testutil/expected_keepers.go -destination=x/topup/testutil/expected_keepers_mocks.go -package=testutil`

// AccountKeeper extends gov's actual expected AccountKeeper with additional
// methods used in tests.
type AccountKeeper interface {
	types.AccountKeeper
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
}

// BankKeeper extends topups's actual expected BankKeeper with additional
// methods used in tests.
type BankKeeper interface {
	bankkeeper.Keeper
}
