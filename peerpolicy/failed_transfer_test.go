package peerpolicy

import (
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestServingFailedTransfersStayNeutral(t *testing.T) {
	for name, family := range map[string]servebudget.Family{"block": servebudget.Block, "chunk": servebudget.Chunk, "snapshot": servebudget.Snapshot, "catchup": servebudget.Catchup} {
		t.Run(name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.PeerBytesPerSecond, cfg.PeerBurstBytes = 1<<20, 1<<20
			g, _ := testGovernor(t, cfg)
			req := servebudget.Request{Family: family, MaxBytes: 1 << 20}
			l, ok := g.Admit(peerA, req)
			require.True(t, ok)
			require.True(t, l.Prepare(req.MaxBytes))
			l.Finish(false)
			p := g.peers[peerA]
			bytes, _, held := p.allowance(family)
			require.Zero(t, bytes.tokens, "partial sends still consume their traffic allowance")
			require.Zero(t, *held)
			require.Zero(t, p.objects[req].copies)
			for range 3 {
				_, ok = g.Admit(peerA, req)
				require.False(t, ok)
			}
			require.Zero(t, g.Snapshot(peerA).Risk, "failed transfers are not completed-serving evidence")
			reason := "peer_capacity"
			if family == servebudget.Catchup {
				reason = "catchup_capacity"
			}
			require.Equal(t, float64(3), testutil.ToFloat64(g.events.WithLabelValues(reason)))
		})
	}
}

func TestServingMixedTransferEvidence(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PeerBytesPerSecond, cfg.PeerBurstBytes = 100, 1000
	g, now := testGovernor(t, cfg)
	serve(t, g, peerA, servebudget.Request{Family: servebudget.Block, MaxBytes: 400}, 400)
	req := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 200}
	l, ok := g.Admit(peerA, req)
	require.True(t, ok)
	require.True(t, l.Prepare(200))
	l.Finish(false)
	*now += time.Second
	req.Height++
	req.MaxBytes = 700
	for range 3 {
		_, ok = g.Admit(peerA, req)
		require.False(t, ok)
	}
	require.Zero(t, g.Snapshot(peerA).Risk, "successful serving still leaves exactly 700 bytes after refill")
	req.MaxBytes++
	for range 3 {
		_, ok = g.Admit(peerA, req)
		require.False(t, ok)
	}
	require.Equal(t, uint64(20), g.Snapshot(peerA).Risk, "successful serving alone now explains the shortfall")
}

func TestServingFailedTransferPrepareGrowth(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PeerBytesPerSecond, cfg.PeerBurstBytes = 100, 1000
	g, _ := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Block, MaxBytes: 500}
	serve(t, g, peerA, req, 10)
	failed, ok := g.Admit(peerA, servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 800})
	require.True(t, ok)
	require.True(t, failed.Prepare(800))
	failed.Finish(false)
	for range 3 {
		l, ok := g.Admit(peerA, req)
		require.True(t, ok)
		require.False(t, l.Prepare(300))
		l.Finish(false)
	}
	require.Zero(t, g.Snapshot(peerA).Risk)
	require.Equal(t, float64(190), g.peers[peerA].bytes.tokens)
}

func TestServingCompletionBurstRecovers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PeerBytesPerSecond, cfg.PeerBurstBytes = 100, 1000
	g, now := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Block, MaxBytes: 1000}
	first, ok := g.Admit(peerA, req)
	require.True(t, ok)
	require.True(t, first.Prepare(1000))
	*now += 10 * time.Second
	req.Height++
	second, ok := g.Admit(peerA, req)
	require.True(t, ok)
	require.True(t, second.Prepare(1000))
	first.Finish(true)
	first.Finish(true)
	second.Finish(true)
	require.Zero(t, g.peers[peerA].completed.tokens, "queued completions cannot create unbounded scoring debt")
	*now += 10 * time.Second
	require.Equal(t, float64(1000), g.peers[peerA].completed.available(*now))
}
