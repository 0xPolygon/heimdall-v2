package listener

import (
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/contracts/rootchain"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
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

// rootChainABIForTest and stateSenderABIForTest parse the same ABI JSON the
// production ContractCaller loads, so tests can decode real events and
// exercise content validation, rather than a name-only stand-in.
func rootChainABIForTest(t *testing.T) abi.ABI {
	t.Helper()
	a, err := abi.JSON(strings.NewReader(rootchain.RootchainMetaData.ABI))
	require.NoError(t, err)
	return a
}

func stateSenderABIForTest(t *testing.T) abi.ABI {
	t.Helper()
	a, err := abi.JSON(strings.NewReader(statesender.StatesenderMetaData.ABI))
	require.NoError(t, err)
	return a
}

// newReceiptResolveTestListener wires both a mocked heimdall REST API (for
// ChainManager params) and a mocked L1 JSON-RPC endpoint (for
// TransactionReceipt) so resolveCheckpointAckLog / getStateSynced can be
// exercised end to end without a real devnet. The real RootChain/StateSender
// ABIs are loaded onto the contract caller and eventMap is populated with the
// real event ID, so a log's content can actually be decoded — a fabricated
// topic/name pair can't exercise the content-validation (headerBlockId /
// stateId) checks that sit downstream of validateReceiptLog.
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

	rootChainABI := rootChainABIForTest(t)
	stateSenderABI := stateSenderABIForTest(t)
	rl.contractCaller.RootChainABI = rootChainABI
	rl.contractCaller.StateSenderABI = stateSenderABI

	var event abi.Event
	switch eventName {
	case helper.NewHeaderBlockEvent:
		event = rootChainABI.Events[eventName]
	case helper.StateSyncedEvent:
		event = stateSenderABI.Events[eventName]
	default:
		t.Fatalf("unsupported event name %q", eventName)
	}
	rl.eventMap = map[common.Hash]*abi.Event{event.ID: &event}

	rpcURL := mockL1Receipt(t, receiptJSON)
	client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	rl.contractCaller.MainChainClient = client

	return rl
}

// newHeaderBlockLog builds a real, ABI-encodable NewHeaderBlock log: the
// proposer/reward topics are fixed filler values (irrelevant to the tests
// here), headerBlockId is the one field self-heal's content-validation
// actually checks.
func newHeaderBlockLog(t *testing.T, rootChainABI abi.ABI, contractAddr common.Address, headerBlockId int64, blockNumber uint64, blockHash, txHash common.Hash, txIndex, logIndex uint) *types.Log {
	t.Helper()

	event := rootChainABI.Events[helper.NewHeaderBlockEvent]
	var nonIndexed abi.Arguments
	for _, arg := range event.Inputs {
		if !arg.Indexed {
			nonIndexed = append(nonIndexed, arg)
		}
	}
	data, err := nonIndexed.Pack(big.NewInt(1), big.NewInt(2), common.HexToHash("0xroot"))
	require.NoError(t, err)

	proposer := common.HexToAddress("0x1111111111111111111111111111111111111111")
	reward := big.NewInt(0)
	topicCols, err := abi.MakeTopics([]interface{}{proposer}, []interface{}{big.NewInt(headerBlockId)}, []interface{}{reward})
	require.NoError(t, err)

	topics := []common.Hash{event.ID}
	for _, col := range topicCols {
		topics = append(topics, col[0])
	}

	return &types.Log{
		Address:     contractAddr,
		Topics:      topics,
		Data:        data,
		BlockNumber: blockNumber,
		TxHash:      txHash,
		TxIndex:     txIndex,
		BlockHash:   blockHash,
		Index:       logIndex,
	}
}

// newStateSyncedLog builds a real, ABI-encodable StateSynced log; id is the
// field self-heal's content-validation actually checks.
func newStateSyncedLog(t *testing.T, stateSenderABI abi.ABI, contractAddr common.Address, stateId int64, blockNumber uint64, blockHash, txHash common.Hash, txIndex, logIndex uint) *types.Log {
	t.Helper()

	event := stateSenderABI.Events[helper.StateSyncedEvent]
	var nonIndexed abi.Arguments
	for _, arg := range event.Inputs {
		if !arg.Indexed {
			nonIndexed = append(nonIndexed, arg)
		}
	}
	data, err := nonIndexed.Pack([]byte{})
	require.NoError(t, err)

	receiver := common.HexToAddress("0x2222222222222222222222222222222222222222")
	topicCols, err := abi.MakeTopics([]interface{}{big.NewInt(stateId)}, []interface{}{receiver})
	require.NoError(t, err)

	topics := []common.Hash{event.ID}
	for _, col := range topicCols {
		topics = append(topics, col[0])
	}

	return &types.Log{
		Address:     contractAddr,
		Topics:      topics,
		Data:        data,
		BlockNumber: blockNumber,
		TxHash:      txHash,
		TxIndex:     txIndex,
		BlockHash:   blockHash,
		Index:       logIndex,
	}
}

