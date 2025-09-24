package app

import (
	"testing"

	abciTypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	helpermocks "github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestCheckAndAddFutureSpan(t *testing.T) {
	_, app, ctx, _ := SetupAppWithABCIctxAndValidators(t, 3)

	// Get validators to create proper span
	validators := app.StakeKeeper.GetAllValidators(ctx)
	valSlice := make([]*stakeTypes.Validator, len(validators))
	for i := range validators {
		valSlice[i] = validators[i]
	}
	valSet := stakeTypes.ValidatorSet{Validators: valSlice}

	// Create validators for selected producers
	selectedProducers := make([]stakeTypes.Validator, len(validators))
	for i, val := range validators {
		selectedProducers[i] = *val
	}

	lastSpan := borTypes.Span{
		Id:                1,
		StartBlock:        100,
		EndBlock:          200,
		BorChainId:        "1",
		ValidatorSet:      valSet,
		SelectedProducers: selectedProducers,
	}
	err := app.BorKeeper.AddNewSpan(ctx, &lastSpan)
	require.NoError(t, err)

	producerValID := selectedProducers[0].ValId
	// The producer is not in the supporting set.
	supportingValidatorIDs := make(map[uint64]struct{})
	for _, v := range validators {
		if v.ValId != producerValID {
			supportingValidatorIDs[v.ValId] = struct{}{}
		}
	}

	t.Run("condition false", func(t *testing.T) {
		majorityMilestone := &milestoneTypes.MilestoneProposition{
			StartBlockNumber: 50, // This will make the condition false
			BlockHashes:      [][]byte{[]byte("hash1")},
		}

		err := app.checkAndAddFutureSpan(ctx, majorityMilestone, lastSpan, supportingValidatorIDs)
		require.NoError(t, err)

		// Check no new span was added
		currentLastSpan, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, lastSpan.Id, currentLastSpan.Id)
	})

	t.Run("condition true", func(t *testing.T) {
		majorityMilestone := &milestoneTypes.MilestoneProposition{
			StartBlockNumber: 150, // This will make the condition true
			BlockHashes:      [][]byte{[]byte("hash1")},
		}

		helper.SetRioHeight(int64(lastSpan.EndBlock + 1))

		// Mock IContractCaller to return the lowercase address.
		mockCaller := new(helpermocks.IContractCaller)
		mockCaller.On("GetBorChainBlockAuthor", mock.Anything, mock.Anything).Return([]common.Address{common.HexToAddress(validators[0].Signer)}, nil)
		app.BorKeeper.SetContractCaller(mockCaller)

		params, err := app.BorKeeper.GetParams(ctx)
		require.NoError(t, err)

		// Set up producer votes so that producer selection can work
		if len(validators) > 1 {
			// All validators vote for the same candidate to ensure consensus
			var consensusCandidateID uint64
			for _, v := range validators {
				if v.ValId != producerValID {
					consensusCandidateID = v.ValId
					break
				}
			}

			allValidatorIDs := make(map[uint64]struct{})
			for _, val := range validators {
				allValidatorIDs[val.ValId] = struct{}{}
				producerVotes := borTypes.ProducerVotes{Votes: []uint64{consensusCandidateID}}
				err := app.BorKeeper.SetProducerVotes(ctx, val.ValId, producerVotes)
				require.NoError(t, err)
			}

			// Set up producer performance scores
			err := app.BorKeeper.UpdateValidatorPerformanceScore(ctx, allValidatorIDs, 1)
			require.NoError(t, err)

			// Set up minimal span state
			params, err := app.BorKeeper.GetParams(ctx)
			require.NoError(t, err)
			params.ProducerCount = 1
			app.BorKeeper.SetParams(ctx, params)
		}

		err = app.checkAndAddFutureSpan(ctx, majorityMilestone, lastSpan, supportingValidatorIDs)
		require.NoError(t, err)

		// Check that a new span was created
		currentLastSpan, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, lastSpan.Id+1, currentLastSpan.Id, "a new span should be created with incremented ID")
		require.Equal(t, lastSpan.EndBlock+1, currentLastSpan.StartBlock, "new span should start after the last span")
		require.Equal(t, currentLastSpan.StartBlock+params.SpanDuration-1, currentLastSpan.EndBlock, "new span should have the exact span duration defined in params")
	})
}

