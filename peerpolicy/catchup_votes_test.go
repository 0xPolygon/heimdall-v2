package peerpolicy

import (
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/stretchr/testify/require"
)

// Comet admits every catch-up vote of a height under one request, before it
// knows which vote it will send, and the writer finishes the lease only after
// the frame is flushed. The next vote must not wait for that flush.
func TestServingCatchupVotesDoNotWaitForFlush(t *testing.T) {
	for _, protected := range []bool{false, true} {
		cfg := DefaultConfig()
		if protected {
			cfg.ProtectedIDs = []string{peerA}
		}
		g, now := testGovernor(t, cfg)
		for height := uint64(1); height <= 20; height++ {
			req := servebudget.Request{Family: servebudget.Catchup, Height: height, Format: 1, MaxBytes: 104857600}
			var inFlight servebudget.Lease
			for vote := 0; vote < 100; vote++ {
				// 200 votes/s stays within the default per-peer catch-up rate.
				*now += 5 * time.Millisecond
				l, ok := g.Admit(peerA, req)
				require.True(t, ok, "vote %d at height %d refused while the previous one was still flushing (protected=%t)", vote, height, protected)
				require.True(t, l.Prepare(400))
				if inFlight != nil {
					inFlight.Finish(true)
				}
				inFlight = l
			}
			inFlight.Finish(true)
		}
		require.Zero(t, g.Snapshot(peerA).Risk)
	}
}

func TestServingBulkObjectStillSerialisedWhilePending(t *testing.T) {
	g, _ := testGovernor(t, DefaultConfig())
	req := servebudget.Request{Family: servebudget.Block, Height: 1, MaxBytes: 1 << 20}
	l, ok := g.Admit(peerA, req)
	require.True(t, ok)
	require.True(t, l.Prepare(1024))
	_, ok = g.Admit(peerA, req)
	require.False(t, ok)
	l.Finish(true)
}
