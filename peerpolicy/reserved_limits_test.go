package peerpolicy

import (
	"fmt"
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestServingReservedPoolLimits(t *testing.T) {
	for _, class := range []string{"ordinary", "catchup", "protected"} {
		for _, resource := range []string{"bytes", "work", "memory", "inflight"} {
			t.Run(class+"/"+resource, func(t *testing.T) {
				cfg := DefaultConfig()
				cfg.BytesPerSecond, cfg.BurstBytes = 16, 16
				cfg.RequestsPerSecond = 8
				cfg.MemoryBytes = 80
				cfg.MaxInflight = 8
				family := servebudget.Block
				limit := 2
				if class != "ordinary" {
					limit = 1
				}
				if class == "catchup" {
					family = servebudget.Catchup
				}
				if class == "protected" {
					cfg.ProtectedIDs = []string{peerA}
				}
				reason := "local_capacity"
				maxBytes := uint64(4)
				switch resource {
				case "bytes":
					reason = "local_bytes"
				case "work":
					limit *= 2
					cfg.BurstBytes = 1024
				case "memory":
					maxBytes = 10
					cfg.BurstBytes = 1024
				case "inflight":
					limit *= 2
					cfg.MemoryBytes = 1024
					cfg.RequestsPerSecond = 128
					cfg.BurstBytes = 1024
				}
				g, now := testGovernor(t, cfg)
				var pending []servebudget.Lease
				for i := 0; i < limit; i++ {
					l, ok := g.Admit(peerA, servebudget.Request{Family: family, Height: uint64(i + 1), MaxBytes: maxBytes})
					require.True(t, ok)
					if resource == "bytes" || resource == "work" {
						require.True(t, l.Prepare(maxBytes))
						l.Finish(true)
					} else {
						pending = append(pending, l)
					}
				}
				req := servebudget.Request{Family: family, Height: 100, MaxBytes: maxBytes}
				_, ok := g.Admit(peerA, req)
				require.False(t, ok, "reserved class must stop at its own capacity")
				require.Equal(t, float64(1), testutil.ToFloat64(g.events.WithLabelValues(reason)))
				require.Zero(t, g.Snapshot(peerA).Risk)
				if len(pending) > 0 {
					pending[0].Finish(false)
					serve(t, g, peerA, req, maxBytes)
					for _, l := range pending {
						l.Finish(false)
					}
					return
				}
				refill := time.Second / time.Duration(limit)
				*now = refill / 2
				_, ok = g.Admit(peerA, req)
				require.False(t, ok, "partial refill cannot fund another request")
				*now = refill
				serve(t, g, peerA, req, maxBytes)
				require.Zero(t, g.Snapshot(peerA).Risk, fmt.Sprintf("%s congestion stays neutral", class))
			})
		}
	}
}
