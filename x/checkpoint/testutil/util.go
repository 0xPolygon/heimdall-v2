package testutil

import (
	"crypto/rand"
	"math/big"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
)

func RandomBytes() []byte {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return b
}

func RandDividendAccounts() []hmTypes.DividendAccount {
	dividendAccounts := make([]hmTypes.DividendAccount, 1)

	dividendAccounts[0] = hmTypes.DividendAccount{
		User:      secp256k1.GenPrivKey().PubKey().Address().String(),
		FeeAmount: big.NewInt(0).String(),
	}

	return dividendAccounts
}
