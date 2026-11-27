package peerpolicy

import (
	"fmt"
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestServingStatePeerHeadroom(t *testing.T) {
	for _, tc := range []struct {
		protected bool
		inflight  uint64
	}{{false, 120}, {false, 240}, {true, 120}, {true, 240}} {
		for _, bytes := range []uint64{1, 16_000_000} {
			t.Run(fmt.Sprintf("protected_%t/inflight_%d/bytes_%d", tc.protected, tc.inflight, bytes), func(t *testing.T) {
				cfg := DefaultConfig()
				cfg.MaxInflight = tc.inflight
				third := fmt.Sprintf("%040x", 3)
				if tc.protected {
					cfg.ProtectedIDs = []string{peerA, peerB, third}
				}
				g, now := testGovernor(t, cfg)
				for height := uint64(1); height <= maxPeerInflight; height++ {
					for _, id := range []string{peerA, peerB} {
						*now += time.Minute
						l, ok := g.Admit(id, servebudget.Request{Family: servebudget.Chunk, Height: height, MaxBytes: bytes})
						if ok {
							t.Cleanup(func() { l.Finish(false) })
							require.True(t, l.Prepare(bytes))
						}
					}
				}
				serve(t, g, third, servebudget.Request{Family: servebudget.Snapshot, MaxBytes: 40_000_000}, 40_000_000)
				require.Zero(t, g.Snapshot(third).Risk)
			})
		}
	}
}

func TestServingUnfundableResponseIsNeutral(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PeerBurstBytes = 100
	g, _ := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Block, MaxBytes: 200}
	for range 3 {
		_, ok := g.Admit(peerA, req)
		require.False(t, ok)
	}
	require.Zero(t, g.Snapshot(peerA).Risk)
	require.Equal(t, float64(3), testutil.ToFloat64(g.events.WithLabelValues("peer_capacity")))
	g, _ = testGovernor(t, cfg)
	serve(t, g, peerA, servebudget.Request{Family: servebudget.Block, MaxBytes: 100}, 100)
	for range 3 {
		_, ok := g.Admit(peerA, req)
		require.False(t, ok)
	}
	require.Zero(t, g.Snapshot(peerA).Risk)
}

func TestServingMixedFamilyCapacityIsNeutral(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PeerBurstBytes = 256 << 20
	g, _ := testGovernor(t, cfg)
	for height := uint64(1); height <= 3; height++ {
		l, ok := g.Admit(peerA, servebudget.Request{Family: servebudget.Block, Height: height, MaxBytes: 48 << 20})
		require.True(t, ok)
		require.True(t, l.Prepare(48<<20))
		t.Cleanup(func() { l.Finish(false) })
	}
	req := servebudget.Request{Family: servebudget.Chunk, MaxBytes: 16_000_000}
	for range 3 {
		_, ok := g.Admit(peerA, req)
		require.False(t, ok)
	}
	require.Zero(t, g.Snapshot(peerA).Risk)
	serve(t, g, peerB, req, req.MaxBytes)
	require.Zero(t, g.Snapshot(peerB).Risk)
}