func TestCheckAndRotateCurrentSpan(t *testing.T) {
	t.Run("condition false - diff too small", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctxAndValidators(t, 3)

		lastMilestone := &milestoneTypes.Milestone{EndBlock: 100}
		app.MilestoneKeeper.AddMilestone(ctx, *lastMilestone)
		lastMilestoneBlock := uint64(50)
		app.MilestoneKeeper.SetLastMilestoneBlock(ctx, lastMilestoneBlock)

		// Get validators to create proper span
		validators := app.StakeKeeper.GetAllValidators(ctx)
		valSlice := make([]*stakeTypes.Validator, len(validators))
		for i := range validators {
			valSlice[i] = validators[i]
		}
		valSet := stakeTypes.ValidatorSet{Validators: valSlice}

		// Create validators for selected producers
		selectedProducers := make([]stakeTypes.Validator, len(validators))
		for i, val := range validators {
			selectedProducers[i] = *val
		}

		lastSpan := borTypes.Span{
			Id:                1,
			StartBlock:        90,
			EndBlock:          190,
			BorChainId:        "1",
			ValidatorSet:      valSet,
			SelectedProducers: selectedProducers,
		}
		err := app.BorKeeper.AddNewSpan(ctx, &lastSpan)
		require.NoError(t, err)

		ctx = ctx.WithBlockHeight(int64(lastMilestoneBlock) + ChangeProducerThreshold) // diff == ChangeProducerThreshold

		err = app.checkAndRotateCurrentSpan(ctx)
		require.NoError(t, err)

		currentLastSpan, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, lastSpan.Id, currentLastSpan.Id)
	})

	t.Run("condition false - not veblop", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctxAndValidators(t, 3)

		lastMilestone := &milestoneTypes.Milestone{EndBlock: 100}
		app.MilestoneKeeper.AddMilestone(ctx, *lastMilestone)
		lastMilestoneBlock := uint64(50)
		app.MilestoneKeeper.SetLastMilestoneBlock(ctx, lastMilestoneBlock)

		// Get validators to create proper span
		validators := app.StakeKeeper.GetAllValidators(ctx)
		valSlice := make([]*stakeTypes.Validator, len(validators))
		for i := range validators {
			valSlice[i] = validators[i]
		}
		valSet := stakeTypes.ValidatorSet{Validators: valSlice}

		// Create validators for selected producers
		selectedProducers := make([]stakeTypes.Validator, len(validators))
		for i, val := range validators {
			selectedProducers[i] = *val
		}

		lastSpan := borTypes.Span{
			Id:                1,
			StartBlock:        90,
			EndBlock:          190,
			BorChainId:        "1",
			ValidatorSet:      valSet,
			SelectedProducers: selectedProducers,
		}
		err := app.BorKeeper.AddNewSpan(ctx, &lastSpan)
		require.NoError(t, err)

		ctx = ctx.WithBlockHeight(int64(lastMilestoneBlock) + ChangeProducerThreshold + 1)
		helper.SetRioHeight(int64(lastMilestone.EndBlock + 2)) // Makes IsRio false

		err = app.checkAndRotateCurrentSpan(ctx)
		require.NoError(t, err)

		currentLastSpan, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, lastSpan.Id, currentLastSpan.Id)

		helper.SetRioHeight(0) // reset
	})

	t.Run("condition true", func(t *testing.T) {
		_, app, ctx, _ := SetupAppWithABCIctxAndValidators(t, 3)

		lastMilestone := &milestoneTypes.Milestone{
			EndBlock:   100,
			BorChainId: "1",
		}
		app.MilestoneKeeper.AddMilestone(ctx, *lastMilestone)
		lastMilestoneBlock := uint64(50)
		app.MilestoneKeeper.SetLastMilestoneBlock(ctx, lastMilestoneBlock)

		validators := app.StakeKeeper.GetAllValidators(ctx)
		valSlice := make([]*stakeTypes.Validator, len(validators))
		for i := range validators {
			valSlice[i] = validators[i]
		}
		valSet := stakeTypes.ValidatorSet{Validators: valSlice}

		// Create validators for selected producers
		selectedProducers := make([]stakeTypes.Validator, len(validators))
		for i, val := range validators {
			selectedProducers[i] = *val
		}

		lastSpan := borTypes.Span{
			Id:                1,
			StartBlock:        90,
			EndBlock:          190,
			BorChainId:        "1",
			ValidatorSet:      valSet,
			SelectedProducers: selectedProducers,
		}
		err := app.BorKeeper.AddNewSpan(ctx, &lastSpan)
		require.NoError(t, err)

		initialActiveProducers := make(map[uint64]struct{})
		for _, val := range validators {
			initialActiveProducers[val.ValId] = struct{}{}
		}

		// Add a few extra producer IDs to ensure we have candidates after current producer is removed
		initialActiveProducers[1] = struct{}{}
		initialActiveProducers[2] = struct{}{}

		app.BorKeeper.UpdateLatestActiveProducer(ctx, initialActiveProducers)
		app.BorKeeper.AddLatestFailedProducer(ctx, uint64(99)) // some other producer

		// Set up comprehensive producer votes and state for successful producer selection
		if len(validators) > 0 {
			// For 3 validators with voting power 100 each:
			// totalPotentialProducers = 3
			// Max possible weighted vote at position 1: totalPotentialProducers * maxVotingPower = 3 * 100 = 300
			// Required threshold: (300 * 2/3) + 1 = 201
			// If all 3 validators vote for same candidate at position 1: 3 * 100 = 300 > 201 ✓

			// Use actual validator IDs - find one that's not the current producer
			var consensusCandidate uint64
			for _, val := range validators {
				// Current producer is validators[0], so use any other validator
				if val.ValId != validators[0].ValId {
					consensusCandidate = val.ValId
					break
				}
			}
			if consensusCandidate == 0 {
				// Fallback: use second validator if available
				if len(validators) > 1 {
					consensusCandidate = validators[1].ValId
				}
			}

			// Set producer votes - all validators vote for the same consensus candidate
			for _, val := range validators {
				// All validators vote for consensus candidate in first position, then fill with other validator IDs
				var votes []uint64
				votes = append(votes, consensusCandidate) // First choice - consensus candidate
				for j, otherVal := range validators {
					if otherVal.ValId != consensusCandidate && len(votes) < 3 {
						votes = append(votes, otherVal.ValId)
					}
					if len(votes) >= 3 {
						break
					}
					_ = j // avoid unused variable
				}

				producerVotes := borTypes.ProducerVotes{Votes: votes}
				err := app.BorKeeper.SetProducerVotes(ctx, val.ValId, producerVotes)
				require.NoError(t, err)

				// Include this validator in the initial active producers
				initialActiveProducers[val.ValId] = struct{}{}
			}

			// Ensure bor params allow for proper producer selection
			params, err := app.BorKeeper.GetParams(ctx)
			require.NoError(t, err)
			params.ProducerCount = 3  // Allow 3 producers
			params.SpanDuration = 100 // Set reasonable span duration
			app.BorKeeper.SetParams(ctx, params)
		}

		ctx = ctx.WithBlockHeight(int64(lastMilestoneBlock) + ChangeProducerThreshold + 1) // diff > ChangeProducerThreshold
		helper.SetRioHeight(int64(lastMilestone.EndBlock + 1))                             // Makes IsRio true

		// Mock IContractCaller with proper producer mapping
		mockCaller := new(helpermocks.IContractCaller)
		producerSignerStr := validators[0].Signer
		producerSignerAddr := common.HexToAddress(producerSignerStr)
		mockCaller.On("GetBorChainBlockAuthor", mock.Anything, lastMilestone.EndBlock+1).Return(&producerSignerAddr, nil)
		app.BorKeeper.SetContractCaller(mockCaller)

		// Call the function
		err = app.checkAndRotateCurrentSpan(ctx)
		require.NoError(t, err)

		// Assert that a new span was actually created
		currentLastSpan, err := app.BorKeeper.GetLastSpan(ctx)
		require.NoError(t, err)
		require.Equal(t, lastSpan.Id+1, currentLastSpan.Id, "a new span should be created with incremented ID")
		require.Equal(t, lastMilestone.EndBlock+1, currentLastSpan.StartBlock, "new span should start after the last milestone")
		require.Equal(t, lastSpan.EndBlock, currentLastSpan.EndBlock, "new span will have the same end block as the last span")

		// Verify other expected state changes
		newLastMilestoneBlock, err := app.MilestoneKeeper.GetLastMilestoneBlock(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(ctx.BlockHeight())+SpanRotationBuffer, newLastMilestoneBlock, "last milestone block should be updated")

		failedProducers, err := app.BorKeeper.GetLatestFailedProducer(ctx)
		require.NoError(t, err)

		currentProducerID := validators[0].ValId
		_, isFailed := failedProducers[currentProducerID]
		require.True(t, isFailed, "current producer should be added to failed list")
	})
}

