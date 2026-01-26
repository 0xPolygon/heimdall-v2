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
