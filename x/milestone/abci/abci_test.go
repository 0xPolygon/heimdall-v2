package abci

import (
	"context"
	"errors"
	"math"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmosTestutil "github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	borKeeper "github.com/0xPolygon/heimdall-v2/x/bor/keeper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

type testChainKeeper struct{}

func (testChainKeeper) GetParams(context.Context) (chainmanagertypes.Params, error) {
	return chainmanagertypes.Params{}, nil
}

type testStakeKeeper struct{}

func (testStakeKeeper) GetSpanEligibleValidators(context.Context) []stakeTypes.Validator {
	return nil
}

func (testStakeKeeper) GetValidatorSet(context.Context) (stakeTypes.ValidatorSet, error) {
	return stakeTypes.ValidatorSet{}, nil
}

func (testStakeKeeper) GetValidatorFromValID(context.Context, uint64) (stakeTypes.Validator, error) {
	return stakeTypes.Validator{}, nil
}

func (testStakeKeeper) GetValIdFromAddress(context.Context, string) (uint64, error) {
	return 0, nil
}

type testMilestoneKeeper struct {
	last *types.Milestone
}

func (k testMilestoneKeeper) GetLastMilestone(context.Context) (*types.Milestone, error) {
	return k.last, nil
}

func TestIsFastForwardMilestone(t *testing.T) {
	tests := []struct {
		name                    string
		latestHeaderNumber      uint64
		latestMilestoneEndBlock uint64
		ffMilestoneThreshold    uint64
		expected                bool
	}{
		{
			name:                    "Header equals milestone block",
			latestHeaderNumber:      100,
			latestMilestoneEndBlock: 100,
			ffMilestoneThreshold:    0,
			expected:                false,
		},
		{
			name:                    "Header less than milestone block",
			latestHeaderNumber:      90,
			latestMilestoneEndBlock: 100,
			ffMilestoneThreshold:    0,
			expected:                false,
		},
		{
			name:                    "Difference equals threshold",
			latestHeaderNumber:      105,
			latestMilestoneEndBlock: 100,
			ffMilestoneThreshold:    5,
			expected:                false, // because 105-100 == 5 (not greater than 5)
		},
		{
			name:                    "Difference less than threshold",
			latestHeaderNumber:      110,
			latestMilestoneEndBlock: 100,
			ffMilestoneThreshold:    15,
			expected:                false,
		},
		{
			name:                    "Difference greater than threshold",
			latestHeaderNumber:      110,
			latestMilestoneEndBlock: 100,
			ffMilestoneThreshold:    5,
			expected:                true,
		},
		{
			name:                    "Threshold zero, header greater than milestone",
			latestHeaderNumber:      101,
			latestMilestoneEndBlock: 100,
			ffMilestoneThreshold:    0,
			expected:                true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isFastForwardMilestone(tc.latestHeaderNumber, tc.latestMilestoneEndBlock, tc.ffMilestoneThreshold)
			if result != tc.expected {
				t.Errorf("isFastForwardMilestone(%d, %d, %d) = %v; expected %v",
					tc.latestHeaderNumber, tc.latestMilestoneEndBlock, tc.ffMilestoneThreshold, result, tc.expected)
			}
		})
	}
}

func TestGetFastForwardMilestoneStartBlock(t *testing.T) {
	tests := []struct {
		name                     string
		latestHeaderNumber       uint64
		latestMilestoneEndBlock  uint64
		ffMilestoneBlockInterval uint64
		expected                 uint64
	}{
		{
			name:                     "Interval is 10",
			latestMilestoneEndBlock:  100,
			ffMilestoneBlockInterval: 10,
			expected:                 110, // 100+10=110
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getFastForwardMilestoneStartBlock(tc.latestMilestoneEndBlock, tc.ffMilestoneBlockInterval)
			if result != tc.expected {
				t.Errorf("getFastForwardMilestoneStartBlock(%d, %d, %d) = %d; expected %d",
					tc.latestHeaderNumber, tc.latestMilestoneEndBlock, tc.ffMilestoneBlockInterval, result, tc.expected)
			}
		})
	}
}

func TestGetMajorityMilestoneProposition_MajorityWins(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(100) // Mock context with block height
	// Two validators: one with 70% power, one with 30%
	v1 := &stakeTypes.Validator{
		Signer:      "0x1111111111111111111111111111111111111111",
		VotingPower: 70,
	}
	v2 := &stakeTypes.Validator{
		Signer:      "0x2222222222222222222222222222222222222222",
		VotingPower: 30,
	}
	validatorSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{v1, v2}}

	// Common milestone data
	parentHash := []byte("parentHash")
	startBlock := uint64(1)
	blockTd := uint64(1)
	hashMajor := []byte("major")
	hashMinor := []byte("minor")

	// Build two different propositions
	propMajor := &types.MilestoneProposition{
		BlockHashes:      [][]byte{hashMajor},
		StartBlockNumber: startBlock,
		ParentHash:       parentHash,
		BlockTds:         []uint64{blockTd},
	}
	propMinor := &types.MilestoneProposition{
		BlockHashes:      [][]byte{hashMinor},
		StartBlockNumber: startBlock,
		ParentHash:       parentHash,
		BlockTds:         []uint64{blockTd},
	}

	// Marshal vote extensions
	ve1 := &sidetxs.VoteExtension{MilestoneProposition: propMajor}
	ve2 := &sidetxs.VoteExtension{MilestoneProposition: propMinor}
	dataMajor, err := ve1.Marshal()
	assert.NoError(t, err)
	dataMinor, err := ve2.Marshal()
	assert.NoError(t, err)

	// Convert signer strings to address bytes using go-ethereum common
	addrBytesMajor := common.HexToAddress(v1.Signer).Bytes()
	addrBytesMinor := common.HexToAddress(v2.Signer).Bytes()

	// Prepare votes
	extVotes := []abciTypes.ExtendedVoteInfo{
		{BlockIdFlag: cmtTypes.BlockIDFlagCommit, VoteExtension: dataMajor, Validator: abciTypes.Validator{Address: addrBytesMajor}},
		{BlockIdFlag: cmtTypes.BlockIDFlagCommit, VoteExtension: dataMinor, Validator: abciTypes.Validator{Address: addrBytesMinor}},
	}
	logger := log.NewTestLogger(t)

	lastEndBlock := startBlock - 1
	lastEndHash := parentHash

	resultProp, _, _, _, err := GetMajorityMilestoneProposition(
		ctx,
		validatorSet,
		extVotes,
		1,
		logger,
		&lastEndBlock,
		lastEndHash,
	)

	assert.NoError(t, err, "expected no error for majority-win scenario")
	assert.NotNil(t, resultProp, "expected a proposition when majority is reached")
	assert.Equal(t, propMajor.BlockHashes, resultProp.BlockHashes, "majority validator's proposition should win")
	assert.Equal(t, propMajor.BlockTds, resultProp.BlockTds, "majority validator's proposition should win")
}

