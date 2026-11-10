package peerpolicy

import (
	"sync/atomic"
	"testing"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/blocksync"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/p2p/conn"
	bc "github.com/cometbft/cometbft/proto/tendermint/blocksync"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/state/mocks"
	"github.com/cometbft/cometbft/store"
	"github.com/cometbft/cometbft/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

type blockWireReceiver struct {
	*p2p.BaseReactor
	received  atomic.Int64
	responses chan int64
	deferred  chan int64
}

func (*blockWireReceiver) GetChannels() []*conn.ChannelDescriptor {
	return []*conn.ChannelDescriptor{{ID: blocksync.BlocksyncChannel, Priority: 1, MessageType: &bc.Message{}}}
}

func (r *blockWireReceiver) Receive(e p2p.Envelope) {
	if response, ok := e.Message.(*bc.BlockDeferredResponse); ok {
		r.deferred <- response.Height
	}
	if response, ok := e.Message.(*bc.BlockResponse); ok {
		r.received.Add(1)
		if r.responses != nil {
			r.responses <- response.Block.Header.Height
		}
	}
}

func blockServingNetwork(t *testing.T, policyCfg Config) (*Governor, *p2p.Switch, *p2p.Switch, *blockWireReceiver, *mocks.Store, *atomic.Int64) {
	t.Helper()
	clock := new(atomic.Int64)
	g, err := newGovernor(policyCfg, prometheus.NewRegistry(), func() time.Duration { return time.Duration(clock.Load()) })
	require.NoError(t, err)
	nativeCfg := cfg.DefaultP2PConfig()
	nativeCfg.ServingPolicy = g
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	blocks := store.NewBlockStore(db)
	for height := int64(1); height <= 100; height++ {
		block := types.MakeBlock(height, nil, &types.Commit{}, nil)
		block.ValidatorsHash = make([]byte, 32)
		block.ProposerAddress = make([]byte, 20)
		parts, err := block.MakePartSet(types.PartSizeBytes)
		require.NoError(t, err)
		blocks.SaveBlock(block, parts, &types.Commit{Height: height})
	}
	state := sm.State{InitialHeight: 1, LastBlockHeight: 100}
	stateStore := &mocks.Store{}
	stateStore.On("Load").Return(state, nil)
	executor := sm.NewBlockExecutor(stateStore, log.NewNopLogger(), nil, nil, nil, blocks)
	reactor := blocksync.NewReactor(state, executor, blocks, false, blocksync.NopMetrics(), 0)
	serving := servingSwitch(t, nativeCfg, reactor)
	receiving := &blockWireReceiver{responses: make(chan int64, 100), deferred: make(chan int64, 100)}
	receiving.BaseReactor = p2p.NewBaseReactor("receive", receiving)
	requester := servingSwitch(t, cfg.DefaultP2PConfig(), receiving)
	require.NoError(t, requester.DialPeerWithAddress(serving.NetAddress()))
	return g, serving, requester, receiving, stateStore, clock
}

func TestServingNativeBlockPipeline(t *testing.T) {
	g, serving, requester, receiving, stateStore, _ := blockServingNetwork(t, DefaultConfig())
	peer := requester.Peers().Get(serving.NodeInfo().ID())
	require.NotNil(t, peer)
	for height := int64(1); height <= 20; height++ {
		require.True(t, peer.Send(p2p.Envelope{ChannelID: blocksync.BlocksyncChannel, Message: &bc.BlockRequest{Height: height}}))
	}
	require.Eventually(t, func() bool { return receiving.received.Load() == 20 }, 5*time.Second, time.Millisecond)
	require.Eventually(t, func() bool {
		g.mu.Lock()
		defer g.mu.Unlock()
		return g.pools[0].memory == 0 && g.pools[0].inflight == 0
	}, time.Second, time.Millisecond)
	require.Zero(t, g.Snapshot(string(requester.NodeInfo().ID())).Risk)
	require.Equal(t, 1, serving.Peers().Size())
	stateStore.AssertNumberOfCalls(t, "Load", 20)
}

func TestServingNativeWorkBackpressure(t *testing.T) {
	for _, protected := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "protected"}[protected], func(t *testing.T) {
			testServingNativeWorkBackpressure(t, protected)
		})
	}
}

func testServingNativeWorkBackpressure(t *testing.T, protected bool) {
	t.Helper()
	policyCfg := DefaultConfig()
	policyCfg.RequestsPerSecond, policyCfg.PeerRequestsPerSecond = 192, 32
	g, serving, requester, receiving, _, clock := blockServingNetwork(t, policyCfg)
	id := string(requester.NodeInfo().ID())
	g.mu.Lock()
	g.protected[id] = protected
	g.peer(id, 0).work.tokens = 0
	g.mu.Unlock()
	peer := requester.Peers().Get(serving.NodeInfo().ID())
	require.NotNil(t, peer)
	send := func(height int64) {
		require.True(t, peer.Send(p2p.Envelope{ChannelID: blocksync.BlocksyncChannel, Message: &bc.BlockRequest{Height: height}}))
	}
	for height := int64(1); height <= 20; height++ {
		send(height)
	}
	require.Eventually(t, func() bool {
		return testutil.ToFloat64(g.events.WithLabelValues("peer_work_capacity")) > 0
	}, time.Second, time.Millisecond)
	clock.Add(int64(time.Second))
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	received := make(map[int64]bool)
	for len(received) < 100 {
		select {
		case height := <-receiving.responses:
			require.False(t, received[height], "duplicate response for %d", height)
			received[height] = true
			clock.Add(int64(time.Second))
			if next := int64(len(received)) + 20; next <= 100 {
				send(next)
			}
		case <-deadline.C:
			t.Fatalf("received only %d of 100 blocks", len(received))
		}
	}
	require.Positive(t, testutil.ToFloat64(g.events.WithLabelValues("peer_work_capacity")))
	require.Equal(t, int64(100), receiving.received.Load())
	require.Zero(t, g.Snapshot(id).Risk)
	require.Equal(t, 1, serving.Peers().Size())
}
