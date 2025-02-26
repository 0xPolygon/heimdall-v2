package abci

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"cosmossdk.io/log"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func GenMilestoneProposition(ctx sdk.Context, milestoneKeeper *keeper.Keeper, contractCaller helper.IContractCaller, reqBlock int64) (*sidetxs.MilestoneProposition, error) {
	milestone, err := milestoneKeeper.GetLastMilestone(ctx)
	if err != nil && !errors.Is(err, types.ErrNoMilestoneFound) {
		return nil, err
	}

	logger := ctx.Logger()

	lastMilestoneBlockNumber, err := milestoneKeeper.GetMilestoneBlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	blocksSinceLastMilestone := reqBlock - lastMilestoneBlockNumber

	logger.Debug("blocksSinceLastMilestone", "blocksSinceLastMilestone", blocksSinceLastMilestone)

	propStartBlock := uint64(0)

	if milestone != nil {
		propStartBlock = milestone.EndBlock + 1
	}

	blockHashes, err := getBlockHashes(ctx, propStartBlock, contractCaller)
	if err != nil {
		return nil, err
	}

	milestoneProp := &sidetxs.MilestoneProposition{
		BlockHashes:      blockHashes,
		StartBlockNumber: propStartBlock,
	}

	return milestoneProp, nil
}