// TestGetMajorityMilestoneProposition_TwoParentsClearThreshold pins parent selection when more than
// one parent hash clears the 1/3 pending threshold. Two disjoint groups vote the same block with
// different parents, both above majorityVP: the honest parent matches lastEndBlockHash, the bogus one
// does not. The canonical parent (lastEndBlockHash) must be selected whenever it clears the threshold,
// regardless of any other parent's voting power, so the honest milestone is returned.
func TestGetMajorityMilestoneProposition_TwoParentsClearThreshold(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(100)

	// Total voting power 100, so majorityVP = 34 is 1/3+1; both 40 and 35 clear it.
	vHonest := &stakeTypes.Validator{Signer: "0x1111111111111111111111111111111111111111", VotingPower: 40}
	vBogus := &stakeTypes.Validator{Signer: "0x2222222222222222222222222222222222222222", VotingPower: 35}
	vIdle := &stakeTypes.Validator{Signer: "0x3333333333333333333333333333333333333333", VotingPower: 25}
	validatorSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{vHonest, vBogus, vIdle}}

	startBlock := uint64(1)
	blockHash := []byte("same-block-hash")
	blockTd := uint64(1)
	honestParent := []byte("honest-parent-hash")
	bogusParent := []byte("bogus-parent-hash") // does not match lastEndBlockHash, so it is never a valid winner

	// Both groups vote the identical block (hash+td); only the parent differs.
	propHonest := &types.MilestoneProposition{
		BlockHashes:      [][]byte{blockHash},
		StartBlockNumber: startBlock,
		ParentHash:       honestParent,
		BlockTds:         []uint64{blockTd},
	}
	propBogus := &types.MilestoneProposition{
		BlockHashes:      [][]byte{blockHash},
		StartBlockNumber: startBlock,
		ParentHash:       bogusParent,
		BlockTds:         []uint64{blockTd},
	}

	veHonest := &sidetxs.VoteExtension{MilestoneProposition: propHonest}
	veBogus := &sidetxs.VoteExtension{MilestoneProposition: propBogus}
	dataHonest, err := veHonest.Marshal()
	assert.NoError(t, err)
	dataBogus, err := veBogus.Marshal()
	assert.NoError(t, err)

	extVotes := []abciTypes.ExtendedVoteInfo{
		{BlockIdFlag: cmtTypes.BlockIDFlagCommit, VoteExtension: dataHonest, Validator: abciTypes.Validator{Address: common.HexToAddress(vHonest.Signer).Bytes()}},
		{BlockIdFlag: cmtTypes.BlockIDFlagCommit, VoteExtension: dataBogus, Validator: abciTypes.Validator{Address: common.HexToAddress(vBogus.Signer).Bytes()}},
	}
	logger := log.NewTestLogger(t)

	lastEndBlock := startBlock - 1
	lastEndHash := honestParent // the honest parent matches the chain's last end block hash

	resultProp, _, _, _, err := GetMajorityMilestoneProposition(
		ctx,
		validatorSet,
		extVotes,
		34, // 1/3+1 of the 100-VP set; both parents clear it
		logger,
		&lastEndBlock,
		lastEndHash,
	)

	assert.NoError(t, err)
	assert.NotNil(t, resultProp, "canonical parent (lastEndBlockHash) clears the threshold and must be selected even though another parent also clears it")
	assert.Equal(t, propHonest.BlockHashes, resultProp.BlockHashes)
}

// TestGetMajorityMilestoneProposition_ByzantineEqualPowerBogusParent covers the byzantine case the
// previous highest-power-with-lex-tie-break tournament lost: a colluding 1/3+1 slice votes the real
// block under a fabricated parent hash that sorts lexicographically before the honest parent, with
// voting power equal to the honest supporters. ParentHash is not bound by ValidateMilestoneProposition,
// so this proposition is structurally valid. Both parents clear the 1/3 pending threshold; the
// aggregator must still return the honest milestone (its parent equals lastEndBlockHash) rather than
// nil. A tournament with an ascending lex tie-break would hand the equal-power case to the bogus
// parent and return nil, silently dropping the pending-stall path.
func TestGetMajorityMilestoneProposition_ByzantineEqualPowerBogusParent(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(100)

	// Total voting power 100, majorityVP = 34. Honest and byzantine groups have equal power; both clear.
	vHonest := &stakeTypes.Validator{Signer: "0x1111111111111111111111111111111111111111", VotingPower: 34}
	vByzantine := &stakeTypes.Validator{Signer: "0x2222222222222222222222222222222222222222", VotingPower: 34}
	vIdle := &stakeTypes.Validator{Signer: "0x3333333333333333333333333333333333333333", VotingPower: 32}
	validatorSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{vHonest, vByzantine, vIdle}}

	startBlock := uint64(1)
	blockTd := uint64(1)
	// Full 32-byte hashes: a byzantine proposition passes ValidateMilestoneProposition's structural
	// checks, so the attack is on real, well-formed input. The honest parent sorts lexicographically
	// after the bogus one, so an ascending-lex tournament at equal power would pick the bogus parent.
	blockHash := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000aa").Bytes()
	honestParent := common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff").Bytes()
	bogusParent := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001").Bytes()
	require.Less(t, common.Bytes2Hex(bogusParent), common.Bytes2Hex(honestParent), "test setup: bogus parent must sort first")

	// Both groups vote the identical real block; only the parent differs.
	propHonest := &types.MilestoneProposition{
		BlockHashes:      [][]byte{blockHash},
		StartBlockNumber: startBlock,
		ParentHash:       honestParent,
		BlockTds:         []uint64{blockTd},
	}
	propByzantine := &types.MilestoneProposition{
		BlockHashes:      [][]byte{blockHash},
		StartBlockNumber: startBlock,
		ParentHash:       bogusParent,
		BlockTds:         []uint64{blockTd},
	}

	veHonest := &sidetxs.VoteExtension{MilestoneProposition: propHonest}
	veByzantine := &sidetxs.VoteExtension{MilestoneProposition: propByzantine}
	dataHonest, err := veHonest.Marshal()
	assert.NoError(t, err)
	dataByzantine, err := veByzantine.Marshal()
	assert.NoError(t, err)

	extVotes := []abciTypes.ExtendedVoteInfo{
		{BlockIdFlag: cmtTypes.BlockIDFlagCommit, VoteExtension: dataHonest, Validator: abciTypes.Validator{Address: common.HexToAddress(vHonest.Signer).Bytes()}},
		{BlockIdFlag: cmtTypes.BlockIDFlagCommit, VoteExtension: dataByzantine, Validator: abciTypes.Validator{Address: common.HexToAddress(vByzantine.Signer).Bytes()}},
	}
	logger := log.NewTestLogger(t)

	lastEndBlock := startBlock - 1
	lastEndHash := honestParent // the canonical parent is the honest one

	resultProp, _, _, _, err := GetMajorityMilestoneProposition(
		ctx,
		validatorSet,
		extVotes,
		34,
		logger,
		&lastEndBlock,
		lastEndHash,
	)

	assert.NoError(t, err)
	assert.NotNil(t, resultProp, "honest parent equals lastEndBlockHash and clears the threshold; a byzantine equal-power bogus parent must not suppress it")
	assert.Equal(t, propHonest.BlockHashes, resultProp.BlockHashes)
}

