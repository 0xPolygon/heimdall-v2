package peerpolicy

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/observation"
	bc "github.com/cometbft/cometbft/proto/tendermint/blocksync"
	mp "github.com/cometbft/cometbft/proto/tendermint/mempool"
	ss "github.com/cometbft/cometbft/proto/tendermint/statesync"
	ct "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func fixture(t *testing.T) (*Tracker, *time.Duration, *prometheus.Registry) {
	t.Helper()
	now := new(time.Duration)
	reg := prometheus.NewRegistry()
	tracker, err := newTracker(reg, func() time.Duration { return *now }, false)
	require.NoError(t, err)
	return tracker, now, reg
}

func TestPeerPolicyRiskAndExpiry(t *testing.T) {
	t.Parallel()
	tr, now, _ := fixture(t)
	tr.Observe("peer", observation.Event{Kind: observation.InvalidEncoding})
	require.Equal(t, uint64(60), tr.Snapshot("peer").Risk)
	require.Equal(t, "throttle", tr.Snapshot("peer").WouldAction)
	// A second module reporting this delivery must not stack the score.
	tr.Observe("peer", observation.Event{Kind: observation.InvalidMessage})
	require.Equal(t, uint64(60), tr.Snapshot("peer").Risk)
	*now = windowWidth
	tr.Observe("peer", observation.Event{Kind: observation.InvalidMessage})
	require.Equal(t, uint64(100), tr.Snapshot("peer").Risk)
	require.Equal(t, "jail", tr.Snapshot("peer").WouldAction)
	*now = 6*windowWidth - time.Nanosecond
	require.Equal(t, uint64(100), tr.Snapshot("peer").Risk)
	*now = 6 * windowWidth
	require.Equal(t, uint64(60), tr.Snapshot("peer").Risk)
	require.NotContains(t, tr.Snapshot("peer").ReasonWindows, "invalid_encoding")
	*now = 7 * windowWidth
	require.Zero(t, tr.Snapshot("peer").Risk)
	require.Equal(t, "none", tr.Snapshot("peer").WouldAction)
}

func TestPeerPolicyVolumeBoundaries(t *testing.T) {
	t.Parallel()
	for f := blockRequests; f < familyCount; f++ {
		t.Run(familyNames[f], func(t *testing.T) {
			t.Parallel()
			p := peerRecord{objects: make(map[objectKey]int)}
			var b bucket
			a := allowances[f]
			require.Equal(t, none, p.traffic(&b, 0, evidence{family: f, items: a.items, bytes: a.bytes}))
			require.Equal(t, a.reason, p.traffic(&b, 0, evidence{family: f, items: 1}))
			b = bucket{}
			require.Equal(t, a.reason, p.traffic(&b, 0, evidence{family: f, bytes: a.bytes + 1}))
			require.Equal(t, a.reason, p.traffic(&b, 0, evidence{family: f, items: math.MaxUint64, bytes: math.MaxUint64}))
			require.Equal(t, a.items+1, b.usage[f].items)
			require.Equal(t, a.bytes+1, b.usage[f].bytes)
		})
	}
}

func TestPeerPolicyRepeatedServing(t *testing.T) {
	t.Parallel()
	tr, now, _ := fixture(t)
	e := observation.Event{Kind: observation.Queued, Bytes: 16 << 20, Message: &bc.BlockResponse{Block: &ct.Block{Header: ct.Header{Height: 10}}}}
	for i := 0; i < 4; i++ {
		tr.Observe("peer", e)
	}
	require.Zero(t, tr.Snapshot("peer").Risk) // initial copy + retry + exactly 32 MiB repeated
	tr.Observe("peer", e)
	require.Equal(t, uint64(20), tr.Snapshot("peer").Risk)
	require.Equal(t, uint64(1), tr.Snapshot("peer").ReasonWindows["serving_repetition"])
	tr.Observe("other-peer", e)
	require.Zero(t, tr.Snapshot("other-peer").Risk)
	*now = 6 * windowWidth
	tr.Observe("peer", e)
	require.Zero(t, tr.Snapshot("peer").Risk)
	// Missing responses never qualify as completed serving.
	for i := 0; i < 8; i++ {
		tr.Observe("missing", observation.Event{Kind: observation.Queued, Bytes: 16 << 20, Message: &ss.ChunkResponse{Missing: true}})
	}
	require.Zero(t, tr.Snapshot("missing").Risk)
}

