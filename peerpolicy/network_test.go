package peerpolicy

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/p2p/conn"
	ss "github.com/cometbft/cometbft/proto/tendermint/statesync"
	"github.com/cometbft/cometbft/proxy/mocks"
	"github.com/cometbft/cometbft/statesync"
	"github.com/cometbft/cometbft/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type wireReceiver struct {
	*p2p.BaseReactor
	received atomic.Int64
}

func (r *wireReceiver) GetChannels() []*conn.ChannelDescriptor {
	return []*conn.ChannelDescriptor{{ID: statesync.ChunkChannel, Priority: 1, MessageType: &ss.Message{}}, {ID: statesync.SnapshotChannel, Priority: 1, MessageType: &ss.Message{}}}
}
func (r *wireReceiver) Receive(p2p.Envelope) { r.received.Add(1) }

func servingSwitch(t *testing.T, config *cfg.P2PConfig, r p2p.Reactor) *p2p.Switch {
	t.Helper()
	key := p2p.NodeKey{PrivKey: ed25519.GenPrivKey()}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := p2p.NewNetAddress(key.ID(), listener.Addr())
	require.NoError(t, listener.Close())
	info := p2p.DefaultNodeInfo{ProtocolVersion: p2p.ProtocolVersion{P2P: version.P2PProtocol, Block: version.BlockProtocol}, DefaultNodeID: key.ID(), ListenAddr: address.DialString(), Network: "serving-test", Version: "test", Channels: servingChannels(r), Moniker: "serving-test"}
	transport := p2p.NewMultiplexTransport(info, key, p2p.MConnConfig(config))
	require.NoError(t, transport.Listen(*address))
	t.Cleanup(func() { require.NoError(t, transport.Close()) })
	sw := p2p.NewSwitch(config, transport)
	sw.SetNodeKey(&key)
	sw.SetNodeInfo(info)
	sw.AddReactor("STATESYNC", r)
	require.NoError(t, sw.Start())
	t.Cleanup(func() { require.NoError(t, sw.Stop()) })
	return sw
}

func TestServingNativeStatesync(t *testing.T) {
	config := DefaultConfig()
	config.RepeatBytes = 1
	g, err := New(config, prometheus.NewRegistry())
	require.NoError(t, err)
	nativeCfg := cfg.DefaultP2PConfig()
	nativeCfg.ServingPolicy = g
	app := &mocks.AppConnSnapshot{}
	app.On("LoadSnapshotChunk", mock.Anything, mock.Anything).Return(&abci.ResponseLoadSnapshotChunk{Chunk: make([]byte, 100)}, nil).Twice()
	reactor := statesync.NewReactor(*cfg.DefaultStateSyncConfig(), app, nil, statesync.NopMetrics())
	serving := servingSwitch(t, nativeCfg, reactor)
	receiving := &wireReceiver{}
	receiving.BaseReactor = p2p.NewBaseReactor("receive", receiving)
	requester := servingSwitch(t, cfg.DefaultP2PConfig(), receiving)
	require.NoError(t, requester.DialPeerWithAddress(serving.NetAddress()))
	id := string(requester.NodeInfo().ID())
	peer := requester.Peers().Get(serving.NodeInfo().ID())
	require.NotNil(t, peer)
	for i := int64(1); i <= 2; i++ {
		require.True(t, peer.Send(p2p.Envelope{ChannelID: statesync.ChunkChannel, Message: &ss.ChunkRequest{Height: 1}}))
		require.Eventually(t, func() bool {
			g.mu.Lock()
			defer g.mu.Unlock()
			p := g.peers[id]
			return p != nil && p.inflight == 0 && receiving.received.Load() == i
		}, 3*time.Second, time.Millisecond)
	}
	for range 3 {
		require.True(t, peer.Send(p2p.Envelope{ChannelID: statesync.ChunkChannel, Message: &ss.ChunkRequest{Height: 1}}))
	}
	require.Eventually(t, func() bool { return g.Snapshot(id).Risk == 20 }, 3*time.Second, time.Millisecond)
	require.Equal(t, int64(2), receiving.received.Load())
	require.Equal(t, 1, serving.Peers().Size())
	app.AssertExpectations(t)
}

func servingChannels(r p2p.Reactor) []byte {
	channels := make([]byte, 0, len(r.GetChannels()))
	for _, c := range r.GetChannels() {
		channels = append(channels, c.ID)
	}
	return channels
}
