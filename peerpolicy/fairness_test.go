package peerpolicy

import (
	"fmt"
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/stretchr/testify/require"
)

func holdResponses(t *testing.T, g *Governor, id string, family servebudget.Family, n uint64) {
	t.Helper()
	for i := uint64(0); i < n; i++ {
		l, ok := g.Admit(id, servebudget.Request{Family: family, Height: i + 1, MaxBytes: 104857604})
		require.True(t, ok)
		require.True(t, l.Prepare(2048))
		t.Cleanup(func() { l.Finish(false) })
	}
}

func TestServingPerPeerHeadroom(t *testing.T) {
	for _, total := range []uint64{24, 80, 240, 4096} {
		for _, class := range []string{"bulk", "catchup", "protected"} {
			t.Run(fmt.Sprint(total, "/", class), func(t *testing.T) {
				cfg := DefaultConfig()
				cfg.MaxInflight = total
				family, index := servebudget.Block, 0
				if class == "catchup" {
					family, index = servebudget.Catchup, 1
				}
				if class == "protected" {
					index = 2
					cfg.ProtectedIDs = []string{peerA, peerB}
				}
				g, now := testGovernor(t, cfg)
				limit := min(uint64(maxPeerInflight), g.pools[index].inflightCap/3)
				for _, id := range []string{peerA, peerB} {
					holdResponses(t, g, id, family, limit)
					*now += time.Second
					_, ok := g.Admit(id, servebudget.Request{Family: family, Height: 100, MaxBytes: 104857604})
					require.False(t, ok)
					require.Zero(t, g.Snapshot(id).Risk)
				}
				third := fmt.Sprintf("%040x", 3)
				if class == "protected" {
					g.protected[third] = true
				}
				l, ok := g.Admit(third, servebudget.Request{Family: family, Height: 100, MaxBytes: 104857604})
				require.True(t, ok, "two peers must leave in-flight headroom")
				l.Finish(false)
			})
		}
	}
}

func TestServingProtectedCatchupHeadroom(t *testing.T) {
	for _, resource := range []string{"inflight", "work", "bytes", "memory"} {
		t.Run(resource, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.ProtectedIDs = []string{peerA, peerB}
			g, _ := testGovernor(t, cfg)
			holdResponses(t, g, peerA, servebudget.Block, 20)
			pool := &g.pools[2]
			switch resource {
			case "inflight":
				pool.inflight = pool.inflightCap
			case "work":
				pool.work.tokens = 0
			case "bytes":
				pool.bytes.tokens = 0
			case "memory":
				pool.memory = pool.memoryCap
			}
			for _, id := range []string{peerA, peerB} {
				l, ok := g.Admit(id, servebudget.Request{Family: servebudget.Catchup, Height: 100, MaxBytes: 104857604})
				require.True(t, ok, "protected bulk must not consume catch-up capacity")
				require.Same(t, &g.pools[1], l.(*lease).pool)
				l.Finish(false)
				require.Zero(t, g.peers[id].catchupInflight)
				require.Zero(t, g.Snapshot(id).Risk)
			}
		})
	}
}

func TestServingPeerFamilyConcurrency(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProtectedIDs = []string{peerA}
	g, _ := testGovernor(t, cfg)
	t.Cleanup(func() {
		require.Zero(t, g.peers[peerA].inflight)
		require.Zero(t, g.peers[peerA].catchupInflight)
	})
	holdResponses(t, g, peerA, servebudget.Catchup, 20)
	holdResponses(t, g, peerA, servebudget.Block, 20)
	require.Equal(t, uint64(40), g.peers[peerA].inflight)
	require.Equal(t, uint64(20), g.peers[peerA].catchupInflight)
	for _, family := range []servebudget.Family{servebudget.Block, servebudget.Catchup} {
		_, ok := g.Admit(peerA, servebudget.Request{Family: family, Height: 100, MaxBytes: 104857604})
		require.False(t, ok)
	}
	require.Zero(t, g.Snapshot(peerA).Risk)
}