func TestPeerPolicyDifferentTransactionsStillCount(t *testing.T) {
	t.Parallel()
	tr, _, _ := fixture(t)
	for i := 0; i < 65; i++ {
		tr.Observe("peer", observation.Event{Kind: observation.Received, Bytes: 1 << 20, Message: &mp.Txs{Txs: [][]byte{[]byte(fmt.Sprint(i))}}})
	}
	s := tr.Snapshot("peer")
	require.Equal(t, uint64(20), s.Risk)
	require.Equal(t, map[string]uint64{"transaction_volume": 1}, s.ReasonWindows)
}

func TestPeerPolicyMemoryBoundsAndReconnect(t *testing.T) {
	t.Parallel()
	tr, _, registry := fixture(t)
	for i := 0; i < maxPeers; i++ {
		tr.Observe(fmt.Sprint(i), observation.Event{Kind: observation.InvalidMessage})
	}
	tr.Observe("0", observation.Event{Kind: observation.InvalidMessage})
	tr.Observe("new", observation.Event{Kind: observation.InvalidMessage})
	require.Len(t, tr.peers, maxPeers)
	require.Equal(t, maxPeers, tr.order.Len())
	families, err := registry.Gather()
	require.NoError(t, err)
	var evictions float64
	for _, family := range families {
		if family.GetName() == "heimdall_peer_reputation_evictions_total" {
			evictions = family.Metric[0].Counter.GetValue()
		}
	}
	require.Equal(t, float64(1), evictions)
	require.Zero(t, tr.Snapshot("1").Risk)
	require.Equal(t, uint64(60), tr.Snapshot("0").Risk)
	for i := uint32(0); i < 2*maxObjects; i++ {
		tr.Observe("0", observation.Event{Kind: observation.Queued, Message: &ss.ChunkResponse{Height: 1, Index: i}})
	}
	p := tr.peers["0"].Value.(*peerRecord)
	require.Len(t, p.objects, maxObjects)
	require.Less(t, p.next, maxObjects)
	require.False(t, p.repeated(0, evidence{family: chunkServing, hasObject: true}))
}

func TestPeerPolicyNeutralAndInputBounds(t *testing.T) {
	t.Parallel()
	tr, _, reg := fixture(t)
	for _, id := range []string{"", strings.Repeat("x", 129)} {
		tr.Observe(id, observation.Event{Kind: observation.InvalidEncoding})
	}
	tr.Observe("silent", observation.Event{Kind: observation.Received, Bytes: 100})
	tr.Observe("unknown", observation.Event{Kind: 255})
	require.Empty(t, tr.peers)
	tr.Observe(strings.Repeat("x", 128), observation.Event{Kind: observation.InvalidEncoding})
	require.Len(t, tr.peers, 1)
	_, err := New(reg, false)
	require.Error(t, err)
	require.Zero(t, weight(none))
	require.Zero(t, weight(reasonCount))
}

func TestPeerPolicyConcurrentObservation(t *testing.T) {
	t.Parallel()
	tr, _, reg := fixture(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				tr.Observe("peer", observation.Event{Kind: observation.InvalidEncoding})
				tr.Snapshot("peer")
				_, err := reg.Gather()
				require.NoError(t, err)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, uint64(60), tr.Snapshot("peer").Risk)
}

func BenchmarkPeerPolicyObserve(b *testing.B) {
	tr, err := New(prometheus.NewRegistry(), false)
	require.NoError(b, err)
	e := observation.Event{Kind: observation.Received, Bytes: 10, Message: &bc.BlockRequest{Height: 100}}
	tr.Observe("peer", e)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Observe("peer", e)
	}
}

func TestPeerPolicyNonzeroWindowBoundaries(t *testing.T) {
	t.Parallel()
	tr, now, _ := fixture(t)
	*now = 2 * windowWidth
	tr.Observe("peer", observation.Event{Kind: observation.InvalidEncoding})
	*now = 5 * windowWidth
	require.Equal(t, map[string]uint64{"invalid_encoding": 1}, tr.Snapshot("peer").ReasonWindows)
	p := peerRecord{objects: make(map[objectKey]int)}
	e := evidence{family: blockServing, hasObject: true}
	require.False(t, p.repeated(2, e))
	require.False(t, p.repeated(2, e))
	require.True(t, p.repeated(7, e))
	require.False(t, p.repeated(8, e))
	require.Equal(t, uint64(10), cappedAdd(6, 5, 10))
	for _, tc := range []struct {
		risk   uint64
		action int
	}{{0, 0}, {39, 0}, {40, 1}, {99, 1}, {100, 2}} {
		require.Equal(t, tc.action, actionFor(tc.risk))
	}
}
