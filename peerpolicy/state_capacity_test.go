package peerpolicy

import (
	"fmt"
	"testing"
	"time"

	"github.com/cometbft/cometbft/blocksync"
	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/stretchr/testify/require"
)

func TestServingStateCapacityHeadroom(t *testing.T) {
	for _, protected := range []bool{false, true} {
		for _, size := range []uint64{1, 16_000_000} {
			t.Run(fmt.Sprintf("protected_%t/size_%d", protected, size), func(t *testing.T) {
				cfg := DefaultConfig()
				for id := 1; protected && id <= 129; id++ {
					cfg.ProtectedIDs = append(cfg.ProtectedIDs, fmt.Sprintf("%040x", id))
				}
				g, now := testGovernor(t, cfg)
				index := 0
				if protected {
					index = 2
				}
				pool := &g.pools[index]
				t.Cleanup(func() {
					require.Zero(t, pool.memory)
					require.Zero(t, pool.stateMemory)
					require.Zero(t, pool.stateInflight)
				})
				for id := 1; id <= 128; id++ {
					*now += 20 * time.Second
					l, ok := g.Admit(fmt.Sprintf("%040x", id), servebudget.Request{Family: servebudget.Chunk, MaxBytes: size})
					if ok {
						t.Cleanup(func() { l.Finish(false) })
						require.True(t, l.Prepare(size))
					}
				}
				require.Positive(t, pool.stateInflight)
				require.LessOrEqual(t, pool.stateMemory, pool.memoryCap/2)
				require.LessOrEqual(t, pool.stateInflight, pool.inflightCap/2)
				if size == 1 {
					require.Equal(t, pool.inflightCap/2, pool.stateInflight)
				} else {
					require.Greater(t, pool.stateMemory+2*size, pool.memoryCap/2)
				}
				*now += 20 * time.Second
				serve(t, g, fmt.Sprintf("%040x", 129), servebudget.Request{Family: servebudget.Block, MaxBytes: blocksync.MaxMsgSize}, blocksync.MaxMsgSize)
			})
		}
	}
}

func TestServingStateLeaseSettlement(t *testing.T) {
	for _, family := range []servebudget.Family{servebudget.Snapshot, servebudget.Chunk} {
		for _, mode := range []string{"unprepared", "rejected", "prepared", "written"} {
			t.Run(fmt.Sprintf("%d/%s", family, mode), func(t *testing.T) {
				g, _ := testGovernor(t, DefaultConfig())
				pool := &g.pools[0]
				l, ok := g.Admit(peerA, servebudget.Request{Family: family, MaxBytes: 100})
				require.True(t, ok)
				require.Equal(t, uint64(200), pool.stateMemory)
				require.Equal(t, uint64(1), pool.stateInflight)
				if mode == "rejected" {
					require.False(t, l.Prepare(101))
					require.Equal(t, uint64(200), pool.stateMemory)
				}
				if mode == "prepared" || mode == "written" {
					require.True(t, l.Prepare(10))
					require.Equal(t, uint64(20), pool.stateMemory)
					require.Equal(t, uint64(20), g.peers[peerA].memory[0])
				}
				l.Finish(mode == "written")
				l.Finish(false)
				require.Zero(t, pool.stateMemory)
				require.Zero(t, pool.stateInflight)
				require.Zero(t, pool.memory)
				require.Zero(t, g.peers[peerA].memory[0])
			})
		}
	}
}