func GetMajorityMilestoneProposition(ctx sdk.Context, validatorSet stakeTypes.ValidatorSet, extVoteInfo []abciTypes.ExtendedVoteInfo, logger log.Logger, lastEndBlock *uint64) (*sidetxs.MilestoneProposition, []byte, string, error) {
	ac := address.HexCodec{}

	// Track voting power per block number
	blockVotingPower := make(map[uint64]int64)
	blockHashVotes := make(map[uint64]map[string]int64) // block -> hash -> voting power
	blockToHash := make(map[uint64][]byte)
	validatorVotes := make(map[string]map[uint64][]byte) // validator -> block -> hash
	validatorAddresses := make(map[string][]byte)
	valAddressToVotingPower := make(map[string]int64)

	totalVotingPower := validatorSet.GetTotalVotingPower()
	majorityVP := totalVotingPower*2/3 + 1

	// First pass - collect all votes
	for _, vote := range extVoteInfo {
		if vote.BlockIdFlag != cmtTypes.BlockIDFlagCommit {
			continue
		}

		voteExtension := new(sidetxs.VoteExtension)
		if err := voteExtension.Unmarshal(vote.VoteExtension); err != nil {
			return nil, nil, "", fmt.Errorf("error while unmarshalling vote extension: %w", err)
		}

		if voteExtension.MilestoneProposition == nil {
			continue
		}

		valAddr, err := ac.BytesToString(vote.Validator.Address)
		if err != nil {
			return nil, nil, "", err
		}

		_, validator := validatorSet.GetByAddress(valAddr)
		if validator == nil {
			return nil, nil, "", fmt.Errorf("failed to get validator %s", valAddr)
		}

		validatorAddresses[valAddr] = vote.Validator.Address
		valAddressToVotingPower[valAddr] = validator.VotingPower
		validatorVotes[valAddr] = make(map[uint64][]byte)

		prop := voteExtension.MilestoneProposition
		for i, blockHash := range prop.BlockHashes {
			blockNum := prop.StartBlockNumber + uint64(i)

			// Record this validator's vote for this block
			validatorVotes[valAddr][blockNum] = blockHash

			// Initialize maps if needed
			if _, ok := blockVotingPower[blockNum]; !ok {
				blockVotingPower[blockNum] = 0
				blockHashVotes[blockNum] = make(map[string]int64)
			}

			// Record block hash -> voting power
			hashStr := common.BytesToHash(blockHash).String()
			blockHashVotes[blockNum][hashStr] += validator.VotingPower

			// Track the hash that currently has the most votes for this block
			// Use a deterministic comparison to break ties
			if blockHashVotes[blockNum][hashStr] > blockVotingPower[blockNum] ||
				(blockHashVotes[blockNum][hashStr] == blockVotingPower[blockNum] &&
					hashStr < common.BytesToHash(blockToHash[blockNum]).String()) {
				blockVotingPower[blockNum] = blockHashVotes[blockNum][hashStr]
				blockToHash[blockNum] = blockHash
			}
		}
	}

	// Find blocks with majority support - use a slice for deterministic ordering
	blockNumbers := make([]uint64, 0, len(blockVotingPower))
	for blockNum := range blockVotingPower {
		blockNumbers = append(blockNumbers, blockNum)
	}
	sort.Slice(blockNumbers, func(i, j int) bool {
		return blockNumbers[i] < blockNumbers[j]
	})

	var majorityBlocks []uint64
	for _, blockNum := range blockNumbers {
		if blockVotingPower[blockNum] >= majorityVP {
			majorityBlocks = append(majorityBlocks, blockNum)
		}
	}

	if len(majorityBlocks) == 0 {
		logger.Debug("No blocks found with majority support")
		return nil, nil, "", nil
	}

	startBlock := uint64(0)

	// Check if we have a block that starts exactly from lastEndBlock + 1
	if lastEndBlock != nil {
		startBlock = *lastEndBlock + 1
	}

	// Check if startBlock is in majorityBlocks
	startBlockFound := false
	for _, blockNum := range majorityBlocks {
		if blockNum == startBlock {
			startBlockFound = true
			break
		}
	}

	if !startBlockFound {
		logger.Debug("No blocks with majority support starting at requested block",
			"requestedStartBlock", startBlock)
		return nil, nil, "", nil
	}

	// Find the first continuous range starting from startBlock
	endBlock := startBlock
	for i := 0; i < len(majorityBlocks); i++ {
		if majorityBlocks[i] == startBlock {
			// Find continuous blocks after startBlock
			for j := i + 1; j < len(majorityBlocks); j++ {
				if majorityBlocks[j] == endBlock+1 {
					endBlock = majorityBlocks[j]
				} else {
					break
				}
			}
			break
		}
	}

	blockCount := endBlock - startBlock + 1
	blockHashes := make([][]byte, 0, blockCount)
	for i := startBlock; i <= endBlock; i++ {
		blockHashes = append(blockHashes, blockToHash[i])
	}

	// Find validators who support the entire winning range
	var supportingValidatorList []string
	for valAddr, blocks := range validatorVotes {
		supports := true
		for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
			hash, hasBlock := blocks[blockNum]
			if !hasBlock || !bytes.Equal(hash, blockToHash[blockNum]) {
				supports = false
				break
			}
		}
		if supports {
			supportingValidatorList = append(supportingValidatorList, valAddr)
		}
	}

	// Sort validators deterministically
	sort.Strings(supportingValidatorList)

	// Verify that we still have 2/3 majority after filtering
	totalSupportingPower := int64(0)
	for _, valAddr := range supportingValidatorList {
		totalSupportingPower += valAddressToVotingPower[valAddr]
	}

	if totalSupportingPower < majorityVP {
		logger.Debug("After filtering validators, no range has 2/3 majority support",
			"totalSupportingPower", totalSupportingPower,
			"requiredPower", majorityVP)
		return nil, nil, "", nil
	}

	// Additional sort by voting power (stable to preserve string order when tied)
	sort.SliceStable(supportingValidatorList, func(i, j int) bool {
		return valAddressToVotingPower[supportingValidatorList[i]] > valAddressToVotingPower[supportingValidatorList[j]]
	})

	if len(supportingValidatorList) == 0 {
		return nil, nil, "", fmt.Errorf("no validators support the winning range")
	}

	// Generate aggregated proposers hash from supporting validators
	aggregatedProposersHash := []byte{}
	for _, valAddr := range supportingValidatorList {
		aggregatedProposersHash = crypto.Keccak256(
			aggregatedProposersHash,
			[]byte{'|'},
			validatorAddresses[valAddr],
		)
	}

	// Create final proposition
	proposition := &sidetxs.MilestoneProposition{
		BlockHashes:      blockHashes,
		StartBlockNumber: startBlock,
	}

	logger.Debug("Found majority milestone proposition",
		"startBlock", startBlock,
		"endBlock", endBlock,
		"blockCount", blockCount,
		"supportingValidators", len(supportingValidatorList))

	return proposition, aggregatedProposersHash, supportingValidatorList[0], nil
}

func getBlockHashes(ctx sdk.Context, startBlock uint64, contractCaller helper.IContractCaller) ([][]byte, error) {
	result := make([][]byte, 0)

	headers, err := contractCaller.GetBorChainBlocksInBatch(ctx, int64(startBlock), int64(startBlock+maxBlocksInProposition-1))
	if err != nil {
		return nil, fmt.Errorf("failed to get headers")
	}

	for _, h := range headers {
		result = append(result, h.Hash().Bytes())
	}

	return result, nil
}

const maxBlocksInProposition = 10
