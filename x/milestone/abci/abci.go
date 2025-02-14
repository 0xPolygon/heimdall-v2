package abci

import (
	"fmt"
	"math/big"

	"cosmossdk.io/log"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func GenMilestoneProposition(ctx sdk.Context, milestoneKeeper *keeper.Keeper, contractCaller helper.IContractCaller) (*sidetxs.MilestoneProposition, error) {
	hasMilestone, err := milestoneKeeper.HasMilestone(ctx)
	if err != nil {
		return nil, err
	}

	propStartBlock := uint64(0)

	if hasMilestone {
		milestone, err := milestoneKeeper.GetLastMilestone(ctx)
		if err != nil {
			return nil, err
		}

		propStartBlock = milestone.EndBlock + 1
	}

	header, err := contractCaller.GetBorChainBlock(ctx, new(big.Int).SetUint64(propStartBlock))
	if err != nil {
		return nil, err
	}

	return &sidetxs.MilestoneProposition{
		BlockHash: header.Hash().Bytes(),
	}, nil
}

func GetMajorityMilestoneProposition(ctx sdk.Context, validatorSet stakeTypes.ValidatorSet, extVoteInfo []abciTypes.ExtendedVoteInfo, logger log.Logger) (*sidetxs.MilestoneProposition, []byte, error) {
	ac := address.HexCodec{}

	hashToProp := make(map[string]*sidetxs.MilestoneProposition)
	hashToVotingPower := make(map[string]int64)
	hashToAggregatedProposersHash := make(map[string][]byte)
	totalVotingPower := validatorSet.GetTotalVotingPower()

	for _, vote := range extVoteInfo {
		// if not BlockIDFlagCommit, skip that vote, as it doesn't have relevant information
		if vote.BlockIdFlag != cmtTypes.BlockIDFlagCommit {
			continue
		}

		voteExtension := new(sidetxs.VoteExtension)
		if err := voteExtension.Unmarshal(vote.VoteExtension); err != nil {
			return nil, nil, fmt.Errorf("error while unmarshalling vote extension: %w", err)
		}

		if voteExtension.MilestoneProposition == nil {
			continue
		}

		hash := common.BytesToHash(voteExtension.MilestoneProposition.BlockHash).String()
		hashToProp[hash] = voteExtension.MilestoneProposition
		if _, ok := hashToVotingPower[hash]; !ok {
			hashToVotingPower[hash] = 0
		}

		valAddr, err := ac.BytesToString(vote.Validator.Address)
		if err != nil {
			return nil, nil, err
		}

		_, validator := validatorSet.GetByAddress(valAddr)
		if validator == nil {
			return nil, nil, fmt.Errorf("failed to get validator %s", valAddr)
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
	}

	var maxVotingPower int64
	var maxHash string
	for hash, votingPower := range hashToVotingPower {
		if votingPower > maxVotingPower {
			maxVotingPower = votingPower
			maxHash = hash
		}
	}

	// If we have at least 2/3 voting power for one milestone proposition, we return it
	majorityVP := totalVotingPower * 2 / 3
	if maxVotingPower >= majorityVP {
		return hashToProp[maxHash], hashToAggregatedProposersHash[maxHash], nil
	}

	logger.Info("No majority milestone proposition found", "maxVotingPower", maxVotingPower, "majorityVP", majorityVP, "milestonePropositions", hashToProp)

	return nil, nil, nil
}
