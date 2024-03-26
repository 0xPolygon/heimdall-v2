package types

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/cbergoon/merkletree"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/ethereum/go-ethereum/crypto"
)

func NewDividendAccount(user string, fee string) DividendAccount {
	return DividendAccount{
		User:      user,
		FeeAmount: fee,
	}
}

func (da *DividendAccount) Strigified() string {
	if da == nil {
		return "nil-DividendAccount"
	}

	return fmt.Sprintf("DividendAccount{%s %v}",
		da.User,
		da.FeeAmount)
}

// MarshallDividendAccount - amino Marshall DividendAccount
func MarshallDividendAccount(cdc codec.BinaryCodec, dividendAccount DividendAccount) (bz []byte, err error) {
	bz, err = cdc.Marshal(&dividendAccount)
	if err != nil {
		return bz, err
	}

	return bz, nil
}

// UnMarshallDividendAccount - amino Unmarshall DividendAccount
func UnMarshallDividendAccount(cdc codec.BinaryCodec, value []byte) (DividendAccount, error) {
	var dividendAccount DividendAccount
	if err := cdc.Unmarshal(value, &dividendAccount); err != nil {
		return dividendAccount, err
	}

	return dividendAccount, nil
}

// SortDividendAccountByAddress - Sorts DividendAccounts  By  Address
func SortDividendAccountByAddress(dividendAccounts []DividendAccount) []DividendAccount {
	sort.Slice(dividendAccounts, func(i, j int) bool {
		return strings.Compare(strings.ToLower(dividendAccounts[i].User), strings.ToLower(dividendAccounts[j].User)) < 0
	})

	return dividendAccounts
}

// TODO H2 Need to check it as []bytes(da.user) might give different result
// CalculateHash hashes the values of a DividendAccount
func (da DividendAccount) CalculateHash() ([]byte, error) {
	fee, _ := big.NewInt(0).SetString(da.FeeAmount, 10)
	divAccountHash := crypto.Keccak256(appendBytes32(
		[]byte(da.User),
		fee.Bytes(),
	))

	return divAccountHash, nil
}

func appendBytes32(data ...[]byte) []byte {
	var result []byte

	for _, v := range data {
		paddedV, err := convertTo32(v)
		if err == nil {
			result = append(result, paddedV[:]...)
		}
	}

	return result
}

//nolint:unparam
func convertTo32(input []byte) (output [32]byte, err error) {
	l := len(input)
	if l > 32 || l == 0 {
		return
	}

	copy(output[32-l:], input[:])

	return
}

// Equals tests for equality of two Contents
func (da DividendAccount) Equals(other merkletree.Content) (bool, error) {
	return da.User == other.(DividendAccount).User, nil
}
