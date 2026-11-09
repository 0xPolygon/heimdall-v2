package peerpolicy

import (
	"fmt"
	"testing"
	"time"

	"github.com/cometbft/cometbft/blocksync"
	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestServingPeerMemoryHeadroom(t *testing.T) {
	for _, tc := range []struct {
		name   string
		family servebudget.Family
		index  int
	}{{"ordinary", servebudget.Block, 0}, {"protected", servebudget.Block, 2}, {"catchup", servebudget.Catchup, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			third := fmt.Sprintf("%040x", 3)
			if tc.index == 2 {
				cfg.ProtectedIDs = []string{peerA, peerB, third}
			}
			family := tc.family
			g, now := testGovernor(t, cfg)
			for height := uint64(1); height <= maxPeerInflight; height++ {
				for _, id := range []string{peerA, peerB} {
					*now += 20 * time.Second
					l, ok := g.Admit(id, servebudget.Request{Family: family, Height: height, MaxBytes: blocksync.MaxMsgSize})
					if ok {
						t.Cleanup(func() { l.Finish(false) })
						require.True(t, l.Prepare(48<<20))
					}
				}
			}
			*now += 20 * time.Second
			for _, id := range []string{peerA, peerB} {
				memory := *g.peers[id].reservedMemory(family)
				require.Positive(t, memory)
				require.LessOrEqual(t, memory, g.pools[tc.index].memoryCap/3)
			}
			require.Positive(t, testutil.ToFloat64(g.events.WithLabelValues("peer_memory_capacity")))
			l, ok := g.Admit(third, servebudget.Request{Family: family, Height: 100, MaxBytes: blocksync.MaxMsgSize})
			require.True(t, ok, "two slow readers must leave room for a maximum response")
			require.True(t, l.Prepare(blocksync.MaxMsgSize))
			l.Finish(true)
			require.Zero(t, *g.peers[third].reservedMemory(family))
			for _, id := range []string{peerA, peerB, third} {
				require.Zero(t, g.Snapshot(id).Risk)
			}
		})
	}
}

func TestServingFamilyWorkTotals(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RequestsPerSecond = 8
	cfg.MaxInflight = 24
	g, now := testGovernor(t, cfg)
	third := fmt.Sprintf("%040x", 3)
	req := servebudget.Request{Family: servebudget.Block, MaxBytes: 10}
	for _, id := range []string{peerA, peerB, third} {
		for range 4 {
			serve(t, g, id, req, 1)
		}
	}
	req.Family = servebudget.Chunk
	_, ok := g.Admit(third, req)
	require.False(t, ok, "family bursts still share the node work limit")
	*now = time.Second
	for range 2 {
		serve(t, g, fmt.Sprintf("%040x", 4), req, 1)
	}
	req.Family = servebudget.Block
	for range 2 {
		serve(t, g, fmt.Sprintf("%040x", 5), req, 1)
	}
	req.Family = servebudget.Chunk
	_, ok = g.Admit(peerB, req)
	require.False(t, ok, "the combined family refill must stay within the node rate")
	require.Equal(t, float64(16), testutil.ToFloat64(g.events.WithLabelValues("admitted")))
}

func TestServingConcurrentSyncFamilies(t *testing.T) {
	for _, bytes := range []uint64{16_000_000, 1024} {
		for _, protected := range []bool{false, true} {
			t.Run(fmt.Sprintf("bytes_%d/protected_%t", bytes, protected), func(t *testing.T) {
				cfg := DefaultConfig()
				third := fmt.Sprintf("%040x", 3)
				if protected {
					cfg.ProtectedIDs = []string{peerA, peerB, third}
				}
				g, now := testGovernor(t, cfg)
				for step := uint64(0); step < 6000; step++ {
					*now = time.Duration(step) * 10 * time.Millisecond
					for _, id := range []string{peerA, peerB} {
						l, ok := g.Admit(id, servebudget.Request{Family: servebudget.Chunk, Height: step + 1, MaxBytes: 16_000_000})
						if ok {
							require.True(t, l.Prepare(bytes))
							l.Finish(true)
						}
					}
					if step >= 200 && step%100 == 0 {
						serve(t, g, third, servebudget.Request{Family: servebudget.Block, Height: step + 1, MaxBytes: blocksync.MaxMsgSize}, 2048)
					}
				}
				require.Zero(t, g.Snapshot(third).Risk)
			})
		}
	}
}
