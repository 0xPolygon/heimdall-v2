package peerpolicy

import (
	"fmt"
	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"strings"
	"sync"
	"testing"
	"time"
)

const peerA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const peerB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func testGovernor(t *testing.T, cfg Config) (*Governor, *time.Duration) {
	t.Helper()
	now := time.Duration(0)
	g, err := newGovernor(cfg, prometheus.NewRegistry(), func() time.Duration { return now })
	require.NoError(t, err)
	return g, &now
}
func serve(t *testing.T, g *Governor, id string, r servebudget.Request, n uint64) {
	t.Helper()
	l, ok := g.Admit(id, r)
	require.True(t, ok)
	require.True(t, l.Prepare(n))
	l.Finish(true)
	l.Finish(false)
}
func TestServingRepeatScore(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RepeatBytes = 100
	g, now := testGovernor(t, cfg)
	r := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 100}
	serve(t, g, peerA, r, 100)
	serve(t, g, peerA, r, 100)
	require.Zero(t, g.Snapshot(peerA).Risk)
	for tick := 0; tick < 5; tick++ {
		*now = time.Duration(tick) * windowWidth
		serve(t, g, peerA, r, 100)
		for range 3 {
			l, ok := g.Admit(peerA, r)
			require.False(t, ok)
			require.Nil(t, l)
		}
		require.Equal(t, uint64((tick+1)*20), g.Snapshot(peerA).Risk)
	}
	require.Equal(t, "jail", g.Snapshot(peerA).WouldAction)
	// Even a score of 100 has no connection or unrelated-object enforcement.
	serve(t, g, peerA, servebudget.Request{Family: servebudget.Chunk, Height: 2, MaxBytes: 100}, 10)
	*now = 10 * windowWidth
	require.Zero(t, g.Snapshot(peerA).Risk)
}
func TestServingSharedHeadroom(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxInflight = 8
	cfg.ProtectedIDs = []string{strings.ToUpper(peerB)}
	g, _ := testGovernor(t, cfg)
	var leases []servebudget.Lease
	for i := 0; i < 4; i++ {
		l, ok := g.Admit(fmt.Sprintf("%040x", i+1), servebudget.Request{Family: servebudget.Family(i % 3), Height: 1, MaxBytes: 100})
		require.True(t, ok)
		leases = append(leases, l)
	}
	_, ok := g.Admit(peerA, servebudget.Request{Family: servebudget.Chunk, MaxBytes: 100})
	require.False(t, ok)
	require.Zero(t, g.Snapshot(peerA).Risk)
	for _, req := range []servebudget.Request{{Family: servebudget.Catchup, MaxBytes: 100}, {Family: servebudget.Block, MaxBytes: 100}} {
		id := peerA
		if req.Family == servebudget.Block {
			id = peerB
		}
		l, ok := g.Admit(id, req)
		require.True(t, ok)
		leases = append(leases, l)
	}
	for _, l := range leases {
		l.Finish(false)
		l.Finish(false)
	}
	for _, p := range g.pools {
		require.Zero(t, p.memory)
		require.Zero(t, p.inflight)
	}
}
func TestServingCompletionAndConcurrency(t *testing.T) {
	g, _ := testGovernor(t, DefaultConfig())
	req := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 1024}
	l, ok := g.Admit(peerA, req)
	require.True(t, ok)
	require.True(t, l.Prepare(100))
	_, ok = g.Admit(peerA, req)
	require.False(t, ok)
	require.Zero(t, g.Snapshot(peerA).Risk)
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() { defer wg.Done(); l.Finish(false) }()
	}
	wg.Wait()
	require.Zero(t, g.peers[peerA].objects[0].copies)
	require.False(t, l.Prepare(100))
	require.Zero(t, g.pools[0].memory)
	serve(t, g, peerA, req, 100)
	require.Equal(t, uint64(1), g.peers[peerA].objects[0].copies)
}
func TestServingEvidenceAndExpiry(t *testing.T) {
	g, now := testGovernor(t, DefaultConfig())
	g.Observe(peerA, servebudget.MalformedRequest)
	require.Equal(t, uint64(20), g.Snapshot(peerA).Risk)
	g.Observe(peerA, servebudget.InvalidResponse)
	require.Equal(t, uint64(80), g.Snapshot(peerA).Risk)
	g.Observe(peerA, servebudget.InvalidResponse)
	require.Equal(t, uint64(100), g.Snapshot(peerA).Risk)
	*now = 60 * time.Second
	require.Zero(t, g.Snapshot(peerA).Risk)
	cfg := DefaultConfig()
	cfg.Observe = false
	g, _ = testGovernor(t, cfg)
	g.Observe(peerA, servebudget.InvalidResponse)
	require.Zero(t, g.Snapshot(peerA).Risk)
}
func TestServingBoundedState(t *testing.T) {
	g, _ := testGovernor(t, DefaultConfig())
	for i := 0; i < maxPeers+50; i++ {
		g.Observe(fmt.Sprintf("%040x", i), servebudget.MalformedRequest)
	}
	require.Len(t, g.peers, maxPeers)
	g.Observe("bad", servebudget.InvalidResponse)
	require.Len(t, g.peers, maxPeers)
}
func TestServingLimits(t *testing.T) {
	for _, kind := range []string{"bytes", "memory", "work"} {
		t.Run(kind, func(t *testing.T) {
			cfg := DefaultConfig()
			switch kind {
			case "bytes":
				cfg.BurstBytes = 4
			case "memory":
				cfg.MemoryBytes = 4
			case "work":
				cfg.RequestsPerSecond = 4
			}
			g, _ := testGovernor(t, cfg)
			r := servebudget.Request{Family: servebudget.Block, MaxBytes: 100}
			if kind == "work" {
				serve(t, g, peerB, r, 10)
				r.Height++
				serve(t, g, peerB, r, 10)
			}
			for range 3 {
				_, ok := g.Admit(peerA, r)
				require.False(t, ok)
			}
			require.Zero(t, g.Snapshot(peerA).Risk)
		})
	}
}
func TestServingPrepareAndRefund(t *testing.T) {
	g, _ := testGovernor(t, DefaultConfig())
	r := servebudget.Request{Family: servebudget.Chunk, MaxBytes: 1024}
	initial := g.pools[0].bytes.tokens
	l, ok := g.Admit(peerA, r)
	require.True(t, ok)
	require.Equal(t, initial-1024, g.pools[0].bytes.tokens)
	require.False(t, l.Prepare(1025))
	l.Finish(false)
	require.Equal(t, initial, g.pools[0].bytes.tokens)
	l, ok = g.Admit(peerA, r)
	require.True(t, ok)
	require.True(t, l.Prepare(10))
	require.Equal(t, initial-10, g.pools[0].bytes.tokens)
	l.Finish(false)
	require.Equal(t, initial-10, g.pools[0].bytes.tokens)
}
func BenchmarkServingAdmission(b *testing.B) {
	now := time.Duration(0)
	g, err := newGovernor(DefaultConfig(), prometheus.NewRegistry(), func() time.Duration { return now })
	require.NoError(b, err)
	r := servebudget.Request{Family: servebudget.Block, MaxBytes: 128 << 10}
	b.ReportAllocs()
	for b.Loop() {
		now += time.Second
		if l, ok := g.Admit(peerA, r); ok {
			l.Finish(false)
		}
	}
}