// TestGetMajorityMilestoneProposition_ParentCheckUsesReturnedStartBlock covers the case where an
// earlier overlapping block has majority support, but the returned pending range must start at
// lastEndBlock+1. The parent-child majority check must use that returned start block; keying it by the
// first majority block would incorrectly drop the valid pending range.
func TestGetMajorityMilestoneProposition_ParentCheckUsesReturnedStartBlock(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(100)

	// Total voting power 100, majorityVP = 34. Both the earlier overlapping block and the returned
	// start block independently clear the threshold.
	vOld := &stakeTypes.Validator{Signer: "0x1111111111111111111111111111111111111111", VotingPower: 40}
	vNew := &stakeTypes.Validator{Signer: "0x2222222222222222222222222222222222222222", VotingPower: 40}
	vIdle := &stakeTypes.Validator{Signer: "0x3333333333333333333333333333333333333333", VotingPower: 20}
	validatorSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{vOld, vNew, vIdle}}

	lastEndBlock := uint64(100)
	returnedStartBlock := lastEndBlock + 1
	oldMajorityBlock := uint64(99)

	oldParent := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000009").Bytes()
	lastEndHash := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000aa").Bytes()
	oldHash := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000bb").Bytes()
	returnedHash := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000cc").Bytes()

	propOld := &types.MilestoneProposition{
		BlockHashes:      [][]byte{oldHash},
		StartBlockNumber: oldMajorityBlock,
		ParentHash:       oldParent,
		BlockTds:         []uint64{1},
	}
	propReturned := &types.MilestoneProposition{
		BlockHashes:      [][]byte{returnedHash},
		StartBlockNumber: returnedStartBlock,
		ParentHash:       lastEndHash,
		BlockTds:         []uint64{2},
	}

	veOld := &sidetxs.VoteExtension{MilestoneProposition: propOld}
	veReturned := &sidetxs.VoteExtension{MilestoneProposition: propReturned}
	dataOld, err := veOld.Marshal()
	assert.NoError(t, err)
	dataReturned, err := veReturned.Marshal()
	assert.NoError(t, err)

	extVotes := []abciTypes.ExtendedVoteInfo{
		{BlockIdFlag: cmtTypes.BlockIDFlagCommit, VoteExtension: dataOld, Validator: abciTypes.Validator{Address: common.HexToAddress(vOld.Signer).Bytes()}},
		{BlockIdFlag: cmtTypes.BlockIDFlagCommit, VoteExtension: dataReturned, Validator: abciTypes.Validator{Address: common.HexToAddress(vNew.Signer).Bytes()}},
	}
	logger := log.NewTestLogger(t)

	resultProp, _, _, _, err := GetMajorityMilestoneProposition(
		ctx,
		validatorSet,
		extVotes,
		34,
		logger,
		&lastEndBlock,
		lastEndHash,
	)

	assert.NoError(t, err)
	assert.NotNil(t, resultProp, "canonical parent clears threshold for lastEndBlock+1; earlier majority blocks must not decide the parent check")
	assert.Equal(t, returnedStartBlock, resultProp.StartBlockNumber)
	assert.Equal(t, propReturned.BlockHashes, resultProp.BlockHashes)
	assert.Equal(t, propReturned.BlockTds, resultProp.BlockTds)
}

func TestValidateMilestonePropositionFork(t *testing.T) {
	t.Parallel()

	t.Run("validates matching parent and last milestone hash", func(t *testing.T) {
		t.Parallel()

		parentHash := []byte("test_hash_123")
		lastMilestoneHash := []byte("test_hash_123")

		err := validateMilestonePropositionFork(parentHash, lastMilestoneHash)
		require.NoError(t, err)
	})

	t.Run("returns error when hashes don't match", func(t *testing.T) {
		t.Parallel()

		parentHash := []byte("parent_hash")
		lastMilestoneHash := []byte("different_hash")

		err := validateMilestonePropositionFork(parentHash, lastMilestoneHash)
		require.Error(t, err)
		require.Contains(t, err.Error(), "first block parent hash does not match last milestone hash")
	})

	t.Run("accepts empty parent hash", func(t *testing.T) {
		t.Parallel()

		var parentHash []byte
		lastMilestoneHash := []byte("some_hash")

		err := validateMilestonePropositionFork(parentHash, lastMilestoneHash)
		require.NoError(t, err)
	})

	t.Run("accepts empty last milestone hash", func(t *testing.T) {
		t.Parallel()

		parentHash := []byte("some_hash")
		var lastMilestoneHash []byte

		err := validateMilestonePropositionFork(parentHash, lastMilestoneHash)
		require.NoError(t, err)
	})

	t.Run("accepts both hashes empty", func(t *testing.T) {
		t.Parallel()

		var parentHash []byte
		var lastMilestoneHash []byte

		err := validateMilestonePropositionFork(parentHash, lastMilestoneHash)
		require.NoError(t, err)
	})

	t.Run("accepts nil parent hash", func(t *testing.T) {
		t.Parallel()

		var parentHash []byte = nil
		lastMilestoneHash := []byte("some_hash")

		err := validateMilestonePropositionFork(parentHash, lastMilestoneHash)
		require.NoError(t, err)
	})

	t.Run("accepts nil last milestone hash", func(t *testing.T) {
		t.Parallel()

		parentHash := []byte("some_hash")
		var lastMilestoneHash []byte = nil

		err := validateMilestonePropositionFork(parentHash, lastMilestoneHash)
		require.NoError(t, err)
	})

	t.Run("validates exact byte match for longer hashes", func(t *testing.T) {
		t.Parallel()

		longHash := []byte("this_is_a_very_long_hash_with_many_bytes_12345678")
		parentHash := make([]byte, len(longHash))
		copy(parentHash, longHash)

		err := validateMilestonePropositionFork(parentHash, longHash)
		require.NoError(t, err)
	})

	t.Run("detects mismatch in long hashes", func(t *testing.T) {
		t.Parallel()

		hash1 := []byte("this_is_a_very_long_hash_with_many_bytes_12345678")
		hash2 := []byte("this_is_a_very_long_hash_with_many_bytes_87654321")

		err := validateMilestonePropositionFork(hash1, hash2)
		require.Error(t, err)
	})
}

