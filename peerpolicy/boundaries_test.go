package peerpolicy

import (
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestServingWorkRefill(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PeerRequestsPerSecond = 2
	g, now := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Chunk, MaxBytes: 10}
	serve(t, g, peerA, req, 1)
	serve(t, g, peerA, req, 1)
	for range 3 {
		_, ok := g.Admit(peerA, req)
		require.False(t, ok)
	}
	require.Equal(t, uint64(20), g.Snapshot(peerA).Risk)
	*now = 250 * time.Millisecond
	_, ok := g.Admit(peerA, req)
	require.False(t, ok)
	*now = 500 * time.Millisecond
	serve(t, g, peerA, req, 1)
	_, ok = g.Admit(peerA, req)
	require.False(t, ok)
	*now = time.Second
	serve(t, g, peerA, req, 1)
	require.Equal(t, float64(1), testutil.ToFloat64(g.windows.WithLabelValues("peer_allowance")))
}

func TestServingRequestOnlyScoreAndOverlap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RepeatBytes = 10
	g, now := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Block, MaxBytes: 10}
	for range 3 {
		serve(t, g, peerA, req, 10)
	}
	for range 3 {
		_, ok := g.Admit(peerA, req)
		require.False(t, ok)
	}
	g.peers[peerA].work.tokens = 0
	for range 3 {
		_, ok := g.Admit(peerA, req)
		require.False(t, ok)
	}
	require.Equal(t, uint64(20), g.Snapshot(peerA).Risk)
	require.Equal(t, int64(-20), g.Snapshot(peerA).Score)
	require.Equal(t, "none", g.Snapshot(peerA).WouldAction)
	*now = windowWidth
	g.Observe(peerA, servebudget.MalformedRequest)
	require.Equal(t, "reduce_bulk", g.Snapshot(peerA).WouldAction)
	g.Observe(peerA, servebudget.MalformedRequest)
	require.Equal(t, "stop_bulk", g.Snapshot(peerA).WouldAction)
	*now = 6 * windowWidth
	require.Equal(t, uint64(40), g.Snapshot(peerA).Risk)
	*now = 7 * windowWidth
	require.Zero(t, g.Snapshot(peerA).Risk)
}

func TestServingRefusedWorkDoesNotCreateCopies(t *testing.T) {
	g, _ := testGovernor(t, DefaultConfig())
	req := servebudget.Request{Family: servebudget.Block, MaxBytes: 10}
	for range 6 {
		l, ok := g.Admit(peerA, req)
		require.True(t, ok)
		l.Finish(false)
	}
	serve(t, g, peerA, req, 10)
	require.Zero(t, g.Snapshot(peerA).Risk)
	require.Equal(t, float64(6), testutil.ToFloat64(g.events.WithLabelValues("incomplete")))
	require.Equal(t, float64(1), testutil.ToFloat64(g.events.WithLabelValues("written")))
}

func TestServingReservedPoolsByteBoundary(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BurstBytes = 400
	cfg.ProtectedIDs = []string{peerB}
	g, _ := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Block, MaxBytes: 200}
	serve(t, g, peerA, req, 200)
	_, ok := g.Admit(peerA, req)
	require.False(t, ok)
	req.MaxBytes = 100
	serve(t, g, peerB, req, 100)
	_, ok = g.Admit(peerB, req)
	require.False(t, ok)
	req.Family = servebudget.Catchup
	serve(t, g, peerA, req, 100)
	_, ok = g.Admit(peerA, req)
	require.False(t, ok)
	require.Zero(t, g.Snapshot(peerA).Risk)
	require.Zero(t, g.Snapshot(peerB).Risk)
}

func TestServingObservationDisabledStillLimits(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Observe = false
	cfg.PeerRequestsPerSecond = 1
	g, _ := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Chunk, MaxBytes: 10}
	serve(t, g, peerA, req, 1)
	for range 5 {
		_, ok := g.Admit(peerA, req)
		require.False(t, ok)
	}
	g.Observe(peerA, servebudget.InvalidResponse)
	require.Zero(t, g.Snapshot(peerA).Risk)
	require.Equal(t, float64(0), testutil.ToFloat64(g.windows.WithLabelValues("peer_allowance")))
	require.Equal(t, float64(5), testutil.ToFloat64(g.events.WithLabelValues("peer_allowance")))
}

func TestServingConfigBoundaries(t *testing.T) {
	for _, inflight := range []uint64{4, 4096} {
		cfg := DefaultConfig()
		cfg.MaxInflight = inflight
		require.NoError(t, cfg.Validate())
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.BytesPerSecond = 3 },
		func(c *Config) { c.RequestsPerSecond = 3 },
		func(c *Config) { c.BurstBytes = 3 },
		func(c *Config) { c.MemoryBytes = 3 },
	} {
		cfg := DefaultConfig()
		change(&cfg)
		_, err := New(cfg, prometheus.NewRegistry())
		require.Error(t, err)
	}
}
