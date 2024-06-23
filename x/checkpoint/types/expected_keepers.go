package types

import (
	"context"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
)

type TopupKeeper interface {
	GetAllDividendAccounts(ctx context.Context) []hmTypes.DividendAccount
}