// TestPreBlockerSpanRotationWithMinorityMilestone tests that span rotation is skipped
// when there's at least 1/3 voting power supporting a new milestone
func TestPreBlockerSpanRotationWithMinorityMilestone(t *testing.T) {
	_, app, ctx, validatorPrivKeys := SetupAppWithABCIctxAndValidators(t, 10)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Set up consensus params to enable vote extensions
	params := cmtTypes.ConsensusParams{
		Abci: &cmtTypes.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	}
	ctx = ctx.WithConsensusParams(params)

	// Setup initial state with milestone and span
	milestone := milestoneTypes.Milestone{
		MilestoneId: "1",
		StartBlock:  0,
		EndBlock:    100,
		Hash:        common.HexToHash("0x1234").Bytes(),
	}
	err := app.MilestoneKeeper.AddMilestone(ctx, milestone)
	require.NoError(t, err)

	// Set last milestone block - this is needed for checkAndRotateCurrentSpan to work
	err = app.MilestoneKeeper.SetLastMilestoneBlock(ctx, uint64(milestone.EndBlock))
	require.NoError(t, err)

	span := &borTypes.Span{
		Id:         1,
		StartBlock: 1,
		EndBlock:   200,
		ValidatorSet: stakeTypes.ValidatorSet{
			Validators: validators,
			Proposer:   validators[0],
		},
		SelectedProducers: []stakeTypes.Validator{*validators[0]},
		BorChainId:        "test",
	}
	err = app.BorKeeper.AddNewSpan(ctx, span)
	require.NoError(t, err)

	// Set up mock contract caller
	mockCaller := new(mocks.IContractCaller)
	producerSigner := common.HexToAddress(validators[0].Signer)
	mockCaller.On("GetBorChainBlockAuthor", mock.Anything, mock.Anything).Return(&producerSigner, nil)
	app.BorKeeper.SetContractCaller(mockCaller)

	// Set context to trigger span rotation conditions
	blockHeight := int64(milestone.EndBlock) + ChangeProducerThreshold + 1
	ctx = ctx.WithBlockHeight(blockHeight)
	// Set rio height to be at or before milestone.EndBlock+1 to ensure IsRio check passes
	helper.SetRioHeight(int64(milestone.EndBlock + 1))

	// Create vote extensions with 40% voting power supporting a new milestone
	// This is more than 1/3 but less than 2/3
	voteExtensions := createVoteExtensionsWithPartialSupport(t, validators, validatorPrivKeys, &milestone, 40, blockHeight-1)

	// Create ExtendedCommitInfo from vote extensions
	extCommit := &abciTypes.ExtendedCommitInfo{
		Round: 0,
		Votes: voteExtensions,
	}
	extCommitBytes, err := extCommit.Marshal()
	require.NoError(t, err)

	req := &abciTypes.RequestFinalizeBlock{
		Height:          ctx.BlockHeight(),
		Txs:             [][]byte{extCommitBytes, []byte("dummy-tx")}, // Add dummy tx to avoid slice bounds error
		ProposerAddress: common.FromHex(validators[0].Signer),
	}

	// Execute PreBlocker
	_, err = app.PreBlocker(ctx, req)
	require.NoError(t, err)

	// Verify that span was NOT rotated
	currentSpan, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.Equal(t, span.Id, currentSpan.Id, "Span should not have been rotated when 1/3+ voting power supports a milestone")
}

