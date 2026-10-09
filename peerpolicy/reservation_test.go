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
