package peerpolicy

import (
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestServingRetryDoesNotAddRefusalEvidence(t *testing.T) {
	for _, repeated := range []bool{false, true} {
		t.Run(map[bool]string{false: "bytes", true: "repeat"}[repeated], func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.PeerBurstBytes, cfg.PeerBytesPerSecond, cfg.RepeatBytes = 300, 300, 1
			g, now := testGovernor(t, cfg)
			req := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 100}
			serve(t, g, peerA, req, 100)
			serve(t, g, peerA, req, 100)
			reason := "repeat_allowance"
			if !repeated {
				req.Height++
				serve(t, g, peerA, req, 100)
				req.Height++
				reason = "peer_allowance"
			}
			lease, ok := g.Admit(peerA, req)
			require.False(t, ok)
			require.Nil(t, lease)
			for range 100 {
				lease, ok = g.Retry(peerA, req)
				require.False(t, ok)
				require.Nil(t, lease)
			}
			require.Zero(t, g.Snapshot(peerA).Risk)
			require.Equal(t, float64(1), testutil.ToFloat64(g.events.WithLabelValues(reason)))
			for range 2 {
				_, ok = g.Admit(peerA, req)
				require.False(t, ok)
			}
			require.Equal(t, uint64(20), g.Snapshot(peerA).Risk)
			if !repeated {
				*now += time.Second
				lease, ok = g.Retry(peerA, req)
				require.True(t, ok)
				require.True(t, lease.Prepare(100))
				lease.Finish(true)
				require.Equal(t, uint64(20), g.Snapshot(peerA).Risk)
			}
		})
	}
}

func TestServingRetryPrepareDoesNotAddEvidence(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RepeatBytes = 150
	g, _ := testGovernor(t, cfg)
	req := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 200}
	serve(t, g, peerA, req, 100)
	serve(t, g, peerA, req, 100)
	for range 4 {
		lease, ok := g.Retry(peerA, req)
		require.True(t, ok)
		require.False(t, lease.Prepare(200))
		lease.Finish(false)
	}
	require.Zero(t, g.Snapshot(peerA).Risk)
}
