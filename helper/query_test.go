package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	httpClient "github.com/cometbft/cometbft/rpc/client/http"
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

func TestGetBeginBlockEvents_BlockResultsFailsAndBlockCommitted(t *testing.T) {
	t.Parallel()

	// latest block height = 200 (block 100 already committed).
	statusJSON := `{
		"node_info": {"protocol_version": {"p2p": "0", "block": "0", "app": "0"}, "id": "", "listen_addr": "", "network": "", "version": "", "channels": "", "moniker": "", "other": {"tx_index": "", "rpc_address": ""}},
		"sync_info": {"latest_block_hash": "", "latest_app_hash": "", "latest_block_height": "200", "latest_block_time": "2024-01-01T00:00:00Z", "earliest_block_hash": "", "earliest_app_hash": "", "earliest_block_height": "1", "earliest_block_time": "2024-01-01T00:00:00Z", "catching_up": false},
		"validator_info": {"address": "", "pub_key": {"type": "tendermint/PubKeyEd25519", "value": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}, "voting_power": "0"}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		switch req.Method {
		case "block_results":
			writeJSONRPC(w, errorResp(req.ID, -32603, "block results not available (pruned)"))
		case "status":
			writeJSONRPC(w, resultResp(req.ID, statusJSON))
		default:
			writeJSONRPC(w, errorResp(req.ID, -32601, fmt.Sprintf("method %s not found", req.Method)))
		}
	}))
	defer ts.Close()

	client, err := httpClient.New(ts.URL, "/websocket")
	require.NoError(t, err)

	// Block 100 was already committed (latest is 200), so this should return an error
	// instead of falling through to subscription
	events, err := GetBeginBlockEvents(context.Background(), client, 100)
	require.Error(t, err)
	require.Nil(t, events)
	require.Contains(t, err.Error(), "BlockResults failed for block 100")
	require.Contains(t, err.Error(), "possibly pruned or unavailable")
}

func TestGetBeginBlockEvents_BlockResultsFailsAndStatusFails(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		switch req.Method {
		case "block_results":
			writeJSONRPC(w, errorResp(req.ID, -32603, "block not found"))
		case "status":
			writeJSONRPC(w, errorResp(req.ID, -32603, "status unavailable"))
		default:
			writeJSONRPC(w, errorResp(req.ID, -32601, fmt.Sprintf("method %s not found", req.Method)))
		}
	}))
	defer ts.Close()

	client, err := httpClient.New(ts.URL, "/websocket")
	require.NoError(t, err)

	// When both BlockResults and Status fail, should return an error (not fall through to subscription)
	events, err := GetBeginBlockEvents(context.Background(), client, 100)
	require.Error(t, err)
	require.Nil(t, events)
	require.Contains(t, err.Error(), "BlockResults failed for block 100")
}
