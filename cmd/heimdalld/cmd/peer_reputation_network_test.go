package heimdalld

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/p2p/conn"
	bc "github.com/cometbft/cometbft/proto/tendermint/blocksync"
	"github.com/cometbft/cometbft/version"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/peerpolicy"
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

func TestPeerReputationDefaultNativeTraffic(t *testing.T) {
	t.Parallel()
	for _, enabled := range []bool{true, false} {
		name := "default"
		if !enabled {
			name = "disabled"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := server.NewDefaultContext()
			cmd := peerReputationCommand()
			if !enabled {
				require.NoError(t, cmd.ParseFlags([]string{"--peer-reputation=false"}))
			}
			require.NoError(t, configurePeerReputation(cmd, ctx, prometheus.NewRegistry()))
			receiver, reactor := reputationSwitch(t, ctx.Config.P2P)
			sender, _ := reputationSwitch(t, cmtcfg.DefaultP2PConfig())
			require.NoError(t, sender.DialPeerWithAddress(receiver.NetAddress()))
			peer := sender.Peers().Get(receiver.NodeInfo().ID())
			require.NotNil(t, peer)
			for height := int64(1); height <= 641; height++ {
				require.True(t, peer.Send(p2p.Envelope{ChannelID: reputationTestChannel, Message: &bc.BlockRequest{Height: height}}))
			}
			require.Eventually(t, func() bool { return reactor.received.Load() == 641 }, 5*time.Second, 10*time.Millisecond)
			if enabled {
				tracker, ok := ctx.Config.P2P.PeerObserver.(*peerpolicy.Tracker)
				require.True(t, ok)
				score := tracker.Snapshot(string(sender.NodeInfo().ID()))
				require.Equal(t, uint64(20), score.Risk)
				require.Equal(t, uint64(1), score.ReasonWindows["request_volume"])
			} else {
				require.Nil(t, ctx.Config.P2P.PeerObserver)
			}
		})
	}
}

func reputationAddress(t *testing.T, id p2p.ID) *p2p.NetAddress {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := p2p.NewNetAddress(id, listener.Addr())
	require.NoError(t, listener.Close())
	return address
}
