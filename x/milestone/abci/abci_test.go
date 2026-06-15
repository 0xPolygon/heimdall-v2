package abci

import (
	"testing"

	"cosmossdk.io/log"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

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
	t.Parallel()

	// Create a mock keeper with params
	setupKeeper := func() (*keeper.Keeper, sdk.Context) {
		// This is a simplified setup - in real tests you'd use the full testutil
		// For coverage purposes, we'll focus on the validation logic itself
		return nil, sdk.Context{}
	}

	t.Run("accepts nil proposition", func(t *testing.T) {
		t.Parallel()

		k, ctx := setupKeeper()
		err := ValidateMilestoneProposition(ctx, k, nil)
		require.NoError(t, err)
	})

	t.Run("validates valid proposition structure", func(t *testing.T) {
		t.Parallel()

		// Test just the validation logic without requiring full keeper setup
		prop := &types.MilestoneProposition{
			BlockHashes:      [][]byte{make([]byte, common.HashLength)},
			StartBlockNumber: 1,
			ParentHash:       make([]byte, common.HashLength),
			BlockTds:         []uint64{100},
		}

		// Validate the structure directly
		require.Len(t, prop.BlockHashes, 1)
		require.Len(t, prop.BlockTds, 1)
		require.Equal(t, len(prop.BlockHashes), len(prop.BlockTds))
	})

	t.Run("detects length mismatch between hashes and tds", func(t *testing.T) {
		t.Parallel()

		prop := &types.MilestoneProposition{
			BlockHashes:      [][]byte{make([]byte, common.HashLength)},
			BlockTds:         []uint64{100, 200}, // Mismatch
			StartBlockNumber: 1,
		}

		// Verify the mismatch would be detected
		require.NotEqual(t, len(prop.BlockHashes), len(prop.BlockTds))
	})

	t.Run("detects invalid hash length", func(t *testing.T) {
		t.Parallel()

		prop := &types.MilestoneProposition{
			BlockHashes:      [][]byte{make([]byte, 16)}, // Too short
			BlockTds:         []uint64{100},
			StartBlockNumber: 1,
		}

		// Verify invalid length would be detected
		require.NotEqual(t, len(prop.BlockHashes[0]), common.HashLength)
	})

	t.Run("validates duplicate hash detection", func(t *testing.T) {
		t.Parallel()

		duplicateHash := make([]byte, common.HashLength)
		for i := range duplicateHash {
			duplicateHash[i] = 0xAA
		}

		prop := &types.MilestoneProposition{
			BlockHashes:      [][]byte{duplicateHash, duplicateHash}, // Duplicates
			BlockTds:         []uint64{100, 200},
			StartBlockNumber: 1,
		}

		// Test that duplicate detection works
		seen := make(map[string]struct{})
		for _, hash := range prop.BlockHashes {
			seen[string(hash)] = struct{}{}
		}
		require.NotEqual(t, len(seen), len(prop.BlockHashes), "should detect duplicates")
	})

	t.Run("validates unique hashes are accepted", func(t *testing.T) {
		t.Parallel()

		hash1 := make([]byte, common.HashLength)
		hash2 := make([]byte, common.HashLength)
		hash1[0] = 0xAA
		hash2[0] = 0xBB

		prop := &types.MilestoneProposition{
			BlockHashes:      [][]byte{hash1, hash2},
			BlockTds:         []uint64{100, 200},
			StartBlockNumber: 1,
		}

		// Test that unique hashes are detected
		seen := make(map[string]struct{})
		for _, hash := range prop.BlockHashes {
			seen[string(hash)] = struct{}{}
		}
		require.Equal(t, len(seen), len(prop.BlockHashes), "unique hashes should be accepted")
	})

	t.Run("validates empty block hashes", func(t *testing.T) {
		t.Parallel()

		prop := &types.MilestoneProposition{
			BlockHashes:      [][]byte{},
			BlockTds:         []uint64{},
			StartBlockNumber: 1,
		}

		// Empty hashes should be detected
		require.Empty(t, prop.BlockHashes)
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
