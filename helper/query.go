package helper

import (
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
func GetNodeStatus(cliCtx cosmosContext.Context) (*ctypes.ResultStatus, error) {
	node, err := cliCtx.GetNode()
	if err != nil {
		return nil, err
	}

	return node.Status(cliCtx.CmdContext)
}

// TODO HV2 Implement this later if needed
/*
QueryTxsByEvents performs a search for transactions for a given set of tags via
cometBFT RPC. It returns a slice of Info object containing txs and metadata.
An error is returned if the query fails.
func QueryTxsByEvents(cliCtx cosmosContext.CLIContext, tags []string, page, limit int) (*sdk.SearchTxsResult, error) {
	if len(tags) == 0 {
		return nil, errors.New("must declare at least one tag to search")
	}

	if page <= 0 {
		return nil, errors.New("page must greater than 0")
	}

	if limit <= 0 {
		return nil, errors.New("limit must greater than 0")
	}

	// XXX: implement ANY
	query := strings.Join(tags, " AND ")

	node, err := cliCtx.GetNode()
	if err != nil {
		return nil, err
	}

	prove := !cliCtx.TrustNode

	resTxs, err := node.TxSearch(query, prove, page, limit)
	if err != nil {
		return nil, err
	}

	if prove {
		for _, tx := range resTxs.Txs {
			err := ValidateTxResult(cliCtx, tx)
			if err != nil {
				return nil, err
			}
		}
	}

	resBlocks, err := getBlocksForTxResults(cliCtx, resTxs.Txs)
	if err != nil {
		return nil, err
	}

	txs, err := formatTxResults(cliCtx.Codec, resTxs.Txs, resBlocks)
	if err != nil {
		return nil, err
	}

	result := sdk.NewSearchTxsResult(resTxs.TotalCount, len(txs), page, limit, txs)

	return &result, nil
}

formatTxResults parses the indexed txs into a slice of TxResponse objects.
func formatTxResults(cdc *codec.Codec, resTxs []*ctypes.ResultTx, resBlocks map[int64]*ctypes.ResultBlock) ([]sdk.TxResponse, error) {
	var err error

	out := make([]sdk.TxResponse, len(resTxs))
	for i := range resTxs {
		out[i], err = formatTxResult(cdc, resTxs[i], resBlocks[resTxs[i].Height])
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

*/

// TODO HV2 Verify function is not available, discuss this with informal team
// ValidateTxResult performs transaction verification.
/*
func ValidateTxResult(cliCtx cosmosContext.Context, resTx *ctypes.ResultTx) error {

		check, err := cliCtx.Verify(resTx.Height)
		if err != nil {
			return err
		}

		err = resTx.Proof.Validate(check.Header.DataHash)

		// Accept if only one tx in block and data hash matches tx hash
		if err != nil &&
			check.Header.NumTxs == 1 &&
			bytes.Equal(check.Header.DataHash, resTx.Hash) &&
			bytes.Equal(check.Header.DataHash, resTx.Tx.Hash()) &&
			resTx.Index == 0 {
			err = nil
		}

		if err != nil {
			return err
		}

	return nil
}
*/

//lint:ignore U1000 ignore unused error
func getBlocksForTxResults(cliCtx cosmosContext.Context, resTxs []*ctypes.ResultTx) (map[int64]*ctypes.ResultBlock, error) {
	node, err := cliCtx.GetNode()
	if err != nil {
		return nil, err
	}

	resBlocks := make(map[int64]*ctypes.ResultBlock)

	for _, resTx := range resTxs {
		if _, ok := resBlocks[resTx.Height]; !ok {
			resBlock, err := node.Block(cliCtx.CmdContext, &resTx.Height)
			if err != nil {
				return nil, err
			}

			resBlocks[resTx.Height] = resBlock
		}
	}

	return resBlocks, nil
}

// TODO HV2 Implement it later if needed
/*
func formatTxResult(cdc *codec.Codec, resTx *ctypes.ResultTx, resBlock *ctypes.ResultBlock) (sdk.TxResponse, error) {
	tx, err := parseTx(cdc, resTx.Tx)
	if err != nil {
		return sdk.TxResponse{}, err
	}

	return sdk.NewResponseResultTx(resTx, tx, resBlock.Block.Time.Format(time.RFC3339)), nil
}

func parseTx(cdc *codec.Codec, txBytes []byte) (sdk.Tx, error) {
	decoder := GetTxDecoder(cdc)
	return decoder(txBytes)
}

//QueryTx query tx from node
func QueryTx(cliCtx cosmosContext.CLIContext, hashHexStr string) (sdk.TxResponse, error) {
	hash, err := hex.DecodeString(hashHexStr)
	if err != nil {
		return sdk.TxResponse{}, err
	}

	node, err := cliCtx.GetNode()
	if err != nil {
		return sdk.TxResponse{}, err
	}

	resTx, err := node.Tx(hash, !cliCtx.TrustNode)
	if err != nil {
		return sdk.TxResponse{}, err
	}

	if !cliCtx.TrustNode {
		if err = ValidateTxResult(cliCtx, resTx); err != nil {
			return sdk.TxResponse{}, err
		}
	}

	resBlocks, err := getBlocksForTxResults(cliCtx, []*ctypes.ResultTx{resTx})
	if err != nil {
		return sdk.TxResponse{}, err
	}

	out, err := formatTxResult(cliCtx.Codec, resTx, resBlocks[resTx.Height])
	if err != nil {
		return out, err
	}

	return out, nil
}

*/

// QueryTxWithProof query tx with proof from node
func QueryTxWithProof(cliCtx cosmosContext.Context, hash []byte) (*ctypes.ResultTx, error) {
	node, err := cliCtx.GetNode()
	if err != nil {
		return nil, err
	}

	return node.Tx(cliCtx.CmdContext, hash, true)
}

// GetBlock returns a block
func GetBlock(cliCtx cosmosContext.Context, height int64) (*ctypes.ResultBlock, error) {
	node, err := cliCtx.GetNode()
	if err != nil {
		return nil, err
	}

	return node.Block(cliCtx.CmdContext, &height)
}

// GetBlockWithClient gets a block given its height
func GetBlockWithClient(client *httpClient.HTTP, height int64) (*cmtTypes.Block, error) {
	c, cancel := context.WithTimeout(context.Background(), CommitTimeout)
	defer cancel()

	// get block using client
	block, err := client.Block(c, &height)
	if err == nil && block != nil {
		return block.Block, nil
	}

	// subscriber
	subscriber := fmt.Sprintf("new-block-%d", height)

	// query for event
	query := cmtTypes.QueryForEvent(cmtTypes.EventNewBlock).String()

	// register for the next event of this type
	eventCh, err := client.Subscribe(c, subscriber, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to subscribe")
	}

	// unsubscribe query
	defer func() {
		if err := client.Unsubscribe(c, subscriber, query); err != nil {
			Logger.Error("GetBlockWithClient | Unsubscribe", "Error", err)
		}
	}()

	for {
		select {
		case event := <-eventCh:
			eventData := event.Data
			switch t := eventData.(type) {
			case cmtTypes.EventDataNewBlock:
				if t.Block.Height == height {
					return t.Block, nil
				}
			default:
				return nil, errors.New("received event is not of block event type")
			}
		case <-c.Done():
			return nil, errors.New("timed out waiting for event")
		}
	}
}

// GetFinalizeBlockEvents gets the beginBlock events for a given height
func GetFinalizeBlockEvents(client *httpClient.HTTP, height int64) ([]abci.Event, error) {
	c, cancel := context.WithTimeout(context.Background(), CommitTimeout)
	defer cancel()

	blockResults, err := client.BlockResults(c, &height)
	if err == nil && blockResults != nil {
		return blockResults.FinalizeBlockEvents, nil
	}

	// subscriber
	subscriber := fmt.Sprintf("finalize-block-%v", height)

	// query for event
	query := cmtTypes.QueryForEvent(cmtTypes.EventNewBlock).String()

	// register for the next event of this type
	eventCh, err := client.Subscribe(c, subscriber, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to subscribe")
	}

	// unsubscribe query
	defer func() {
		err = client.Unsubscribe(c, subscriber, query)
		if err != nil {
			Logger.Error("error while unsubscribing", "error", err)
		}
	}()

	for {
		select {
		case event := <-eventCh:
			eventData := event.Data
			switch t := eventData.(type) {
			case cmtTypes.EventDataNewBlock:
				if t.Block.Height == height {
					// TODO HV2 Fetching all the events ,not the begin block one
					return t.ResultFinalizeBlock.GetEvents(), nil
				}
			default:
				return nil, errors.New("received event is not of block event type")
			}
		case <-c.Done():
			return nil, errors.New("timed out waiting for event")
		}
	}
}

// GetBeginBlockEvents get block through per height
func GetBeginBlockEvents(client *httpClient.HTTP, height int64) ([]abci.Event, error) {
	var events []abci.Event
	var err error

	c, cancel := context.WithTimeout(context.Background(), CommitTimeout)
	defer cancel()

	// get block using client
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
				return events, errors.New("timed out waiting for event")
			}
		case <-c.Done():
			return events, errors.New("timed out waiting for event")
		}
	}

	return events, err
}
