package listener

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
	"github.com/0xPolygon/heimdall-v2/helper"
)

// TestFindRecoverableStateSyncBoundary exercises the binary search directly
// against a synthetic subgraph + L1 RPC, covering the cases the algorithm is
// built around: the whole gap already old enough, the whole gap still too
// recent, a genuine mixed boundary in the middle, and a probe that fails
// outright instead of resolving an age.
func TestFindRecoverableStateSyncBoundary(t *testing.T) {
	t.Run("all ids already old enough resolves to hi from the fast path", func(t *testing.T) {
		ids := []int64{500, 501, 502, 503}
		rl, _ := newStateSyncBoundaryTestListener(t, ids, func(int64) bool { return true }, nil)

		got, err := rl.findRecoverableStateSyncBoundary(t.Context(), 500, 503)
		require.NoError(t, err)
		require.Equal(t, int64(503), got)
	})

	t.Run("all ids too recent resolves to lo-1", func(t *testing.T) {
		ids := []int64{600, 601, 602, 603}
		rl, _ := newStateSyncBoundaryTestListener(t, ids, func(int64) bool { return false }, nil)

		got, err := rl.findRecoverableStateSyncBoundary(t.Context(), 600, 603)
		require.NoError(t, err)
		require.Equal(t, int64(599), got)
	})

	t.Run("mixed range binary-searches the exact boundary", func(t *testing.T) {
		const cutoff = 703 // 700..703 old enough, 704..707 still too recent
		ids := []int64{700, 701, 702, 703, 704, 705, 706, 707}
		rl, _ := newStateSyncBoundaryTestListener(t, ids, func(id int64) bool { return id <= cutoff }, nil)

		got, err := rl.findRecoverableStateSyncBoundary(t.Context(), 700, 707)
		require.NoError(t, err)
		require.Equal(t, int64(cutoff), got)
	})

	t.Run("a failed probe propagates instead of silently resolving a boundary", func(t *testing.T) {
		rl, _ := newStateSyncBoundaryTestListener(t, []int64{800}, func(int64) bool { return true }, map[int64]bool{800: true})

		_, err := rl.findRecoverableStateSyncBoundary(t.Context(), 800, 802)
		require.Error(t, err)
		require.Contains(t, err.Error(), "resolving stateId 800 for age check")
		require.Contains(t, err.Error(), "no state synced event found")
	})

	t.Run("a single-id gap probes the age check exactly once", func(t *testing.T) {
		rl, hits := newStateSyncBoundaryTestListener(t, []int64{900}, func(int64) bool { return true }, nil)

		got, err := rl.findRecoverableStateSyncBoundary(t.Context(), 900, 900)
		require.NoError(t, err)
		require.Equal(t, int64(900), got)
		require.Equal(t, []int64{900}, hits.snapshot(), "lo==hi must not re-probe the same id as hi")
	})
}

// TestResolveRecoverableStateSyncRange_FallsBackOnProbeFailure exercises the
// wrapper's failure handling directly: a boundary-discovery error must not
// abandon the whole cycle, since a single bad row elsewhere in the gap would
// otherwise block recovery of every unrelated missing state sync in it.
func TestResolveRecoverableStateSyncRange_FallsBackOnProbeFailure(t *testing.T) {
	rl, _ := newStateSyncBoundaryTestListener(t, []int64{1000}, func(int64) bool { return true }, map[int64]bool{1000: true})

	start, end, ok := rl.resolveRecoverableStateSyncRange(t.Context(), 1000, 1002)
	require.True(t, ok, "a probe failure must fall back to the full gap, not skip the cycle")
	require.Equal(t, int64(1000), start)
	require.Equal(t, int64(1002), end)
}

const stateSyncBoundaryStateSenderAddr = "0xB59f30f2A5C39A0B7C8b1e4b6C9E6a52B4a8A0FE"

// subgraphCallLog records, in order, every stateId the subgraph mock was
// queried for — tests use it to assert on probe counts, not just outcomes.
type subgraphCallLog struct {
	mu  sync.Mutex
	ids []int64
}

func (c *subgraphCallLog) record(id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ids = append(c.ids, id)
}

func (c *subgraphCallLog) snapshot() []int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]int64(nil), c.ids...)
}

