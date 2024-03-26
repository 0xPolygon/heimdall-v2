package testutil

import (
	"context"

	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
)

type ModuleCommunicatorMock struct {
	accounts []hmTypes.DividendAccount
}

func (m ModuleCommunicatorMock) GetAllDividendAccounts(ctx context.Context) []hmTypes.DividendAccount {
	account := hmTypes.DividendAccount{
		User:      "0x0000000000000000",
		FeeAmount: "1",
	}
	m.accounts = []hmTypes.DividendAccount{account}

	return m.accounts
}
