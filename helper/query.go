package helper

import (
	"context"
	"fmt"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	httpClient "github.com/cometbft/cometbft/rpc/client/http"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmTypes "github.com/cometbft/cometbft/types"
	cosmosContext "github.com/cosmos/cosmos-sdk/client"
	"github.com/pkg/errors"
)

const (
	// CommitTimeout commit timeout
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

// QueryTxsByEvents performs a search for transactions for a given set of tags via
// Tendermint RPC. It returns a slice of Info object containing txs and metadata.
// An error is returned if the query fails.
// func QueryTxsByEvents(cliCtx cosmosContext.CLIContext, tags []string, page, limit int) (*sdk.SearchTxsResult, error) {
// 	if len(tags) == 0 {
// 		return nil, errors.New("must declare at least one tag to search")
// 	}

// 	if page <= 0 {
// 		return nil, errors.New("page must greater than 0")
// 	}

// 	if limit <= 0 {
// 		return nil, errors.New("limit must greater than 0")
// 	}

// 	// XXX: implement ANY
// 	query := strings.Join(tags, " AND ")

// 	node, err := cliCtx.GetNode()
// 	if err != nil {
// 		return nil, err
// 	}

// 	prove := !cliCtx.TrustNode

// 	resTxs, err := node.TxSearch(query, prove, page, limit)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if prove {
// 		for _, tx := range resTxs.Txs {
// 			err := ValidateTxResult(cliCtx, tx)
// 			if err != nil {
// 				return nil, err
// 			}
// 		}
// 	}

// 	resBlocks, err := getBlocksForTxResults(cliCtx, resTxs.Txs)
// 	if err != nil {
// 		return nil, err
// 	}

// 	txs, err := formatTxResults(cliCtx.Codec, resTxs.Txs, resBlocks)
// 	if err != nil {
// 		return nil, err
// 	}

// 	result := sdk.NewSearchTxsResult(resTxs.TotalCount, len(txs), page, limit, txs)

// 	return &result, nil
// }

// formatTxResults parses the indexed txs into a slice of TxResponse objects.
// func formatTxResults(cdc *codec.Codec, resTxs []*ctypes.ResultTx, resBlocks map[int64]*ctypes.ResultBlock) ([]sdk.TxResponse, error) {
// 	var err error

// 	out := make([]sdk.TxResponse, len(resTxs))
// 	for i := range resTxs {
// 		out[i], err = formatTxResult(cdc, resTxs[i], resBlocks[resTxs[i].Height])
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	return out, nil
// }

//TODO HV2 Verify function is not available, discuss this with
// ValidateTxResult performs transaction verification.
// func ValidateTxResult(cliCtx cosmosContext.Context, resTx *ctypes.ResultTx) error {

// 		check, err := cliCtx.Verify(resTx.Height)
// 		if err != nil {
// 			return err
// 		}

// 		err = resTx.Proof.Validate(check.Header.DataHash)

// 		// Accept if only one tx in block and data hash matches tx hash
// 		if err != nil &&
// 			check.Header.NumTxs == 1 &&
// 			bytes.Equal(check.Header.DataHash, resTx.Hash) &&
// 			bytes.Equal(check.Header.DataHash, resTx.Tx.Hash()) &&
// 			resTx.Index == 0 {
// 			err = nil
// 		}

// 		if err != nil {
// 			return err
// 		}

// 	return nil
// }

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

// func formatTxResult(cdc *codec.Codec, resTx *ctypes.ResultTx, resBlock *ctypes.ResultBlock) (sdk.TxResponse, error) {
// 	tx, err := parseTx(cdc, resTx.Tx)
// 	if err != nil {
// 		return sdk.TxResponse{}, err
// 	}

// 	return sdk.NewResponseResultTx(resTx, tx, resBlock.Block.Time.Format(time.RFC3339)), nil
// }

// func parseTx(cdc *codec.Codec, txBytes []byte) (sdk.Tx, error) {
// 	decoder := GetTxDecoder(cdc)
// 	return decoder(txBytes)
// }

// QueryTx query tx from node
// func QueryTx(cliCtx cosmosContext.CLIContext, hashHexStr string) (sdk.TxResponse, error) {
// 	hash, err := hex.DecodeString(hashHexStr)
// 	if err != nil {
// 		return sdk.TxResponse{}, err
// 	}

// 	node, err := cliCtx.GetNode()
// 	if err != nil {
// 		return sdk.TxResponse{}, err
// 	}

// 	resTx, err := node.Tx(hash, !cliCtx.TrustNode)
// 	if err != nil {
// 		return sdk.TxResponse{}, err
// 	}

// 	if !cliCtx.TrustNode {
// 		if err = ValidateTxResult(cliCtx, resTx); err != nil {
// 			return sdk.TxResponse{}, err
// 		}
// 	}

// 	resBlocks, err := getBlocksForTxResults(cliCtx, []*ctypes.ResultTx{resTx})
// 	if err != nil {
// 		return sdk.TxResponse{}, err
// 	}

// 	out, err := formatTxResult(cliCtx.Codec, resTx, resBlocks[resTx.Height])
// 	if err != nil {
// 		return out, err
// 	}

// 	return out, nil
// }

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

// GetBlockWithClient get block through per height
func GetBlockWithClient(client *httpClient.HTTP, height int64) (*tmTypes.Block, error) {
	c, cancel := context.WithTimeout(context.Background(), CommitTimeout)
	defer cancel()

	// get block using client
	block, err := client.Block(c, &height)
	if err == nil && block != nil {
		return block.Block, nil
	}

	// subscriber
	subscriber := fmt.Sprintf("new-block-%v", height)

	// query for event
	query := tmTypes.QueryForEvent(tmTypes.EventNewBlock).String()

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
			case tmTypes.EventDataNewBlock:
				if t.Block.Height == height {
					return t.Block, nil
				}
			default:
				return nil, errors.New("timed out waiting for event")
			}
		case <-c.Done():
			return nil, errors.New("timed out waiting for event")
		}
	}
}

