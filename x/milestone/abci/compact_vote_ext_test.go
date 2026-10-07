package abci

import (
	"math"
	"testing"

	"cosmossdk.io/log"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

const compactForkHeight int64 = 100

var compactSigners = []string{
	"0x1111111111111111111111111111111111111111",
	"0x2222222222222222222222222222222222222222",
	"0x3333333333333333333333333333333333333333",
}

// setCompactFork activates the compact format at compactForkHeight with Kyoto on and Ithaca at the
// given height (0 matches the local/devnet defaults).
func setCompactFork(t *testing.T, ithaca int64) {
	t.Helper()
	origIthaca, origKyoto, origCompact := helper.GetIthacaHeight(), helper.GetKyotoHeight(), helper.GetCompactVoteExtHeight()
	t.Cleanup(func() {
		helper.SetIthacaHeight(origIthaca)
		helper.SetKyotoHeight(origKyoto)
		helper.SetCompactVoteExtHeight(origCompact)
	})
	helper.SetIthacaHeight(ithaca)
	helper.SetKyotoHeight(1)
	helper.SetCompactVoteExtHeight(compactForkHeight)
}

// testProp proposes blocks 10-11 with tail hash fill32(0x02).
func testProp(parent []byte, latest uint64, latestHash []byte) *types.MilestoneProposition {
	return &types.MilestoneProposition{
		StartBlockNumber:  10,
		BlockHashes:       [][]byte{fill32(0x01), fill32(0x02)},
		BlockTds:          []uint64{1, 2},
		ParentHash:        parent,
		LatestBlockNumber: latest,
		LatestBlockHash:   latestHash,
	}
}

func overflowProp(latest uint64, latestHash []byte) *types.MilestoneProposition {
	prop := testProp(nil, latest, latestHash)
	prop.StartBlockNumber = math.MaxUint64
	return prop
}

// votesFrom gives compactSigners[i] the proposition props[i].
func votesFrom(t *testing.T, props ...*types.MilestoneProposition) []abciTypes.ExtendedVoteInfo {
	t.Helper()
	votes := make([]abciTypes.ExtendedVoteInfo, len(props))
	for i, prop := range props {
		votes[i] = commitVote(t, compactSigners[i], prop)
	}
	return votes
}

func equalPowerSet(signers []string) *stakeTypes.ValidatorSet {
	set := &stakeTypes.ValidatorSet{}
	for _, s := range signers {
		set.Validators = append(set.Validators, &stakeTypes.Validator{Signer: s, VotingPower: 10})
	}
	return set
}

func requireErr(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		require.NoError(t, err)
		return
	}
	require.ErrorContains(t, err, want)
}

func TestCompactProposition(t *testing.T) {
	setCompactFork(t, 1)
	parent, tail := fill32(0x09), fill32(0x02)

	tests := []struct {
		name           string
		height         int64
		prop           *types.MilestoneProposition
		wantParent     []byte
		wantLatestHash []byte
	}{
		{"pre-fork untouched", compactForkHeight - 1, testProp(parent, 11, tail), parent, tail},
		{"head at tail drops hash", compactForkHeight, testProp(parent, 11, tail), parent[:8], nil},
		{"head beyond tail keeps hash", compactForkHeight + 1, testProp(parent, 15, fill32(0x05)), parent[:8], fill32(0x05)},
		{"no head", compactForkHeight, testProp(parent, 0, nil), parent[:8], nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := compactProposition(tc.height, tc.prop)
			require.Equal(t, tc.wantParent, got.ParentHash)
			require.Equal(t, tc.wantLatestHash, got.LatestBlockHash)
		})
	}
}

func TestValidateMilestonePropositionCompact(t *testing.T) {
	ctx, milestoneKeeper := newTestMilestoneKeeper(t)
	parent, tail := fill32(0x09), fill32(0x02)
	maxStart := &types.MilestoneProposition{
		StartBlockNumber:  math.MaxUint64,
		BlockHashes:       [][]byte{fill32(0x01)},
		BlockTds:          []uint64{1},
		LatestBlockNumber: math.MaxUint64,
	}

	tests := []struct {
		name    string
		ithaca  int64
		height  int64
		prop    *types.MilestoneProposition
		wantErr string
	}{
		{"legacy before fork", 1, compactForkHeight - 1, testProp(parent, 11, tail), ""},
		{"compact head before fork", 1, compactForkHeight - 1, testProp(parent[:8], 11, nil), "latest block number set without latest block hash"},
		{"compact at fork", 1, compactForkHeight, testProp(parent[:8], 11, nil), ""},
		{"compact after fork", 1, compactForkHeight + 1, testProp(parent[:8], 11, nil), ""},
		{"no parent", 1, compactForkHeight, testProp(nil, 11, nil), ""},
		{"no head", 1, compactForkHeight, testProp(parent[:8], 0, nil), ""},
		{"head beyond tail with hash", 1, compactForkHeight, testProp(parent[:8], 12, fill32(0x05)), ""},
		{"head at max block", 1, compactForkHeight, maxStart, ""},
		{"full parent", 1, compactForkHeight, testProp(parent, 11, nil), "invalid compact parent hash length"},
		{"full parent without ithaca", 0, compactForkHeight, testProp(parent, 0, nil), "invalid compact parent hash length"},
		{"redundant tail hash", 1, compactForkHeight, testProp(parent[:8], 11, tail), "must be omitted"},
		{"head beyond tail without hash", 1, compactForkHeight, testProp(parent[:8], 12, nil), "latest block number set without latest block hash"},
		{"head behind tail", 1, compactForkHeight, testProp(parent[:8], 10, fill32(0x05)), "behind proposition end"},
		{"short head hash", 1, compactForkHeight, testProp(parent[:8], 12, fill32(0x05)[:8]), "invalid latest block hash length"},
		{"overflowing start with omitted head", 1, compactForkHeight, overflowProp(5, nil), "latest block number set without latest block hash"},
		{"overflowing start with head hash", 1, compactForkHeight, overflowProp(5, fill32(0x05)), "proposition start block overflow"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setCompactFork(t, tc.ithaca)
			requireErr(t, ValidateMilestoneProposition(ctx.WithBlockHeight(tc.height), milestoneKeeper, tc.prop), tc.wantErr)
		})
	}
}

