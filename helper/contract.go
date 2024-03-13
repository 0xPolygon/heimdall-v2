package helper

import (
	"math/big"

	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

// TODO HV2: this is a dummy implementation of the contract caller
// with only the necessary methods for the x/bor
// we will replace this with the actual implementation

type ContractCaller struct{}

// GetMainChainBlock returns main chain block header
func (c *ContractCaller) GetMainChainBlock(blockNum *big.Int) (header *ethTypes.Header, err error) {
	return &ethTypes.Header{}, nil
}
