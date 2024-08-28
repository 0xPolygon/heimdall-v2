package types

import (
	bytes "bytes"

	"github.com/ethereum/go-ethereum/common"
)

const (
	HashLength = common.HashLength
)

// ZeroHeimdallHash represents zero address
var ZeroHeimdallHash = HeimdallHash{Hash: common.Hash{}.Bytes()}

// Empty returns boolean for whether an HeimdallHash is empty
func (hh HeimdallHash) Empty() bool {
	return bytes.Equal(hh.GetHash(), ZeroHeimdallHash.GetHash())
}