func TestExpandParentHash(t *testing.T) {
	setCompactFork(t, 1)
	last := fill32(0x09)

	tests := []struct {
		name     string
		veHeight int64
		parent   []byte
		lastEnd  []byte
		want     []byte
	}{
		{"matching prefix", compactForkHeight, last[:8], last, last},
		{"prefix before fork", compactForkHeight - 1, last[:8], last, last[:8]},
		{"mismatching prefix", compactForkHeight, fill32(0x08)[:8], last, fill32(0x08)[:8]},
		{"full hash", compactForkHeight, last, last, last},
		{"no last milestone", compactForkHeight, last[:8], nil, last[:8]},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, expandParentHash(tc.veHeight, tc.parent, tc.lastEnd))
		})
	}
}

// FinalizeBlock at H tallies extensions produced at H-1, so the prefix only expands from fork+1.
func TestGetMajorityMilestonePropositionCompactParent(t *testing.T) {
	const twoThirds = 21
	lastEndHash, lastEndBlock := fill32(0x09), uint64(9)
	prop := func(parent []byte) *types.MilestoneProposition {
		return &types.MilestoneProposition{StartBlockNumber: 10, BlockHashes: [][]byte{fill32(0x01)}, BlockTds: []uint64{1}, ParentHash: parent}
	}
	full, short, wrong := prop(lastEndHash), prop(lastEndHash[:8]), prop(fill32(0x08)[:8])

	tests := []struct {
		name      string
		ithaca    int64
		height    int64
		votes     []abciTypes.ExtendedVoteInfo
		wantFound bool
	}{
		{"legacy extensions at fork", 1, compactForkHeight, votesFrom(t, full, full, full), true},
		{"compact extensions before they are valid", 1, compactForkHeight, votesFrom(t, short, short, short), false},
		{"compact extensions expanded", 1, compactForkHeight + 1, votesFrom(t, short, short, short), true},
		{"compact extensions expanded without ithaca", 0, compactForkHeight + 1, votesFrom(t, short, short, short), true},
		{"mismatching prefix not counted", 1, compactForkHeight + 1, votesFrom(t, short, short, wrong), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setCompactFork(t, tc.ithaca)
			ctx := sdk.Context{}.WithBlockHeight(tc.height)
			got, _, _, _, err := GetMajorityMilestoneProposition(ctx, equalPowerSet(compactSigners), tc.votes, twoThirds, log.NewTestLogger(t), &lastEndBlock, lastEndHash)
			require.NoError(t, err)
			require.Equal(t, tc.wantFound, got != nil)
		})
	}
}

func TestGetMajorityActualHeadCompactImpliedTail(t *testing.T) {
	setCompactFork(t, 1)
	const minMajorityVP, maxBlock = 11, 100
	signers := compactSigners[:2]
	legacy, compact := testProp(nil, 11, fill32(0x02)), testProp(nil, 11, nil)

	tests := []struct {
		name      string
		height    int64
		votes     []abciTypes.ExtendedVoteInfo
		wantFound bool
	}{
		{"legacy extensions at fork", compactForkHeight, votesFrom(t, legacy, legacy), true},
		{"compact extensions before they are valid", compactForkHeight, votesFrom(t, compact, compact), false},
		{"compact extensions", compactForkHeight + 1, votesFrom(t, compact, compact), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			num, hash, found, err := GetMajorityActualHead(sdk.Context{}.WithBlockHeight(tc.height), equalPowerSet(signers), tc.votes, minMajorityVP, maxBlock)
			require.NoError(t, err)
			require.Equal(t, tc.wantFound, found)
			if found {
				require.Equal(t, uint64(11), num)
				require.Equal(t, fill32(0x02), hash)
			}
		})
	}
}

func TestImpliedLatestHead(t *testing.T) {
	tail := fill32(0x02)
	tests := []struct {
		name     string
		prop     *types.MilestoneProposition
		wantHash []byte
	}{
		{"head at tail", testProp(nil, 11, nil), tail},
		{"head beyond tail", testProp(nil, 12, nil), nil},
		{"no head", testProp(nil, 0, nil), nil},
		{"hash present", testProp(nil, 12, fill32(0x05)), fill32(0x05)},
		{"hash present at tail", testProp(nil, 11, fill32(0x05)), fill32(0x05)},
		{"overflow", overflowProp(11, nil), nil},
		{"empty proposition", &types.MilestoneProposition{LatestBlockNumber: math.MaxUint64}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := *tc.prop
			require.Equal(t, tc.wantHash, impliedLatestHead(tc.prop).LatestBlockHash)
			require.Equal(t, before, *tc.prop, "input must not be mutated")
		})
	}
}
