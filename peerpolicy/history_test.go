package peerpolicy

import (
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/stretchr/testify/require"
)

func TestServingRetainsLiveHistory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RequestsPerSecond = 2 * (maxObjects + 4)
	cfg.PeerRequestsPerSecond = maxObjects + 4
	cfg.RepeatBytes = 1
	g, now := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 10}
	serve(t, g, peerA, req, 10)
	serve(t, g, peerA, req, 10)
	for height := uint64(2); height <= maxObjects; height++ {
		next := req
		next.Height = height
		serve(t, g, peerA, next, 10)
	}
	next := req
	next.Height = maxObjects + 1
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
	require.Equal(t, uint64(2), g.peers[peerA].objects[0].copies)
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
