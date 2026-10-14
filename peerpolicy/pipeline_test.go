package peerpolicy

import (
	"testing"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/stretchr/testify/require"
)

func TestServingBlockPipeline(t *testing.T) {
	g, _ := testGovernor(t, DefaultConfig())
	var pending []servebudget.Lease
	// Model one native blocksync pipeline while the transport batches its flush.
	// Reads reserve the protocol maximum; encoded small blocks must free that
	// headroom before completion, without receiving early completion credit.
	for i := range maxPeerInflight {
		l, ok := g.Admit(peerA, servebudget.Request{Family: servebudget.Block, Height: uint64(i + 1), MaxBytes: 104857604})
		require.True(t, ok, "request %d should fit the native pipeline", i+1)
		require.True(t, l.Prepare(2048))
		pending = append(pending, l)
		require.Equal(t, uint64((i+1)*4096), g.pools[0].memory)
		require.Zero(t, g.peers[peerA].objects[i].copies)
	}
	require.Equal(t, uint64(maxPeerInflight), g.pools[0].inflight)
	for _, l := range pending {
		l.Finish(true)
	}
	require.Zero(t, g.pools[0].memory)
	require.Zero(t, g.pools[0].inflight)
	for i := range maxPeerInflight {
		require.Equal(t, uint64(1), g.peers[peerA].objects[i].copies)
	}
}
