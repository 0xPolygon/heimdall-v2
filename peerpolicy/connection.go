package peerpolicy

import "github.com/cometbft/cometbft/p2p/observation"

var _ observation.ConnectionPolicy = (*Tracker)(nil)

// AllowPeer shares the score ledger with Observe. Checking a connection neither
// refreshes evidence nor extends its expiry. Unknown and evicted IDs are neutral.
func (t *Tracker) AllowPeer(id string) bool {
	if !t.enforce {
		return true
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	elem := t.peers[id]
	if elem == nil {
		return true
	}
	return elem.Value.(*peerRecord).risk(int64(t.now()/windowWidth)) < 100
}
