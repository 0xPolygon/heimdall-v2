package helper

import (
	"bytes"
	"context"
	"fmt"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	httpClient "github.com/cometbft/cometbft/rpc/client/http"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	cmtTypes "github.com/cometbft/cometbft/types"
	cosmosContext "github.com/cosmos/cosmos-sdk/client"
	"github.com/pkg/errors"
)

const (
	CommitTimeout = 2 * time.Minute
)

// GetNodeStatus returns node status
func GetNodeStatus(ctx context.Context, cliCtx cosmosContext.Context) (*ctypes.ResultStatus, error) {
	node, err := cliCtx.GetNode()
	if err != nil {
		return nil, err
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return node.Status(ctxWithTimeout)
}

// QueryTxWithProof query tx with proof from the node
func QueryTxWithProof(cliCtx cosmosContext.Context, hash []byte) (*ctypes.ResultTx, error) {
	node, err := cliCtx.GetNode()
	if err != nil {
		return nil, err
	}

	ctx := cliCtx.CmdContext

	if ctx == nil {
		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ctx = ctxWithTimeout
	}

	return node.Tx(ctx, hash, true)
}

// QueryTxBytesFromBlock returns the raw bytes of the tx with the given hash
// inside the block at the given height. It reads from the BlockStore via the
// node's /block RPC and does not depend on the cometbft tx_index — so it works
// even when the node is configured with `indexer = "null"`.
//
// Used by the bridge checkpoint flow which previously called node.Tx(hash, true)
// just to retrieve the tx bytes for sign-bytes recomputation. The bridge already
// has the block height in hand, so the indexer detour is unnecessary.
func QueryTxBytesFromBlock(cliCtx cosmosContext.Context, hash []byte, height int64) ([]byte, error) {
	node, err := cliCtx.GetNode()
	if err != nil {
		return nil, err
	}

	ctx := cliCtx.CmdContext
	if ctx == nil {
		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ctx = ctxWithTimeout
	}

	blk, err := node.Block(ctx, &height)
	if err != nil {
		return nil, err
	}
	if blk == nil || blk.Block == nil {
		return nil, fmt.Errorf("block %d not available", height)
	}
	return findTxInBlock(blk.Block.Txs, hash)
}

// findTxInBlock scans the block's tx list for the tx whose hash matches `hash`
// and returns its raw bytes. Pure function — extracted for testability.
func findTxInBlock(txs cmtTypes.Txs, hash []byte) ([]byte, error) {
	for _, raw := range txs {
		if bytes.Equal(raw.Hash(), hash) {
			return raw, nil
		}
	}
	return nil, fmt.Errorf("tx %X not found in block", hash)
}

// GetBeginBlockEvents get block through per height
func GetBeginBlockEvents(ctx context.Context, client *httpClient.HTTP, height int64) ([]abci.Event, error) {
	c, cancel := context.WithTimeout(ctx, CommitTimeout)
	defer cancel()

	// Try to get block results directly
	blockResults, err := client.BlockResults(c, &height)
	if err == nil && blockResults != nil {
		return blockResults.FinalizeBlockEvents, nil
	}

	// Only fallthrough to subscription if the block hasn't been committed yet.
	// Otherwise, return the BlockResults error.
	// Subscribing for an already-committed block (e.g., block pruned) would time out.
	latestStatus, statusErr := client.Status(c)
	if statusErr != nil || latestStatus == nil || height <= latestStatus.SyncInfo.LatestBlockHeight {
		return nil, fmt.Errorf("BlockResults failed for block %d (possibly pruned or unavailable): %w", height, err)
	}

	// Block is in the future, subscribe and wait for it
	subscriber := fmt.Sprintf("new-block-%v", height)
	query := cmtTypes.QueryForEvent(cmtTypes.EventNewBlock).String()

	eventCh, subscribeErr := client.Subscribe(c, subscriber, query)
	if subscribeErr != nil {
		return nil, errors.Wrap(subscribeErr, "failed to subscribe")
	}

	defer func() {
		if unsubscribeErr := client.Unsubscribe(c, subscriber, query); unsubscribeErr != nil {
			Logger.Error("GetBeginBlockEvents: error unsubscribing", "error", unsubscribeErr)
		}
	}()

	for {
		select {
		case event := <-eventCh:
			switch t := event.Data.(type) {
			case cmtTypes.EventDataNewBlock:
				if t.Block.Height == height {
					return t.ResultFinalizeBlock.GetEvents(), nil
				}
			default:
				Logger.Error("GetBeginBlockEvents", "unexpected event type", fmt.Sprintf("%+v", t))
				return nil, fmt.Errorf("unexpected event type: %T", t)
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-c.Done():
			return nil, errors.New("timed out waiting for event")
		}
	}
}
