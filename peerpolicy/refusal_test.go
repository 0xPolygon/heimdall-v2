package peerpolicy

import (
	"fmt"
	"testing"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestServingPendingRepeatAdmissionIsNeutral(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RepeatBytes = 150
	g, _ := testGovernor(t, cfg)
	first := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 200}
	second := first
	second.Height = 2
	for range 2 {
		serve(t, g, peerA, first, 100)
		serve(t, g, peerA, second, 100)
	}
	pending, ok := g.Admit(peerA, first)
	require.True(t, ok)
	require.True(t, pending.Prepare(100))
	for range 3 {
		_, ok := g.Admit(peerA, second)
		require.False(t, ok)
	}
	require.Zero(t, g.Snapshot(peerA).Risk)
	require.Equal(t, float64(3), testutil.ToFloat64(g.events.WithLabelValues("repeat_reserved")))
	pending.Finish(false)
	require.Zero(t, g.Snapshot(peerA).Risk)
	require.Zero(t, g.peers[peerA].windows[0].repeatedBytes)
	serve(t, g, peerA, second, 100)
	require.Zero(t, g.Snapshot(peerA).Risk)
}

func TestServingPendingRepeatGrowthIsNeutral(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RepeatBytes = 300
	g, _ := testGovernor(t, cfg)
	first := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 400}
	second := first
	second.Height = 2
	for range 2 {
		serve(t, g, peerA, first, 100)
		serve(t, g, peerA, second, 100)
	}
	pending, ok := g.Admit(peerA, first)
	require.True(t, ok)
	require.True(t, pending.Prepare(200))
	for range 3 {
		growing, ok := g.Admit(peerA, second)
		require.True(t, ok)
		require.False(t, growing.Prepare(101))
		growing.Finish(false)
	}
	require.Zero(t, g.Snapshot(peerA).Risk)
	require.Equal(t, float64(3), testutil.ToFloat64(g.events.WithLabelValues("repeat_reserved")))
	pending.Finish(false)
	for range 3 {
		growing, ok := g.Admit(peerA, second)
		require.True(t, ok)
		require.False(t, growing.Prepare(301))
		growing.Finish(false)
	}
	require.Equal(t, uint64(20), g.Snapshot(peerA).Risk)
	require.Equal(t, float64(3), testutil.ToFloat64(g.events.WithLabelValues("repeat_allowance")))
	require.Zero(t, g.peers[peerA].repeatHeld)
}

func TestServingMixedRefusalsQualifyOnce(t *testing.T) {
	for _, order := range [][]bool{{true, true, false}, {false, false, true}, {true, false, true}} {
		for _, observe := range []bool{true, false} {
			t.Run(fmt.Sprint(order, observe), func(t *testing.T) {
				cfg := DefaultConfig()
				cfg.Observe, cfg.RepeatBytes = observe, 100
				g, _ := testGovernor(t, cfg)
				req := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 100}
				for range 3 {
					serve(t, g, peerA, req, 100)
				}
				for i, repeat := range append(order, order...) {
					g.peers[peerA].bytes.tokens = 0
					g.peers[peerA].completed.tokens = 0
					next := req
					next.Height = 2
					if repeat {
						g.peers[peerA].bytes.tokens = 100
						next = req
					}
					_, ok := g.Admit(peerA, next)
					require.False(t, ok)
					expected := uint64(0)
					if observe && i >= 2 {
						expected = 20
					}
					require.Equal(t, expected, g.Snapshot(peerA).Risk)
				}
				expected := float64(0)
				if observe {
					expected = 1
				}
				require.Equal(t, expected, testutil.ToFloat64(g.windows.WithLabelValues("mixed_allowance")))
				require.Zero(t, testutil.ToFloat64(g.windows.WithLabelValues("peer_allowance")))
				require.Zero(t, testutil.ToFloat64(g.windows.WithLabelValues("repeat_allowance")))
			})
		}
	}
}
