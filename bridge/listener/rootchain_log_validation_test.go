package listener

import (
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
	chainmanagerTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

// newMemLevelDB opens an in-memory leveldb instance, so tests exercising
// persistLastRootBlock/fromBlockAfterLastPersisted can read and write the
// cursor for real instead of needing a live on-disk DB.
func newMemLevelDB(t *testing.T) *leveldb.DB {
	t.Helper()
	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestRootChainListener_ValidateLogAgainstQuery(t *testing.T) {
	t.Parallel()

	// Must be a name in rootChainEvents (a real queried event), not an
	// arbitrary string — validateLogAgainstQuery checks membership there.
	// Belongs to the RootChain contract.
	rootChainTopic := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	rootChainEvent := &abi.Event{Name: helper.NewHeaderBlockEvent}

	// A second real, queried event, belonging to a different contract
	// (StakingInfo) — used to test cross-contract topic/address mismatches.
	stakingInfoTopic := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	stakingInfoEvent := &abi.Event{Name: helper.StakedEvent}

	// In eventMap (so the topic resolves) but its event name isn't in
	// rootChainEvents — the gap the query-scope check exists to close.
	unqueriedTopic := common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444")
	unqueriedEvent := &abi.Event{Name: "OwnershipTransferred"}

	// In eventMap and rootChainEvents (so it clears both earlier checks) but
	// deliberately absent from eventContract — the gap the explicit ok-check
	// on that lookup exists to close. Production listeners always populate
	// both maps together; this simulates them falling out of sync.
	unboundContractTopic := common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555")
	unboundContractEvent := &abi.Event{Name: helper.StakeUpdateEvent}

	rootChainAddress := common.HexToAddress("0xaaaa000000000000000000000000000000aaaa")
	stakingInfoAddress := common.HexToAddress("0xbbbb000000000000000000000000000000bbbb")
	stateSenderAddress := common.HexToAddress("0xcccc000000000000000000000000000000cccc")

	contractAddresses := map[rootChainContract]common.Address{
		rootChainContractRootChain:   rootChainAddress,
		rootChainContractStakingInfo: stakingInfoAddress,
		rootChainContractStateSender: stateSenderAddress,
	}

	fromBlock := big.NewInt(100)
	toBlock := big.NewInt(200)

	newListener := func() *RootChainListener {
		rl := &RootChainListener{
			eventMap: map[common.Hash]*abi.Event{
				rootChainTopic:       rootChainEvent,
				stakingInfoTopic:     stakingInfoEvent,
				unqueriedTopic:       unqueriedEvent,
				unboundContractTopic: unboundContractEvent,
			},
			eventContract: map[common.Hash]rootChainContract{
				rootChainTopic:   rootChainContractRootChain,
				stakingInfoTopic: rootChainContractStakingInfo,
				unqueriedTopic:   rootChainContractRootChain,
				// unboundContractTopic intentionally has no eventContract entry.
			},
		}
		rl.BaseListener.Logger = log.NewNopLogger()
		return rl
	}

	t.Run("accepts a log matching its event's contract, range, and topic", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: 150,
		}

		event, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.True(t, ok)
		require.Same(t, rootChainEvent, event)
	})

	t.Run("accepts a log from a different watched contract when its topic matches", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     stakingInfoAddress,
			Topics:      []common.Hash{stakingInfoTopic},
			BlockNumber: 150,
		}

		event, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.True(t, ok)
		require.Same(t, stakingInfoEvent, event)
	})

	t.Run("accepts a log at the exact range boundaries", func(t *testing.T) {
		t.Parallel()

		rl := newListener()

		lowerBound := types.Log{Address: rootChainAddress, Topics: []common.Hash{rootChainTopic}, BlockNumber: fromBlock.Uint64()}
		_, ok := rl.validateLogAgainstQuery(lowerBound, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.True(t, ok)

		upperBound := types.Log{Address: rootChainAddress, Topics: []common.Hash{rootChainTopic}, BlockNumber: toBlock.Uint64()}
		_, ok = rl.validateLogAgainstQuery(upperBound, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.True(t, ok)
	})

	t.Run("rejects a log from an address outside all watched contracts", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     common.HexToAddress("0xdddd000000000000000000000000000000dddd"),
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.False(t, ok)
	})

	t.Run("rejects a log whose topic belongs to a different contract than its address", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		// A real, watched contract address (StakingInfo) paired with a topic
		// that belongs to RootChain — right address, wrong topic-family.
		vLog := types.Log{
			Address:     stakingInfoAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.False(t, ok)
	})

	t.Run("rejects a log whose address belongs to a different contract than its topic", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		// The topic resolves to a StakingInfo event, but the log claims to
		// come from RootChainAddress — right topic, wrong address.
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{stakingInfoTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.False(t, ok)
	})

	t.Run("rejects a log below the queried block range", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: fromBlock.Uint64() - 1,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.False(t, ok)
	})

	t.Run("rejects a log above the queried block range", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: toBlock.Uint64() + 1,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.False(t, ok)
	})

	t.Run("rejects a log with no topics", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.False(t, ok)
	})

	t.Run("rejects a log with a topic outside the configured event set", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.False(t, ok)
	})

	t.Run("rejects a log whose event resolves in eventMap but isn't in the query's topic set", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{unqueriedTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.False(t, ok)
	})

	t.Run("rejects a log whose topic resolves in eventMap but has no eventContract entry", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{unboundContractTopic},
			BlockNumber: 150,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.False(t, ok)
	})

	t.Run("rejects a removed (reorg'd) log", func(t *testing.T) {
		t.Parallel()

		rl := newListener()
		vLog := types.Log{
			Address:     rootChainAddress,
			Topics:      []common.Hash{rootChainTopic},
			BlockNumber: 150,
			Removed:     true,
		}

		_, ok := rl.validateLogAgainstQuery(vLog, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.False(t, ok)
	})
}

