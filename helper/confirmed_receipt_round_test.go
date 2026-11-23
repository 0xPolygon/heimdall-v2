package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// roundL1 is a fake Ethereum endpoint: receipts live in block 100, the finalized and latest
// heads are block 90. It counts head lookups so tests can check how many round trips happen.
type roundL1 struct {
	finalizedCalls, latestCalls atomic.Int64
	noFinalized                 bool
	failFinalizedOnce           bool
	beforeFinalized             func() // called before answering a finalized lookup
	latestNumber                uint64 // 0 = same as finalized (90)
	receiptBlock                uint64 // 0 = block 100
	failLatest                  bool   // the latest-head lookup returns an RPC error
	failReceipt                 bool   // the receipt lookup returns an RPC error
}

func (l *roundL1) serve(t *testing.T) *httptest.Server {
	t.Helper()
	zero32 := "0x" + strings.Repeat("0", 64)
	bloom := "0x" + strings.Repeat("0", 512)
	headerAt := func(n uint64) string {
		return fmt.Sprintf(`{"parentHash":%q,"sha3Uncles":%q,"miner":"0x0000000000000000000000000000000000000000","stateRoot":%q,"transactionsRoot":%q,"receiptsRoot":%q,"logsBloom":%q,"difficulty":"0x0","number":"0x%x","gasLimit":"0x1c9c380","gasUsed":"0x0","timestamp":"0x1","extraData":"0x","mixHash":%q,"nonce":"0x0000000000000000","baseFeePerGas":"0x1","hash":%q}`,
			zero32, zero32, zero32, zero32, zero32, bloom, n, zero32, zero32)
	}
	latest := l.latestNumber
	if latest == 0 {
		latest = 90
	}
	receiptBlock := l.receiptBlock
	if receiptBlock == 0 {
		receiptBlock = 100
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var q struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		_ = json.Unmarshal(body, &q)
		w.Header().Set("Content-Type", "application/json")
		if q.Method == "eth_getBlockByNumber" {
			var tag string
			_ = json.Unmarshal(q.Params[0], &tag)
			if tag == "finalized" {
				calls := l.finalizedCalls.Add(1)
				if l.beforeFinalized != nil {
					l.beforeFinalized()
				}
				if l.noFinalized || (l.failFinalizedOnce && calls == 1) {
					fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"finalized not supported"}}`, q.ID)
					return
				}
				fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, q.ID, headerAt(90))
				return
			}
			l.latestCalls.Add(1)
			if l.failLatest {
				fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"latest unavailable"}}`, q.ID)
				return
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, q.ID, headerAt(latest))
			return
		}
		if l.failReceipt {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"receipt unavailable"}}`, q.ID)
			return
		}
		var hash string
		_ = json.Unmarshal(q.Params[0], &hash)
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"blockNumber":"0x%x","status":"0x1","cumulativeGasUsed":"0x5208","gasUsed":"0x5208","logsBloom":%q,"logs":[],"transactionHash":%q}}`,
			q.ID, receiptBlock, bloom, hash)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newRoundCaller(t *testing.T, l1 *roundL1) *ContractCaller {
	t.Helper()
	rc, err := rpc.Dial(l1.serve(t).URL)
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	return &ContractCaller{
		MainChainRPCClient: rc,
		MainChainClient:    ethclient.NewClient(rc),
		MainChainTimeout:   5 * time.Second,
		prefetchMu:         &sync.RWMutex{},
	}
}

// confirmMany asks for n distinct receipts that all sit above the finalized/latest head.
func confirmMany(t *testing.T, c *ContractCaller, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		_, err := c.GetConfirmedTxReceipt(context.Background(), common.BigToHash(big.NewInt(int64(i+1))), 6)
		require.Error(t, err, "a receipt above the head must not be confirmed")
	}
}

// TestGetConfirmedTxReceipt_RoundFetchesFinalizedOnce: inside one ExtendVote round, 49 side txs
// that point at not-yet-final L1 txs cost one finalized-head lookup, not 49.
func TestGetConfirmedTxReceipt_RoundFetchesFinalizedOnce(t *testing.T) {
	l1 := &roundL1{}
	c := newRoundCaller(t, l1)
	c.BeginPrefetchRound()
	confirmMany(t, c, 49)
	c.EndPrefetchRound()
	require.EqualValues(t, 1, l1.finalizedCalls.Load())
	require.Zero(t, l1.latestCalls.Load())
}

