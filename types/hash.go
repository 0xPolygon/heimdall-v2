package types

import (
	bytes "bytes"
)

// ZeroHeimdallHash represents zero address
var ZeroHeimdallHash = HeimdallHash{}

// Empty returns boolean for whether an AccAddress is empty
func (aa HeimdallHash) Empty() bool {
	return bytes.Equal(aa.GetHash(), ZeroHeimdallHash.GetHash())
}
