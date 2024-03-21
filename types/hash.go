package types

import (
	bytes "bytes"
)

// ZeroHeimdallHash represents zero address
var ZeroHeimdallHash = HeimdallHash{}

// Empty returns boolean for whether an HeimdallHash is empty
func (hh HeimdallHash) Empty() bool {
	return bytes.Equal(hh.GetHash(), ZeroHeimdallHash.GetHash())
}
