package peerpolicy

import (
	"fmt"
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestServingRetainsLiveHistory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RepeatBytes = 1
	g, now := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 10}
	serve(t, g, peerA, req, 10)
	serve(t, g, peerA, req, 10)
	for height := uint64(2); height <= defaultPeerHistory; height++ {
		next := req
		next.Height = height
		require.NotNil(t, g.newObject(g.peers[peerA], next, 0))
	}
	next := req
	next.Height = defaultPeerHistory + 1
	_, ok := g.Admit(peerA, next)
	require.False(t, ok, "full live history must refuse new objects")
	_, ok = g.Admit(peerA, req)
	require.False(t, ok, "original repeat allowance must survive table pressure")
	*now = 6*windowWidth - time.Nanosecond
	_, ok = g.Admit(peerA, next)
	require.False(t, ok)
	*now = 6 * windowWidth
	serve(t, g, peerA, next, 10)
	serve(t, g, peerA, req, 10)
	require.Zero(t, g.Snapshot(peerA).Risk)
}

func TestServingPendingHistorySurvivesExpiry(t *testing.T) {
	g, now := testGovernor(t, DefaultConfig())
	req := servebudget.Request{Family: servebudget.Chunk, Height: 1, MaxBytes: 10}
	l, ok := g.Admit(peerA, req)
	require.True(t, ok)
	*now = 10 * windowWidth
	_, ok = g.Admit(peerA, req)
	require.False(t, ok)
	require.True(t, l.Prepare(10))
	l.Finish(true)
	serve(t, g, peerA, req, 10)
	require.Equal(t, uint64(2), g.peers[peerA].objects[req].copies)
}

func TestServingHistorySupportsDefaultThroughput(t *testing.T) {
	cfg := DefaultConfig()
	g, now := testGovernor(t, cfg)
	// Both independent peer allowances can be active during a single history
	// horizon. Unique heights must not exhaust history before their rate limits.
	for second := uint64(0); second < 60; second++ {
		*now = time.Duration(second) * time.Second
		for offset := uint64(0); offset < cfg.PeerRequestsPerSecond; offset++ {
			height := second*cfg.PeerRequestsPerSecond + offset + 1
			for _, family := range []servebudget.Family{servebudget.Block, servebudget.Catchup} {
				serve(t, g, peerA, servebudget.Request{Family: family, Height: height, MaxBytes: 10}, 10)
			}
		}
	}
	require.Zero(t, g.Snapshot(peerA).Risk)
}

func TestServingHistoryAllocatedOnDemand(t *testing.T) {
	g, _ := testGovernor(t, DefaultConfig())
	g.Observe(peerA, servebudget.MalformedRequest)
	require.Empty(t, g.peers[peerA].objects)
	req := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 10}
	serve(t, g, peerA, req, 10)
	require.Len(t, g.peers[peerA].objects, 1)
}

func TestServingHistoryNodeWideBound(t *testing.T) {
	cfg := DefaultConfig()
	g, now := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Block, MaxBytes: 10}
	peers := defaultNodeHistory / defaultPeerHistory
	for i := range peers {
		id := fmt.Sprintf("%040x", i)
		for height := uint64(1); height <= defaultPeerHistory; height++ {
			next := req
			next.Height = height
			require.NotNil(t, g.newObject(g.peer(id, 0), next, 0))
		}
		require.Len(t, g.peers[id].objects, defaultPeerHistory, "per-peer history stays bounded")
	}
	require.Equal(t, uint64(defaultNodeHistory), g.objects)

	spare := fmt.Sprintf("%040x", peers)
	_, ok := g.Admit(spare, req)
	require.False(t, ok, "a full table refuses rather than displacing live entries")
	require.Equal(t, float64(1), testutil.ToFloat64(g.events.WithLabelValues("history_capacity")))
	require.Zero(t, g.Snapshot(spare).Risk, "history capacity is neutral")
	require.Equal(t, uint64(defaultNodeHistory), g.objects)

	// The next interval reclaims settled entries node-wide, including those of
	// peers whose own admissions would never revisit them.
	*now = windowCount * windowWidth
	serve(t, g, spare, req, 10)
	require.Equal(t, uint64(1), g.objects)
}

func TestServingHistoryReclaimedOnEviction(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RequestsPerSecond = 2048
	g, now := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Block, MaxBytes: 10}
	for i := range maxPeers {
		*now = time.Duration(i) * time.Millisecond
		serve(t, g, fmt.Sprintf("%040x", i), req, 10)
	}
	require.Equal(t, uint64(maxPeers), g.objects)
	*now += time.Second
	serve(t, g, peerA, req, 10)
	require.Len(t, g.peers, maxPeers)
	require.Equal(t, uint64(maxPeers), g.objects, "eviction returns the peer's entries to the budget")
}

func TestServingRefusedRequestAllocatesNoHistory(t *testing.T) {
	g, _ := testGovernor(t, DefaultConfig())
	p := g.peer(peerA, 0)
	p.bytes.tokens = 0
	for height := uint64(1); height <= 64; height++ {
		_, ok := g.Admit(peerA, servebudget.Request{Family: servebudget.Block, Height: height, MaxBytes: 10})
		require.False(t, ok)
	}
	require.Empty(t, p.objects, "a refused request must not claim a history slot")
	require.Zero(t, g.objects)
	require.Equal(t, uint64(20), g.Snapshot(peerA).Risk)
}

func BenchmarkServingHistory(b *testing.B) {
	for _, size := range []int{1, defaultPeerHistory} {
		b.Run(fmt.Sprintf("objects_%d", size), func(b *testing.B) {
			g, _ := testGovernor(b, DefaultConfig())
			p := g.peer(peerA, 0)
			for i := 0; i < size; i++ {
				g.newObject(p, servebudget.Request{Family: servebudget.Block, Height: uint64(i + 1), MaxBytes: 10}, 0)
			}
			req := servebudget.Request{Family: servebudget.Block, Height: uint64(size), MaxBytes: 10}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				g.lookupObject(p, req, 0)
			}
		})
	}
}

func TestServingHistoryExpiresWithoutDroppingPending(t *testing.T) {
	g, _ := testGovernor(t, DefaultConfig())
	p := g.peer(peerA, 0)
	one := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 10}
	two := one
	two.Height++
	old := g.newObject(p, one, 0)
	pending := g.newObject(p, two, 0)
	pending.pending = 1
	next := one
	next.Height = 3
	g.newObject(p, next, windowCount)
	require.NotContains(t, p.objects, one)
	require.Same(t, pending, g.lookupObject(p, two, windowCount))
	require.Nil(t, g.lookupObject(p, one, windowCount))
	require.NotSame(t, old, g.newObject(p, one, windowCount))
	require.Equal(t, uint64(len(p.objects)), g.objects)
	g.expire(p, 2*windowCount)
	require.Len(t, p.objects, 1)
	pending.pending = 0
	g.expire(p, 2*windowCount+1)
	require.Nil(t, p.objects)
	require.Zero(t, g.objects)
}
