package peerpolicy

import (
	"testing"
	"time"

	"github.com/cometbft/cometbft/blocksync"
	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/stretchr/testify/require"
)

func TestServingLargeResponseRetainsBandwidthCap(t *testing.T) {
	for _, protected := range []bool{false, true} {
		cfg := DefaultConfig()
		if protected {
			cfg.ProtectedIDs = []string{peerA}
		}
		g, now := testGovernor(t, cfg)
		req := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: blocksync.MaxMsgSize}
		serve(t, g, peerA, req, blocksync.MaxMsgSize)
		req.Height++
		*now = 14 * time.Second
		l, ok := g.Retry(peerA, req)
		require.False(t, ok)
		require.Nil(t, l)
		*now = time.Minute
		l, ok = g.Retry(peerA, req)
		require.True(t, ok, "bounded requester deferral must cover byte refill (protected=%t)", protected)
		l.Finish(false)
	}
}
