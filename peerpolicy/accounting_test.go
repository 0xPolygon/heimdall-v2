package peerpolicy

import (
	"fmt"
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestServingMemoryBoundary(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MemoryBytes = 400
	g, _ := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Block, MaxBytes: 100}
	l, ok := g.Admit(peerA, req)
	require.True(t, ok, "encoded and decoded reservations exactly fill the ordinary pool")
	require.Equal(t, uint64(200), g.pools[0].memory)
	_, ok = g.Admit(peerB, req)
	require.False(t, ok)
	require.Equal(t, float64(1), testutil.ToFloat64(g.events.WithLabelValues("local_capacity")))
	l.Finish(false)
	require.Zero(t, g.pools[0].memory)
	serve(t, g, peerB, req, 100)
	require.Zero(t, g.pools[0].memory)
}

func TestServingPeerInflightBoundary(t *testing.T) {
	g, _ := testGovernor(t, DefaultConfig())
	var leases []servebudget.Lease
	for i := range maxPeerInflight {
		l, ok := g.Admit(peerA, servebudget.Request{Family: servebudget.Chunk, Height: uint64(i), MaxBytes: 10})
		require.True(t, ok)
		leases = append(leases, l)
	}
	_, ok := g.Admit(peerA, servebudget.Request{Family: servebudget.Chunk, Height: maxPeerInflight, MaxBytes: 10})
	require.False(t, ok)
	require.Equal(t, float64(1), testutil.ToFloat64(g.events.WithLabelValues("peer_busy")))
	require.Zero(t, g.Snapshot(peerA).Risk)
	leases[0].Finish(false)
	serve(t, g, peerA, servebudget.Request{Family: servebudget.Chunk, Height: maxPeerInflight, MaxBytes: 10}, 10)
	for _, l := range leases {
		l.Finish(false)
	}
	require.Zero(t, g.peers[peerA].inflight)
	require.Zero(t, g.pools[0].inflight)
}

func TestServingLedgerEviction(t *testing.T) {
	g, now := testGovernor(t, DefaultConfig())
	for i := range maxPeers {
		*now = time.Duration(i) * time.Millisecond
		g.Observe(fmt.Sprintf("%040x", i), servebudget.MalformedRequest)
	}
	oldest := fmt.Sprintf("%040x", 0)
	// Refresh one old peer; the next oldest idle peer must leave instead.
	*now += time.Second
	g.Observe(oldest, servebudget.MalformedRequest)
	g.Observe(peerA, servebudget.InvalidResponse)
	require.Contains(t, g.peers, oldest)
	require.NotContains(t, g.peers, fmt.Sprintf("%040x", 1))
	require.Equal(t, float64(1), testutil.ToFloat64(g.events.WithLabelValues("evicted")))
	for _, p := range g.peers {
		p.inflight = 1
	}
	l, ok := g.Admit(peerB, servebudget.Request{MaxBytes: 10})
	require.False(t, ok)
	require.Nil(t, l)
	g.Observe(peerB, servebudget.InvalidResponse)
	require.NotContains(t, g.peers, peerB)
	require.Len(t, g.peers, maxPeers)
}

func TestServingLeaseAccounting(t *testing.T) {
	for _, size := range []uint64{64, 128 << 10, 256 << 10} {
		for _, written := range []bool{false, true} {
			t.Run(fmt.Sprintf("size_%d_written_%t", size, written), func(t *testing.T) {
				g, _ := testGovernor(t, DefaultConfig())
				req := servebudget.Request{Family: servebudget.Block, MaxBytes: 1 << 20}
				// The third completed transfer consumes the repeated-byte allowance.
				serve(t, g, peerA, req, 128<<10)
				serve(t, g, peerA, req, 128<<10)
				l, ok := g.Admit(peerA, req)
				require.True(t, ok)
				initial := g.pools[0].bytes.capacity - 2*(128<<10)
				peerInitial := g.peers[peerA].bytes.capacity - 2*(128<<10)
				require.Equal(t, uint64(128<<10), g.peers[peerA].repeatHeld)
				require.True(t, l.Prepare(size))
				require.Equal(t, initial-float64(size), g.pools[0].bytes.tokens)
				require.Equal(t, peerInitial-float64(size), g.peers[peerA].bytes.tokens)
				require.Equal(t, size, g.peers[peerA].repeatHeld)
				l.Finish(written)
				l.Finish(!written)
				require.Zero(t, g.peers[peerA].repeatHeld)
				require.Zero(t, g.pools[0].memory)
				require.Zero(t, g.pools[0].inflight)
				expected := uint64(0)
				if written {
					expected = size
				}
				require.Equal(t, expected, g.peers[peerA].windows[0].repeatedBytes)
				require.Equal(t, initial-float64(size), g.pools[0].bytes.tokens)
			})
		}
	}
}

