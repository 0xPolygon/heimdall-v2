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
	var events []abci.Event
	var err error

	c, cancel := context.WithTimeout(ctx, CommitTimeout)
	defer cancel()

	// get block using the client
	blockResults, err := client.BlockResults(c, &height)
	if err == nil && blockResults != nil {
		events = blockResults.FinalizeBlockEvents
		return events, nil
	}

	// subscriber
	subscriber := fmt.Sprintf("new-block-%v", height)

	// query for event
	query := cmtTypes.QueryForEvent(cmtTypes.EventNewBlock).String()

	// register for the next event of this type
	eventCh, err := client.Subscribe(c, subscriber, query)
	if err != nil {
		return events, errors.Wrap(err, "failed to subscribe")
	}

	// unsubscribe query
	defer func() {
		if unsubscribeErr := client.Unsubscribe(c, subscriber, query); unsubscribeErr != nil && err == nil {
			err = unsubscribeErr
			events = nil // Set events to nil when returning an error
		}
	}()

	for {
		select {
		case event := <-eventCh:
			eventData := event.Data
			switch t := eventData.(type) {
			case cmtTypes.EventDataNewBlock:
				if t.Block.Height == height {
					events = t.ResultFinalizeBlock.GetEvents()
					return events, err
				}
			default:
				Logger.Error("GetBeginBlockEvents", "unexpected event type", fmt.Sprintf("%+v", t))
				return events, fmt.Errorf("unexpected event type: %T", t)
			}
		case <-ctx.Done():
			// Parent context canceled - return immediately
			return events, ctx.Err()
		case <-c.Done():
			return events, errors.New("timed out waiting for event")
		}
	}
}