func TestValidateMilestoneProposition(t *testing.T) {
	ctx, milestoneKeeper := newTestMilestoneKeeper(t)

	makeProp := func(blockHashes [][]byte, blockTds []uint64, start uint64, latestNum uint64, latestHash []byte) *types.MilestoneProposition {
		return &types.MilestoneProposition{
			BlockHashes:       blockHashes,
			BlockTds:          blockTds,
			StartBlockNumber:  start,
			LatestBlockNumber: latestNum,
			LatestBlockHash:   latestHash,
		}
	}

	t.Run("accepts nil proposition", func(t *testing.T) {
		require.NoError(t, ValidateMilestoneProposition(ctx, milestoneKeeper, nil))
	})

	t.Run("accepts a valid proposition with matching latest head", func(t *testing.T) {
		prop := makeProp([][]byte{fill32(0x01)}, []uint64{100}, 10, 10, fill32(0x01))
		require.NoError(t, ValidateMilestoneProposition(ctx, milestoneKeeper, prop))
	})

	t.Run("rejects too many blocks", func(t *testing.T) {
		prop := makeProp([][]byte{fill32(0x01), fill32(0x02), fill32(0x03)}, []uint64{1, 2, 3}, 10, 10, fill32(0x03))
		params, err := milestoneKeeper.GetParams(ctx)
		require.NoError(t, err)
		params.MaxMilestonePropositionLength = 2
		require.NoError(t, milestoneKeeper.SetParams(ctx, params))

		err = ValidateMilestoneProposition(ctx, milestoneKeeper, prop)
		require.Error(t, err)
		require.Contains(t, err.Error(), "too many blocks in proposition")
	})

	t.Run("rejects empty block list", func(t *testing.T) {
		prop := makeProp(nil, nil, 10, 0, nil)
		err := ValidateMilestoneProposition(ctx, milestoneKeeper, prop)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no blocks in proposition")
	})

	t.Run("rejects length mismatch", func(t *testing.T) {
		prop := makeProp([][]byte{fill32(0x01)}, []uint64{1, 2}, 10, 10, fill32(0x01))
		err := ValidateMilestoneProposition(ctx, milestoneKeeper, prop)
		require.Error(t, err)
		require.Contains(t, err.Error(), "len mismatch between hashes and tds")
	})

	t.Run("rejects invalid block hash length", func(t *testing.T) {
		prop := makeProp([][]byte{make([]byte, 16)}, []uint64{1}, 10, 10, make([]byte, 16))
		err := ValidateMilestoneProposition(ctx, milestoneKeeper, prop)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid block hash length")
	})

	t.Run("rejects duplicate block hashes", func(t *testing.T) {
		hash := fill32(0xAA)
		prop := makeProp([][]byte{hash, hash}, []uint64{1, 2}, 10, 11, fill32(0xAA))
		err := ValidateMilestoneProposition(ctx, milestoneKeeper, prop)
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate block hashes found")
	})

	t.Run("rejects latest block number without hash", func(t *testing.T) {
		prop := makeProp([][]byte{fill32(0x01)}, []uint64{1}, 10, 10, nil)
		prop.LatestBlockNumber = 12
		err := ValidateMilestoneProposition(ctx, milestoneKeeper, prop)
		require.Error(t, err)
		require.Contains(t, err.Error(), "latest block number set without latest block hash")
	})

	t.Run("rejects latest block hash length", func(t *testing.T) {
		prop := makeProp([][]byte{fill32(0x01)}, []uint64{1}, 10, 10, make([]byte, 16))
		err := ValidateMilestoneProposition(ctx, milestoneKeeper, prop)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid latest block hash length")
	})

	t.Run("rejects latest head behind proposition tail", func(t *testing.T) {
		prop := makeProp([][]byte{fill32(0x01), fill32(0x02)}, []uint64{1, 2}, 10, 10, fill32(0x02))
		prop.LatestBlockNumber = 10
		err := ValidateMilestoneProposition(ctx, milestoneKeeper, prop)
		require.Error(t, err)
		require.Contains(t, err.Error(), "latest block number 10 behind proposition end 11")
	})

	t.Run("rejects latest tail hash mismatch", func(t *testing.T) {
		prop := makeProp([][]byte{fill32(0x01), fill32(0x02)}, []uint64{1, 2}, 10, 11, fill32(0x03))
		err := ValidateMilestoneProposition(ctx, milestoneKeeper, prop)
		require.Error(t, err)
		require.Contains(t, err.Error(), "latest block hash does not match proposition tail")
	})

	t.Run("accepts latest head beyond proposition tail", func(t *testing.T) {
		prop := makeProp([][]byte{fill32(0x01), fill32(0x02)}, []uint64{1, 2}, 10, 99, fill32(0xFF))
		require.NoError(t, ValidateMilestoneProposition(ctx, milestoneKeeper, prop))
	})
}

