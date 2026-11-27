package peerpolicy

import (
	"fmt"
	"testing"
	"time"

	"github.com/cometbft/cometbft/blocksync"
	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/stretchr/testify/require"
)

func TestServingPeerRateHeadroom(t *testing.T) {
	for _, class := range []string{"ordinary", "protected", "catchup"} {
		for _, readers := range []int{1, 2} {
			t.Run(fmt.Sprintf("%s/%d", class, readers), func(t *testing.T) {
				cfg := DefaultConfig()
				third := fmt.Sprintf("%040x", 3)
				family := servebudget.Block
				if class == "protected" {
					cfg.ProtectedIDs = []string{peerA, peerB, third}
				} else if class == "catchup" {
					family = servebudget.Catchup
					cfg.ProtectedIDs = []string{third}
				}
				g, now := testGovernor(t, cfg)
				for step := uint64(0); step < 120000; step++ {
					*now = time.Duration(step) * time.Millisecond
					req := servebudget.Request{Family: family, Height: step + 1, MaxBytes: 1}
					for _, id := range []string{peerA, peerB}[:readers] {
						l, ok := g.Admit(id, req)
						if ok {
							require.True(t, l.Prepare(1))
							l.Finish(true)
						}
					}
					if step%1000 == 0 {
						serve(t, g, third, req, 1)
					}
				}
				require.Zero(t, g.Snapshot(third).Risk)
			})
		}
	}
}

func TestServingPeerByteBurstHeadroom(t *testing.T) {
	for _, class := range []string{"ordinary", "protected", "catchup"} {
		t.Run(class, func(t *testing.T) {
			cfg := DefaultConfig()
			third := fmt.Sprintf("%040x", 3)
			family := servebudget.Block
			if class == "protected" {
				cfg.ProtectedIDs = []string{peerA, peerB, third}
			} else if class == "catchup" {
				family = servebudget.Catchup
				cfg.ProtectedIDs = []string{third}
			}
			g, _ := testGovernor(t, cfg)
			for _, id := range []string{peerA, peerB} {
				for height := uint64(1); height <= 64; height++ {
					l, ok := g.Admit(id, servebudget.Request{Family: family, Height: height, MaxBytes: 8 << 20})
					if ok {
						require.True(t, l.Prepare(8<<20))
						l.Finish(true)
					}
				}
			}
			serve(t, g, third, servebudget.Request{Family: family, MaxBytes: blocksync.MaxMsgSize}, blocksync.MaxMsgSize)
			require.Zero(t, g.Snapshot(third).Risk)
		})
	}
}

func TestServingMinimumPeerRefill(t *testing.T) {
	for _, tc := range []struct {
		name      string
		family    servebudget.Family
		protected bool
		bytes     uint64
	}{
		{"ordinary_block", servebudget.Block, false, 2},
		{"ordinary_state", servebudget.Chunk, false, 2},
		{"protected_block", servebudget.Block, true, 2},
		{"protected_state", servebudget.Chunk, true, 2},
		{"catchup", servebudget.Catchup, false, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.BytesPerSecond, cfg.RequestsPerSecond, cfg.BurstBytes = 12, 12, 36
			cfg.MaxInflight = 24
			if tc.protected {
				cfg.ProtectedIDs = []string{peerA}
			}
			g, now := testGovernor(t, cfg)
			req := servebudget.Request{Family: tc.family, MaxBytes: tc.bytes}
			serve(t, g, peerA, req, tc.bytes)
			req.Height++
			for range 3 {
				_, ok := g.Admit(peerA, req)
				require.False(t, ok)
			}
			require.Zero(t, g.Snapshot(peerA).Risk)
			*now = 2999 * time.Millisecond
			_, ok := g.Admit(peerA, req)
			require.False(t, ok)
			*now = 3*time.Second + time.Nanosecond
			serve(t, g, peerA, req, tc.bytes)
			require.Zero(t, g.Snapshot(peerA).Risk)
		})
	}
}
