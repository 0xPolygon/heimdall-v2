package abci

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"

	"cosmossdk.io/log"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var pendingMilestoneProposition *sidetxs.MilestoneProposition

func GenMilestoneProposition(ctx sdk.Context, milestoneKeeper *keeper.Keeper, contractCaller helper.IContractCaller, reqBlock int64) (*sidetxs.MilestoneProposition, error) {
	milestone, err := milestoneKeeper.GetLastMilestone(ctx)
	if err != nil && err != types.ErrNoMilestoneFound {
		return nil, err
	}

	pendingMilestone := GetPendingMilestoneProposition()

	logger := ctx.Logger()

	lastMilestoneBlockNumber, err := milestoneKeeper.GetMilestoneBlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	blocksSinceLastMilestone := reqBlock - lastMilestoneBlockNumber

	logger.Debug("blocksSinceLastMilestone", "blocksSinceLastMilestone", blocksSinceLastMilestone)

	// TODO: make blocksSinceLastMilestone limit configurable
	propStartBlock := uint64(0)
	if pendingMilestone != nil && milestone != nil && blocksSinceLastMilestone > 6 {
		propStartBlock = milestone.EndBlock + 1
	} else {
		if pendingMilestone != nil {
			propStartBlock = pendingMilestone.StartBlockNumber + uint64(len(pendingMilestone.BlockHashes))
		} else if milestone != nil {
			propStartBlock = milestone.EndBlock + 1
		} else {
			propStartBlock = 0
		}
	}

	blockHashes, err := getBlockHashes(ctx, propStartBlock, contractCaller)
	if err != nil {
		return nil, err
	}

	milestoneProp := &sidetxs.MilestoneProposition{
		BlockHashes:      blockHashes,
		StartBlockNumber: propStartBlock,
	}

	SetPendingMilestoneProposition(milestoneProp)

	return milestoneProp, nil
}

func GetMajorityMilestoneProposition(ctx sdk.Context, validatorSet stakeTypes.ValidatorSet, extVoteInfo []abciTypes.ExtendedVoteInfo, logger log.Logger) (*sidetxs.MilestoneProposition, []byte, string, error) {
	ac := address.HexCodec{}

	hashToProp := make(map[string]*sidetxs.MilestoneProposition)
	hashToVotingPower := make(map[string]int64)
	hashToAggregatedProposersHash := make(map[string][]byte)
	hashVoters := make(map[string][]string)
	valAddressToVotingPower := make(map[string]int64)
	totalVotingPower := validatorSet.GetTotalVotingPower()

	for _, vote := range extVoteInfo {
		// if not BlockIDFlagCommit, skip that vote, as it doesn't have relevant information
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

		valAddressToVotingPower[valAddr] = validator.VotingPower

		blockHashesCount := uint64(len(voteExtension.MilestoneProposition.BlockHashes))
		prefix := make([][]byte, 0)
		for i := uint64(0); i < blockHashesCount; i++ {
			prefix = append(prefix, voteExtension.MilestoneProposition.BlockHashes[i])

			prefixCopy := make([][]byte, len(prefix))
			copy(prefixCopy, prefix)

			startBlockBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(startBlockBytes, voteExtension.MilestoneProposition.StartBlockNumber)
			hashInput := bytes.Join(append(prefixCopy, startBlockBytes), []byte{'|'})
			hash := common.BytesToHash(hashInput).String()

			hashToProp[hash] = &sidetxs.MilestoneProposition{
				BlockHashes:      prefixCopy,
				StartBlockNumber: voteExtension.MilestoneProposition.StartBlockNumber,
			}
			if _, ok := hashToVotingPower[hash]; !ok {
				hashToVotingPower[hash] = 0
			}

			hashToVotingPower[hash] += validator.VotingPower

			if _, ok := hashToAggregatedProposersHash[hash]; !ok {
				hashToAggregatedProposersHash[hash] = []byte{}
			}

			hashToAggregatedProposersHash[hash] = crypto.Keccak256(
				hashToAggregatedProposersHash[hash],
				[]byte{'|'},
				vote.Validator.Address,
			)

			if _, ok := hashVoters[hash]; !ok {
				hashVoters[hash] = []string{}
			}

			hashVoters[hash] = append(hashVoters[hash], valAddr)
		}
	}

	var maxVotingPower int64
	var maxHash string
	for hash, votingPower := range hashToVotingPower {
		if votingPower > maxVotingPower {
			maxVotingPower = votingPower
			maxHash = hash
		} else if votingPower == maxVotingPower &&
			len(hashToProp[hash].BlockHashes) > len(hashToProp[maxHash].BlockHashes) {
			maxHash = hash
		}
	}

	// If we have at least 2/3 voting power for one milestone proposition, we return it
	majorityVP := totalVotingPower * 2 / 3
	if maxVotingPower >= majorityVP {

		voters := hashVoters[maxHash]
		sort.SliceStable(voters, func(i, j int) bool {
			return valAddressToVotingPower[voters[i]] > valAddressToVotingPower[voters[j]]
		})

		if len(voters) == 0 {
			return nil, nil, "", fmt.Errorf("no voters found for majority milestone proposition")
		}

		return hashToProp[maxHash], hashToAggregatedProposersHash[maxHash], voters[0], nil
	}

	logger.Debug("No majority milestone proposition found", "maxVotingPower", maxVotingPower, "majorityVP", majorityVP, "milestonePropositions", hashToProp)

	return nil, nil, "", nil
}

func SetPendingMilestoneProposition(prop *sidetxs.MilestoneProposition) {
	pendingMilestoneProposition = prop
}

func GetPendingMilestoneProposition() *sidetxs.MilestoneProposition {
	return pendingMilestoneProposition
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