func TestServingPrepareAllowanceBoundaries(t *testing.T) {
	for _, kind := range []string{"local", "peer", "repeat"} {
		for _, spare := range []uint64{99, 100} {
			t.Run(fmt.Sprintf("%s_%d", kind, spare), func(t *testing.T) {
				cfg := DefaultConfig()
				g, _ := testGovernor(t, cfg)
				req := servebudget.Request{Family: servebudget.Block, MaxBytes: 1 << 20}
				serve(t, g, peerA, req, 100)
				serve(t, g, peerA, req, 100)
				l, ok := g.Admit(peerA, req)
				require.True(t, ok)
				switch kind {
				case "local":
					g.pools[0].bytes.tokens = float64(spare)
				case "peer":
					g.peers[peerA].bytes.tokens = float64(spare)
				case "repeat":
					g.cfg.RepeatBytes = 100 + spare
				}
				require.Equal(t, spare == 100, l.Prepare(200))
				if spare == 99 {
					reason := map[string]string{"local": "local_bytes", "peer": "peer_allowance", "repeat": "repeat_allowance"}[kind]
					require.Equal(t, float64(1), testutil.ToFloat64(g.events.WithLabelValues(reason)))
				}
				l.Finish(false)
				require.Zero(t, g.peers[peerA].repeatHeld)
			})
		}
	}
}

func TestServingConcurrentRepeatReservations(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RepeatBytes = 300
	g, _ := testGovernor(t, cfg)
	first := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 200}
	second := first
	second.Height = 2
	for range 2 {
		serve(t, g, peerA, first, 100)
		serve(t, g, peerA, second, 100)
	}
	l, ok := g.Admit(peerA, first)
	require.True(t, ok)
	require.True(t, l.Prepare(200))
	m, ok := g.Admit(peerA, second)
	require.True(t, ok, "the combined reservation exactly meets the limit")
	require.False(t, m.Prepare(101))
	require.True(t, m.Prepare(100))
	l.Finish(true)
	m.Finish(true)
	require.Equal(t, uint64(300), g.peers[peerA].windows[0].repeatedBytes)
	require.Zero(t, g.peers[peerA].repeatHeld)
	_, ok = g.Admit(peerA, first)
	require.False(t, ok)
}

func TestServingMetricsAndMode(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Observe = enabled
			cfg.PeerRequestsPerSecond = 1
			reg := prometheus.NewRegistry()
			g, err := newGovernor(cfg, reg, func() time.Duration { return 0 })
			require.NoError(t, err)
			req := servebudget.Request{Family: servebudget.Chunk, MaxBytes: 100}
			serve(t, g, peerA, req, 10)
			for range 3 {
				_, ok := g.Admit(peerA, req)
				require.False(t, ok)
			}
			g.Observe(peerA, servebudget.MalformedRequest)
			g.Observe(peerA, servebudget.InvalidResponse)
			mode := "disabled"
			if enabled {
				mode = "observe"
			}
			require.Equal(t, mode, g.Mode())
			require.Equal(t, mode, g.Snapshot(peerA).Mode)
			metrics, err := reg.Gather()
			require.NoError(t, err)
			names := make(map[string]bool)
			for _, m := range metrics {
				names[m.GetName()] = true
			}
			require.True(t, names["heimdall_peer_serving_events_total"])
			require.True(t, names["heimdall_peer_reputation_risk"])
			require.Equal(t, enabled, names["heimdall_peer_reputation_windows_total"])
			if enabled {
				require.Equal(t, float64(1), testutil.ToFloat64(g.events.WithLabelValues("invalid_response")))
				require.Equal(t, float64(1), testutil.ToFloat64(g.events.WithLabelValues("malformed_request")))
				require.Equal(t, uint64(100), g.Snapshot(peerA).Risk)
			}
		})
	}
}

func TestServingConfigInclusiveBounds(t *testing.T) {
	cfg := Config{BytesPerSecond: 4, BurstBytes: 4, MemoryBytes: 4, MaxInflight: 4, RequestsPerSecond: 4, PeerBytesPerSecond: 1 << 40, PeerBurstBytes: 1 << 40, PeerRequestsPerSecond: 1 << 40, RepeatBytes: 1 << 40}
	for i := range 256 {
		cfg.ProtectedIDs = append(cfg.ProtectedIDs, fmt.Sprintf("%040x", i))
	}
	require.NoError(t, cfg.Validate())
	cfg.ProtectedIDs = append(cfg.ProtectedIDs, peerA)
	require.Error(t, cfg.Validate())
}