func TestServingCatchupIsNeutral(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PeerRequestsPerSecond = 1
	g, _ := testGovernor(t, cfg)
	serve(t, g, peerA, servebudget.Request{Family: servebudget.Catchup, MaxBytes: 100}, 10)
	for range 3 {
		_, ok := g.Admit(peerA, servebudget.Request{Family: servebudget.Catchup, Height: 2, MaxBytes: 100})
		require.False(t, ok)
	}
	require.Zero(t, g.Snapshot(peerA).Risk)
	serve(t, g, peerA, servebudget.Request{Family: servebudget.Block, MaxBytes: 100}, 10)
}

func TestServingPrepareGrowth(t *testing.T) {
	for _, kind := range []string{"admit", "global", "peer", "repeat"} {
		t.Run(kind, func(t *testing.T) {
			g, _ := testGovernor(t, DefaultConfig())
			r := servebudget.Request{Family: servebudget.Block, MaxBytes: 1 << 20}
			l, ok := g.Admit(peerA, r)
			require.True(t, ok)
			switch kind {
			case "global":
				g.pools[0].bytes.tokens = 0
			case "peer":
				g.peers[peerA].bytes.tokens = 0
			case "repeat":
				active := l.(*lease)
				active.repeat = true
				active.p.repeatHeld = active.bytes
				g.cfg.RepeatBytes = active.bytes
			}
			require.Equal(t, kind == "admit", l.Prepare(512<<10))
			if kind == "admit" {
				require.False(t, l.Prepare(512<<10))
			}
			l.Finish(false)
			require.Zero(t, g.pools[0].memory)
			require.Zero(t, g.peers[peerA].repeatHeld)
		})
	}
}
func TestServingPeerQuotaAndCongestion(t *testing.T) {
	g, _ := testGovernor(t, DefaultConfig())
	r := servebudget.Request{Family: servebudget.Block, MaxBytes: 100}
	serve(t, g, peerA, r, 10)
	g.peers[peerA].bytes.tokens = 0
	g.pools[0].bytes.tokens = 0
	for range 3 {
		_, ok := g.Admit(peerA, r)
		require.False(t, ok)
	}
	require.Zero(t, g.Snapshot(peerA).Risk)
	g.pools[0].bytes.tokens = 1000
	for range 3 {
		_, ok := g.Admit(peerA, r)
		require.False(t, ok)
	}
	require.Equal(t, uint64(20), g.Snapshot(peerA).Risk)
}
func TestServingConfigAndRegistration(t *testing.T) {
	for _, edit := range []func(*Config){func(c *Config) { c.BytesPerSecond = 0 }, func(c *Config) { c.MemoryBytes = 1 << 41 }, func(c *Config) { c.MaxInflight = 3 }, func(c *Config) { c.MaxInflight = 4097 }, func(c *Config) { c.ProtectedIDs = []string{"invalid"} }, func(c *Config) { c.ProtectedIDs = make([]string, 257) }} {
		cfg := DefaultConfig()
		edit(&cfg)
		_, err := New(cfg, prometheus.NewRegistry())
		require.Error(t, err)
	}
	reg := prometheus.NewRegistry()
	_, err := New(DefaultConfig(), reg)
	require.NoError(t, err)
	_, err = New(DefaultConfig(), reg)
	require.Error(t, err)
}
func TestServingLeaseSurvivesEviction(t *testing.T) {
	g, _ := testGovernor(t, DefaultConfig())
	l, ok := g.Admit(peerA, servebudget.Request{Family: servebudget.Block, MaxBytes: 100})
	require.True(t, ok)
	for i := 0; i < maxPeers+1; i++ {
		g.Observe(fmt.Sprintf("%040x", i), servebudget.MalformedRequest)
	}
	require.NotNil(t, g.peers[peerA])
	require.True(t, l.Prepare(10))
	l.Finish(true)
	require.Zero(t, g.pools[0].memory)
}
