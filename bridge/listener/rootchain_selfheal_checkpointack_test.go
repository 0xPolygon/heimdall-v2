package listener

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
)

// mockL1Receipt starts a JSON-RPC HTTP server that answers
// eth_getTransactionReceipt with the given receipt JSON (an object matching
// go-ethereum's RPC receipt shape) for every request, so ethclient.Dial
// against it can exercise a real TransactionReceipt call.
func mockL1Receipt(t *testing.T, receiptJSON string) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		require.Equal(t, "eth_getTransactionReceipt", req.Method)

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":` + receiptJSON + `}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func zeroBloomHex() string {
	b := make([]byte, 512)
	for i := range b {
		b[i] = '0'
	}
	return `"0x` + string(b) + `"`
}

// checkpointAckTestRoutes maps a checkpoint REST path to the raw JSON body to
// serve for it. Missing routes 404, matching bridge/util's error handling.
type checkpointAckTestRoutes map[string]string

func newCheckpointAckTestListener(t *testing.T, routes checkpointAckTestRoutes) *RootChainListener {
	t.Helper()

	heimdall := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, body)
	}))
	t.Cleanup(heimdall.Close)

	cfg := helper.CustomAppConfig{
		Config: *serverconfig.DefaultConfig(),
		Custom: helper.GetDefaultHeimdallConfig(),
	}
	cfg.Config.API.Address = heimdall.URL
	helper.SetTestConfig(cfg)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	rl := &RootChainListener{
		BaseListener: BaseListener{
			Logger: log.NewNopLogger(),
			cliCtx: client.Context{}.WithCodec(cdc),
		},
	}
	return rl
}

// testEventTopic is the fixed topic hash used by newReceiptResolveTestListener
// for whichever single event the caller is testing (only one event is ever
// registered in eventMap per test, so any fixed hash works).
var testEventTopic = common.HexToHash("0x9999999999999999999999999999999999999999999999999999999999999999")

// newReceiptResolveTestListener wires both a mocked heimdall REST API (for
// ChainManager params) and a mocked L1 JSON-RPC endpoint (for
// TransactionReceipt) so resolveCheckpointAckLog / getStateSynced can be
// exercised end to end without a real devnet. eventName/testEventTopic are
// registered in eventMap so eventTopicByName resolves, matching what
// resolveCheckpointAckLog/getStateSynced look up in production.
//
// Must NOT run in parallel: helper.SetTestConfig mutates global config.
func newReceiptResolveTestListener(t *testing.T, rootChainAddr, stateSenderAddr, eventName, receiptJSON string) *RootChainListener {
	t.Helper()

	rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{
		util.ChainManagerParamsURL: `{"params":{"chain_params":{
			"root_chain_address": "` + rootChainAddr + `",
			"state_sender_address": "` + stateSenderAddr + `"
		}}}`,
	})
	rl.eventMap = map[common.Hash]*abi.Event{testEventTopic: {Name: eventName}}

	rpcURL := mockL1Receipt(t, receiptJSON)
	client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	rl.contractCaller.MainChainClient = client

	return rl
}

