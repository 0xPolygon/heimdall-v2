package peerpolicy

import (
	"testing"
	"time"

	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/stretchr/testify/require"
)

func TestServingProtectedAllowanceMatchesOrdinary(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProtectedIDs = []string{peerB}
	g, now := testGovernor(t, cfg)
	for _, family := range []servebudget.Family{servebudget.Block, servebudget.Chunk} {
		for i := uint64(0); i < 2000; i++ {
			*now += 10 * time.Millisecond
			req := servebudget.Request{Family: family, Height: i + 1, MaxBytes: 1 << 20}
			ordinary, ok := g.Admit(peerA, req)
			protected, protectedOK := g.Admit(peerB, req)
			if ok {
				require.True(t, ordinary.Prepare(req.MaxBytes))
				ordinary.Finish(true)
				require.True(t, protectedOK, "protected peer lost service available to ordinary peer at request %d", i)
			}
			if protectedOK {
				require.True(t, protected.Prepare(req.MaxBytes))
				protected.Finish(true)
			}
		}
	}
}

func TestServingDefaultSmallBlockProgress(t *testing.T) {
	for _, protected := range []bool{false, true} {
		cfg := DefaultConfig()
		if protected {
			cfg.ProtectedIDs = []string{peerA}
		}
		g, now := testGovernor(t, cfg)
		for i := uint64(0); i < 3000; i++ {
			*now = time.Duration(i) * 10 * time.Millisecond
			req := servebudget.Request{Family: servebudget.Block, Height: i + 1, MaxBytes: 104857605}
			l, ok := g.Admit(peerA, req)
			require.True(t, ok, "100 small blocks/s should fit default work allowance (protected=%t height=%d)", protected, i+1)
			require.True(t, l.Prepare(2048))
			l.Finish(true)
		}
		require.Zero(t, g.Snapshot(peerA).Risk)
	}
}

func TestServingDefaultPeerRefill(t *testing.T) {
	for _, tc := range []struct {
		name      string
		protected bool
		catchup   bool
		mib       float64
		requests  float64
	}{
		{name: "ordinary", mib: 1.78, requests: 113.67},
		{name: "protected", protected: true, mib: 1.78, requests: 113.67},
		{name: "catchup", catchup: true, mib: 3.56, requests: 227.33},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			if tc.protected {
				cfg.ProtectedIDs = []string{peerA}
			}
			g, _ := testGovernor(t, cfg)
			p := g.peer(peerA, 0)
			bytes, work := &p.bytes, &p.work
			if tc.catchup {
				bytes, work = &p.catchupBytes, &p.catchupWork
			}
			bytes.tokens, work.tokens = 0, 0
			require.InDelta(t, tc.mib, bytes.available(time.Second)/(1<<20), 0.005)
			require.InDelta(t, tc.requests, work.available(time.Second), 0.005)
		})
	}
}
