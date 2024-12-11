package testutil

import (
	"crypto/rand"
	"math/big"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
)

func RandomBytes() []byte {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return b
}

func RandDividendAccounts() []hmTypes.DividendAccount {
	dividendAccs := make([]hmTypes.DividendAccount, 1)

	dividendAccs[0] = hmTypes.DividendAccount{
		User:      secp256k1.GenPrivKey().PubKey().Address().String(),
		FeeAmount: big.NewInt(0).String(),
	}

	return dividendAccs
}
