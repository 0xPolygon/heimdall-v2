package abci

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
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

func GenMilestoneProposition(ctx sdk.Context, milestoneKeeper *keeper.Keeper, contractCaller helper.IContractCaller) (*types.MilestoneProposition, error) {
	milestone, err := milestoneKeeper.GetLastMilestone(ctx)
	if err != nil && !errors.Is(err, types.ErrNoMilestoneFound) {
		return nil, err
	}

	propStartBlock := uint64(0)

	var lastMilestoneHash []byte
	var lastMilestoneBlockNumber uint64

	if milestone != nil {
		propStartBlock = milestone.EndBlock + 1

		latestHeader, err := contractCaller.GetBorChainBlock(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest header")
		}

		params, err := milestoneKeeper.GetParams(ctx)
		if err != nil {
			return nil, err
		}

		if isFastForwardMilestone(latestHeader.Number.Uint64(), milestone.EndBlock, params.FfMilestoneThreshold) {
			propStartBlock = getFastForwardMilestoneStartBlock(latestHeader.Number.Uint64(), milestone.EndBlock, params.FfMilestoneBlockInterval)
		}

		lastMilestoneHash = milestone.Hash
		lastMilestoneBlockNumber = milestone.EndBlock
	}

	params, err := milestoneKeeper.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	parentHash, blockHashes, err := getBlockHashes(ctx, propStartBlock, params.MaxMilestonePropositionLength, lastMilestoneHash, lastMilestoneBlockNumber, contractCaller)
	if err != nil {
		return nil, err
	}

	if err := validateMilestonePropositionFork(parentHash, lastMilestoneHash); err != nil {
		return nil, err
	}

	milestoneProp := &types.MilestoneProposition{
		BlockHashes:      blockHashes,
		StartBlockNumber: propStartBlock,
		ParentHash:       parentHash,
	}

	return milestoneProp, nil
}

func isFastForwardMilestone(latestHeaderNumber, latestMilestoneEndBlock, ffMilestoneThreshold uint64) bool {
	return latestHeaderNumber > latestMilestoneEndBlock && latestHeaderNumber-latestMilestoneEndBlock > ffMilestoneThreshold
}

func getFastForwardMilestoneStartBlock(latestHeaderNumber, latestMilestoneEndBlock, ffMilestoneBlockInterval uint64) uint64 {
	latestHeaderMilestoneDistanceInBlocks := ((latestHeaderNumber - latestMilestoneEndBlock) / ffMilestoneBlockInterval) * ffMilestoneBlockInterval
	return latestMilestoneEndBlock + latestHeaderMilestoneDistanceInBlocks + 1
}