// newStateSyncBoundaryTestListener wires a self-heal listener against a
// synthetic subgraph and L1 RPC serving one real, ABI-encodable StateSynced
// log per id in ids (skipping ids in missing, to simulate a subgraph miss),
// each with a main-chain block time old enough per isOld(id). The returned
// subgraphCallLog records every id actually probed.
func newStateSyncBoundaryTestListener(t *testing.T, ids []int64, isOld func(int64) bool, missing map[int64]bool) (*RootChainListener, *subgraphCallLog) {
	t.Helper()

	stateSenderABI := stateSenderABIForTest(t)
	event := stateSenderABI.Events[helper.StateSyncedEvent]
	contractAddr := common.HexToAddress(stateSyncBoundaryStateSenderAddr)

	const oldEnoughAge = -48 * time.Hour
	const tooRecentAge = -1 * time.Hour
	now := time.Now()

	txHashByID := map[int64]common.Hash{}
	receipts := map[common.Hash]string{}
	blocks := map[uint64]string{}

	for _, id := range ids {
		if missing[id] {
			continue
		}

		blockNumber := uint64(10_000 + id) //nolint:gosec // test fixture, id is always small and positive
		blockHash := common.BigToHash(big.NewInt(id + 1_000_000))
		txHash := common.BigToHash(big.NewInt(id))

		vLog := newStateSyncedLog(t, stateSenderABI, contractAddr, id, blockNumber, blockHash, txHash, 0, 0)
		receipts[txHash] = receiptJSONWithLog(t, txHash, blockHash, blockNumber, 0, vLog)

		age := oldEnoughAge
		if !isOld(id) {
			age = tooRecentAge
		}
		blocks[blockNumber] = newStateSyncBlockJSON(t, blockNumber, now.Add(age))

		txHashByID[id] = txHash
	}

	hits := &subgraphCallLog{}
	l1URL := newStateSyncL1RPCServer(t, receipts, blocks)
	subgraphURL := newStateSyncSubgraphServer(t, txHashByID, hits)

	rl := newCheckpointAckTestListener(t, checkpointAckTestRoutes{
		util.ChainManagerParamsURL: `{"params":{"chain_params":{
			"state_sender_address": "` + stateSyncBoundaryStateSenderAddr + `"
		}}}`,
	})
	rl.subGraphClient = &subGraphClient{graphUrl: subgraphURL, httpClient: &http.Client{Timeout: 5 * time.Second}}
	rl.contractCaller.StateSenderABI = stateSenderABI
	rl.eventMap = map[common.Hash]*abi.Event{event.ID: &event}
	rl.contractCaller.MainChainTimeout = 5 * time.Second

	client, err := ethclient.Dial(l1URL)
	require.NoError(t, err)
	rl.contractCaller.MainChainClient = client

	return rl, hits
}

// newStateSyncBlockJSON renders a minimal but structurally valid
// eth_getBlockByNumber result: just enough header fields for ethclient's
// decoder, plus empty transaction/uncle lists matching an empty txHash/
// uncleHash so its cross-checks against the header pass.
func newStateSyncBlockJSON(t *testing.T, blockNumber uint64, ts time.Time) string {
	t.Helper()

	header := &types.Header{
		UncleHash:  types.EmptyUncleHash,
		TxHash:     types.EmptyTxsHash,
		Difficulty: big.NewInt(0),
		Number:     new(big.Int).SetUint64(blockNumber),
		Time:       uint64(ts.Unix()), //nolint:gosec // test fixture, always a small positive value
		Extra:      []byte{},
	}
	headerJSON, err := json.Marshal(header)
	require.NoError(t, err)

	return string(headerJSON[:len(headerJSON)-1]) + `,"transactions":[],"uncles":[]}`
}

// newStateSyncL1RPCServer serves eth_getTransactionReceipt and
// eth_getBlockByNumber from fixed fixtures keyed by tx hash / block number,
// the two L1 calls stateSyncOldEnough's dependency chain makes.
func newStateSyncL1RPCServer(t *testing.T, receipts map[common.Hash]string, blocks map[uint64]string) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.Unmarshal(body, &req))

		var result string
		switch req.Method {
		case "eth_getTransactionReceipt":
			var txHashHex string
			require.NoError(t, json.Unmarshal(req.Params[0], &txHashHex))
			receiptJSON, ok := receipts[common.HexToHash(txHashHex)]
			require.True(t, ok, "no receipt fixture for tx %s", txHashHex)
			result = receiptJSON
		case "eth_getBlockByNumber":
			var blockNumHex string
			require.NoError(t, json.Unmarshal(req.Params[0], &blockNumHex))
			num, err := hexutil.DecodeUint64(blockNumHex)
			require.NoError(t, err)
			blockJSON, ok := blocks[num]
			require.True(t, ok, "no block fixture for number %s", blockNumHex)
			result = blockJSON
		default:
			t.Fatalf("unexpected L1 RPC method %q", req.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":` + result + `}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

var stateSyncSubgraphIDRe = regexp.MustCompile(`stateId:\s*(-?\d+)`)

// newStateSyncSubgraphServer answers a getStateSynced query with the
// registered tx hash for the requested stateId, or an empty result set
// (a genuine subgraph miss) for any id not in txHashByID. Every request is
// recorded on hits, in order, regardless of outcome.
func newStateSyncSubgraphServer(t *testing.T, txHashByID map[int64]common.Hash, hits *subgraphCallLog) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		m := stateSyncSubgraphIDRe.FindSubmatch(body)
		require.NotNil(t, m, "no stateId found in subgraph query: %s", body)
		id, err := strconv.ParseInt(string(m[1]), 10, 64)
		require.NoError(t, err)
		hits.record(id)

		txHash, ok := txHashByID[id]
		if !ok {
			writeJSON(w, `{"data":{"stateSynceds":[]}}`)
			return
		}
		writeJSON(w, fmt.Sprintf(`{"data":{"stateSynceds":[{"logIndex":"0","transactionHash":"%s"}]}}`, txHash.Hex()))
	}))
	t.Cleanup(server.Close)
	return server.URL
}
