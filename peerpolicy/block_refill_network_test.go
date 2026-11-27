package peerpolicy

import (
	"testing"
	"time"

	"github.com/cometbft/cometbft/blocksync"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/p2p/servebudget"
	bc "github.com/cometbft/cometbft/proto/tendermint/blocksync"
	"github.com/stretchr/testify/require"
)

func TestServingNativeByteRefill(t *testing.T) {
	for _, protected := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "protected"}[protected], func(t *testing.T) {
			t.Parallel()
			g, server, client, receiver, stateStore, clock := blockServingNetwork(t, DefaultConfig())
			id := string(client.NodeInfo().ID())
			g.mu.Lock()
			g.protected[id] = protected
			g.mu.Unlock()
			serve(t, g, id, servebudget.Request{Family: servebudget.Block, Height: 101, MaxBytes: blocksync.MaxMsgSize}, blocksync.MaxMsgSize)
			peer := client.Peers().Get(server.NodeInfo().ID())
			require.NotNil(t, peer)
			require.True(t, peer.Send(p2p.Envelope{ChannelID: blocksync.BlocksyncChannel, Message: &bc.BlockRequest{Height: 1, AcceptDeferred: true}}))
			select {
			case height := <-receiver.deferred:
				require.Equal(t, int64(1), height)
			case <-time.After(time.Second):
				t.Fatal("requester was not told to wait for serving capacity")
			}
			stateStore.AssertNotCalled(t, "Load")
			clock.Store(int64(time.Minute))
			select {
			case height := <-receiver.responses:
				require.Equal(t, int64(1), height)
			case <-time.After(2 * time.Second):
				t.Fatal("queued request did not resume after byte refill")
			}
			require.Zero(t, g.Snapshot(id).Risk)
			require.Equal(t, 1, server.Peers().Size())
			stateStore.AssertNumberOfCalls(t, "Load", 1)
		})
	}
}