func newTestMilestoneKeeper(t *testing.T) (sdk.Context, *keeper.Keeper) {
	t.Helper()

	key := storetypes.NewKVStoreKey(types.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := cosmosTestutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx.WithBlockHeight(1)
	encCfg := moduletestutil.MakeTestEncodingConfig()
	caller := &mocks.IContractCaller{}

	k := keeper.NewKeeper(
		encCfg.Codec,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		storeService,
		caller,
	)
	k.InitGenesis(ctx, types.DefaultGenesisState())
	return ctx, &k
}

// fill32 returns a 32-byte hash filled with seed.
func fill32(seed byte) []byte {
	h := make([]byte, common.HashLength)
	for i := range h {
		h[i] = seed
	}
	return h
}

// actualHeadVote builds a committed vote extension whose milestone proposition reports
// (number, fill32(hashSeed)) as its actual latest bor head.
func actualHeadVote(t *testing.T, signer string, number uint64, hashSeed byte) abciTypes.ExtendedVoteInfo {
	t.Helper()
	hash := fill32(hashSeed)
	ve := &sidetxs.VoteExtension{MilestoneProposition: &types.MilestoneProposition{
		StartBlockNumber:  number,
		BlockHashes:       [][]byte{hash},
		BlockTds:          []uint64{1},
		LatestBlockNumber: number,
		LatestBlockHash:   hash,
	}}
	data, err := ve.Marshal()
	require.NoError(t, err)
	return abciTypes.ExtendedVoteInfo{
		BlockIdFlag:   cmtTypes.BlockIDFlagCommit,
		VoteExtension: data,
		Validator:     abciTypes.Validator{Address: common.HexToAddress(signer).Bytes()},
	}
}

// voteNoLatestHead builds a committed vote whose proposition omits the actual-head fields.
func voteNoLatestHead(t *testing.T, signer string) abciTypes.ExtendedVoteInfo {
	t.Helper()
	ve := &sidetxs.VoteExtension{MilestoneProposition: &types.MilestoneProposition{
		StartBlockNumber: 1,
		BlockHashes:      [][]byte{fill32(0x01)},
		BlockTds:         []uint64{1},
	}}
	data, err := ve.Marshal()
	require.NoError(t, err)
	return abciTypes.ExtendedVoteInfo{
		BlockIdFlag:   cmtTypes.BlockIDFlagCommit,
		VoteExtension: data,
		Validator:     abciTypes.Validator{Address: common.HexToAddress(signer).Bytes()},
	}
}

func TestGetMajorityActualHead(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(100)
	const (
		s1 = "0x1111111111111111111111111111111111111111"
		s2 = "0x2222222222222222222222222222222222222222"
		s3 = "0x3333333333333333333333333333333333333333"
	)

	t.Run("greatest voting power wins, not the highest block number", func(t *testing.T) {
		valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
			{Signer: s1, VotingPower: 40}, {Signer: s2, VotingPower: 35}, {Signer: s3, VotingPower: 25},
		}}
		// 300 has 40 VP (a fabricated higher head from a 1/3+1 minority); 200 has 60 VP (the converged
		// honest majority). Both clear 34, but the most-voted head must win — a byzantine minority cannot
		// install a higher fabricated head by number.
		votes := []abciTypes.ExtendedVoteInfo{
			actualHeadVote(t, s1, 300, 0xAA),
			actualHeadVote(t, s2, 200, 0xBB),
			actualHeadVote(t, s3, 200, 0xBB),
		}
		head, hash, found, err := GetMajorityActualHead(ctx, valSet, votes, 34, math.MaxUint64)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(200), head, "the most-voted head wins, not the highest-numbered minority head")
		require.Equal(t, fill32(0xBB), hash)
	})

	t.Run("lone far head below 1/3 is ignored; real agreed head wins", func(t *testing.T) {
		valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
			{Signer: s1, VotingPower: 40}, {Signer: s2, VotingPower: 35}, {Signer: s3, VotingPower: 25},
		}}
		votes := []abciTypes.ExtendedVoteInfo{
			actualHeadVote(t, s1, 150, 0xAA),
			actualHeadVote(t, s2, 150, 0xAA),
			actualHeadVote(t, s3, 99999, 0xEE), // byzantine far head, only 25 < 34
		}
		head, _, found, err := GetMajorityActualHead(ctx, valSet, votes, 34, math.MaxUint64)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(150), head, "the lone far head must not be selected")
	})

	t.Run("no head clears the threshold", func(t *testing.T) {
		valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
			{Signer: s1, VotingPower: 33}, {Signer: s2, VotingPower: 33}, {Signer: s3, VotingPower: 33},
		}}
		votes := []abciTypes.ExtendedVoteInfo{
			actualHeadVote(t, s1, 100, 0xA1),
			actualHeadVote(t, s2, 200, 0xB2),
			actualHeadVote(t, s3, 300, 0xC3),
		}
		_, _, found, err := GetMajorityActualHead(ctx, valSet, votes, 34, math.MaxUint64)
		require.NoError(t, err)
		require.False(t, found, "no distinct head reaches 1/3")
	})

	t.Run("head with exactly the threshold voting power is included", func(t *testing.T) {
		// head 200 has exactly minMajorityVP (34) and is the only head clearing it; 100 and 300 each have
		// 33 (<34) so neither qualifies.
		valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
			{Signer: s1, VotingPower: 34}, {Signer: s2, VotingPower: 33}, {Signer: s3, VotingPower: 33},
		}}
		votes := []abciTypes.ExtendedVoteInfo{
			actualHeadVote(t, s1, 200, 0xAA),
			actualHeadVote(t, s2, 100, 0xBB),
			actualHeadVote(t, s3, 300, 0xCC),
		}
		head, _, found, err := GetMajorityActualHead(ctx, valSet, votes, 34, math.MaxUint64)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(200), head, "VP exactly equal to the threshold must count (>= comparison)")
	})

	t.Run("same-height fork resolves to the greater voting power", func(t *testing.T) {
		// Two hashes at the same number 200: 0xAA has 40+25=65 VP, 0xBB has 35. The greater-VP hash wins.
		valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
			{Signer: s1, VotingPower: 40}, {Signer: s2, VotingPower: 35}, {Signer: s3, VotingPower: 25},
		}}
		votes := []abciTypes.ExtendedVoteInfo{
			actualHeadVote(t, s1, 200, 0xAA),
			actualHeadVote(t, s2, 200, 0xBB),
			actualHeadVote(t, s3, 200, 0xAA),
		}
		head, hash, found, err := GetMajorityActualHead(ctx, valSet, votes, 34, math.MaxUint64)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(200), head)
		require.Equal(t, fill32(0xAA), hash, "the hash with greater voting power wins")
	})

	t.Run("equal voting power ties break deterministically", func(t *testing.T) {
		// 100 and 200 each clear the threshold at exactly equal power (34); 300 has 32 (<34). The choice
		// must be deterministic (lexicographically smaller tally key).
		valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
			{Signer: s1, VotingPower: 34}, {Signer: s2, VotingPower: 34}, {Signer: s3, VotingPower: 32},
		}}
		votes := []abciTypes.ExtendedVoteInfo{
			actualHeadVote(t, s1, 100, 0xAA),
			actualHeadVote(t, s2, 200, 0xBB),
			actualHeadVote(t, s3, 300, 0xCC),
		}
		head, _, found, err := GetMajorityActualHead(ctx, valSet, votes, 34, math.MaxUint64)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(100), head, "equal power resolves to the lexicographically smaller key, deterministically")
	})

	t.Run("a validator voting twice is counted once", func(t *testing.T) {
		valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
			{Signer: s1, VotingPower: 34}, {Signer: s2, VotingPower: 33}, {Signer: s3, VotingPower: 33},
		}}
		// s1 votes head 200 twice. With minVP=68, dedup keeps 200 at 34 (<68) → not found; if the
		// duplicate were counted, 200 would reach 68 and wrongly clear.
		votes := []abciTypes.ExtendedVoteInfo{
			actualHeadVote(t, s1, 200, 0xAA),
			actualHeadVote(t, s1, 200, 0xAA),
			actualHeadVote(t, s2, 100, 0xBB),
		}
		_, _, found, err := GetMajorityActualHead(ctx, valSet, votes, 68, math.MaxUint64)
		require.NoError(t, err)
		require.False(t, found, "a duplicate vote must not double-count the validator's power")
	})

	t.Run("votes without latest-head fields are skipped", func(t *testing.T) {
		valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
			{Signer: s1, VotingPower: 40}, {Signer: s2, VotingPower: 35}, {Signer: s3, VotingPower: 25},
		}}
		votes := []abciTypes.ExtendedVoteInfo{
			actualHeadVote(t, s1, 200, 0xAA),
			voteNoLatestHead(t, s2),
			{BlockIdFlag: cmtTypes.BlockIDFlagCommit, Validator: abciTypes.Validator{Address: common.HexToAddress(s3).Bytes()}, VoteExtension: func() []byte {
				ve := &sidetxs.VoteExtension{MilestoneProposition: nil}
				b, err := ve.Marshal()
				require.NoError(t, err)
				return b
			}()},
		}
		head, _, found, err := GetMajorityActualHead(ctx, valSet, votes, 34, math.MaxUint64)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(200), head)
	})

	t.Run("out-of-range head is dropped even when it has the greater voting power", func(t *testing.T) {
		// s1's head 99999 is beyond maxBlock (500) and holds 51 VP — more than the in-range head 200 (s2+s3
		// = 49 VP). Both clear 34, so without the bound greatest-VP selection would pick the out-of-range
		// head. The bound must drop it so the legitimate in-range head is selected; this is what makes the
		// maxBlock filter, not VP, decide the outcome here.
		valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
			{Signer: s1, VotingPower: 51}, {Signer: s2, VotingPower: 34}, {Signer: s3, VotingPower: 15},
		}}
		votes := []abciTypes.ExtendedVoteInfo{
			actualHeadVote(t, s1, 99999, 0xAA),
			actualHeadVote(t, s2, 200, 0xBB),
			actualHeadVote(t, s3, 200, 0xBB),
		}
		head, hash, found, err := GetMajorityActualHead(ctx, valSet, votes, 34, 500)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, uint64(200), head, "a head beyond maxBlock must be dropped even with greater VP")
		require.Equal(t, fill32(0xBB), hash)
	})

	t.Run("a lone out-of-range head clearing the threshold is dropped, not tracked", func(t *testing.T) {
		// s1 alone clears 34 on a fabricated far head (99999 > maxBlock 500); honest s2/s3 split below
		// the threshold. The far head must be filtered so nothing is found, never poisoning tracking.
		valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
			{Signer: s1, VotingPower: 40}, {Signer: s2, VotingPower: 30}, {Signer: s3, VotingPower: 30},
		}}
		votes := []abciTypes.ExtendedVoteInfo{
			actualHeadVote(t, s1, 99999, 0xEE),
			actualHeadVote(t, s2, 100, 0xA1),
			actualHeadVote(t, s3, 200, 0xB2),
		}
		_, _, found, err := GetMajorityActualHead(ctx, valSet, votes, 34, 500)
		require.NoError(t, err)
		require.False(t, found, "an out-of-range head must be dropped even when it alone clears the threshold")
	})
}

