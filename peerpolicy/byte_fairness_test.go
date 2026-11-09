package peerpolicy

import (
	"fmt"
	"testing"
	"time"

	"github.com/cometbft/cometbft/blocksync"
	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/stretchr/testify/require"
)

func TestServingPeerByteRefillHeadroom(t *testing.T) {
	for _, class := range []string{"ordinary", "protected", "catchup"} {
		for _, written := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/written_%t", class, written), func(t *testing.T) {
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
				for step := uint64(0); step < 30000; step++ {
					*now = time.Duration(step) * 10 * time.Millisecond
					for _, id := range []string{peerA, peerB} {
						l, ok := g.Admit(id, servebudget.Request{Family: family, Height: step + 1, MaxBytes: 1 << 20})
						if ok {
							require.True(t, l.Prepare(1<<20))
							l.Finish(written)
						}
					}
					if step%10000 == 0 {
						serve(t, g, third, servebudget.Request{Family: family, Height: step + 1, MaxBytes: blocksync.MaxMsgSize}, blocksync.MaxMsgSize)
					}
				}
				require.Zero(t, g.Snapshot(third).Risk)
			})
		}
	}
}