// TestPreBlockerSpanRotationWithoutMinorityMilestone tests that span rotation occurs
// when there's less than 1/3 voting power supporting a new milestone
func TestPreBlockerSpanRotationWithoutMinorityMilestone(t *testing.T) {
	_, app, ctx, validatorPrivKeys := SetupAppWithABCIctxAndValidators(t, 10)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Set up consensus params to enable vote extensions
	params := cmtTypes.ConsensusParams{
		Abci: &cmtTypes.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	}
	ctx = ctx.WithConsensusParams(params)

	// Setup initial state with milestone and span
	milestone := milestoneTypes.Milestone{
		MilestoneId: "1",
		StartBlock:  0,
		EndBlock:    100,
		Hash:        common.HexToHash("0x1234").Bytes(),
	}
	err := app.MilestoneKeeper.AddMilestone(ctx, milestone)
	require.NoError(t, err)

	// Set last milestone block - this is needed for checkAndRotateCurrentSpan to work
	err = app.MilestoneKeeper.SetLastMilestoneBlock(ctx, uint64(milestone.EndBlock))
	require.NoError(t, err)

	span := &borTypes.Span{
		Id:         1,
		StartBlock: 1,
		EndBlock:   200,
		ValidatorSet: stakeTypes.ValidatorSet{
			Validators: validators,
			Proposer:   validators[0],
		},
		SelectedProducers: []stakeTypes.Validator{*validators[0]},
		BorChainId:        "test",
	}
	err = app.BorKeeper.AddNewSpan(ctx, span)
	require.NoError(t, err)

	// Set up mock contract caller
	mockCaller := new(mocks.IContractCaller)
	producerSigner := common.HexToAddress(validators[0].Signer)
	mockCaller.On("GetBorChainBlockAuthor", mock.Anything, mock.Anything).Return(&producerSigner, nil)
	app.BorKeeper.SetContractCaller(mockCaller)

	// Set context to trigger span rotation conditions
	blockHeight := int64(milestone.EndBlock) + ChangeProducerThreshold + 1
	ctx = ctx.WithBlockHeight(blockHeight)
	// Set rio height to be at or before milestone.EndBlock+1 to ensure IsRio check passes
	helper.SetRioHeight(int64(milestone.EndBlock + 1))

	// Create vote extensions with only 20% voting power supporting a new milestone
	// This is less than 1/3
	voteExtensions := createVoteExtensionsWithPartialSupport(t, validators, validatorPrivKeys, &milestone, 20, blockHeight-1)

	// Create ExtendedCommitInfo from vote extensions
	extCommit := &abciTypes.ExtendedCommitInfo{
		Round: 0,
		Votes: voteExtensions,
	}
	extCommitBytes, err := extCommit.Marshal()
	require.NoError(t, err)

	req := &abciTypes.RequestFinalizeBlock{
		Height:          ctx.BlockHeight(),
		Txs:             [][]byte{extCommitBytes, []byte("dummy-tx")}, // Add dummy tx to avoid slice bounds error
		ProposerAddress: common.FromHex(validators[0].Signer),
	}

	// Execute PreBlocker
	_, err = app.PreBlocker(ctx, req)
	require.NoError(t, err)

	// Verify that span WAS rotated
	currentSpan, err := app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)
	require.NotEqual(t, span.Id, currentSpan.Id, "Span should have been rotated when less than 1/3 voting power supports a milestone")
}