// TestCheckpointAckReady exercises checkpointAckReady's ready/not-ready/error
// branches against a mocked heimdall REST API — no L1 RPC involved, since
// checkpointAckReady only reads Heimdall's own checkpoint state.
//
// Must NOT run in parallel: helper.SetTestConfig mutates global config.
func TestCheckpointAckReady(t *testing.T) {
	const (
		paramsBody = `{"params":{"child_chain_block_interval":"10000"}}`
	)

	t.Run("ready when buffered checkpoint matches the L1 checkpoint", func(t *testing.T) {
		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{
			util.CheckpointParamsURL:   paramsBody,
			util.CountCheckpointURL:    `{"ack_count":"5"}`,
			util.BufferedCheckpointURL: `{"checkpoint":{"id":"6"}}`,
		})

		id, ready, err := rl.checkpointAckReady(&newHeaderBlock{HeaderBlockId: "60000"})
		require.NoError(t, err)
		require.True(t, ready)
		require.Equal(t, uint64(6), id)
	})

	t.Run("not ready when heimdall already synced to the L1 checkpoint", func(t *testing.T) {
		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{
			util.CheckpointParamsURL: paramsBody,
			util.CountCheckpointURL:  `{"ack_count":"6"}`,
		})

		id, ready, err := rl.checkpointAckReady(&newHeaderBlock{HeaderBlockId: "60000"})
		require.NoError(t, err)
		require.False(t, ready)
		require.Equal(t, uint64(6), id)
	})

	t.Run("not ready when buffered checkpoint is empty", func(t *testing.T) {
		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{
			util.CheckpointParamsURL:   paramsBody,
			util.CountCheckpointURL:    `{"ack_count":"5"}`,
			util.BufferedCheckpointURL: `{"checkpoint":{"id":"0"}}`,
		})

		_, ready, err := rl.checkpointAckReady(&newHeaderBlock{HeaderBlockId: "60000"})
		require.NoError(t, err)
		require.False(t, ready)
	})

	t.Run("not ready when buffered checkpoint id doesn't match the L1 checkpoint", func(t *testing.T) {
		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{
			util.CheckpointParamsURL:   paramsBody,
			util.CountCheckpointURL:    `{"ack_count":"5"}`,
			util.BufferedCheckpointURL: `{"checkpoint":{"id":"7"}}`,
		})

		_, ready, err := rl.checkpointAckReady(&newHeaderBlock{HeaderBlockId: "60000"})
		require.NoError(t, err)
		require.False(t, ready)
	})

	t.Run("errors on malformed L1 header block id", func(t *testing.T) {
		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{})

		_, ready, err := rl.checkpointAckReady(&newHeaderBlock{HeaderBlockId: "not-a-number"})
		require.Error(t, err)
		require.False(t, ready)
	})

	t.Run("errors when checkpoint params are unavailable", func(t *testing.T) {
		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{})

		_, ready, err := rl.checkpointAckReady(&newHeaderBlock{HeaderBlockId: "60000"})
		require.Error(t, err)
		require.False(t, ready)
	})

	t.Run("errors when checkpoint ack count is unavailable", func(t *testing.T) {
		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{
			util.CheckpointParamsURL: paramsBody,
		})

		_, ready, err := rl.checkpointAckReady(&newHeaderBlock{HeaderBlockId: "60000"})
		require.Error(t, err)
		require.False(t, ready)
	})

	t.Run("errors when buffered checkpoint is unavailable", func(t *testing.T) {
		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{
			util.CheckpointParamsURL: paramsBody,
			util.CountCheckpointURL:  `{"ack_count":"5"}`,
		})

		_, ready, err := rl.checkpointAckReady(&newHeaderBlock{HeaderBlockId: "60000"})
		require.Error(t, err)
		require.False(t, ready)
	})
}

