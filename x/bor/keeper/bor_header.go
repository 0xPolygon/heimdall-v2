package keeper

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/0xPolygon/heimdall-v2/helper"
)

// fetchBorBlockAuthor fetches a Bor block's author in an isolated goroutine
// so a panic in the Bor RPC/gRPC client surfaces as an error instead of
// crashing the node. getBorBlockForSpanSeed calls this repeatedly while
// scanning for a seed block author, so isolating it once here covers all of
// those calls.
func (k *Keeper) fetchBorBlockAuthor(ctx context.Context, blockNum *big.Int) (*common.Address, error) {
	return helper.RunIsolated(func() (*common.Address, error) {
		return k.contractCaller.GetBorChainBlockAuthor(ctx, blockNum)
	})
}

// fetchBorHeader fetches a Bor header in an isolated goroutine so a panic in
// the Bor RPC/gRPC client surfaces as an error instead of crashing the node,
// and normalizes a nil header or a nil Number into an error so every caller
// can dereference the result directly without repeating either check.
func fetchBorHeader(ctx context.Context, contractCaller helper.IContractCaller, blockNum *big.Int) (*ethTypes.Header, error) {
	header, err := helper.RunIsolated(func() (*ethTypes.Header, error) {
		return contractCaller.GetBorChainBlock(ctx, blockNum)
	})
	if err != nil {
		return nil, err
	}
	if header == nil {
		return nil, errors.New("nil header returned")
	}
	if header.Number == nil {
		return nil, errors.New("header has nil Number")
	}
	return header, nil
}
