package types

import (
	"sort"
	"strings"

	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// SortValidatorByAddress sorts a slice of validators by address
// to sort it we compare the values of the Signer(HeimdallAddress i.e. [20]byte)
func SortValidatorByAddress(a []staketypes.Validator) []staketypes.Validator {
	sort.Slice(a, func(i, j int) bool {
		return strings.Compare(a[i].Signer, a[j].Signer) < 0
	})

	return a
}

// SortSpanByID sorts spans by SpanID
func SortSpansById(a []*Span) {
	sort.Slice(a, func(i, j int) bool {
		return a[i].Id < a[j].Id
	})
}