// TestResolveCheckpointAckLog exercises resolveCheckpointAckLog end to end
// against a mocked ChainManager-params REST endpoint and a mocked L1
// JSON-RPC endpoint serving the transaction receipt — a real address-mismatch
// rejection this way, not just the isolated validateReceiptLog unit test.
func TestResolveCheckpointAckLog(t *testing.T) {
	const rootChainAddr = "0x107a27363FE1Ba8578B8a8c76acDB237cDB35533"
	const otherAddr = "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

	receiptWithLogFrom := func(addr string) string {
		return `{
			"status": "0x1",
			"transactionHash": "0x0000000000000000000000000000000000000000000000000000000000000abc",
			"transactionIndex": "0x0",
			"blockHash": "0x0000000000000000000000000000000000000000000000000000000000000001",
			"blockNumber": "0x1",
			"cumulativeGasUsed": "0x0",
			"gasUsed": "0x0",
			"logsBloom": ` + zeroBloomHex() + `,
			"logs": [{
				"address": "` + addr + `",
				"topics": ["0x9999999999999999999999999999999999999999999999999999999999999999"],
				"data": "0x",
				"blockNumber": "0x1",
				"transactionHash": "0x0000000000000000000000000000000000000000000000000000000000000abc",
				"transactionIndex": "0x0",
				"blockHash": "0x0000000000000000000000000000000000000000000000000000000000000001",
				"logIndex": "0x3",
				"removed": false
			}]
		}`
	}

	t.Run("accepts a log from the expected RootChain address", func(t *testing.T) {
		rl := newReceiptResolveTestListener(t, rootChainAddr, "0x0000000000000000000000000000000000000001", helper.NewHeaderBlockEvent, receiptWithLogFrom(rootChainAddr))

		log, err := rl.resolveCheckpointAckLog(t.Context(), &newHeaderBlock{TransactionHash: "0x0000000000000000000000000000000000000000000000000000000000000abc", LogIndex: "3"})
		require.NoError(t, err)
		require.NotNil(t, log)
	})

	t.Run("rejects a log from an address that doesn't match RootChain", func(t *testing.T) {
		rl := newReceiptResolveTestListener(t, rootChainAddr, "0x0000000000000000000000000000000000000001", helper.NewHeaderBlockEvent, receiptWithLogFrom(otherAddr))

		_, err := rl.resolveCheckpointAckLog(t.Context(), &newHeaderBlock{TransactionHash: "0x0000000000000000000000000000000000000000000000000000000000000abc", LogIndex: "3"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not match expected contract")
	})

	t.Run("rejects a log with the right address but a different topic", func(t *testing.T) {
		receiptJSON := `{
			"status": "0x1",
			"transactionHash": "0x0000000000000000000000000000000000000000000000000000000000000abc",
			"transactionIndex": "0x0",
			"blockHash": "0x0000000000000000000000000000000000000000000000000000000000000001",
			"blockNumber": "0x1",
			"cumulativeGasUsed": "0x0",
			"gasUsed": "0x0",
			"logsBloom": ` + zeroBloomHex() + `,
			"logs": [{
				"address": "` + rootChainAddr + `",
				"topics": ["0x8888888888888888888888888888888888888888888888888888888888888888"],
				"data": "0x",
				"blockNumber": "0x1",
				"transactionHash": "0x0000000000000000000000000000000000000000000000000000000000000abc",
				"transactionIndex": "0x0",
				"blockHash": "0x0000000000000000000000000000000000000000000000000000000000000001",
				"logIndex": "0x3",
				"removed": false
			}]
		}`
		rl := newReceiptResolveTestListener(t, rootChainAddr, "0x0000000000000000000000000000000000000001", helper.NewHeaderBlockEvent, receiptJSON)

		_, err := rl.resolveCheckpointAckLog(t.Context(), &newHeaderBlock{TransactionHash: "0x0000000000000000000000000000000000000000000000000000000000000abc", LogIndex: "3"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "topic does not match")
	})

	t.Run("errors when chain manager params are unavailable", func(t *testing.T) {
		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{})

		_, err := rl.resolveCheckpointAckLog(t.Context(), &newHeaderBlock{TransactionHash: "0x0000000000000000000000000000000000000000000000000000000000000abc", LogIndex: "3"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "chain manager params")
	})

	t.Run("errors when the NewHeaderBlock topic isn't registered", func(t *testing.T) {
		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{
			util.ChainManagerParamsURL: `{"params":{"chain_params":{"root_chain_address": "` + rootChainAddr + `"}}}`,
		})
		// eventMap deliberately left nil/empty: eventTopicByName must fail closed.

		_, err := rl.resolveCheckpointAckLog(t.Context(), &newHeaderBlock{TransactionHash: "0x0000000000000000000000000000000000000000000000000000000000000abc", LogIndex: "3"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "no known topic")
	})

	t.Run("errors when the L1 receipt fetch fails", func(t *testing.T) {
		rl := newReceiptResolveTestListener(t, rootChainAddr, "0x0000000000000000000000000000000000000001", helper.NewHeaderBlockEvent, receiptWithLogFrom(rootChainAddr))
		brokenClient, err := ethclient.Dial("http://127.0.0.1:1")
		require.NoError(t, err)
		rl.contractCaller.MainChainClient = brokenClient

		_, err = rl.resolveCheckpointAckLog(t.Context(), &newHeaderBlock{TransactionHash: "0x0000000000000000000000000000000000000000000000000000000000000abc", LogIndex: "3"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get transaction receipt")
	})
}

// TestGetStateSynced_ReceiptValidation exercises getStateSynced's receipt
// fetch and address-validation path (the subgraph-query half is covered by
// TestSubgraphErrorChecks_OtherEntities).
func TestGetStateSynced_ReceiptValidation(t *testing.T) {
	const stateSenderAddr = "0xb661f1577e8749456FA749d44e8A77A9d604e603"
	const otherAddr = "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

	receiptWithLogFrom := func(addr string) string {
		return `{
			"status": "0x1",
			"transactionHash": "0x0000000000000000000000000000000000000000000000000000000000000abc",
			"transactionIndex": "0x0",
			"blockHash": "0x0000000000000000000000000000000000000000000000000000000000000001",
			"blockNumber": "0x1",
			"cumulativeGasUsed": "0x0",
			"gasUsed": "0x0",
			"logsBloom": ` + zeroBloomHex() + `,
			"logs": [{
				"address": "` + addr + `",
				"topics": ["0x9999999999999999999999999999999999999999999999999999999999999999"],
				"data": "0x",
				"blockNumber": "0x1",
				"transactionHash": "0x0000000000000000000000000000000000000000000000000000000000000abc",
				"transactionIndex": "0x0",
				"blockHash": "0x0000000000000000000000000000000000000000000000000000000000000001",
				"logIndex": "0x0",
				"removed": false
			}]
		}`
	}

	newStateSyncedTestListener := func(t *testing.T, receiptJSON string) *RootChainListener {
		t.Helper()
		rl := newReceiptResolveTestListener(t, "0x0000000000000000000000000000000000000001", stateSenderAddr, helper.StateSyncedEvent, receiptJSON)
		return rl
	}

	t.Run("accepts a log from the expected StateSender address", func(t *testing.T) {
		graph := newSubgraph(`{"data":{"stateSynceds":[{"logIndex":"0","transactionHash":"0xabc"}]}}`)
		defer graph.Close()

		rl := newStateSyncedTestListener(t, receiptWithLogFrom(stateSenderAddr))
		rl.subGraphClient = &subGraphClient{graphUrl: graph.URL, httpClient: http.DefaultClient}

		log, err := rl.getStateSynced(t.Context(), 5)
		require.NoError(t, err)
		require.NotNil(t, log)
	})

	t.Run("rejects a log from an address that doesn't match StateSender", func(t *testing.T) {
		graph := newSubgraph(`{"data":{"stateSynceds":[{"logIndex":"0","transactionHash":"0xabc"}]}}`)
		defer graph.Close()

		rl := newStateSyncedTestListener(t, receiptWithLogFrom(otherAddr))
		rl.subGraphClient = &subGraphClient{graphUrl: graph.URL, httpClient: http.DefaultClient}

		_, err := rl.getStateSynced(t.Context(), 5)
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not match expected contract")
	})

	t.Run("errors when chain manager params are unavailable", func(t *testing.T) {
		graph := newSubgraph(`{"data":{"stateSynceds":[{"logIndex":"0","transactionHash":"0xabc"}]}}`)
		defer graph.Close()

		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{})
		rl.subGraphClient = &subGraphClient{graphUrl: graph.URL, httpClient: http.DefaultClient}

		_, err := rl.getStateSynced(t.Context(), 5)
		require.Error(t, err)
		require.Contains(t, err.Error(), "chain manager params")
	})

	t.Run("errors when the StateSynced topic isn't registered", func(t *testing.T) {
		graph := newSubgraph(`{"data":{"stateSynceds":[{"logIndex":"0","transactionHash":"0xabc"}]}}`)
		defer graph.Close()

		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{
			util.ChainManagerParamsURL: `{"params":{"chain_params":{"state_sender_address": "` + stateSenderAddr + `"}}}`,
		})
		rl.subGraphClient = &subGraphClient{graphUrl: graph.URL, httpClient: http.DefaultClient}
		// eventMap deliberately left nil/empty: eventTopicByName must fail closed.

		_, err := rl.getStateSynced(t.Context(), 5)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no known topic")
	})

	t.Run("errors when the L1 receipt fetch fails", func(t *testing.T) {
		graph := newSubgraph(`{"data":{"stateSynceds":[{"logIndex":"0","transactionHash":"0xabc"}]}}`)
		defer graph.Close()

		rl := newStateSyncedTestListener(t, receiptWithLogFrom(stateSenderAddr))
		rl.subGraphClient = &subGraphClient{graphUrl: graph.URL, httpClient: http.DefaultClient}
		brokenClient, err := ethclient.Dial("http://127.0.0.1:1")
		require.NoError(t, err)
		rl.contractCaller.MainChainClient = brokenClient

		_, err = rl.getStateSynced(t.Context(), 5)
		require.Error(t, err)
		require.Contains(t, err.Error(), "connection refused")
	})
}
