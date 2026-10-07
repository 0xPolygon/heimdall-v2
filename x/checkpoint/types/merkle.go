package types

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/0xPolygon/heimdall-v2/common/cache"
	"github.com/0xPolygon/heimdall-v2/helper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
)

var (
	defaultTTL  = 10 * time.Second
	rootCache   *cache.Cache[[]byte]
	existsCache *cache.Cache[bool]
	initOnce    sync.Once
)

// wrapBorQueryError wraps a RunIsolated failure with borTypes.ErrFailedToQueryBor,
// treating a recovered panic the same as an ordinary error so both reach the
// tolerateBorErr carve-out in app/abci.go identically. See
// contract-interactions.md's Panic Isolation section for why panics aren't
// special-cased here.
func wrapBorQueryError(err error, context string) error {
	return fmt.Errorf("%w: %s: %w", borTypes.ErrFailedToQueryBor, context, err)
}

// IsValidCheckpoint validates if checkpoint rootHash matches or not.
//
// Also reached from ProcessProposal/VerifyVoteExtension; the RunIsolated
// wraps below are intentional there too (see contract-interactions.md).
func IsValidCheckpoint(ctx context.Context, start uint64, end uint64, rootHash []byte, checkpointLength uint64, contractCaller helper.IContractCaller, confirmations uint64) (bool, error) {
	initOnce.Do(func() {
		rootCache = cache.NewCache[[]byte](defaultTTL)
		existsCache = cache.NewCache[bool](defaultTTL)
	})

	existsKey := fmt.Sprintf("%d-%d", end, confirmations)
	exists, err := existsCache.Get(existsKey)

	if !exists || err != nil {
		if err != nil {
			helper.Logger.With("module", "x/checkpoint/types").Debug("Blocks existence not found in cache, querying contract",
				"end", end,
				"confirmations", confirmations,
				"existsKey", existsKey,
				"error", err,
			)
		}
		exists, err := helper.RunIsolated(func() (bool, error) {
			return contractCaller.CheckIfBlocksExist(ctx, end+confirmations)
		})
		if err != nil {
			return false, wrapBorQueryError(err, fmt.Sprintf(
				"block existence check failed (end=%d confirmations=%d target=%d)",
				end, confirmations, end+confirmations,
			))
		}
		if !exists {
			return false, fmt.Errorf(
				"%w: end=%d confirmations=%d target=%d",
				borTypes.ErrBorBlockNotFound,
				end,
				confirmations,
				end+confirmations,
			)
		}

		existsCache.Set(existsKey, exists)
	}

	rootKey := fmt.Sprintf("%d-%d-%d", start, end, checkpointLength)
	root, err := rootCache.Get(rootKey)
	if err != nil {
		helper.Logger.With("module", "x/checkpoint/types").Debug("Root hash not found in cache, querying contract",
			"start", start,
			"end", end,
			"checkpointLength", checkpointLength,
			"rootKey", rootKey,
			"error", err,
		)

		root, err = helper.RunIsolated(func() ([]byte, error) {
			return contractCaller.GetRootHash(ctx, start, end, checkpointLength)
		})
		if err != nil {
			return false, wrapBorQueryError(err, fmt.Sprintf(
				"root hash query failed (start=%d end=%d checkpointLength=%d)",
				start, end, checkpointLength,
			))
		}

		if len(root) > 0 {
			rootCache.Set(rootKey, root)
		}
	}

	if bytes.Equal(root, rootHash) {
		return true, nil
	}

	return false, nil
}