func TestRootChainListener_ValidateAndHandleLogs(t *testing.T) {
	t.Parallel()

	knownTopic := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	knownEvent := &abi.Event{Name: helper.StakedEvent}
	rootChainAddress := common.HexToAddress("0xaaaa000000000000000000000000000000aaaa")
	contractAddresses := map[rootChainContract]common.Address{
		rootChainContractRootChain: rootChainAddress,
	}
	fromBlock := big.NewInt(100)
	toBlock := big.NewInt(200)

	newListener := func() *RootChainListener {
		rl := &RootChainListener{
			eventMap:      map[common.Hash]*abi.Event{knownTopic: knownEvent},
			eventContract: map[common.Hash]rootChainContract{knownTopic: rootChainContractRootChain},
		}
		rl.BaseListener.Logger = log.NewNopLogger()
		return rl
	}

	t.Run("a bad log anywhere in the batch fails the whole batch and dispatches nothing", func(t *testing.T) {
		t.Parallel()

		rl := newListener()

		// A valid log ahead of an invalid one: if dispatch ran before the whole
		// batch was validated, this would call handleLog for the valid entry
		// (which needs ABI/codec plumbing this test deliberately doesn't set up,
		// so it would panic) before returning the error for the invalid one.
		logs := []types.Log{
			{Address: rootChainAddress, Topics: []common.Hash{knownTopic}, BlockNumber: 150},
			{Address: common.HexToAddress("0xcccc000000000000000000000000000000cccc"), Topics: []common.Hash{knownTopic}, BlockNumber: 150},
		}

		var err error
		require.NotPanics(t, func() {
			err = rl.validateAndHandleLogs(logs, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		})
		require.ErrorIs(t, err, errUnexpectedRootChainLog)
	})

	t.Run("an empty batch succeeds with no dispatch", func(t *testing.T) {
		t.Parallel()

		rl := newListener()

		err := rl.validateAndHandleLogs(nil, contractAddresses, fromBlock, toBlock, newRootChainRejectionState())
		require.NoError(t, err)
	})
}

// mockEthGetLogs starts a JSON-RPC HTTP server that answers eth_getLogs with
// the given logs JSON for every request, regardless of the requested block
// range — simulating an endpoint that doesn't honor FilterLogs' filter, the
// exact scenario validateLogAgainstQuery exists to catch. requestCount is
// incremented on every call from the handler goroutine, so it must be an
// atomic counter — the TCP round-trip between that goroutine and the test
// goroutine reading it later is not a happens-before edge the race detector
// recognizes.
func mockEthGetLogs(t *testing.T, logsJSON string, requestCount *atomic.Int64) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		require.Equal(t, "eth_getLogs", req.Method)
		requestCount.Add(1)

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":` + logsJSON + `}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// TestProcessRootChainBlockRange_DedupesRejectionMetricAcrossBisection proves
// the fix for the metric-inflation bug: a persistently-bad log returned by
// every sub-query during range bisection must only increment
// rootchain_listener_log_rejected_total once per top-level call, not once
// per bisection level that re-encounters it.
func TestProcessRootChainBlockRange_DedupesRejectionMetricAcrossBisection(t *testing.T) {
	knownTopic := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	knownEvent := &abi.Event{Name: helper.NewHeaderBlockEvent}
	// Not in eventMap at all, and BlockNumber left at its zero value, which
	// falls outside every sub-range this test bisects into (all >= 100): this
	// same log re-fails the range check on every single sub-query, regardless
	// of which bisected fromBlock/toBlock it's queried against. Which check
	// rejects it doesn't matter for this test — only that it's the same
	// logKey every time.
	badLogTopic := common.HexToHash("0x9999999999999999999999999999999999999999999999999999999999999999")

	badLog := &types.Log{
		Address: common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
		Topics:  []common.Hash{badLogTopic},
		TxHash:  common.HexToHash("0xbad"),
		Index:   0,
	}
	logsJSON, err := json.Marshal([]*types.Log{badLog})
	require.NoError(t, err)

	rl := &RootChainListener{
		eventMap:      map[common.Hash]*abi.Event{knownTopic: knownEvent},
		eventContract: map[common.Hash]rootChainContract{knownTopic: rootChainContractRootChain},
	}
	rl.BaseListener.Logger = log.NewNopLogger()

	var requestCount atomic.Int64
	rpcURL := mockEthGetLogs(t, string(logsJSON), &requestCount)
	client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	rl.contractCaller.MainChainClient = client
	rl.contractCaller.MainChainTimeout = 5 * time.Second

	rootChainContext := &RootChainListenerContext{
		ChainmanagerParams: &chainmanagerTypes.Params{
			ChainParams: chainmanagerTypes.ChainParams{
				RootChainAddress:   "0x1111111111111111111111111111111111111111",
				StakingInfoAddress: "0x2222222222222222222222222222222222222222",
				StateSenderAddress: "0x3333333333333333333333333333333333333333",
			},
		},
	}

	before := testutil.ToFloat64(metrics.RootChainListenerLogRejected)

	state := newRootChainRejectionState()
	// A 4-block range bisects into several sub-queries (range, then halves,
	// down to single blocks) before processRootChainBlockRange gives up —
	// every one of them re-fetches and re-rejects the same bad log.
	err = rl.processRootChainBlockRange(rootChainContext, big.NewInt(100), big.NewInt(103), state)
	require.Error(t, err)

	require.Greater(t, requestCount.Load(), int64(1), "the range must actually have been bisected into more than one sub-query")

	after := testutil.ToFloat64(metrics.RootChainListenerLogRejected)
	require.Equal(t, float64(1), after-before, "one bad log across a bisected range must increment the rejection counter exactly once")
}

// newQuarantineTestListener wires a listener against a mock eth_getLogs
// endpoint that always returns the same unrecognized-topic log for the
// single block [100,100], plus a real in-memory storage client so the
// success/quarantine path (which persists the cursor) can be exercised.
func newQuarantineTestListener(t *testing.T) (*RootChainListener, *RootChainListenerContext) {
	t.Helper()

	knownTopic := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	knownEvent := &abi.Event{Name: helper.NewHeaderBlockEvent}
	badLogTopic := common.HexToHash("0x9999999999999999999999999999999999999999999999999999999999999999")

	badLog := &types.Log{
		Address:     common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
		Topics:      []common.Hash{badLogTopic},
		TxHash:      common.HexToHash("0xbad"),
		Index:       0,
		BlockNumber: 100,
	}
	logsJSON, err := json.Marshal([]*types.Log{badLog})
	require.NoError(t, err)

	rl := &RootChainListener{
		eventMap:      map[common.Hash]*abi.Event{knownTopic: knownEvent},
		eventContract: map[common.Hash]rootChainContract{knownTopic: rootChainContractRootChain},
	}
	rl.BaseListener.Logger = log.NewNopLogger()
	rl.storageClient = newMemLevelDB(t)

	var requestCount atomic.Int64
	rpcURL := mockEthGetLogs(t, string(logsJSON), &requestCount)
	client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	rl.contractCaller.MainChainClient = client
	rl.contractCaller.MainChainTimeout = 5 * time.Second

	rootChainContext := &RootChainListenerContext{
		ChainmanagerParams: &chainmanagerTypes.Params{
			ChainParams: chainmanagerTypes.ChainParams{
				RootChainAddress:   "0x1111111111111111111111111111111111111111",
				StakingInfoAddress: "0x2222222222222222222222222222222222222222",
				StateSenderAddress: "0x3333333333333333333333333333333333333333",
			},
		},
	}

	return rl, rootChainContext
}

// TestProcessRootChainBlockRange_QuarantinesAfterMaxRejections proves the
// bounded-quarantine behavior: a log that fails validation on fewer than
// maxRootChainLogRejections separate poll cycles keeps withholding the
// cursor exactly as before, but on the Nth cycle it's quarantined — the
// cursor advances past it, a distinct quarantine metric fires once, and its
// entry is dropped from the persistent failure-count map.
func TestProcessRootChainBlockRange_QuarantinesAfterMaxRejections(t *testing.T) {
	rl, rootChainContext := newQuarantineTestListener(t)

	rejectedBefore := testutil.ToFloat64(metrics.RootChainListenerLogRejected)
	quarantinedBefore := testutil.ToFloat64(metrics.RootChainListenerLogQuarantined)

	// Each call simulates one ProcessHeader poll cycle: a fresh
	// rootChainRejectionState, but the same rl (so its persistent
	// logFailureCounts carries over across cycles).
	for cycle := 1; cycle < maxRootChainLogRejections; cycle++ {
		state := newRootChainRejectionState()
		err := rl.processRootChainBlockRange(rootChainContext, big.NewInt(100), big.NewInt(100), state)
		require.Error(t, err, "cycle %d (below the threshold) must still withhold the cursor", cycle)
	}

	_, err := rl.storageClient.Get([]byte(lastRootBlockKey), nil)
	require.ErrorIs(t, err, leveldb.ErrNotFound, "the cursor must not have advanced before the threshold is reached")

	// The Nth cycle crosses the threshold.
	state := newRootChainRejectionState()
	err = rl.processRootChainBlockRange(rootChainContext, big.NewInt(100), big.NewInt(100), state)
	require.NoError(t, err, "the Nth cycle must quarantine the log and advance the cursor instead of erroring")

	rejectedAfter := testutil.ToFloat64(metrics.RootChainListenerLogRejected)
	require.Equal(t, float64(maxRootChainLogRejections), rejectedAfter-rejectedBefore, "the rejection counter fires once per cycle, including the quarantining one")

	quarantinedAfter := testutil.ToFloat64(metrics.RootChainListenerLogQuarantined)
	require.Equal(t, float64(1), quarantinedAfter-quarantinedBefore, "the quarantine counter fires exactly once")

	lastBlockBytes, err := rl.storageClient.Get([]byte(lastRootBlockKey), nil)
	require.NoError(t, err)
	require.Equal(t, "100", string(lastBlockBytes), "the cursor must have advanced past the quarantined block")

	require.Empty(t, rl.logFailureCounts, "a quarantined log's failure count must be dropped, not kept forever")
}

func TestPruneStaleLogFailureCounts(t *testing.T) {
	rl := &RootChainListener{logFailureCounts: map[string]uint64{
		"stale":     5,
		"recurring": 3,
	}}

	rl.pruneStaleLogFailureCounts(map[string]struct{}{"recurring": {}})

	require.NotContains(t, rl.logFailureCounts, "stale", "a log not rejected again this cycle must be dropped")
	require.Contains(t, rl.logFailureCounts, "recurring", "a log rejected again this cycle must be kept")
	require.Equal(t, uint64(3), rl.logFailureCounts["recurring"])
}

// TestRejectRootChainLog_CapsTrackedFailures proves logFailureCounts can't
// grow without bound: an endpoint returning a fresh, distinct bad log on
// every poll never lets any single entry reach maxRootChainLogRejections,
// so nothing is ever quarantined or pruned to bound the map naturally. Once
// the cap is reached, a brand-new key is refused, but an already-tracked
// one still increments.
func TestRejectRootChainLog_CapsTrackedFailures(t *testing.T) {
	knownTopic := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	knownEvent := &abi.Event{Name: helper.NewHeaderBlockEvent}
	rl := &RootChainListener{
		eventMap:      map[common.Hash]*abi.Event{knownTopic: knownEvent},
		eventContract: map[common.Hash]rootChainContract{knownTopic: rootChainContractRootChain},
	}
	rl.BaseListener.Logger = log.NewNopLogger()

	const block = uint64(100)
	existingLog := unrecognizedTopicLog(block, common.HexToHash("0xaaaa12"))
	rl.logFailureCounts = make(map[string]uint64, maxTrackedLogFailures)
	rl.logFailureCounts[rootChainLogKey(*existingLog)] = 5
	for i := 1; i < maxTrackedLogFailures; i++ {
		rl.logFailureCounts[strconv.Itoa(i)] = 1
	}
	require.Len(t, rl.logFailureCounts, maxTrackedLogFailures)

	contractAddresses := map[rootChainContract]common.Address{}
	fromBlock, toBlock := big.NewInt(int64(block)), big.NewInt(int64(block))

	state := newRootChainRejectionState()
	_, ok := rl.validateLogAgainstQuery(*existingLog, contractAddresses, fromBlock, toBlock, state)
	require.False(t, ok)
	require.Equal(t, uint64(6), rl.logFailureCounts[rootChainLogKey(*existingLog)], "an already-tracked key keeps incrementing at the cap")
	require.Len(t, rl.logFailureCounts, maxTrackedLogFailures)

	newLog := unrecognizedTopicLog(block, common.HexToHash("0xaaaa13"))
	state = newRootChainRejectionState()
	_, ok = rl.validateLogAgainstQuery(*newLog, contractAddresses, fromBlock, toBlock, state)
	require.False(t, ok)
	require.NotContains(t, rl.logFailureCounts, rootChainLogKey(*newLog), "a brand-new key is refused once the cap is reached")
	require.Len(t, rl.logFailureCounts, maxTrackedLogFailures)
}

// mockEthGetLogsByRange starts a JSON-RPC HTTP server that answers
// eth_getLogs by inspecting the request's fromBlock/toBlock and calling
// logsFor to decide what to return for that exact sub-range — so different
// single-block queries within the same bisected range can be given
// different (or no) logs, unlike mockEthGetLogs' one-fixed-response-always.
func mockEthGetLogsByRange(t *testing.T, logsFor func(fromBlock, toBlock uint64) []*types.Log) string {
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
		require.Equal(t, "eth_getLogs", req.Method)

		var filter struct {
			FromBlock string `json:"fromBlock"`
			ToBlock   string `json:"toBlock"`
		}
		require.NoError(t, json.Unmarshal(req.Params[0], &filter))
		fromBlock, err := strconv.ParseUint(strings.TrimPrefix(filter.FromBlock, "0x"), 16, 64)
		require.NoError(t, err)
		toBlock, err := strconv.ParseUint(strings.TrimPrefix(filter.ToBlock, "0x"), 16, 64)
		require.NoError(t, err)

		logsJSON, err := json.Marshal(logsFor(fromBlock, toBlock))
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":` + string(logsJSON) + `}`))
		require.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// twoBlockListener wires a listener plus chain-manager context whose
// eth_getLogs mock is driven by logsFor, for tests that need different
// blocks within one range to behave differently.
func twoBlockListener(t *testing.T, logsFor func(fromBlock, toBlock uint64) []*types.Log) (*RootChainListener, *RootChainListenerContext) {
	t.Helper()

	knownTopic := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	knownEvent := &abi.Event{Name: helper.NewHeaderBlockEvent}

	rl := &RootChainListener{
		eventMap:      map[common.Hash]*abi.Event{knownTopic: knownEvent},
		eventContract: map[common.Hash]rootChainContract{knownTopic: rootChainContractRootChain},
	}
	rl.BaseListener.Logger = log.NewNopLogger()
	rl.storageClient = newMemLevelDB(t)

	rpcURL := mockEthGetLogsByRange(t, logsFor)
	client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)
	rl.contractCaller.MainChainClient = client
	rl.contractCaller.MainChainTimeout = 5 * time.Second

	rootChainContext := &RootChainListenerContext{
		ChainmanagerParams: &chainmanagerTypes.Params{
			ChainParams: chainmanagerTypes.ChainParams{
				RootChainAddress:   "0x1111111111111111111111111111111111111111",
				StakingInfoAddress: "0x2222222222222222222222222222222222222222",
				StateSenderAddress: "0x3333333333333333333333333333333333333333",
			},
		},
	}

	return rl, rootChainContext
}

func unrecognizedTopicLog(blockNumber uint64, txHash common.Hash) *types.Log {
	return &types.Log{
		Address:     common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
		Topics:      []common.Hash{common.HexToHash("0x9999999999999999999999999999999999999999999999999999999999999999")},
		TxHash:      txHash,
		BlockNumber: blockNumber,
	}
}

// TestProcessRootChainBlockRange_QuarantineIsScopedToItsOwnLog proves
// quarantine is keyed by the exact rejected log (txHash+logIndex), not by
// block number or a single shared slot, so a *different*, not-yet-eligible
// bad log at a different block within the same poll cycle can never be
// mistaken for the one that actually crossed the threshold.
func TestProcessRootChainBlockRange_QuarantineIsScopedToItsOwnLog(t *testing.T) {
	const blockN, blockM = uint64(200), uint64(201)
	logX := unrecognizedTopicLog(blockN, common.HexToHash("0xaaaa1"))
	logY := unrecognizedTopicLog(blockM, common.HexToHash("0xaaaa2"))

	rl, rootChainContext := twoBlockListener(t, func(fromBlock, toBlock uint64) []*types.Log {
		var logs []*types.Log
		if fromBlock <= blockN && blockN <= toBlock {
			logs = append(logs, logX)
		}
		if fromBlock <= blockM && blockM <= toBlock {
			logs = append(logs, logY)
		}
		return logs
	})

	// X has already failed maxRootChainLogRejections-1 times in earlier
	// cycles; this cycle's rejection is its 100th. Y is fresh.
	rl.logFailureCounts = map[string]uint64{rootChainLogKey(*logX): maxRootChainLogRejections - 1}

	quarantinedBefore := testutil.ToFloat64(metrics.RootChainListenerLogQuarantined)

	state := newRootChainRejectionState()
	err := rl.processRootChainBlockRange(rootChainContext, big.NewInt(int64(blockN)), big.NewInt(int64(blockM)), state)
	require.Error(t, err, "Y's own, unrelated, not-yet-eligible failure must still withhold the range")

	quarantinedAfter := testutil.ToFloat64(metrics.RootChainListenerLogQuarantined)
	require.Equal(t, float64(1), quarantinedAfter-quarantinedBefore, "X was quarantined on its own merits")

	lastBlockBytes, err := rl.storageClient.Get([]byte(lastRootBlockKey), nil)
	require.NoError(t, err)
	require.Equal(t, strconv.FormatUint(blockN, 10), string(lastBlockBytes), "the cursor advanced through X's own block, not Y's")

	require.NotContains(t, rl.logFailureCounts, rootChainLogKey(*logX), "X's count was cleared by its own quarantine")
	require.Equal(t, uint64(1), rl.logFailureCounts[rootChainLogKey(*logY)], "Y was only counted once, on its own merits, not quarantined")
}

// TestProcessRootChainBlockRange_QuarantineNeverResolvesAWiderRange proves
// quarantine may only exclude a log at the exact single-block range it
// bisects down to, never at a still-being-bisected wider range. Without
// that check, a quarantine-eligible log reported at the start of a
// wide chunk would let validateAndHandleLogs return success for the whole
// chunk on the strength of one FilterLogs round trip — advancing the cursor
// across every other block in it without any of them ever having been
// independently queried.
func TestProcessRootChainBlockRange_QuarantineNeverResolvesAWiderRange(t *testing.T) {
	const badBlock = uint64(200)
	badLog := unrecognizedTopicLog(badBlock, common.HexToHash("0xaaaa9"))

	var requestCount atomic.Int64
	rl, rootChainContext := twoBlockListener(t, func(fromBlock, toBlock uint64) []*types.Log {
		requestCount.Add(1)
		if fromBlock <= badBlock && badBlock <= toBlock {
			return []*types.Log{badLog}
		}
		return nil
	})
	// Already eligible: this cycle's rejection crosses the threshold on the
	// very first encounter, at the top-level (unbisected) range.
	rl.logFailureCounts = map[string]uint64{rootChainLogKey(*badLog): maxRootChainLogRejections - 1}

	quarantinedBefore := testutil.ToFloat64(metrics.RootChainListenerLogQuarantined)

	state := newRootChainRejectionState()
	err := rl.processRootChainBlockRange(rootChainContext, big.NewInt(int64(badBlock)), big.NewInt(int64(badBlock+3)), state)
	require.NoError(t, err, "every block besides badBlock is genuinely clean, so bisection resolves the whole range")

	quarantinedAfter := testutil.ToFloat64(metrics.RootChainListenerLogQuarantined)
	require.Equal(t, float64(1), quarantinedAfter-quarantinedBefore, "the bad log is quarantined exactly once, at its own block")

	lastBlockBytes, err := rl.storageClient.Get([]byte(lastRootBlockKey), nil)
	require.NoError(t, err)
	require.Equal(t, strconv.FormatUint(badBlock+3, 10), string(lastBlockBytes))

	// The load-bearing assertion: a bug that lets quarantine resolve the
	// whole 4-block chunk in the single top-level FilterLogs call would
	// still land on the same end cursor and the same quarantine count, but
	// with requestCount stuck at 1 — blocks badBlock+1..badBlock+3 would
	// never have been independently queried at all.
	require.Greater(t, requestCount.Load(), int64(1),
		"the range must be bisected down to the single block that actually owns the quarantined log, never resolved in one wide-range request")
}

// TestProcessRootChainBlockRange_QuarantinesAnOutOfRangeLog proves a log
// whose own claimed block number is permanently outside the range being
// swept — an endpoint returning garbage from before the sweep even started,
// never inside any sub-range bisection can reach — still gets quarantined
// and stops blocking the cursor, instead of withholding it forever because
// its own blockNumber can never equal the single block being resolved.
func TestProcessRootChainBlockRange_QuarantinesAnOutOfRangeLog(t *testing.T) {
	const queriedBlock = uint64(500)
	badLog := &types.Log{
		Address:     common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
		Topics:      []common.Hash{common.HexToHash("0x9999999999999999999999999999999999999999999999999999999999999999")},
		TxHash:      common.HexToHash("0xaaaa10"),
		BlockNumber: 0,
	}

	rl, rootChainContext := twoBlockListener(t, func(fromBlock, toBlock uint64) []*types.Log {
		return []*types.Log{badLog}
	})

	quarantinedBefore := testutil.ToFloat64(metrics.RootChainListenerLogQuarantined)

	for cycle := 1; cycle < maxRootChainLogRejections; cycle++ {
		state := newRootChainRejectionState()
		err := rl.processRootChainBlockRange(rootChainContext, big.NewInt(int64(queriedBlock)), big.NewInt(int64(queriedBlock)), state)
		require.Error(t, err, "cycle %d (below the threshold) must still withhold the cursor", cycle)
	}

	state := newRootChainRejectionState()
	err := rl.processRootChainBlockRange(rootChainContext, big.NewInt(int64(queriedBlock)), big.NewInt(int64(queriedBlock)), state)
	require.NoError(t, err, "the Nth cycle must quarantine the out-of-range log and advance the cursor instead of erroring forever")

	quarantinedAfter := testutil.ToFloat64(metrics.RootChainListenerLogQuarantined)
	require.Equal(t, float64(1), quarantinedAfter-quarantinedBefore)

	lastBlockBytes, err := rl.storageClient.Get([]byte(lastRootBlockKey), nil)
	require.NoError(t, err)
	require.Equal(t, strconv.FormatUint(queriedBlock, 10), string(lastBlockBytes))
}

// TestProcessRootChainBlockRange_QuarantineClearsWholeChunkInOneCycle proves
// a persistently-misbehaving log an endpoint returns for every query
// regardless of the requested range gets excluded at every single-block leaf
// that re-encounters it within the same cycle, not just the first one
// bisection reaches — so a wide, multi-block chunk clears entirely in one
// cycle once the log crosses the quarantine threshold, instead of only the
// first excluded block's cursor advancing and every other leaf failing
// because the quarantine entry was already consumed. The quarantine metric
// still fires exactly once despite the log being excluded at several leaves.
func TestProcessRootChainBlockRange_QuarantineClearsWholeChunkInOneCycle(t *testing.T) {
	rl, rootChainContext := newQuarantineTestListener(t)
	badLogKey := rootChainLogKey(types.Log{TxHash: common.HexToHash("0xbad"), Index: 0})
	rl.logFailureCounts = map[string]uint64{badLogKey: maxRootChainLogRejections - 1}

	quarantinedBefore := testutil.ToFloat64(metrics.RootChainListenerLogQuarantined)

	state := newRootChainRejectionState()
	err := rl.processRootChainBlockRange(rootChainContext, big.NewInt(100), big.NewInt(103), state)
	require.NoError(t, err, "the whole 4-block chunk must clear in one cycle, not just the block first excluded")

	quarantinedAfter := testutil.ToFloat64(metrics.RootChainListenerLogQuarantined)
	require.Equal(t, float64(1), quarantinedAfter-quarantinedBefore, "the same log is excluded at several leaves but quarantined only once")

	lastBlockBytes, err := rl.storageClient.Get([]byte(lastRootBlockKey), nil)
	require.NoError(t, err)
	require.Equal(t, "103", string(lastBlockBytes))

	require.Empty(t, rl.logFailureCounts, "the quarantined log's failure count is dropped")
}

// TestQueryAndBroadcastEvents_TransientFailureNeverTouchesQuarantine proves a
// transient, non-content failure (the L1 client not being ready) can't be
// confused with a validation-rejection quarantine: it fails before
// validateAndHandleLogs is ever called, so a pending quarantine entry from
// elsewhere in the same cycle is left completely untouched.
func TestQueryAndBroadcastEvents_TransientFailureNeverTouchesQuarantine(t *testing.T) {
	rl := &RootChainListener{}
	rl.BaseListener.Logger = log.NewNopLogger()
	// MainChainClient deliberately left nil: the transient-unavailable path.

	rootChainContext := &RootChainListenerContext{
		ChainmanagerParams: &chainmanagerTypes.Params{
			ChainParams: chainmanagerTypes.ChainParams{RootChainAddress: "0x1111111111111111111111111111111111111111"},
		},
	}

	state := newRootChainRejectionState()
	pending := &rootChainLogDetail{logKey: "unrelated-log", blockNumber: 999}
	state.quarantine["unrelated-log"] = pending

	err := rl.queryAndBroadcastEvents(rootChainContext, big.NewInt(500), big.NewInt(500), state)
	require.ErrorIs(t, err, errMainChainClientUnavailable)

	require.Same(t, pending, state.quarantine["unrelated-log"], "a transient failure must never consume a pending quarantine for a different log")
}

// TestProcessRootChainBlockRange_QuarantineDispatchesOtherValidLogsInTheSameBlock
// proves a block containing both a persistently-bad log and a genuinely
// valid one still dispatches the valid one when the bad one is quarantined,
// instead of losing it along with the bad log.
func TestProcessRootChainBlockRange_QuarantineDispatchesOtherValidLogsInTheSameBlock(t *testing.T) {
	const block = uint64(300)
	validTopic := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	validLog := &types.Log{
		Address:     common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Topics:      []common.Hash{validTopic},
		TxHash:      common.HexToHash("0xaaaa3"),
		BlockNumber: block,
	}
	badLog := unrecognizedTopicLog(block, common.HexToHash("0xbad"))

	rl, rootChainContext := twoBlockListener(t, func(fromBlock, toBlock uint64) []*types.Log {
		return []*types.Log{validLog, badLog}
	})
	rl.logFailureCounts = map[string]uint64{rootChainLogKey(*badLog): maxRootChainLogRejections - 1}

	// handleLog itself always runs; every handleXLog variant's actual send
	// is gated behind a validator-set REST check this test doesn't stand up.
	// onLogDispatched is the seam that makes dispatch observable regardless,
	// so this test would fail if the dispatch loop were ever deleted.
	var dispatched []common.Hash
	rl.onLogDispatched = func(vLog types.Log, _ *abi.Event) {
		dispatched = append(dispatched, vLog.TxHash)
	}

	quarantinedBefore := testutil.ToFloat64(metrics.RootChainListenerLogQuarantined)

	state := newRootChainRejectionState()
	var err error
	require.NotPanics(t, func() {
		err = rl.processRootChainBlockRange(rootChainContext, big.NewInt(int64(block)), big.NewInt(int64(block)), state)
	}, "handleLog for the valid log must run without panicking")
	require.NoError(t, err, "the batch succeeds: the only failing log was quarantined, the other one dispatched cleanly")

	require.Equal(t, []common.Hash{validLog.TxHash}, dispatched,
		"the valid log must dispatch exactly once; the quarantined log must never dispatch")

	quarantinedAfter := testutil.ToFloat64(metrics.RootChainListenerLogQuarantined)
	require.Equal(t, float64(1), quarantinedAfter-quarantinedBefore)

	lastBlockBytes, err := rl.storageClient.Get([]byte(lastRootBlockKey), nil)
	require.NoError(t, err)
	require.Equal(t, strconv.FormatUint(block, 10), string(lastBlockBytes))
}
