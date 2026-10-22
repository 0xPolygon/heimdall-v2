package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	httpClient "github.com/cometbft/cometbft/rpc/client/http"
	cmtTypes "github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/require"
)

// jsonRPCRequest mirrors the CometBFT JSON-RPC request format for parsing in tests.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// jsonRPCResponse is the JSON-RPC response format expected by the CometBFT client.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func writeJSONRPC(w http.ResponseWriter, resp jsonRPCResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func errorResp(id json.RawMessage, code int, msg string) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonRPCError{Code: code, Message: msg},
	}
}

func resultResp(id json.RawMessage, rawJSON string) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  json.RawMessage(rawJSON),
	}
}

func TestGetBeginBlockEvents_BlockResultsSuccess(t *testing.T) {
	t.Parallel()

	blockResultJSON := `{
		"height": "100",
		"finalize_block_events": [
			{
				"type": "test-event",
				"attributes": [
					{"key": "key1", "value": "value1", "index": false}
				]
			}
		]
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		switch req.Method {
		case "block_results":
			writeJSONRPC(w, resultResp(req.ID, blockResultJSON))
		default:
			writeJSONRPC(w, errorResp(req.ID, -32601, fmt.Sprintf("method %s not found", req.Method)))
		}
	}))
	defer ts.Close()

	client, err := httpClient.New(ts.URL, "/websocket")
	require.NoError(t, err)

	events, err := GetBeginBlockEvents(context.Background(), client, 100)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "test-event", events[0].Type)
	require.Equal(t, "key1", events[0].Attributes[0].Key)
	require.Equal(t, "value1", events[0].Attributes[0].Value)
}

func TestGetBeginBlockEvents_BlockResultsFails(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSONRPC(w, errorResp(req.ID, -32603, "block results not available (pruned)"))
	}))
	defer ts.Close()

	client, err := httpClient.New(ts.URL, "/websocket")
	require.NoError(t, err)

	events, err := GetBeginBlockEvents(context.Background(), client, 100)
	require.Error(t, err)
	require.Nil(t, events)
	require.Contains(t, err.Error(), "BlockResults failed for block 100")
	require.Contains(t, err.Error(), "possibly pruned or unavailable")
}

// TestFindTxInBlock_HappyPath verifies the helper returns the matching tx
// bytes for a hash that exists in the block.
func TestFindTxInBlock_HappyPath(t *testing.T) {
	t.Parallel()

	tx1 := cmtTypes.Tx("checkpoint-tx-bytes-aaaa")
	tx2 := cmtTypes.Tx("other-tx-bytes-bbbb")
	tx3 := cmtTypes.Tx("decoy-tx-bytes-cccc")
	txs := cmtTypes.Txs{tx1, tx2, tx3}

	got, err := findTxInBlock(txs, tx2.Hash())
	require.NoError(t, err)
	require.Equal(t, []byte(tx2), got)
}

// TestFindTxInBlock_NotFound verifies a non-matching hash returns an error
// rather than wrong bytes — protects the bridge from silently using the
// wrong checkpoint tx.
func TestFindTxInBlock_NotFound(t *testing.T) {
	t.Parallel()

	tx1 := cmtTypes.Tx("checkpoint-tx-bytes-aaaa")
	tx2 := cmtTypes.Tx("other-tx-bytes-bbbb")
	txs := cmtTypes.Txs{tx1, tx2}

	missing := cmtTypes.Tx("does-not-exist").Hash()
	got, err := findTxInBlock(txs, missing)
	require.Error(t, err)
	require.Nil(t, got)
	require.Contains(t, err.Error(), "not found in block")
}

// TestFindTxInBlock_EmptyBlock verifies the helper returns an error for a
// block with no txs.
func TestFindTxInBlock_EmptyBlock(t *testing.T) {
	t.Parallel()

	tx := cmtTypes.Tx("some-tx-bytes")
	got, err := findTxInBlock(cmtTypes.Txs{}, tx.Hash())
	require.Error(t, err)
	require.Nil(t, got)
}
