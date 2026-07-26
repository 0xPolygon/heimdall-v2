package abci

import (
	"testing"

	"cosmossdk.io/log"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// A votes only block 1 (correct parent); B votes blocks 1-2 (correct parent);
// C votes blocks 1-2 but under a non-matching parent. Block-level majority and
// the first-block parent check pass on the honest power, but the long range [1,2] only
// clears 2/3 if C's non-matching-parent vote is counted as range support.
func TestGetMajorityMilestoneProposition_RangeSupportRequiresParentHash(t *testing.T) {
	origKyoto := helper.GetKyotoHeight()
	t.Cleanup(func() { helper.SetKyotoHeight(origKyoto) })

	vA := &stakeTypes.Validator{Signer: "0x1111111111111111111111111111111111111111", VotingPower: 33}
	vB := &stakeTypes.Validator{Signer: "0x2222222222222222222222222222222222222222", VotingPower: 34}
	vC := &stakeTypes.Validator{Signer: "0x3333333333333333333333333333333333333333", VotingPower: 33}
	validatorSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{vA, vB, vC}}

	parent := []byte("correctParent")
	wrongParent := []byte("wrongParent")
	h1 := []byte("hash1")
	h2 := []byte("hash2")

	mkVE := func(t *testing.T, parentHash []byte, hashes [][]byte) []byte {
		t.Helper()
		tds := make([]uint64, len(hashes))
		for i := range tds {
			tds[i] = 1
		}
		ve := &sidetxs.VoteExtension{MilestoneProposition: &types.MilestoneProposition{
			BlockHashes:      hashes,
			StartBlockNumber: 1,
			ParentHash:       parentHash,
			BlockTds:         tds,
		}}
		data, err := ve.Marshal()
		require.NoError(t, err)
		return data
	}

	extVotes := []abciTypes.ExtendedVoteInfo{
		{BlockIdFlag: cmtTypes.BlockIDFlagCommit, VoteExtension: mkVE(t, parent, [][]byte{h1}), Validator: abciTypes.Validator{Address: common.HexToAddress(vA.Signer).Bytes()}},
		{BlockIdFlag: cmtTypes.BlockIDFlagCommit, VoteExtension: mkVE(t, parent, [][]byte{h1, h2}), Validator: abciTypes.Validator{Address: common.HexToAddress(vB.Signer).Bytes()}},
		{BlockIdFlag: cmtTypes.BlockIDFlagCommit, VoteExtension: mkVE(t, wrongParent, [][]byte{h1, h2}), Validator: abciTypes.Validator{Address: common.HexToAddress(vC.Signer).Bytes()}},
	}
	logger := log.NewTestLogger(t)
	lastEndBlock := uint64(0)
	majorityVP := int64(67)

	t.Run("below kyoto, wrong-parent vote bridges the long range", func(t *testing.T) {
		helper.SetKyotoHeight(0)
		ctx := sdk.Context{}.WithBlockHeight(100)
		prop, _, _, _, err := GetMajorityMilestoneProposition(ctx, validatorSet, extVotes, majorityVP, logger, &lastEndBlock, parent)
		require.NoError(t, err)
		require.NotNil(t, prop)
		require.Len(t, prop.BlockHashes, 2)
	})

	t.Run("at/after kyoto, long range not counted without matching parent", func(t *testing.T) {
		helper.SetKyotoHeight(1)
		ctx := sdk.Context{}.WithBlockHeight(100)
		prop, _, _, _, err := GetMajorityMilestoneProposition(ctx, validatorSet, extVotes, majorityVP, logger, &lastEndBlock, parent)
		require.NoError(t, err)
		require.Nil(t, prop)
	})
}
