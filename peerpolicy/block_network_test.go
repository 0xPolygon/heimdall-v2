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
	"github.com/stretchr/testify/require"
)

type blockWireReceiver struct {
	*p2p.BaseReactor
	received atomic.Int64
}

func (*blockWireReceiver) GetChannels() []*conn.ChannelDescriptor {
	return []*conn.ChannelDescriptor{{ID: blocksync.BlocksyncChannel, Priority: 1, MessageType: &bc.Message{}}}
}
func (r *blockWireReceiver) Receive(e p2p.Envelope) {
	if _, ok := e.Message.(*bc.BlockResponse); ok {
		r.received.Add(1)
	}
}

func TestServingNativeBlockPipeline(t *testing.T) {
	g, err := New(DefaultConfig(), prometheus.NewRegistry())
	require.NoError(t, err)
	nativeCfg := cfg.DefaultP2PConfig()
	nativeCfg.ServingPolicy = g
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	blocks := store.NewBlockStore(db)
	for height := int64(1); height <= 20; height++ {
		block := types.MakeBlock(height, nil, &types.Commit{}, nil)
		block.ValidatorsHash = make([]byte, 32)
		block.ProposerAddress = make([]byte, 20)
		parts, err := block.MakePartSet(types.PartSizeBytes)
		require.NoError(t, err)
		blocks.SaveBlock(block, parts, &types.Commit{Height: height})
	}
	state := sm.State{InitialHeight: 1, LastBlockHeight: 20}
	stateStore := &mocks.Store{}
	stateStore.On("Load").Return(state, nil)
	executor := sm.NewBlockExecutor(stateStore, log.NewNopLogger(), nil, nil, nil, blocks)
	reactor := blocksync.NewReactor(state, executor, blocks, false, blocksync.NopMetrics(), 0)
	serving := servingSwitch(t, nativeCfg, reactor)
	receiving := &blockWireReceiver{}
	receiving.BaseReactor = p2p.NewBaseReactor("receive", receiving)
	requester := servingSwitch(t, cfg.DefaultP2PConfig(), receiving)
	require.NoError(t, requester.DialPeerWithAddress(serving.NetAddress()))
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