func GetMajorityMilestoneProposition(
	validatorSet *stakeTypes.ValidatorSet,
	extVoteInfo []abciTypes.ExtendedVoteInfo,
	logger log.Logger,
	lastEndBlock *uint64,
	lastEndBlockHash []byte,
) (*types.MilestoneProposition, []byte, string, error) {
	ac := address.HexCodec{}

	// Track voting power per block number
	blockVotingPower := make(map[uint64]int64)
	blockHashVotes := make(map[uint64]map[string]int64) // block -> hash -> voting power
	blockToHash := make(map[uint64][]byte)
	validatorVotes := make(map[string]map[uint64][]byte) // validator -> block -> hash
	validatorAddresses := make(map[string][]byte)
	valAddressToVotingPower := make(map[string]int64)
	parentHashes := make(map[string]struct{})
	parentHashToVotingPower := make(map[string]int64)

	// Track which validators we've already processed to prevent duplicate votes
	processedValidators := make(map[string]bool)

	totalVotingPower := validatorSet.GetTotalVotingPower()
	majorityVP := totalVotingPower*2/3 + 1

	getParentChildKey := func(parent, child string) string {
		return fmt.Sprintf("%s-%s", parent, child)
	}

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

		// Skip if we've already processed a vote from this validator
		if processedValidators[valAddr] {
			logger.Debug("Skipping duplicate vote from validator", "validator", valAddr)
			continue
		}

		// Mark this validator as processed
		processedValidators[valAddr] = true

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

			key := getParentChildKey(common.BytesToHash(prop.ParentHash).String(), common.BytesToHash(blockHash).String())
			parentHashToVotingPower[key] += validator.VotingPower
		}
		parentHashes[common.BytesToHash(prop.ParentHash).String()] = struct{}{}
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

	var majorityParentHash string
	isParentHashMajority := false

	for parentHash := range parentHashes {
		key := getParentChildKey(parentHash, common.BytesToHash(blockToHash[majorityBlocks[0]]).String())
		if parentHashToVotingPower[key] >= majorityVP {
			isParentHashMajority = true
			majorityParentHash = parentHash
			break
		}
	}

	if !isParentHashMajority {
		logger.Debug("No parent hash found with majority support")
		return nil, nil, "", nil
	}

	if majorityParentHash != common.BytesToHash(lastEndBlockHash).String() {
		logger.Debug("Parent hash does not match last end block hash",
			"majorityParentHash", majorityParentHash,
			"lastEndBlockHash", common.BytesToHash(lastEndBlockHash).String())
		return nil, nil, "", nil
	}

	startBlock := uint64(0)

	// Check if we have a block that starts exactly from lastEndBlock + 1
	if lastEndBlock != nil {
		startBlock = *lastEndBlock + 1

		if majorityBlocks[0] > startBlock {
			startBlock = majorityBlocks[0]
		}
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
	var aggregatedProposersHash []byte
	for _, valAddr := range supportingValidatorList {
		aggregatedProposersHash = crypto.Keccak256(
			aggregatedProposersHash,
			[]byte{'|'},
			validatorAddresses[valAddr],
		)
	}

	// Create final proposition
	proposition := &types.MilestoneProposition{
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

func getBlockHashes(ctx sdk.Context, startBlock, maxBlocksInProposition uint64, lastMilestoneHash []byte, lastMilestoneBlock uint64, contractCaller helper.IContractCaller) ([]byte, [][]byte, error) {
	headers, err := contractCaller.GetBorChainBlocksInBatch(ctx, int64(startBlock), int64(startBlock+maxBlocksInProposition-1))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get headers: %w", err)
	}

	result := make([][]byte, 0, len(headers))

	var parentHash []byte
	if len(headers) > 0 && len(lastMilestoneHash) > 0 {
		parentHash = headers[0].ParentHash.Bytes()
		if startBlock-lastMilestoneBlock > 1 {
			header, err := contractCaller.GetBorChainBlock(ctx, big.NewInt(int64(lastMilestoneBlock+1)))
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get headers: %w", err)
			}

			parentHash = header.ParentHash.Bytes()
		}
	}

	for _, h := range headers {
		result = append(result, h.Hash().Bytes())
	}

	return parentHash, result, nil
}

func validateMilestonePropositionFork(parentHash []byte, lastMilestoneHash []byte) error {
	if len(parentHash) > 0 && len(lastMilestoneHash) > 0 {
		if !bytes.Equal(parentHash, lastMilestoneHash) {
			return fmt.Errorf("first block parent hash does not match last milestone hash")
		}
	}
	return nil
}

func ValidateMilestoneProposition(ctx sdk.Context, milestoneKeeper *keeper.Keeper, milestoneProp *types.MilestoneProposition) error {
	if milestoneProp == nil {
		return nil
	}

	params, err := milestoneKeeper.GetParams(ctx)
	if err != nil {
		return err
	}

	if len(milestoneProp.BlockHashes) > int(params.MaxMilestonePropositionLength) {
		return fmt.Errorf("too many blocks in proposition")
	}

	if len(milestoneProp.BlockHashes) == 0 {
		return fmt.Errorf("no blocks in proposition")
	}

	for _, blockHash := range milestoneProp.BlockHashes {
		if len(blockHash) != common.HashLength {
			return fmt.Errorf("invalid block hash length")
		}
	}

	return nil
}
