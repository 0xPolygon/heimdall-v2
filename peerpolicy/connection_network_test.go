package peerpolicy

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/p2p/conn"
	"github.com/cometbft/cometbft/p2p/observation"
	bc "github.com/cometbft/cometbft/proto/tendermint/blocksync"
	"github.com/cometbft/cometbft/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

const reputationTestChannel = 0x40

type reputationWireReactor struct {
	*p2p.BaseReactor
	received atomic.Int64
}

func (r *reputationWireReactor) GetChannels() []*conn.ChannelDescriptor {
	return []*conn.ChannelDescriptor{{ID: reputationTestChannel, Priority: 1, SendQueueCapacity: 1024, MessageType: &bc.Message{}}}
}

func (r *reputationWireReactor) Receive(p2p.Envelope) { r.received.Add(1) }

func reputationSwitch(t *testing.T, cfg *cmtcfg.P2PConfig) (*p2p.Switch, *reputationWireReactor) {
	t.Helper()
	r := &reputationWireReactor{}
	r.BaseReactor = p2p.NewBaseReactor("reputation-wire", r)
	key := p2p.NodeKey{PrivKey: ed25519.GenPrivKey()}
	address := reputationAddress(t, key.ID())
	info := p2p.DefaultNodeInfo{
		ProtocolVersion: p2p.ProtocolVersion{P2P: version.P2PProtocol, Block: version.BlockProtocol},
		DefaultNodeID:   key.ID(), ListenAddr: address.DialString(), Network: "reputation-test",
		Version: "test", Channels: []byte{reputationTestChannel}, Moniker: "reputation-test",
	}
	transport := p2p.NewMultiplexTransport(info, key, p2p.MConnConfig(cfg))
	t.Cleanup(func() { require.NoError(t, transport.Close()) })
	require.NoError(t, transport.Listen(*address))
	sw := p2p.NewSwitch(cfg, transport)
	sw.SetNodeKey(&key)
	sw.SetNodeInfo(info)
	sw.AddReactor("BLOCKSYNC", r)
	require.NoError(t, sw.Start())
	t.Cleanup(func() { require.NoError(t, sw.Stop()) })
	return sw, r
}

func reputationAddress(t *testing.T, id p2p.ID) *p2p.NetAddress {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := p2p.NewNetAddress(id, listener.Addr())
	require.NoError(t, listener.Close())
	return address
}

func TestPeerPolicyNativeConnectionExpiry(t *testing.T) {
	t.Parallel()
	for _, enforce := range []bool{false, true} {
		t.Run(map[bool]string{false: "observe", true: "enforce"}[enforce], func(t *testing.T) {
			t.Parallel()
			var now atomic.Int64
			tracker, err := newTracker(prometheus.NewRegistry(), func() time.Duration { return time.Duration(now.Load()) }, enforce)
			require.NoError(t, err)
			cfg := cmtcfg.DefaultP2PConfig()
			cfg.PeerObserver = tracker
			if enforce {
				cfg.PeerPolicy = tracker
			}
			receiver, reactor := reputationSwitch(t, cfg)
			sender, _ := reputationSwitch(t, cmtcfg.DefaultP2PConfig())
			id := string(sender.NodeInfo().ID())
			tracker.Observe(id, observation.Event{Kind: observation.InvalidMessage})
			require.NoError(t, sender.DialPeerWithAddress(receiver.NetAddress()))
			peer := sender.Peers().Get(receiver.NodeInfo().ID())
			for tick := 1; tick <= 2; tick++ {
				now.Store(int64(time.Duration(tick) * windowWidth))
				for i := 0; i < 641; i++ {
					require.True(t, peer.Send(p2p.Envelope{ChannelID: reputationTestChannel, Message: &bc.BlockRequest{Height: int64(i + 1)}}))
				}
				require.Eventually(t, func() bool { return tracker.Snapshot(id).Risk == uint64(60+20*tick) }, 5*time.Second, 10*time.Millisecond)
			}
			if !enforce {
				require.Eventually(t, func() bool { return reactor.received.Load() == 1282 }, time.Second, 10*time.Millisecond)
				require.Equal(t, 1, receiver.Peers().Size())
				return
			}
			require.Eventually(t, func() bool { return receiver.Peers().Size() == 0 && sender.Peers().Size() == 0 }, time.Second, 10*time.Millisecond)
			require.Equal(t, int64(1281), reactor.received.Load())
			// Outbound admission checks the same authenticated identity and score.
			require.ErrorContains(t, receiver.DialPeerWithAddress(sender.NetAddress()), "connection rejected by peer policy")
			require.Eventually(t, func() bool { return sender.Peers().Size() == 0 }, time.Second, 10*time.Millisecond)
			// The same remote identity cannot bypass policy with an inbound reconnect.
			require.NoError(t, sender.DialPeerWithAddress(receiver.NetAddress()))
			require.Eventually(t, func() bool { return sender.Peers().Size() == 0 }, time.Second, 10*time.Millisecond)
			require.Zero(t, receiver.Peers().Size())
			now.Store(int64(6 * windowWidth))
			require.Equal(t, uint64(40), tracker.Snapshot(id).Risk)
			require.NoError(t, sender.DialPeerWithAddress(receiver.NetAddress()))
			require.Eventually(t, func() bool { return receiver.Peers().Size() == 1 }, time.Second, 10*time.Millisecond)
		})
	}
}
