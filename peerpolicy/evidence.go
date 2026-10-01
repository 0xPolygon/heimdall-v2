// Package peerpolicy implements bounded, observation-only native peer scoring.
package peerpolicy

import (
	"encoding/binary"

	"github.com/cometbft/cometbft/p2p/observation"
	bc "github.com/cometbft/cometbft/proto/tendermint/blocksync"
	mp "github.com/cometbft/cometbft/proto/tendermint/mempool"
	ss "github.com/cometbft/cometbft/proto/tendermint/statesync"
)

type reason uint8

const (
	none reason = iota
	invalidEncoding
	invalidMessage
	requestVolume
	transactionVolume
	responseVolume
	servingVolume
	servingRepetition
	reasonCount
)

var reasonNames = [...]string{"none", "invalid_encoding", "invalid_message", "request_volume", "transaction_volume", "response_volume", "serving_volume", "serving_repetition"}

type family uint8

const (
	other family = iota
	blockRequests
	snapshotRequests
	chunkRequests
	transactions
	responses
	blockServing
	chunkServing
	familyCount
)

var familyNames = [...]string{"other", "block_requests", "snapshot_requests", "chunk_requests", "transactions", "responses", "block_serving", "chunk_serving"}

type evidence struct {
	reason       reason
	family       family
	items, bytes uint64
	object       [16]byte
	hasObject    bool
}

type allowance struct {
	items, bytes uint64
	reason       reason
}

// Experimental ten-second profiles. These do not enforce production limits.
var allowances = [familyCount]allowance{
	blockRequests:    {640, 1 << 20, requestVolume},
	snapshotRequests: {64, 1 << 20, requestVolume},
	chunkRequests:    {640, 1 << 20, requestVolume},
	transactions:     {32768, 64 << 20, transactionVolume},
	responses:        {10240, 160 << 20, responseVolume},
	blockServing:     {10240, 160 << 20, servingVolume},
	chunkServing:     {10240, 160 << 20, servingVolume},
}

func classify(event observation.Event) evidence {
	e := evidence{items: 1}
	if event.Bytes > 0 {
		e.bytes = uint64(event.Bytes)
	}
	switch event.Kind {
	case observation.InvalidEncoding:
		e.reason = invalidEncoding
	case observation.InvalidMessage:
		e.reason = invalidMessage
	case observation.Received:
		e = received(event, e)
	case observation.Queued:
		e = queued(event, e)
	}
	return e
}

func received(event observation.Event, e evidence) evidence {
	switch msg := event.Message.(type) {
	case *bc.BlockRequest, *bc.StatusRequest:
		e.family = blockRequests
	case *ss.SnapshotsRequest:
		e.family = snapshotRequests
	case *ss.ChunkRequest:
		e.family = chunkRequests
	case *mp.Txs:
		e.family, e.items = transactions, uint64(len(msg.Txs))
	case *bc.BlockResponse, *ss.ChunkResponse, *ss.SnapshotsResponse:
		e.family = responses
	}
	return e
}

func queued(event observation.Event, e evidence) evidence {
	switch msg := event.Message.(type) {
	case *bc.BlockResponse:
		if msg.Block == nil {
			return e
		}
		height := msg.Block.Header.Height
		if height < 0 {
			return e
		}
		e.family, e.hasObject = blockServing, true
		binary.BigEndian.PutUint64(e.object[:8], uint64(height))
	case *ss.ChunkResponse:
		if !msg.Missing {
			e.family, e.hasObject = chunkServing, true
			binary.BigEndian.PutUint64(e.object[:8], msg.Height)
			binary.BigEndian.PutUint32(e.object[8:12], msg.Format)
			binary.BigEndian.PutUint32(e.object[12:], msg.Index)
		}
	}
	return e
}

func weight(r reason) uint64 {
	switch r {
	case invalidEncoding, invalidMessage:
		return 60
	case requestVolume, transactionVolume, responseVolume, servingVolume, servingRepetition:
		return 20
	default:
		return 0
	}
}
