package abci

import (
	"testing"

	"cosmossdk.io/log"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
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