// TestGetConfirmedTxReceipt_RoundFallbackFetchesHeadsOnce: when the finalized API is not
// available, the finalized lookup is still tried for every receipt (an earlier failure must
// never move a vote onto the weaker rule), but the latest head is looked up once per round.
func TestGetConfirmedTxReceipt_RoundFallbackFetchesHeadsOnce(t *testing.T) {
	l1 := &roundL1{noFinalized: true}
	c := newRoundCaller(t, l1)
	c.BeginPrefetchRound()
	confirmMany(t, c, 49)
	c.EndPrefetchRound()
	require.EqualValues(t, 49, l1.finalizedCalls.Load())
	require.EqualValues(t, 1, l1.latestCalls.Load())
}

// TestGetConfirmedTxReceipt_FinalizedFailureDoesNotDowngradeLaterReceipts: a finalized-head failure
// (for example a query on the shared caller, cancelled mid-call) must not make later receipts in
// the round skip the finalized check. The receipt whose own lookup failed still falls back to the
// N-confirmations rule, as on the base branch; this test does not cover it. Here the first lookup fails, the next ones succeed, and a
// receipt above the finalized head must still be rejected even though it has enough
// confirmations against the latest head.
func TestGetConfirmedTxReceipt_FinalizedFailureDoesNotDowngradeLaterReceipts(t *testing.T) {
	l1 := &roundL1{failFinalizedOnce: true, latestNumber: 120}
	c := newRoundCaller(t, l1)
	c.BeginPrefetchRound()
	_, _ = c.GetConfirmedTxReceipt(context.Background(), common.BigToHash(big.NewInt(1)), 6)
	_, err := c.GetConfirmedTxReceipt(context.Background(), common.BigToHash(big.NewInt(2)), 6)
	c.EndPrefetchRound()
	require.Error(t, err, "receipt at block 100 is above finalized 90 and must not be confirmed")
}

// TestGetConfirmedTxReceipt_StaleLookupDoesNotFillNewRound: a finalized lookup that started
// before the round (e.g. a query on the shared caller) and finishes inside it must not put its
// older head into the round cache, where the in-round fast reject would trust it.
func TestGetConfirmedTxReceipt_StaleLookupDoesNotFillNewRound(t *testing.T) {
	l1 := &roundL1{}
	c := newRoundCaller(t, l1)
	l1.beforeFinalized = func() {
		if l1.finalizedCalls.Load() == 1 {
			c.BeginPrefetchRound() // the round starts while the outside lookup is in flight
		}
	}
	_, _ = c.GetConfirmedTxReceipt(context.Background(), common.BigToHash(big.NewInt(1)), 6)
	c.prefetchMu.RLock()
	cached := c.finalizedHeaderCache
	c.prefetchMu.RUnlock()
	c.EndPrefetchRound()
	require.Nil(t, cached, "a lookup from an earlier generation must not fill the round cache")
}

// TestGetConfirmedTxReceipt_OutsideRoundUnchanged: callers outside ExtendVote keep the old
// behavior and re-check the finalized head on every call.
func TestGetConfirmedTxReceipt_OutsideRoundUnchanged(t *testing.T) {
	l1 := &roundL1{}
	c := newRoundCaller(t, l1)
	confirmMany(t, c, 3)
	require.EqualValues(t, 3, l1.finalizedCalls.Load())
}

// TestGetConfirmedTxReceipt_NewRoundRefetches: the per-round cache does not leak into the
// next round, so finality progress between rounds is always seen.
func TestGetConfirmedTxReceipt_NewRoundRefetches(t *testing.T) {
	l1 := &roundL1{noFinalized: true}
	c := newRoundCaller(t, l1)
	for round := 0; round < 2; round++ {
		c.BeginPrefetchRound()
		confirmMany(t, c, 5)
		c.EndPrefetchRound()
	}
	require.EqualValues(t, 10, l1.finalizedCalls.Load())
	require.EqualValues(t, 2, l1.latestCalls.Load())
}