func TestGetMajorityActualHeadErrorsOnInvalidVoteExtension(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(100)
	valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
		{Signer: "0x1111111111111111111111111111111111111111", VotingPower: 40},
	}}
	votes := []abciTypes.ExtendedVoteInfo{
		{
			BlockIdFlag:   cmtTypes.BlockIDFlagCommit,
			VoteExtension: []byte{0xFF, 0x00, 0x01},
			Validator:     abciTypes.Validator{Address: common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes()},
		},
	}

	_, _, _, err := GetMajorityActualHead(ctx, valSet, votes, 34, math.MaxUint64)
	require.Error(t, err)
}

func TestGetMajorityActualHeadErrorsOnMissingValidatorWhenChecksEnabled(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(helper.GetTallyFixHeight())
	valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
		{Signer: "0x1111111111111111111111111111111111111111", VotingPower: 40},
	}}
	votes := []abciTypes.ExtendedVoteInfo{
		actualHeadVote(t, "0x2222222222222222222222222222222222222222", 100, 0xAA),
	}

	_, _, _, err := GetMajorityActualHead(ctx, valSet, votes, 34, math.MaxUint64)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Failed to get validator")
}

func TestGetMajorityActualHeadSkipsNonCommitAndMissingValidatorWhenChecksDisabled(t *testing.T) {
	height := helper.GetDisableValSetCheckHeight() + 1
	if helper.GetDisableValSetCheckHeight() >= helper.GetTallyFixHeight() {
		height = helper.GetTallyFixHeight() - 1
	}
	ctx := sdk.Context{}.WithBlockHeight(height)
	valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
		{Signer: "0x1111111111111111111111111111111111111111", VotingPower: 40},
		{Signer: "0x2222222222222222222222222222222222222222", VotingPower: 40},
	}}
	votes := []abciTypes.ExtendedVoteInfo{
		{
			BlockIdFlag:   cmtTypes.BlockIDFlagNil,
			VoteExtension: []byte{0x01, 0x02, 0x03},
			Validator:     abciTypes.Validator{Address: common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes()},
		},
		actualHeadVote(t, "0x1111111111111111111111111111111111111111", 100, 0xAA),
	}

	head, _, found, err := GetMajorityActualHead(ctx, valSet, votes, 34, math.MaxUint64)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, uint64(100), head)
}

func TestGetMajorityMilestonePropositionErrorsOnInvalidVoteExtension(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(100)
	valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
		{Signer: "0x1111111111111111111111111111111111111111", VotingPower: 40},
	}}
	votes := []abciTypes.ExtendedVoteInfo{
		{
			BlockIdFlag:   cmtTypes.BlockIDFlagCommit,
			VoteExtension: []byte{0xFF, 0x00, 0x01},
			Validator:     abciTypes.Validator{Address: common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes()},
		},
	}
	logger := log.NewTestLogger(t)

	_, _, _, _, err := GetMajorityMilestoneProposition(ctx, valSet, votes, 34, logger, nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error while unmarshalling vote extension")
}

func TestGetMajorityMilestonePropositionErrorsOnMissingValidatorWhenChecksEnabled(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(helper.GetTallyFixHeight())
	valSet := &stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
		{Signer: "0x1111111111111111111111111111111111111111", VotingPower: 40},
	}}
	prop := &types.MilestoneProposition{
		BlockHashes:      [][]byte{fill32(0xAA)},
		StartBlockNumber: 1,
		ParentHash:       fill32(0xBB),
		BlockTds:         []uint64{1},
	}
	data, err := (&sidetxs.VoteExtension{MilestoneProposition: prop}).Marshal()
	require.NoError(t, err)
	votes := []abciTypes.ExtendedVoteInfo{
		{
			BlockIdFlag:   cmtTypes.BlockIDFlagCommit,
			VoteExtension: data,
			Validator:     abciTypes.Validator{Address: common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes()},
		},
	}
	logger := log.NewTestLogger(t)

	_, _, _, _, err = GetMajorityMilestoneProposition(ctx, valSet, votes, 34, logger, nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Failed to get validator")
}

