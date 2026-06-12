package abci

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

// TestMilestonePropositionHeadID covers the hash+td identity of a pending proposition's head,
// which POS-3629 uses to reset the stall clock when a contested tip flaps at a constant height.
func TestMilestonePropositionHeadID(t *testing.T) {
	h1 := []byte("hash-block-1-aaaaaaaaaaaaaaaaaaa")
	h2 := []byte("hash-block-2-bbbbbbbbbbbbbbbbbbb")
	h2alt := []byte("hash-block-2-cccccccccccccccccc")

	t.Run("nil proposition", func(t *testing.T) {
		require.Nil(t, MilestonePropositionHeadID(nil))
	})

	t.Run("empty hashes", func(t *testing.T) {
		require.Nil(t, MilestonePropositionHeadID(&types.MilestoneProposition{}))
	})

	t.Run("mismatched tds length returns nil", func(t *testing.T) {
		prop := &types.MilestoneProposition{BlockHashes: [][]byte{h1, h2}, StartBlockNumber: 100, BlockTds: []uint64{10}}
		require.Nil(t, MilestonePropositionHeadID(prop))
	})

	t.Run("uses the head (last) block, hash prefix preserved", func(t *testing.T) {
		prop := &types.MilestoneProposition{BlockHashes: [][]byte{h1, h2}, StartBlockNumber: 100, BlockTds: []uint64{10, 20}}
		id := MilestonePropositionHeadID(prop)
		require.Len(t, id, len(h2)+8)
		require.Equal(t, h2, id[:len(h2)])
	})

	t.Run("identical content yields identical id", func(t *testing.T) {
		a := &types.MilestoneProposition{BlockHashes: [][]byte{h1, h2}, StartBlockNumber: 100, BlockTds: []uint64{10, 20}}
		b := &types.MilestoneProposition{BlockHashes: [][]byte{h1, h2}, StartBlockNumber: 100, BlockTds: []uint64{10, 20}}
		require.Equal(t, MilestonePropositionHeadID(a), MilestonePropositionHeadID(b))
	})

	t.Run("differs when head hash changes at the same height", func(t *testing.T) {
		base := &types.MilestoneProposition{BlockHashes: [][]byte{h1, h2}, StartBlockNumber: 100, BlockTds: []uint64{10, 20}}
		alt := &types.MilestoneProposition{BlockHashes: [][]byte{h1, h2alt}, StartBlockNumber: 100, BlockTds: []uint64{10, 20}}
		require.NotEqual(t, MilestonePropositionHeadID(base), MilestonePropositionHeadID(alt))
	})

	t.Run("differs when head td changes", func(t *testing.T) {
		base := &types.MilestoneProposition{BlockHashes: [][]byte{h2}, StartBlockNumber: 100, BlockTds: []uint64{20}}
		alt := &types.MilestoneProposition{BlockHashes: [][]byte{h2}, StartBlockNumber: 100, BlockTds: []uint64{21}}
		require.NotEqual(t, MilestonePropositionHeadID(base), MilestonePropositionHeadID(alt))
	})
}
