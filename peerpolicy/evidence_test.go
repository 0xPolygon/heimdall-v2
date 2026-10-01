package peerpolicy

import (
	"encoding/binary"
	"testing"

	"github.com/cometbft/cometbft/p2p/observation"
	bc "github.com/cometbft/cometbft/proto/tendermint/blocksync"
	mp "github.com/cometbft/cometbft/proto/tendermint/mempool"
	ss "github.com/cometbft/cometbft/proto/tendermint/statesync"
	ct "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

func TestPeerPolicyClassification(t *testing.T) {
	t.Parallel()
	cases := []struct {
		msg    proto.Message
		family family
		items  uint64
	}{
		{&bc.BlockRequest{}, blockRequests, 1},
		{&ss.SnapshotsRequest{}, snapshotRequests, 1},
		{&ss.ChunkRequest{}, chunkRequests, 1},
		{&mp.Txs{Txs: [][]byte{{1}, {2}}}, transactions, 2},
		{&bc.BlockResponse{}, responses, 1},
		{&ss.ChunkResponse{}, responses, 1},
		{&ss.SnapshotsResponse{}, responses, 1},
		{&bc.NoBlockResponse{}, other, 1},
	}
	for _, tc := range cases {
		e := classify(observation.Event{Kind: observation.Received, Message: tc.msg, Bytes: 20})
		require.Equal(t, tc.family, e.family)
		require.Equal(t, tc.items, e.items)
		require.Equal(t, uint64(20), e.bytes)
		require.Equal(t, none, e.reason)
		require.False(t, e.hasObject)
	}
	require.Zero(t, classify(observation.Event{Bytes: -1}).bytes)
}

func TestPeerPolicyObjectIdentity(t *testing.T) {
	t.Parallel()
	block := classify(observation.Event{Kind: observation.Queued, Message: &bc.BlockResponse{Block: &ct.Block{Header: ct.Header{Height: 123}}}})
	require.Equal(t, blockServing, block.family)
	require.True(t, block.hasObject)
	require.Equal(t, uint64(123), binary.BigEndian.Uint64(block.object[:8]))
	chunk := classify(observation.Event{Kind: observation.Queued, Message: &ss.ChunkResponse{Height: 123, Format: 4, Index: 5}})
	require.Equal(t, chunkServing, chunk.family)
	require.True(t, chunk.hasObject)
	require.Equal(t, uint64(123), binary.BigEndian.Uint64(chunk.object[:8]))
	require.Equal(t, uint32(4), binary.BigEndian.Uint32(chunk.object[8:12]))
	require.Equal(t, uint32(5), binary.BigEndian.Uint32(chunk.object[12:]))
	for _, msg := range []proto.Message{&bc.BlockResponse{}, &ss.ChunkResponse{Missing: true}, &bc.NoBlockResponse{}} {
		e := classify(observation.Event{Kind: observation.Queued, Message: msg})
		require.Equal(t, other, e.family)
		require.False(t, e.hasObject)
	}
	p := peerRecord{objects: make(map[objectKey]int)}
	for i := 0; i < 2; i++ {
		require.False(t, p.repeated(0, block))
	}
	require.True(t, p.repeated(0, block))
	require.False(t, p.repeated(0, chunk))
	require.False(t, p.repeated(windowCount, block))
}

func TestPeerPolicyBlockHeightBoundaries(t *testing.T) {
	t.Parallel()
	for _, height := range []int64{-1, 0, 1} {
		e := classify(observation.Event{Kind: observation.Queued, Message: &bc.BlockResponse{Block: &ct.Block{Header: ct.Header{Height: height}}}})
		require.Equal(t, height >= 0, e.hasObject)
	}
}
