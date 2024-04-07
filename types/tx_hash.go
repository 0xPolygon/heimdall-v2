package types

import (
	"github.com/ethereum/go-ethereum/common"
)

// EthHash get eth hash
func (aa TxHash) EthHash() common.Hash {
	return common.Hash(aa.Bytes())
}

// Bytes returns the raw address bytes.
func (aa TxHash) Bytes() []byte {
	return aa.GetHash()
}
