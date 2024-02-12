package types

import (
	"context"

	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
)

type ModuleCommunicator interface {
	GetAllDividendAccounts(ctx context.Context) []hmTypes.DividendAccount
}