func TestValidateLatestHead(t *testing.T) {
	t.Parallel()

	mk := func(start uint64, nBlocks int, latestNum uint64, latestHash []byte) *types.MilestoneProposition {
		hashes := make([][]byte, nBlocks)
		for i := range hashes {
			hashes[i] = fill32(byte(i + 1))
		}
		return &types.MilestoneProposition{
			StartBlockNumber:  start,
			BlockHashes:       hashes,
			LatestBlockNumber: latestNum,
			LatestBlockHash:   latestHash,
		}
	}

	cases := []struct {
		name string
		prop *types.MilestoneProposition
		ok   bool
	}{
		{"both absent is ok", mk(10, 1, 0, nil), true},
		{"number without hash rejected", mk(10, 1, 12, nil), false},
		{"present, head == proposition end, hash matches tail", mk(10, 1, 10, fill32(1)), true},
		{"present, head == proposition end, hash mismatches tail rejected", mk(10, 1, 10, fill32(0x9)), false},
		{"present, multi-block head == proposition end, hash matches tail", mk(10, 5, 14, fill32(5)), true}, // propEnd 14, tail hash fill32(5)
		{"present, head beyond proposition end", mk(10, 5, 99, fill32(0x9)), true},
		{"bad hash length rejected", mk(10, 1, 10, make([]byte, 16)), false},
		{"head behind proposition end rejected", mk(10, 5, 12, fill32(0x9)), false}, // propEnd 14
		{"start-block overflow rejected", mk(math.MaxUint64, 2, math.MaxUint64, fill32(0x9)), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateLatestHead(tc.prop)
			if tc.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestActualHeadFields(t *testing.T) {
	orig := helper.GetIthacaHeight()
	t.Cleanup(func() { helper.SetIthacaHeight(orig) })

	ctx := sdk.Context{}.WithBlockHeight(100)
	header := &ethTypes.Header{Number: big.NewInt(4242)}

	// Fork OFF: never emit the fields, even with a header (pre-fork VEs must stay free of the new fields).
	helper.SetIthacaHeight(0)
	num, hash := actualHeadFields(ctx, header)
	require.Zero(t, num)
	require.Nil(t, hash)

	// Fork ON but no latest header (e.g. no prior milestone / no reachable Bor): empty.
	helper.SetIthacaHeight(1)
	num, hash = actualHeadFields(ctx, nil)
	require.Zero(t, num)
	require.Nil(t, hash)

	// Fork ON with a header: populated from the header.
	num, hash = actualHeadFields(ctx, header)
	require.Equal(t, uint64(4242), num)
	require.Equal(t, header.Hash().Bytes(), hash)
}

// TestGetBlockInfoReturnsRefreshedHeader pins that getBlockInfo returns the header it actually built
// the proposition from — including the one it refreshes internally when the caller's cached header is
// behind propStartBlock ("Heimdall faster than Bor"). Returning the stale input instead would let the
// actual-head fields fall behind the proposition end and fail validateLatestHead (POS-3629).
func TestGetBlockInfoReturnsRefreshedHeader(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(100)
	const startBlock = uint64(100)

	batch := func(from, to int64) ([]*ethTypes.Header, []uint64, []common.Address) {
		var hdrs []*ethTypes.Header
		var tds []uint64
		var authors []common.Address
		for i := from; i <= to; i++ {
			hdrs = append(hdrs, &ethTypes.Header{Number: big.NewInt(i)})
			tds = append(tds, uint64(i))
			authors = append(authors, common.Address{})
		}
		return hdrs, tds, authors
	}

	t.Run("stale cached header is refreshed and the fresh one is returned", func(t *testing.T) {
		mc := &mocks.IContractCaller{}
		fresh := &ethTypes.Header{Number: big.NewInt(105)}
		mc.On("GetBorChainBlock", mock.Anything, mock.Anything).Return(fresh, nil)
		h, td, a := batch(100, 105) // milestoneEnd = 100 + min(6,10) - 1 = 105
		mc.On("GetBorChainBlockInfoInBatch", mock.Anything, int64(100), int64(105)).Return(h, td, a, nil)

		stale := &ethTypes.Header{Number: big.NewInt(90)} // behind startBlock → triggers refresh
		_, blockHashes, _, _, eff, err := getBlockInfo(ctx, mc, startBlock, 10, stale, nil, 0)
		require.NoError(t, err)
		require.NotNil(t, eff)
		require.Equal(t, uint64(105), eff.Number.Uint64(), "must return the refreshed header, not the stale input (90)")
		require.GreaterOrEqual(t, eff.Number.Uint64(), startBlock+uint64(len(blockHashes))-1,
			"effective head must be >= the proposition's last block so validateLatestHead passes")
	})

	t.Run("fresh cached header is returned unchanged", func(t *testing.T) {
		mc := &mocks.IContractCaller{}
		h, td, a := batch(100, 105)
		mc.On("GetBorChainBlockInfoInBatch", mock.Anything, int64(100), int64(105)).Return(h, td, a, nil)

		cached := &ethTypes.Header{Number: big.NewInt(105)} // already >= startBlock → no refresh
		_, _, _, _, eff, err := getBlockInfo(ctx, mc, startBlock, 10, cached, nil, 0)
		require.NoError(t, err)
		require.Same(t, cached, eff, "no refresh path must return the caller's header")
		mc.AssertNotCalled(t, "GetBorChainBlock", mock.Anything, mock.Anything)
	})
}

// TestGetBlockInfoErrorPaths covers the branches that return early when Bor data
// is missing or malformed, so the coverage for the changed return signature stays
// high on failure paths too.
func TestGetBlockInfoErrorPaths(t *testing.T) {
	ctx := sdk.Context{}.WithBlockHeight(100)

	t.Run("fails when the latest header cannot be fetched", func(t *testing.T) {
		mc := &mocks.IContractCaller{}
		fetchErr := errors.New("fetch failed")
		mc.On("GetBorChainBlock", mock.Anything, mock.Anything).Return((*ethTypes.Header)(nil), fetchErr)

		_, _, _, _, eff, err := getBlockInfo(ctx, mc, 100, 10, nil, nil, 0)
		require.Error(t, err)
		require.Nil(t, eff)
		require.Contains(t, err.Error(), "failed to get the latest header")
	})

	t.Run("fails when batch retrieval returns no headers", func(t *testing.T) {
		mc := &mocks.IContractCaller{}
		mc.On("GetBorChainBlockInfoInBatch", mock.Anything, int64(100), int64(105)).Return([]*ethTypes.Header{}, []uint64{}, []common.Address{}, nil)

		latest := &ethTypes.Header{Number: big.NewInt(105)}
		_, _, _, _, eff, err := getBlockInfo(ctx, mc, 100, 10, latest, nil, 0)
		require.ErrorIs(t, err, ErrNoNewHeadersFound)
		require.Nil(t, eff)
	})

	t.Run("fails when parent hash lookup fails", func(t *testing.T) {
		mc := &mocks.IContractCaller{}
		latest := &ethTypes.Header{Number: big.NewInt(105)}
		hdrs := []*ethTypes.Header{{Number: big.NewInt(101)}}
		tds := []uint64{101}
		authors := []common.Address{{}}
		mc.On("GetBorChainBlockInfoInBatch", mock.Anything, int64(101), int64(105)).Return(hdrs, tds, authors, nil)
		parentErr := errors.New("parent failed")
		mc.On("GetBorChainBlock", mock.Anything, mock.Anything).Return((*ethTypes.Header)(nil), parentErr)

		_, _, _, _, eff, err := getBlockInfo(ctx, mc, 101, 10, latest, []byte{0x01}, 99)
		require.Error(t, err)
		require.Nil(t, eff)
		require.Contains(t, err.Error(), "failed to get header for parent hash")
	})
}

func TestGenMilestoneProposition(t *testing.T) {
	milestoneKey := storetypes.NewKVStoreKey(types.StoreKey)
	borKey := storetypes.NewKVStoreKey(borTypes.StoreKey)
	transientKey := storetypes.NewTransientStoreKey("milestone_test_transient")
	milestoneCtx := cosmosTestutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{
			"milestone": milestoneKey,
			"bor":       borKey,
		},
		map[string]*storetypes.TransientStoreKey{
			"transient": transientKey,
		},
		nil,
	).WithBlockHeight(1)
	encCfg := moduletestutil.MakeTestEncodingConfig()
	milestoneKeeper := keeper.NewKeeper(
		encCfg.Codec,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.NewKVStoreService(milestoneKey),
		&mocks.IContractCaller{},
	)
	milestoneKeeper.InitGenesis(milestoneCtx, types.DefaultGenesisState())
	params, err := milestoneKeeper.GetParams(milestoneCtx)
	require.NoError(t, err)
	params.MaxMilestonePropositionLength = 5
	params.FfMilestoneThreshold = 10
	params.FfMilestoneBlockInterval = 5
	require.NoError(t, milestoneKeeper.SetParams(milestoneCtx, params))

	latestHeader := &ethTypes.Header{Number: big.NewInt(111)}
	latestHash := latestHeader.Hash().Bytes()
	lastMilestone := &types.Milestone{
		EndBlock:   100,
		Hash:       fill32(0x0A),
		BorChainId: "1",
	}
	require.NoError(t, milestoneKeeper.AddMilestone(milestoneCtx, *lastMilestone))
	parentHeader := &ethTypes.Header{
		Number:     big.NewInt(101),
		ParentHash: common.BytesToHash(lastMilestone.Hash),
	}

	// Build a minimal Bor keeper that can answer CanVoteProducers and keep spans
	// in-memory. The stake/chain keepers are stubs because this test only needs the
	// span state and the milestone proposition path.
	storeService := runtime.NewKVStoreService(borKey)
	bk := borKeeper.NewKeeper(
		encCfg.Codec,
		storeService,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		testChainKeeper{},
		testStakeKeeper{},
		testMilestoneKeeper{last: lastMilestone},
		&mocks.IContractCaller{},
	)
	require.NoError(t, bk.AddNewSpan(milestoneCtx, &borTypes.Span{
		Id:         1,
		StartBlock: 1,
		EndBlock:   100,
		BorChainId: "1",
		ValidatorSet: stakeTypes.ValidatorSet{Validators: []*stakeTypes.Validator{
			{ValId: 1, VotingPower: 100, Signer: "0x0000000000000000000000000000000000000001"},
		}},
		SelectedProducers: []stakeTypes.Validator{{ValId: 1, VotingPower: 100, Signer: "0x0000000000000000000000000000000000000001"}},
	}))

	origRio := helper.GetRioHeight()
	origSpan := helper.GetIthacaHeight()
	t.Cleanup(func() {
		helper.SetRioHeight(origRio)
		helper.SetIthacaHeight(origSpan)
	})
	helper.SetIthacaHeight(1)

	batchHeaders := func(from, to int64) ([]*ethTypes.Header, []uint64, []common.Address) {
		var hdrs []*ethTypes.Header
		var tds []uint64
		var authors []common.Address
		for i := from; i <= to; i++ {
			hdrs = append(hdrs, &ethTypes.Header{Number: big.NewInt(i)})
			tds = append(tds, uint64(i))
			authors = append(authors, common.Address{})
		}
		return hdrs, tds, authors
	}

	t.Run("returns the full proposition when veBlop voting is disabled", func(t *testing.T) {
		helper.SetRioHeight(999999)
		mc := &mocks.IContractCaller{}
		mc.On("GetBorChainBlock", mock.Anything, (*big.Int)(nil)).Return(latestHeader, nil)
		mc.On("GetBorChainBlock", mock.Anything, mock.MatchedBy(func(n *big.Int) bool { return n != nil && n.Uint64() == 101 })).Return(parentHeader, nil)
		h, td, a := batchHeaders(105, 109)
		mc.On("GetBorChainBlockInfoInBatch", mock.Anything, int64(105), int64(109)).Return(h, td, a, nil)
		prop, err := GenMilestoneProposition(milestoneCtx, &bk, &milestoneKeeper, mc, func(sdk.Context, uint64) ([]common.Address, error) {
			return nil, nil
		})
		require.NoError(t, err)
		require.NotNil(t, prop)
		require.Equal(t, uint64(105), prop.StartBlockNumber)
		require.Equal(t, latestHeader.Number.Uint64(), prop.LatestBlockNumber)
		require.Equal(t, latestHash, prop.LatestBlockHash)
		require.Len(t, prop.BlockHashes, 5)
	})

	t.Run("filters by block author when veBlop voting is enabled", func(t *testing.T) {
		helper.SetRioHeight(101)
		mc := &mocks.IContractCaller{}
		mc.On("GetBorChainBlock", mock.Anything, (*big.Int)(nil)).Return(latestHeader, nil)
		mc.On("GetBorChainBlock", mock.Anything, mock.MatchedBy(func(n *big.Int) bool { return n != nil && n.Uint64() == 101 })).Return(parentHeader, nil)
		h, td, a := batchHeaders(105, 109)
		mc.On("GetBorChainBlockInfoInBatch", mock.Anything, int64(105), int64(109)).Return(h, td, a, nil)
		prop, err := GenMilestoneProposition(milestoneCtx, &bk, &milestoneKeeper, mc, func(sdk.Context, uint64) ([]common.Address, error) {
			return []common.Address{{}}, nil
		})
		require.NoError(t, err)
		require.NotNil(t, prop)
		require.Len(t, prop.BlockHashes, 5)
		require.Equal(t, latestHash, prop.LatestBlockHash)
	})
}

func TestShouldErrorOnValidatorNotFound(t *testing.T) {
	t.Parallel()

	// Note: These tests depend on helper.GetTallyFixHeight() and helper.GetDisableValSetCheckHeight()
	// We test the logic based on typical values

	t.Run("returns true for heights at or above tally fix", func(t *testing.T) {
		t.Parallel()

		tallyFixHeight := helper.GetTallyFixHeight()

		result := ShouldErrorOnValidatorNotFound(tallyFixHeight)
		require.True(t, result)

		result = ShouldErrorOnValidatorNotFound(tallyFixHeight + 1)
		require.True(t, result)

		result = ShouldErrorOnValidatorNotFound(tallyFixHeight + 1000)
		require.True(t, result)
	})

	t.Run("returns false for heights between disable check and tally fix", func(t *testing.T) {
		t.Parallel()

		disableCheckHeight := helper.GetDisableValSetCheckHeight()
		tallyFixHeight := helper.GetTallyFixHeight()

		if disableCheckHeight < tallyFixHeight {
			// Test a height in the middle range
			middleHeight := disableCheckHeight + (tallyFixHeight-disableCheckHeight)/2
			result := ShouldErrorOnValidatorNotFound(middleHeight)
			require.False(t, result)
		}
	})

	t.Run("returns true for heights below disable check", func(t *testing.T) {
		t.Parallel()

		disableCheckHeight := helper.GetDisableValSetCheckHeight()

		if disableCheckHeight > 0 {
			result := ShouldErrorOnValidatorNotFound(disableCheckHeight - 1)
			require.True(t, result)
		}

		result := ShouldErrorOnValidatorNotFound(0)
		// Will be true if 0 < disableCheckHeight
		if disableCheckHeight > 0 {
			require.True(t, result)
		}
	})

	t.Run("validates boundary conditions", func(t *testing.T) {
		t.Parallel()

		disableCheckHeight := helper.GetDisableValSetCheckHeight()
		tallyFixHeight := helper.GetTallyFixHeight()

		// Exact boundary at disabling the check height.
		// height >= tallyFixHeight || height < disableCheckHeight
		// At disableCheckHeight: NOT < disableCheckHeight, so depends on the first condition
		resultDisable := ShouldErrorOnValidatorNotFound(disableCheckHeight)
		// If disableCheckHeight >= tallyFixHeight, returns true; otherwise false
		if disableCheckHeight >= tallyFixHeight {
			require.True(t, resultDisable)
		} else {
			require.False(t, resultDisable)
		}

		// Exact boundary at tally fix height
		resultTally := ShouldErrorOnValidatorNotFound(tallyFixHeight)
		// Should return true (>= condition)
		require.True(t, resultTally)
	})

	t.Run("handles very large heights", func(t *testing.T) {
		t.Parallel()

		result := ShouldErrorOnValidatorNotFound(1000000000)
		require.True(t, result)
	})

	t.Run("handles negative heights", func(t *testing.T) {
		t.Parallel()

		result := ShouldErrorOnValidatorNotFound(-1)
		// Negative heights should return true (< disableCheckHeight)
		require.True(t, result)

		result = ShouldErrorOnValidatorNotFound(-1000)
		require.True(t, result)
	})
}

func TestErrNoNewHeadersFound(t *testing.T) {
	t.Parallel()

	t.Run("error message is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, ErrNoNewHeadersFound)
		require.Contains(t, ErrNoNewHeadersFound.Error(), "no new headers")
	})

	t.Run("error can be compared", func(t *testing.T) {
		t.Parallel()

		testErr := ErrNoNewHeadersFound
		require.Equal(t, ErrNoNewHeadersFound, testErr)
	})
}
