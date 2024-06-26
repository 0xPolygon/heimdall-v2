package types

import (
	bytes "bytes"
	"math/big"
	"sort"

	"github.com/cbergoon/merkletree"
	addCodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

// GetAccountRootHash returns rootHash of Validator Account State Tree
func GetAccountRootHash(dividendAccounts []DividendAccount) ([]byte, error) {
	tree, err := GetAccountTree(dividendAccounts)
	if err != nil {
		return nil, err
	}

	return tree.Root.Hash, nil
}

// GetAccountTree returns rootHash of Validator Account State Tree
func GetAccountTree(dividendAccounts []DividendAccount) (*merkletree.MerkleTree, error) {
	// Sort the dividendAccounts by ID
	dividendAccounts = SortDividendAccountByAddress(dividendAccounts)
	list := make([]merkletree.Content, len(dividendAccounts))

	for i := 0; i < len(dividendAccounts); i++ {
		list[i] = dividendAccounts[i]
	}

	tree, err := merkletree.NewTreeWithHashStrategy(list, sha3.NewLegacyKeccak256)
	if err != nil {
		return nil, err
	}

	return tree, nil
}

// SortDividendAccountByAddress - Sorts DividendAccounts  By  Address
func SortDividendAccountByAddress(dividendAccounts []DividendAccount) []DividendAccount {
	sort.Slice(dividendAccounts, func(i, j int) bool {
		// TODO HV2 Try to catch the err in the following or we can just assume that
		// these dividendAccounts[i].User is of correct form.
		divAccBytesI, _ := addCodec.NewHexCodec().StringToBytes(dividendAccounts[i].User)

		divAccBytesJ, _ := addCodec.NewHexCodec().StringToBytes(dividendAccounts[j].User)

		return bytes.Compare(divAccBytesI, divAccBytesJ) < 0
	})

	return dividendAccounts
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

// CalculateHash hashes the values of a DividendAccount
func (da DividendAccount) CalculateHash() ([]byte, error) {
	// TODO HV2 Please try to catch the error or We can assume these
	// following values are correct in format
	fee, _ := big.NewInt(0).SetString(da.FeeAmount, 10)
	addressBytes, _ := addCodec.NewHexCodec().StringToBytes(da.User)

	divAccountHash := crypto.Keccak256(AppendBytes32(
		addressBytes,
		fee.Bytes(),
	))

	return divAccountHash, nil
}

func (da DividendAccount) Equals(other merkletree.Content) (bool, error) {
	return da.User == other.(DividendAccount).User, nil
}
