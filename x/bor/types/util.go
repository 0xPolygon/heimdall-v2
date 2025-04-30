package types

import (
	"sort"
	"strings"

	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// SortValidatorByAddress sorts a slice of validators by address
// to sort it we compare the values of the Signer(HeimdallAddress i.e. [20]byte)
func SortValidatorByAddress(a []staketypes.Validator) []staketypes.Validator {
	sort.Slice(a, func(i, j int) bool {
		return strings.Compare(a[i].Signer, a[j].Signer) < 0
	})

	return a
}

// SortSpansById sorts spans by SpanID
func SortSpansById(a []Span) {
	sort.Slice(a, func(i, j int) bool {
		return a[i].Id < a[j].Id
	})
}

func GetAddr(validator staketypes.Validator) (string, error) {
	pub, err := crypto.UnmarshalPubkey(validator.PubKey)
	if err != nil {
		return "", err
	}
	return crypto.PubkeyToAddress(*pub).Hex(), nil
}

func GetAddrs(validators []staketypes.Validator) ([]string, error) {
	addrs := make([]string, len(validators))
	for i, val := range validators {
		addr, err := GetAddr(val)
		if err != nil {
			return nil, err
		}
		addrs[i] = addr
	}
	return addrs, nil
}