// TestPreBlockerSpanRotationWithMajorityMilestone tests that span rotation is skipped
// when there's a 2/3 majority milestone (existing behavior)
func TestPreBlockerSpanRotationWithMajorityMilestone(t *testing.T) {
	_, app, ctx, validatorPrivKeys := SetupAppWithABCIctxAndValidators(t, 10)
	validators := app.StakeKeeper.GetAllValidators(ctx)

	// Set up consensus params to enable vote extensions
	params := cmtTypes.ConsensusParams{
		Abci: &cmtTypes.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	}
	ctx = ctx.WithConsensusParams(params)

	// Setup initial state with milestone and span
	milestone := milestoneTypes.Milestone{
		MilestoneId: "1",
		StartBlock:  0,
		EndBlock:    100,
		Hash:        common.HexToHash("0x1234").Bytes(),
	}
	err := app.MilestoneKeeper.AddMilestone(ctx, milestone)
	require.NoError(t, err)

	// Set last milestone block - this is needed for checkAndRotateCurrentSpan to work
	err = app.MilestoneKeeper.SetLastMilestoneBlock(ctx, uint64(milestone.EndBlock))
	require.NoError(t, err)

	span := &borTypes.Span{
		Id:         1,
		StartBlock: 1,
		EndBlock:   200,
		ValidatorSet: stakeTypes.ValidatorSet{
			Validators: validators,
			Proposer:   validators[0],
		},
		SelectedProducers: []stakeTypes.Validator{*validators[0]},
		BorChainId:        "test",
	}
	err = app.BorKeeper.AddNewSpan(ctx, span)
	require.NoError(t, err)

	// Set context to trigger span rotation conditions
	blockHeight := int64(milestone.EndBlock) + ChangeProducerThreshold + 1
	ctx = ctx.WithBlockHeight(blockHeight)
	// Set rio height to be at or before milestone.EndBlock+1 to ensure IsRio check passes
	helper.SetRioHeight(int64(milestone.EndBlock + 1))

	// Create vote extensions with 70% voting power supporting a new milestone
	// This is more than 2/3
	voteExtensions := createVoteExtensionsWithPartialSupport(t, validators, validatorPrivKeys, &milestone, 70, blockHeight-1)

	// Create ExtendedCommitInfo from vote extensions
	extCommit := &abciTypes.ExtendedCommitInfo{
		Round: 0,
		Votes: voteExtensions,
	}
	extCommitBytes, err := extCommit.Marshal()
	require.NoError(t, err)

	req := &abciTypes.RequestFinalizeBlock{
		Height:          ctx.BlockHeight(),
		Txs:             [][]byte{extCommitBytes, []byte("dummy-tx")}, // Add dummy tx to avoid slice bounds error
		ProposerAddress: common.FromHex(validators[0].Signer),
	}

	// Execute PreBlocker
	_, err = app.PreBlocker(ctx, req)
	require.NoError(t, err)

	// When there's a 2/3 majority milestone, it gets processed normally
	// This can include creating a new span if the milestone warrants it
	_, err = app.BorKeeper.GetLastSpan(ctx)
	require.NoError(t, err)

	// Verify that milestone was added
	latestMilestone, err := app.MilestoneKeeper.GetLastMilestone(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(101), latestMilestone.EndBlock, "New milestone should have been added with correct end block")
}

