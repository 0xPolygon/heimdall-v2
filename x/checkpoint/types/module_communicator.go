package types

import (
	"context"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
)

type ModuleCommunicator interface {
	GetAllDividendAccounts(ctx context.Context) []hmTypes.DividendAccount
}
