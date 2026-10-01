package abci

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/0xPolygon/heimdall-v2/helper"
)

// borBlockBatchInfo packs GetBorChainBlockInfoInBatch's multiple return
// values so a single call to helper.RunIsolated can carry all of them
// across the isolation boundary.
type borBlockBatchInfo struct {
	headers []*ethTypes.Header
	tds     []uint64
	authors []common.Address
}

// fetchBorHeader fetches a Bor header in an isolated goroutine so a panic in
// the Bor RPC/gRPC client surfaces as an error instead of crashing the node,
// and normalizes a nil header or a nil Number into an error so every caller
// shares one check instead of dereferencing either directly.
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

// resolveLatestHeader returns latestHeader as-is if it already covers
// startBlockNum, otherwise fetches it (or refreshes it once, in case Bor
// produced the block since latestHeader was cached, which handles Heimdall
// blocking faster than Bor).
func resolveLatestHeader(ctx sdk.Context, contractCaller helper.IContractCaller, latestHeader *ethTypes.Header, startBlockNum uint64) (*ethTypes.Header, uint64, error) {
	var err error
	if latestHeader == nil {
		latestHeader, err = fetchBorHeader(ctx, contractCaller, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get the latest header: %w", err)
		}
	}

	latestBlockNum := latestHeader.Number.Uint64()
	if latestBlockNum >= startBlockNum {
		return latestHeader, latestBlockNum, nil
	}

	latestHeader, err = fetchBorHeader(ctx, contractCaller, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to refresh the latest header: %w", err)
	}
	latestBlockNum = latestHeader.Number.Uint64()
	if latestBlockNum < startBlockNum {
		// Bor hasn't produced the block yet. GenMilestoneProposition will
		// propagate this, and app/abci.go will handle it gracefully.
		return nil, 0, ErrNoNewHeadersFound
	}
	return latestHeader, latestBlockNum, nil
}

// resolveParentHash returns the parent hash of the proposition's first
// block. It only needs a fresh RPC call when the proposition doesn't
// immediately follow the last milestone (a gap), since headers[0]'s own
// parent hash isn't the one we need in that case.
func resolveParentHash(ctx sdk.Context, contractCaller helper.IContractCaller, headers []*ethTypes.Header, lastMilestoneHash []byte, lastMilestoneBlock, startBlockNum uint64) ([]byte, error) {
	if len(headers) == 0 || len(lastMilestoneHash) == 0 {
		return nil, nil
	}

	if startBlockNum-lastMilestoneBlock <= 1 {
		return headers[0].ParentHash.Bytes(), nil
	}

	header, err := fetchBorHeader(ctx, contractCaller, big.NewInt(int64(lastMilestoneBlock+1)))
	if err != nil {
		return nil, fmt.Errorf("failed to get header for parent hash: %w", err)
	}

	return header.ParentHash.Bytes(), nil
}
