package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMilestonePropositionAccessorsAndEqual(t *testing.T) {
	hash := []byte{0xAA, 0xBB, 0xCC}
	prop := &MilestoneProposition{
		BlockHashes:       [][]byte{[]byte{0x01}},
		BlockTds:          []uint64{1},
		StartBlockNumber:  10,
		ParentHash:        []byte{0x02},
		LatestBlockNumber: 11,
		LatestBlockHash:   hash,
	}

	require.Equal(t, uint64(11), prop.GetLatestBlockNumber())
	require.Equal(t, hash, prop.GetLatestBlockHash())

	same := &MilestoneProposition{
		BlockHashes:       [][]byte{[]byte{0x01}},
		BlockTds:          []uint64{1},
		StartBlockNumber:  10,
		ParentHash:        []byte{0x02},
		LatestBlockNumber: 11,
		LatestBlockHash:   append([]byte(nil), hash...),
	}
	require.True(t, prop.Equal(same))

	different := &MilestoneProposition{
		BlockHashes:       [][]byte{[]byte{0x01}},
		BlockTds:          []uint64{1},
		StartBlockNumber:  10,
		ParentHash:        []byte{0x02},
		LatestBlockNumber: 12,
		LatestBlockHash:   append([]byte(nil), hash...),
	}
	require.False(t, prop.Equal(different))

	var nilProp *MilestoneProposition
	require.Zero(t, nilProp.GetLatestBlockNumber())
	require.Nil(t, nilProp.GetLatestBlockHash())
}