// TestCheckConfirmations pins the N-confirmations rule at its edges: a receipt in the latest
// block has zero confirmations, and exactly `required` confirmations is enough.
func TestCheckConfirmations(t *testing.T) {
	tests := []struct {
		name                    string
		latest, receipt, needed uint64
		wantErr                 bool
	}{
		{"receipt ahead of latest", 99, 100, 0, true},
		{"receipt in latest block, none needed", 100, 100, 0, false},
		{"receipt in latest block, one needed", 100, 100, 1, true},
		{"one short of required", 105, 100, 6, true},
		{"exactly required", 106, 100, 6, false},
		{"more than required", 200, 100, 6, false},
		{"far short of required", 100, 95, 10, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkConfirmations(tt.latest, tt.receipt, tt.needed)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGetConfirmedTxReceipt_FallbackConfirmsAtRequiredDepth: without a finalized API the
// receipt is confirmed once the latest head is exactly `required` blocks above it, not before.
func TestGetConfirmedTxReceipt_FallbackConfirmsAtRequiredDepth(t *testing.T) {
	for _, tc := range []struct {
		latest uint64
		ok     bool
	}{{105, false}, {106, true}} {
		t.Run(fmt.Sprintf("latest=%d", tc.latest), func(t *testing.T) {
			c := newRoundCaller(t, &roundL1{noFinalized: true, latestNumber: tc.latest})
			for _, inRound := range []bool{false, true} {
				if inRound {
					c.BeginPrefetchRound()
				}
				receipt, err := c.GetConfirmedTxReceipt(context.Background(), common.BigToHash(big.NewInt(1)), 6)
				if inRound {
					c.EndPrefetchRound()
				}
				if tc.ok {
					require.NoError(t, err, "inRound=%v", inRound)
					require.NotNil(t, receipt)
					require.EqualValues(t, 100, receipt.BlockNumber.Uint64())
				} else {
					require.Error(t, err, "inRound=%v", inRound)
					require.Nil(t, receipt)
				}
			}
		})
	}
}

// TestGetConfirmedTxReceipt_FinalizedBoundary: a receipt exactly at the finalized head is
// confirmed, both on the fresh lookup and from the round cache.
func TestGetConfirmedTxReceipt_FinalizedBoundary(t *testing.T) {
	l1 := &roundL1{receiptBlock: 90}
	c := newRoundCaller(t, l1)
	c.BeginPrefetchRound()
	defer c.EndPrefetchRound()
	for i := range 2 { // first: fresh finalized lookup; second: cached finalized
		receipt, err := c.GetConfirmedTxReceipt(context.Background(), common.BigToHash(big.NewInt(int64(i+1))), 6)
		require.NoError(t, err, "call %d", i)
		require.NotNil(t, receipt, "call %d", i)
	}
	require.EqualValues(t, 1, l1.finalizedCalls.Load(), "the second receipt must be decided from the cache")
	require.Zero(t, l1.latestCalls.Load())
}

// TestGetConfirmedTxReceipt_LookupErrors: an RPC error on the receipt or on the latest head
// is returned, never turned into a confirmed receipt.
func TestGetConfirmedTxReceipt_LookupErrors(t *testing.T) {
	for name, l1 := range map[string]*roundL1{
		"receipt": {failReceipt: true},
		"latest":  {noFinalized: true, failLatest: true, latestNumber: 200},
	} {
		t.Run(name, func(t *testing.T) {
			c := newRoundCaller(t, l1)
			for _, inRound := range []bool{false, true} {
				if inRound {
					c.BeginPrefetchRound()
				}
				receipt, err := c.GetConfirmedTxReceipt(context.Background(), common.BigToHash(big.NewInt(1)), 6)
				if inRound {
					c.EndPrefetchRound()
				}
				require.Error(t, err, "inRound=%v", inRound)
				require.Nil(t, receipt, "inRound=%v", inRound)
			}
		})
	}
}

// TestGetConfirmedTxReceipt_LookupAcrossRoundsDoesNotFillNextRound: a finalized lookup that
// starts in one round and finishes after that round ended and a new one began must not fill
// the new round's cache. The generation must never repeat across End/Begin.
func TestGetConfirmedTxReceipt_LookupAcrossRoundsDoesNotFillNextRound(t *testing.T) {
	l1 := &roundL1{}
	c := newRoundCaller(t, l1)
	l1.beforeFinalized = func() {
		if l1.finalizedCalls.Load() == 1 {
			c.EndPrefetchRound()
			c.BeginPrefetchRound()
		}
	}
	c.BeginPrefetchRound()
	_, _ = c.GetConfirmedTxReceipt(context.Background(), common.BigToHash(big.NewInt(1)), 6)
	c.prefetchMu.RLock()
	cached := c.finalizedHeaderCache
	c.prefetchMu.RUnlock()
	c.EndPrefetchRound()
	require.Nil(t, cached, "a lookup from the previous round must not fill the next round's cache")
}

// TestGetConfirmedTxReceipt_AfterRoundEndsBehavesAsOutside: once a round ends, callers are
// outside a round again and re-check the finalized head on every call.
func TestGetConfirmedTxReceipt_AfterRoundEndsBehavesAsOutside(t *testing.T) {
	l1 := &roundL1{}
	c := newRoundCaller(t, l1)
	c.BeginPrefetchRound()
	c.EndPrefetchRound()
	confirmMany(t, c, 3)
	require.EqualValues(t, 3, l1.finalizedCalls.Load())
}