// GetBeginBlockEvents get block through per height
func GetBeginBlockEvents(client *httpClient.HTTP, height int64) ([]abci.Event, error) {
	c, cancel := context.WithTimeout(context.Background(), CommitTimeout)
	defer cancel()

	//TODO HV2 It give all the events, not only begin blocks
	// get block using client
	blockResults, err := client.BlockResults(c, &height)
	if err == nil && blockResults != nil {
		return blockResults.FinalizeBlockEvents, nil
	}

	// subscriber
	subscriber := fmt.Sprintf("new-block-%v", height)

	// query for event
	query := tmTypes.QueryForEvent(tmTypes.EventNewBlock).String()

	// register for the next event of this type
	eventCh, err := client.Subscribe(c, subscriber, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to subscribe")
	}

	// unsubscribe query
	defer func() {
		_ = client.Unsubscribe(c, subscriber, query)
	}()

	for {
		select {
		case event := <-eventCh:
			eventData := event.Data
			switch t := eventData.(type) {
			case tmTypes.EventDataNewBlock:
				if t.Block.Height == height {
					//TODO HV2 Fetching all the events ,not the begin block one
					return t.ResultFinalizeBlock.GetEvents(), nil
				}
			default:
				return nil, errors.New("timed out waiting for event")
			}
		case <-c.Done():
			return nil, errors.New("timed out waiting for event")
		}
	}
}

// FetchVotes fetches votes and extracts sigs from it
func FetchVotes(
	client *httpClient.HTTP,
	height int64,
) (votes []tmTypes.CommitSig, sigs []byte, chainID string, err error) {
	// get block client
	blockDetails, err := GetBlockWithClient(client, height+1)

	if err != nil {
		return nil, nil, "", err
	}

	// extract votes from response
	preCommits := blockDetails.LastCommit.Signatures

	// extract signs from votes
	valSigs := GetVoteSigs(preCommits)

	// extract chainID
	chainID = blockDetails.ChainID

	// return
	return preCommits, valSigs, chainID, nil
}

//TODO HV2 Can use this logic if need
// FetchSideTxSigs fetches side tx sigs from it
// func FetchSideTxSigs(
// 	client *httpClient.HTTP,
// 	height int64,
// 	txHash []byte,
// 	sideTxData []byte,
// ) ([][3]*big.Int, error) {
// 	// get block client
// 	blockDetails, err := GetBlockWithClient(client, height)

// 	if err != nil {
// 		return nil, err
// 	}

// 	// extract votes from response
// 	preCommits := blockDetails.LastCommit.Signatures

// 	// extract side-tx signs from votes
// 	return GetSideTxSigs(txHash, sideTxData, preCommits)
// }
