package peerpolicy

import (
	"fmt"
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestServingSustainedWorkBackpressure(t *testing.T) {
	g, now := testGovernor(t, DefaultConfig())
	admitted, refused := 0, 0
	for i := 0; i < 5000; i++ {
		*now = time.Duration(i) * 10 * time.Millisecond
		l, ok := g.Admit(peerA, servebudget.Request{Family: servebudget.Block, Height: uint64(i + 1), MaxBytes: 104857604})
		if !ok {
			refused++
			continue
		}
		admitted++
		require.True(t, l.Prepare(2048))
		l.Finish(true)
	}
	require.Greater(t, admitted, 1500)
	require.LessOrEqual(t, admitted, 32+50*32)
	require.Positive(t, refused)
	require.Equal(t, float64(refused), testutil.ToFloat64(g.events.WithLabelValues("peer_work_capacity")))
	require.Zero(t, g.Snapshot(peerA).Risk)
	require.Equal(t, "none", g.Snapshot(peerA).WouldAction)
}

func TestServingWorkBackpressurePreservesEvidence(t *testing.T) {
	for _, repeat := range []bool{false, true} {
		cfg := DefaultConfig()
		cfg.RepeatBytes = 100
		g, _ := testGovernor(t, cfg)
		req := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 100}
		for range 3 {
			serve(t, g, peerA, req, 100)
		}
		p := g.peers[peerA]
		p.work.tokens = 0
		reason := "repeat_allowance"
		if !repeat {
			req.Height++
			p.bytes.tokens = 0
			reason = "peer_allowance"
		}
		for range 3 {
			_, ok := g.Admit(peerA, req)
			require.False(t, ok)
		}
		require.Equal(t, uint64(20), g.Snapshot(peerA).Risk)
		require.Equal(t, float64(3), testutil.ToFloat64(g.events.WithLabelValues(reason)))
	}
}

func TestServingConfiguredHistoryThroughput(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PeerRequestsPerSecond = 128
	cfg.RequestsPerSecond = 512
	g, now := testGovernor(t, cfg)
	for i := uint64(0); i < 8000; i++ {
		*now = time.Duration(i) * 8 * time.Millisecond
		for _, family := range []servebudget.Family{servebudget.Block, servebudget.Catchup} {
			serve(t, g, peerA, servebudget.Request{Family: family, Height: i + 1, MaxBytes: 104857604}, 2048)
		}
	}
	require.Greater(t, len(g.peers[peerA].objects), defaultPeerHistory)
	require.LessOrEqual(t, uint64(len(g.peers[peerA].objects)), g.objectLimit)
	require.Zero(t, testutil.ToFloat64(g.events.WithLabelValues("history_capacity")))
	require.Zero(t, g.Snapshot(peerA).Risk)
}

func TestServingHistoryConfigurationBounds(t *testing.T) {
	for _, peer := range []bool{false, true} {
		cfg := DefaultConfig()
		limit := (uint64(maxNodeHistory) - cfg.MaxInflight) / 61
		value := &cfg.RequestsPerSecond
		if peer {
			limit = (maxPeerHistory - 2*maxPeerInflight) / (2 * 61)
			value = &cfg.PeerRequestsPerSecond
		}
		*value = limit
		require.NoError(t, cfg.Validate())
		*value = limit + 1
		require.ErrorContains(t, cfg.Validate(), "history capacity")
	}
	cfg := DefaultConfig()
	objects, history := cfg.historyLimits()
	require.Equal(t, uint64(defaultPeerHistory), objects)
	require.Equal(t, uint64(defaultNodeHistory), history)
	cfg.RequestsPerSecond = 2048
	_, history = cfg.historyLimits()
	require.Equal(t, uint64(2048*61)+cfg.MaxInflight, history)
	cfg.MaxInflight = 11
	require.Error(t, cfg.Validate())
}

func TestServingConfiguredNodeHistoryThroughput(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RequestsPerSecond = 2048
	g, now := testGovernor(t, cfg)
	for second := uint64(0); second < 60; second++ {
		*now = time.Duration(second) * time.Second
		for peer := 0; peer < 32; peer++ {
			id := fmt.Sprintf("%040x", peer+1)
			for offset := uint64(0); offset < 32; offset++ {
				req := servebudget.Request{Family: servebudget.Block, Height: second*32 + offset + 1, MaxBytes: 10}
				serve(t, g, id, req, 1)
				if peer < 16 {
					req.Family = servebudget.Catchup
					serve(t, g, id, req, 1)
				}
			}
		}
	}
	require.Greater(t, g.objects, uint64(defaultNodeHistory))
	require.LessOrEqual(t, g.objects, g.historyLimit)
	require.Zero(t, testutil.ToFloat64(g.events.WithLabelValues("history_capacity")))
}
