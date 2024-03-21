package helper

import (
	"bytes"
	"fmt"
	"sort"

	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	codec "github.com/cosmos/cosmos-sdk/codec/address"
)

// ToBytes32 is a convenience method for converting a byte slice to a fix
// sized 32 byte array. This method will truncate the input if it is larger
// than 32 bytes.
func ToBytes32(x []byte) [32]byte {
	var y [32]byte

	copy(y[:], x)

	return y
}

// SortValidatorByAddress sorts a slice of validators by address
// to sort it we compare the values of the Signer(HeimdallAddress i.e. [20]byte)
func SortValidatorByAddress(a []borTypes.Validator) []borTypes.Validator {
	sort.Slice(a, func(i, j int) bool {
		first, err := codec.NewHexCodec().StringToBytes(a[i].Signer)
		if err != nil {
			panic(fmt.Sprintf("failed to convert signer address string to bytes: %v", err))
		}
		second, err := codec.NewHexCodec().StringToBytes(a[j].Signer)
		if err != nil {
			panic(fmt.Sprintf("failed to convert signer address string to bytes: %v", err))
		}

		return bytes.Compare(first, second) < 0
	})

	return a
}
