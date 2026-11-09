package peerpolicy

import (
	"testing"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestServingUnseenByteAdmission(t *testing.T) {
	for _, tc := range []struct{ scope, reason string }{{"pool", "local_bytes"}, {"peer", "peer_allowance"}} {
		t.Run(tc.scope, func(t *testing.T) {
			g, _ := testGovernor(t, DefaultConfig())
			p := g.peer(peerA, 0)
			bucket := &g.pools[0].bytes
			if tc.scope == "peer" {
				bucket = &p.bytes
			}
			bucket.tokens = 128 << 10
			poolWork, peerWork := g.pools[0].work.tokens, p.work.tokens
			req := servebudget.Request{Family: servebudget.Block, MaxBytes: 1 << 20}
			for range 3 {
				l, ok := g.Admit(peerA, req)
				if l != nil {
					l.Finish(false)
				}
				require.False(t, ok)
				require.Nil(t, l)
			}
			require.Equal(t, float64(128<<10), bucket.tokens)
			require.Equal(t, poolWork, g.pools[0].work.tokens)
			require.Equal(t, peerWork, p.work.tokens)
			require.Zero(t, g.pools[0].memory)
			require.Zero(t, p.inflight)
			require.Equal(t, float64(3), testutil.ToFloat64(g.events.WithLabelValues(tc.reason)))
			if tc.scope == "pool" {
				require.Zero(t, g.Snapshot(peerA).Risk)
			}
			bucket.tokens = float64(req.MaxBytes)
			serve(t, g, peerA, req, req.MaxBytes)
			require.Zero(t, bucket.tokens)
		})
	}
}

// A conformant reader that pipelines concurrent requests drains its burst with
// reservations it still holds. That shortfall is congestion, not excess, so it
// must stay neutral; only an allowance spent by completed serving is scored.
func TestServingConcurrentReaderIsNeutral(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PeerBytesPerSecond, cfg.PeerBurstBytes = 1<<20, 1<<20
	g, _ := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Block, MaxBytes: 1 << 20}
	first, ok := g.Admit(peerA, req)
	require.True(t, ok)
	for height := uint64(1); height < maxPeerInflight; height++ {
		next := req
		next.Height = height
		l, ok := g.Admit(peerA, next)
		require.False(t, ok, "one maximum reservation at a time fits the peer burst")
		require.Nil(t, l)
	}
	require.Zero(t, g.Snapshot(peerA).Risk)
	require.Equal(t, float64(maxPeerInflight-1), testutil.ToFloat64(g.events.WithLabelValues("peer_reserved")))
	require.Zero(t, testutil.ToFloat64(g.events.WithLabelValues("peer_allowance")))

	require.True(t, first.Prepare(req.MaxBytes))
	first.Finish(true)
	require.Zero(t, g.peers[peerA].held[0])
	done := req
	done.Height = maxPeerInflight
	for range 3 {
		_, ok := g.Admit(peerA, done)
		require.False(t, ok)
	}
	require.Equal(t, uint64(20), g.Snapshot(peerA).Risk, "completed serving still exposes real excess")
	require.Equal(t, float64(3), testutil.ToFloat64(g.events.WithLabelValues("peer_allowance")))

	g.pools[0].bytes.tokens = 0
	_, ok = g.Admit(peerA, done)
	require.False(t, ok)
	require.Equal(t, float64(3), testutil.ToFloat64(g.events.WithLabelValues("peer_allowance")), "local congestion stays neutral")
	require.Equal(t, uint64(20), g.Snapshot(peerA).Risk)
}

func TestServingMaximumReservationAndCompletionCache(t *testing.T) {
	for _, written := range []bool{false, true} {
		t.Run(map[bool]string{false: "incomplete", true: "completed"}[written], func(t *testing.T) {
			g, _ := testGovernor(t, DefaultConfig())
			req := servebudget.Request{Family: servebudget.Block, MaxBytes: 1 << 20}
			p := g.peer(peerA, 0)
			g.pools[0].bytes.tokens = float64(req.MaxBytes)
			p.bytes.tokens = float64(req.MaxBytes)
			l, ok := g.Admit(peerA, req)
			require.True(t, ok)
			require.Zero(t, g.pools[0].bytes.tokens)
			require.Zero(t, p.bytes.tokens)
			require.True(t, l.Prepare(128<<10))
			require.Equal(t, float64(req.MaxBytes-(128<<10)), g.pools[0].bytes.tokens)
			require.Equal(t, g.pools[0].bytes.tokens, p.bytes.tokens)
			l.Finish(written)
			cached, ok := g.Admit(peerA, req)
			require.Equal(t, written, ok)
			if written {
				require.True(t, cached.Prepare(128<<10))
				cached.Finish(true)
				require.Equal(t, float64(req.MaxBytes-2*(128<<10)), p.bytes.tokens)
				require.Equal(t, p.bytes.tokens, g.pools[0].bytes.tokens)
			} else {
				require.Nil(t, cached)
			}
		})
	}
}