// receiptJSONWithLog marshals a real types.Receipt (matching the go-ethereum
// RPC receipt shape) carrying the given log, so mockL1Receipt serves
// something ethclient can actually decode end to end.
func receiptJSONWithLog(t *testing.T, txHash, blockHash common.Hash, blockNumber uint64, txIndex uint, log *types.Log) string {
	t.Helper()

	receipt := &types.Receipt{
		Status:           types.ReceiptStatusSuccessful,
		TxHash:           txHash,
		BlockNumber:      new(big.Int).SetUint64(blockNumber),
		BlockHash:        blockHash,
		TransactionIndex: txIndex,
		Logs:             []*types.Log{log},
	}
	b, err := json.Marshal(receipt)
	require.NoError(t, err)
	return string(b)
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
	const requestedHeaderBlockId = 60000

	txHash := common.HexToHash("0xabc")
	blockHash := common.HexToHash("0x01")
	const blockNumber uint64 = 1
	const txIndex uint = 0
	const logIndex uint = 3

	t.Run("accepts a log from the expected RootChain address with the requested headerBlockId", func(t *testing.T) {
		rootChainABI := rootChainABIForTest(t)
		log := newHeaderBlockLog(t, rootChainABI, common.HexToAddress(rootChainAddr), requestedHeaderBlockId, blockNumber, blockHash, txHash, txIndex, logIndex)
		receiptJSON := receiptJSONWithLog(t, txHash, blockHash, blockNumber, txIndex, log)
		rl := newReceiptResolveTestListener(t, rootChainAddr, "0x0000000000000000000000000000000000000001", helper.NewHeaderBlockEvent, receiptJSON)

		got, err := rl.resolveCheckpointAckLog(t.Context(), &newHeaderBlock{TransactionHash: txHash.Hex(), LogIndex: "3", HeaderBlockId: "60000"})
		require.NoError(t, err)
		require.NotNil(t, got)
	})

	t.Run("rejects a log from an address that doesn't match RootChain", func(t *testing.T) {
		rootChainABI := rootChainABIForTest(t)
		log := newHeaderBlockLog(t, rootChainABI, common.HexToAddress(otherAddr), requestedHeaderBlockId, blockNumber, blockHash, txHash, txIndex, logIndex)
		receiptJSON := receiptJSONWithLog(t, txHash, blockHash, blockNumber, txIndex, log)
		rl := newReceiptResolveTestListener(t, rootChainAddr, "0x0000000000000000000000000000000000000001", helper.NewHeaderBlockEvent, receiptJSON)

		_, err := rl.resolveCheckpointAckLog(t.Context(), &newHeaderBlock{TransactionHash: txHash.Hex(), LogIndex: "3", HeaderBlockId: "60000"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not match expected contract")
	})

	t.Run("rejects a log with the right address but a different topic", func(t *testing.T) {
		rootChainABI := rootChainABIForTest(t)
		log := newHeaderBlockLog(t, rootChainABI, common.HexToAddress(rootChainAddr), requestedHeaderBlockId, blockNumber, blockHash, txHash, txIndex, logIndex)
		log.Topics[0] = common.HexToHash("0x8888888888888888888888888888888888888888888888888888888888888888")
		receiptJSON := receiptJSONWithLog(t, txHash, blockHash, blockNumber, txIndex, log)
		rl := newReceiptResolveTestListener(t, rootChainAddr, "0x0000000000000000000000000000000000000001", helper.NewHeaderBlockEvent, receiptJSON)

		_, err := rl.resolveCheckpointAckLog(t.Context(), &newHeaderBlock{TransactionHash: txHash.Hex(), LogIndex: "3", HeaderBlockId: "60000"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "topic does not match")
	})

	t.Run("rejects a structurally valid log whose decoded headerBlockId doesn't match what was requested", func(t *testing.T) {
		rootChainABI := rootChainABIForTest(t)
		log := newHeaderBlockLog(t, rootChainABI, common.HexToAddress(rootChainAddr), requestedHeaderBlockId, blockNumber, blockHash, txHash, txIndex, logIndex)
		receiptJSON := receiptJSONWithLog(t, txHash, blockHash, blockNumber, txIndex, log)
		rl := newReceiptResolveTestListener(t, rootChainAddr, "0x0000000000000000000000000000000000000001", helper.NewHeaderBlockEvent, receiptJSON)

		// Requested headerBlockId (60001) differs from what the log actually
		// encodes (60000): structural validation alone would accept this.
		_, err := rl.resolveCheckpointAckLog(t.Context(), &newHeaderBlock{TransactionHash: txHash.Hex(), LogIndex: "3", HeaderBlockId: "60001"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not match requested")
	})

	t.Run("errors when chain manager params are unavailable", func(t *testing.T) {
		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{})

		_, err := rl.resolveCheckpointAckLog(t.Context(), &newHeaderBlock{TransactionHash: txHash.Hex(), LogIndex: "3", HeaderBlockId: "60000"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "chain manager params")
	})

	t.Run("errors when the NewHeaderBlock topic isn't registered", func(t *testing.T) {
		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{
			util.ChainManagerParamsURL: `{"params":{"chain_params":{"root_chain_address": "` + rootChainAddr + `"}}}`,
		})
		// eventMap deliberately left nil/empty: eventTopicByName must fail closed.

		_, err := rl.resolveCheckpointAckLog(t.Context(), &newHeaderBlock{TransactionHash: txHash.Hex(), LogIndex: "3", HeaderBlockId: "60000"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "no known topic")
	})

	t.Run("errors when the L1 receipt fetch fails", func(t *testing.T) {
		rootChainABI := rootChainABIForTest(t)
		log := newHeaderBlockLog(t, rootChainABI, common.HexToAddress(rootChainAddr), requestedHeaderBlockId, blockNumber, blockHash, txHash, txIndex, logIndex)
		receiptJSON := receiptJSONWithLog(t, txHash, blockHash, blockNumber, txIndex, log)
		rl := newReceiptResolveTestListener(t, rootChainAddr, "0x0000000000000000000000000000000000000001", helper.NewHeaderBlockEvent, receiptJSON)
		brokenClient, err := ethclient.Dial("http://127.0.0.1:1")
		require.NoError(t, err)
		rl.contractCaller.MainChainClient = brokenClient

		_, err = rl.resolveCheckpointAckLog(t.Context(), &newHeaderBlock{TransactionHash: txHash.Hex(), LogIndex: "3", HeaderBlockId: "60000"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get transaction receipt")
	})
}

// TestGetStateSynced_ReceiptValidation exercises getStateSynced's receipt
// fetch, address-validation, and content-validation (decoded id vs requested
// stateId) paths (the subgraph-query half is covered by
// TestSubgraphErrorChecks_OtherEntities).
func TestGetStateSynced_ReceiptValidation(t *testing.T) {
	const stateSenderAddr = "0xb661f1577e8749456FA749d44e8A77A9d604e603"
	const otherAddr = "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	const requestedStateId = 5

	txHash := common.HexToHash("0xabc")
	blockHash := common.HexToHash("0x01")
	const blockNumber uint64 = 1
	const txIndex uint = 0
	const logIndex uint = 0

	newStateSyncedTestListener := func(t *testing.T, receiptJSON string) *RootChainListener {
		t.Helper()
		rl := newReceiptResolveTestListener(t, "0x0000000000000000000000000000000000000001", stateSenderAddr, helper.StateSyncedEvent, receiptJSON)
		return rl
	}

	t.Run("accepts a log from the expected StateSender address with the requested stateId", func(t *testing.T) {
		graph := newSubgraph(`{"data":{"stateSynceds":[{"logIndex":"0","transactionHash":"0xabc"}]}}`)
		defer graph.Close()

		stateSenderABI := stateSenderABIForTest(t)
		log := newStateSyncedLog(t, stateSenderABI, common.HexToAddress(stateSenderAddr), requestedStateId, blockNumber, blockHash, txHash, txIndex, logIndex)
		receiptJSON := receiptJSONWithLog(t, txHash, blockHash, blockNumber, txIndex, log)
		rl := newStateSyncedTestListener(t, receiptJSON)
		rl.subGraphClient = &subGraphClient{graphUrl: graph.URL, httpClient: http.DefaultClient}

		got, err := rl.getStateSynced(t.Context(), requestedStateId)
		require.NoError(t, err)
		require.NotNil(t, got)
	})

	t.Run("rejects a log from an address that doesn't match StateSender", func(t *testing.T) {
		graph := newSubgraph(`{"data":{"stateSynceds":[{"logIndex":"0","transactionHash":"0xabc"}]}}`)
		defer graph.Close()

		stateSenderABI := stateSenderABIForTest(t)
		log := newStateSyncedLog(t, stateSenderABI, common.HexToAddress(otherAddr), requestedStateId, blockNumber, blockHash, txHash, txIndex, logIndex)
		receiptJSON := receiptJSONWithLog(t, txHash, blockHash, blockNumber, txIndex, log)
		rl := newStateSyncedTestListener(t, receiptJSON)
		rl.subGraphClient = &subGraphClient{graphUrl: graph.URL, httpClient: http.DefaultClient}

		_, err := rl.getStateSynced(t.Context(), requestedStateId)
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not match expected contract")
	})

	t.Run("rejects a structurally valid log whose decoded stateId doesn't match what was requested", func(t *testing.T) {
		graph := newSubgraph(`{"data":{"stateSynceds":[{"logIndex":"0","transactionHash":"0xabc"}]}}`)
		defer graph.Close()

		stateSenderABI := stateSenderABIForTest(t)
		// Log actually encodes id 6, but the subgraph hit was fetched under a
		// query for stateId 5 — a mismatched subgraph response would slip
		// through structural validation alone.
		log := newStateSyncedLog(t, stateSenderABI, common.HexToAddress(stateSenderAddr), 6, blockNumber, blockHash, txHash, txIndex, logIndex)
		receiptJSON := receiptJSONWithLog(t, txHash, blockHash, blockNumber, txIndex, log)
		rl := newStateSyncedTestListener(t, receiptJSON)
		rl.subGraphClient = &subGraphClient{graphUrl: graph.URL, httpClient: http.DefaultClient}

		_, err := rl.getStateSynced(t.Context(), requestedStateId)
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not match requested")
	})

	t.Run("errors when chain manager params are unavailable", func(t *testing.T) {
		graph := newSubgraph(`{"data":{"stateSynceds":[{"logIndex":"0","transactionHash":"0xabc"}]}}`)
		defer graph.Close()

		rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{})
		rl.subGraphClient = &subGraphClient{graphUrl: graph.URL, httpClient: http.DefaultClient}

		_, err := rl.getStateSynced(t.Context(), requestedStateId)
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

		_, err := rl.getStateSynced(t.Context(), requestedStateId)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no known topic")
	})

	t.Run("errors when the L1 receipt fetch fails", func(t *testing.T) {
		graph := newSubgraph(`{"data":{"stateSynceds":[{"logIndex":"0","transactionHash":"0xabc"}]}}`)
		defer graph.Close()

		stateSenderABI := stateSenderABIForTest(t)
		log := newStateSyncedLog(t, stateSenderABI, common.HexToAddress(stateSenderAddr), requestedStateId, blockNumber, blockHash, txHash, txIndex, logIndex)
		receiptJSON := receiptJSONWithLog(t, txHash, blockHash, blockNumber, txIndex, log)
		rl := newStateSyncedTestListener(t, receiptJSON)
		rl.subGraphClient = &subGraphClient{graphUrl: graph.URL, httpClient: http.DefaultClient}
		brokenClient, err := ethclient.Dial("http://127.0.0.1:1")
		require.NoError(t, err)
		rl.contractCaller.MainChainClient = brokenClient

		_, err = rl.getStateSynced(t.Context(), requestedStateId)
		require.Error(t, err)
		require.Contains(t, err.Error(), "connection refused")
	})
}

// TestConfirmHeaderBlockId exercises confirmHeaderBlockId's own error paths
// directly: a malformed log index and a receipt that doesn't actually
// contain the requested event (decode failure).
func TestConfirmHeaderBlockId(t *testing.T) {
	const rootChainAddr = "0x107a27363FE1Ba8578B8a8c76acDB237cDB35533"

	rl := &RootChainListener{}

	t.Run("rejects a malformed log index", func(t *testing.T) {
		err := rl.confirmHeaderBlockId(nil, rootChainAddr, "not-a-number", "60000")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid log index")
	})

	t.Run("fails to decode when the receipt has no matching log", func(t *testing.T) {
		err := rl.confirmHeaderBlockId(&types.Receipt{}, rootChainAddr, "3", "60000")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode")
	})

	t.Run("increments the self-heal rejection metric on every rejection branch", func(t *testing.T) {
		rootChainABI := rootChainABIForTest(t)
		rlWithABI := &RootChainListener{}
		rlWithABI.contractCaller.RootChainABI = rootChainABI

		mismatchLog := newHeaderBlockLog(t, rootChainABI, common.HexToAddress(rootChainAddr), 60000, 42,
			common.HexToHash("0xblockhash"), common.HexToHash("0xabc"), 1, 0)
		mismatchReceipt := &types.Receipt{Logs: []*types.Log{mismatchLog}}

		before := testutil.ToFloat64(metrics.SelfHealValidationRejected)

		err := rlWithABI.confirmHeaderBlockId(nil, rootChainAddr, "not-a-number", "60000")
		require.Error(t, err)

		err = rlWithABI.confirmHeaderBlockId(&types.Receipt{}, rootChainAddr, "3", "60000")
		require.Error(t, err)

		err = rlWithABI.confirmHeaderBlockId(mismatchReceipt, rootChainAddr, "0", "60001")
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not match requested")

		after := testutil.ToFloat64(metrics.SelfHealValidationRejected)
		require.Equal(t, float64(3), after-before)
	})
}

// TestConfirmStateSyncedId exercises confirmStateSyncedId's own error paths
// directly: a malformed log index and a receipt that doesn't actually
// contain the requested event (decode failure).
func TestConfirmStateSyncedId(t *testing.T) {
	const stateSenderAddr = "0xb661f1577e8749456FA749d44e8A77A9d604e603"

	rl := &RootChainListener{}

	t.Run("rejects a malformed log index", func(t *testing.T) {
		err := rl.confirmStateSyncedId(nil, stateSenderAddr, "not-a-number", 5)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid log index")
	})

	t.Run("fails to decode when the receipt has no matching log", func(t *testing.T) {
		err := rl.confirmStateSyncedId(&types.Receipt{}, stateSenderAddr, "0", 5)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode")
	})

	t.Run("rejects an oversized decoded id whose Int64() truncation would falsely match", func(t *testing.T) {
		stateSenderABI := stateSenderABIForTest(t)
		rlWithABI := &RootChainListener{}
		rlWithABI.contractCaller.StateSenderABI = stateSenderABI

		// 2^64+5: doesn't fit in int64, but its low 64 bits equal 5, so the
		// old buggy decoded.Id.Int64() != stateId comparison would have
		// wrongly treated this as a match against a requested stateId of 5.
		oversized := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(5))
		event := stateSenderABI.Events[helper.StateSyncedEvent]
		var nonIndexed abi.Arguments
		for _, arg := range event.Inputs {
			if !arg.Indexed {
				nonIndexed = append(nonIndexed, arg)
			}
		}
		data, err := nonIndexed.Pack([]byte{})
		require.NoError(t, err)
		receiver := common.HexToAddress("0x2222222222222222222222222222222222222222")
		topicCols, err := abi.MakeTopics([]interface{}{oversized}, []interface{}{receiver})
		require.NoError(t, err)
		topics := []common.Hash{event.ID}
		for _, col := range topicCols {
			topics = append(topics, col[0])
		}
		log := &types.Log{Address: common.HexToAddress(stateSenderAddr), Topics: topics, Data: data, Index: 0}
		receipt := &types.Receipt{Logs: []*types.Log{log}}

		err = rlWithABI.confirmStateSyncedId(receipt, stateSenderAddr, "0", 5)
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not match requested")
	})

	t.Run("increments the self-heal rejection metric on every rejection branch", func(t *testing.T) {
		stateSenderABI := stateSenderABIForTest(t)
		rlWithABI := &RootChainListener{}
		rlWithABI.contractCaller.StateSenderABI = stateSenderABI

		mismatchLog := newStateSyncedLog(t, stateSenderABI, common.HexToAddress(stateSenderAddr), 5, 42,
			common.HexToHash("0xblockhash"), common.HexToHash("0xabc"), 1, 0)
		mismatchReceipt := &types.Receipt{Logs: []*types.Log{mismatchLog}}

		before := testutil.ToFloat64(metrics.SelfHealValidationRejected)

		err := rlWithABI.confirmStateSyncedId(nil, stateSenderAddr, "not-a-number", 5)
		require.Error(t, err)

		err = rlWithABI.confirmStateSyncedId(&types.Receipt{}, stateSenderAddr, "0", 5)
		require.Error(t, err)

		err = rlWithABI.confirmStateSyncedId(mismatchReceipt, stateSenderAddr, "0", 6)
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not match requested")

		after := testutil.ToFloat64(metrics.SelfHealValidationRejected)
		require.Equal(t, float64(3), after-before)
	})
}
