package peerpolicy

import (
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/observation"
	bc "github.com/cometbft/cometbft/proto/tendermint/blocksync"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestPeerPolicyConnectionThreshold(t *testing.T) {
	t.Parallel()
	for _, enforce := range []bool{false, true} {
		t.Run(map[bool]string{false: "observe", true: "enforce"}[enforce], func(t *testing.T) {
			tr, now, _ := fixture(t)
			tr.enforce = enforce
			require.True(t, tr.AllowPeer("unknown"))
			for tick := 0; tick < 5; tick++ {
				*now = windowWidth * time.Duration(tick)
				tr.Observe("peer", observation.Event{Kind: observation.Received, Message: &bc.BlockRequest{Height: 1}, Bytes: 2 << 20})
				require.Equal(t, uint64((tick+1)*20), tr.Snapshot("peer").Risk)
				require.Equal(t, !enforce || tick < 4, tr.AllowPeer("peer"))
				require.Equal(t, tr.AllowPeer("peer"), tr.Snapshot("peer").ConnectionAllowed)
			}
			// Repeated checks must not refresh evidence or create new records.
			for i := 0; i < 10; i++ {
				tr.AllowPeer("peer")
				tr.AllowPeer("unknown")
			}
			require.Len(t, tr.peers, 1)
			*now = 6 * windowWidth
			require.Equal(t, uint64(80), tr.Snapshot("peer").Risk)
			require.True(t, tr.AllowPeer("peer"))
			reset, err := New(prometheus.NewRegistry(), enforce)
			require.NoError(t, err)
			require.True(t, reset.AllowPeer("peer"))
		})
	}
}

func TestPeerPolicyConnectionMode(t *testing.T) {
	t.Parallel()
	for _, enforce := range []bool{false, true} {
		mode := map[bool]string{false: "observe", true: "enforce"}[enforce]
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			reg := prometheus.NewRegistry()
			tracker, err := New(reg, enforce)
			require.NoError(t, err)
			snapshot := tracker.Snapshot("unknown")
			require.Equal(t, mode, snapshot.Mode)
			require.True(t, snapshot.ConnectionAllowed)
			families, err := reg.Gather()
			require.NoError(t, err)
			var found bool
			for _, family := range families {
				if family.GetName() == "heimdall_peer_reputation_evictions_total" {
					found = true
					require.Equal(t, mode, family.Metric[0].Label[0].GetValue())
				}
			}
			require.True(t, found)
		})
	}
}