// Helper function to create vote extensions with specified percentage of voting power supporting a milestone
func createVoteExtensionsWithPartialSupport(t *testing.T, validators []*stakeTypes.Validator, validatorPrivKeys []secp256k1.PrivKey, lastMilestone *milestoneTypes.Milestone, supportPercentage int, voteExtHeight int64) []abciTypes.ExtendedVoteInfo {
	var voteExtensions []abciTypes.ExtendedVoteInfo
	totalVotingPower := int64(0)
	supportingVotingPower := int64(0)

	// Calculate total voting power
	for _, v := range validators {
		totalVotingPower += v.VotingPower
	}

	targetSupportingPower := (totalVotingPower * int64(supportPercentage)) / 100

	// Create milestone proposition
	newMilestone := &milestoneTypes.MilestoneProposition{
		StartBlockNumber: lastMilestone.EndBlock + 1,
		BlockHashes:      [][]byte{common.HexToHash("0x5678").Bytes()},
		ParentHash:       lastMilestone.Hash,
		BlockTds:         []uint64{1},
	}

	// Create dummy non-rp vote extension
	dummyNonRpExt, err := GetDummyNonRpVoteExtension(voteExtHeight, "test-chain")
	require.NoError(t, err)

	for i, validator := range validators {
		var voteExt []byte

		// Create vote extension with milestone proposition if we haven't reached target supporting power
		if supportingVotingPower < targetSupportingPower {
			// Create vote extension with milestone proposition
			voteExtension := &sidetxs.VoteExtension{
				BlockHash:            []byte("test-block-hash"),
				Height:               voteExtHeight,
				MilestoneProposition: newMilestone,
				SideTxResponses:      []sidetxs.SideTxResponse{},
			}
			encoded, err := proto.Marshal(voteExtension)
			require.NoError(t, err)
			voteExt = encoded
			supportingVotingPower += validator.VotingPower
		} else {
			// Create vote extension without milestone proposition
			voteExtension := &sidetxs.VoteExtension{
				BlockHash:            []byte("test-block-hash"),
				Height:               voteExtHeight,
				MilestoneProposition: nil,
				SideTxResponses:      []sidetxs.SideTxResponse{},
			}
			encoded, err := proto.Marshal(voteExtension)
			require.NoError(t, err)
			voteExt = encoded
		}

		// Use validator private key to get consensus address
		consAddr := validatorPrivKeys[i].PubKey().Address()
		voteExtensions = append(voteExtensions, abciTypes.ExtendedVoteInfo{
			Validator: abciTypes.Validator{
				Address: consAddr,
				Power:   validator.VotingPower,
			},
			VoteExtension:           voteExt,
			ExtensionSignature:      []byte("dummy-signature"),
			NonRpVoteExtension:      dummyNonRpExt,
			NonRpExtensionSignature: []byte("dummy-non-rp-signature"),
			BlockIdFlag:             cmtTypes.BlockIDFlagCommit,
		})
	}

	return voteExtensions
}
