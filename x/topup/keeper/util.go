package keeper

import (
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"

	hTypes "github.com/0xPolygon/heimdall-v2/types"
)

// CalculateDividendAccountHash hashes the values of a DividendAccount
func CalculateDividendAccountHash(da hTypes.DividendAccount) ([]byte, error) {
	fee, _ := big.NewInt(0).SetString(da.FeeAmount, 10)
	divAccountHash := crypto.Keccak256(AppendBytes32(
		[]byte(da.User),
		fee.Bytes(),
	))

	return divAccountHash, nil
}

func AppendBytes32(data ...[]byte) []byte {
	var result []byte

	for _, v := range data {
		paddedV, err := convertTo32(v)
		if err == nil {
			result = append(result, paddedV[:]...)
		}
	}

	return result
}

func convertTo32(input []byte) (output [32]byte, err error) {
	l := len(input)
	if l > 32 || l == 0 {
		return
	}

	copy(output[32-l:], input[:])

	return
}
