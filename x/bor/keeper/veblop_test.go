package keeper_test

import (
	"fmt"

	"github.com/golang/mock/gomock"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func (s *KeeperTestSuite) TestCalculateProducerSet() {
	require := s.Require()
	ctx := s.ctx
	borKeeper := s.borKeeper
	stakeKeeper := s.stakeKeeper

	val1 := staketypes.Validator{ValId: 1, VotingPower: 100}
	val2 := staketypes.Validator{ValId: 2, VotingPower: 90}
	val3 := staketypes.Validator{ValId: 3, VotingPower: 80}

	prodA := uint64(101)
	prodB := uint64(102)
	prodC := uint64(103)

	testCases := []struct {
		name               string
		params             types.Params
		allValidatorsInSet []staketypes.Validator
		validatorDetails   map[uint64]staketypes.Validator
		setupVotes         func()
		expectedCandidates []uint64
		expectedError      bool
	}{
		{
			name:               "Successful selection - example from design doc",
			params:             types.Params{ProducerCount: 3}, // Final producer limit
			allValidatorsInSet: []staketypes.Validator{val1, val2, val3},
			validatorDetails: map[uint64]staketypes.Validator{
				1: val1,
				2: val2,
				3: val3,
			},
			setupVotes: func() {
				require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{prodB, prodA}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, val2.ValId, types.ProducerVotes{Votes: []uint64{prodB, prodC}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, val3.ValId, types.ProducerVotes{Votes: []uint64{prodA, prodC}}))
			},
			expectedCandidates: []uint64{prodB, prodA, prodC},
			expectedError:      false,
		},
		{
			name:               "No votes cast",
			params:             types.Params{ProducerCount: 3},
			allValidatorsInSet: []staketypes.Validator{val1, val2, val3},
			validatorDetails:   map[uint64]staketypes.Validator{1: val1},
			setupVotes: func() {
				// Ensure no votes from previous tests by clearing for relevant validators if necessary
			},
			expectedCandidates: []uint64{},
			expectedError:      false,
		},
		{
			name:               "No validators in the system",
			params:             types.Params{ProducerCount: 3},
			allValidatorsInSet: []staketypes.Validator{},
			validatorDetails:   map[uint64]staketypes.Validator{},
			setupVotes:         func() {},
			expectedCandidates: []uint64{},
			expectedError:      false,
		},
		{
			name:               "Zero total stake",
			params:             types.Params{ProducerCount: 3},
			allValidatorsInSet: []staketypes.Validator{{ValId: 1, VotingPower: 0}},
			validatorDetails:   map[uint64]staketypes.Validator{1: {ValId: 1, VotingPower: 0}},
			setupVotes: func() {
				require.NoError(borKeeper.SetProducerVotes(ctx, 1, types.ProducerVotes{Votes: []uint64{prodA}}))
			},
			expectedCandidates: []uint64{},
			expectedError:      false,
		},
		{
			name:               "ProducerCount is 0",
			params:             types.Params{ProducerCount: 0},
			allValidatorsInSet: []staketypes.Validator{val1, val2, val3},
			validatorDetails:   map[uint64]staketypes.Validator{1: val1, 2: val2, 3: val3},
			setupVotes: func() {
				require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{prodA}}))
			},
			expectedCandidates: []uint64{},
			expectedError:      false,
		},
		{
			name:   "All candidates fail with positional weighting",
			params: types.Params{ProducerCount: 2},
			allValidatorsInSet: []staketypes.Validator{
				{ValId: 1, VotingPower: 10},
				{ValId: 2, VotingPower: 10},
				{ValId: 3, VotingPower: 1000},
			},
			validatorDetails: map[uint64]staketypes.Validator{
				1: {ValId: 1, VotingPower: 10},
				2: {ValId: 2, VotingPower: 10},
				3: {ValId: 3, VotingPower: 1000},
			},
			setupVotes: func() {
				require.NoError(borKeeper.SetProducerVotes(ctx, 1, types.ProducerVotes{Votes: []uint64{prodA}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, 2, types.ProducerVotes{Votes: []uint64{prodB}}))
			},
			expectedCandidates: []uint64{},
			expectedError:      false,
		},
		{
			name:               "More qualified than ProducerCount",
			params:             types.Params{ProducerCount: 1},
			allValidatorsInSet: []staketypes.Validator{val1, val2},
			validatorDetails:   map[uint64]staketypes.Validator{1: val1, 2: val2},
			setupVotes: func() {
				require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{prodA}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, val2.ValId, types.ProducerVotes{Votes: []uint64{prodA, prodB}}))
			},
			expectedCandidates: []uint64{prodA},
			expectedError:      false,
		},
		{
			name:               "Hard stop rule",
			params:             types.Params{ProducerCount: 2},
			allValidatorsInSet: []staketypes.Validator{val1, val2, val3},
			validatorDetails:   map[uint64]staketypes.Validator{1: val1, 2: val2, 3: val3},
			setupVotes: func() {
				require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{prodC, prodA}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, val2.ValId, types.ProducerVotes{Votes: []uint64{prodC, prodB}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, val3.ValId, types.ProducerVotes{Votes: []uint64{prodC, prodB}}))
			},
			expectedCandidates: []uint64{prodC},
			expectedError:      false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := borKeeper.ClearProducerVotes(ctx)
			require.NoError(err)

			require.NoError(borKeeper.SetParams(ctx, tc.params))

			valSet := staketypes.ValidatorSet{Validators: make([]*staketypes.Validator, len(tc.allValidatorsInSet))}
			for i := range tc.allValidatorsInSet {
				valSet.Validators[i] = &tc.allValidatorsInSet[i]
			}
			// Set up mock expectations for this specific test case
			stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).AnyTimes()

			// Set up specific validator details for the test case
			for valIDInDetails, valDetailFromDetails := range tc.validatorDetails {
				localValDetail := valDetailFromDetails
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, valIDInDetails).Return(localValDetail, nil).AnyTimes()
			}

			tc.setupVotes()

			candidates, err := borKeeper.CalculateProducerSet(ctx, tc.params.ProducerCount)

			if tc.expectedError {
				require.Error(err)
			} else {
				require.NoError(err)
				require.ElementsMatch(tc.expectedCandidates, candidates, "Test: '%s'. Expected: %v, Got: %v", tc.name, tc.expectedCandidates, candidates)
			}
		})
	}
}

func (s *KeeperTestSuite) TestSelectNextSpanProducer() {
	require := s.Require()
	ctx := s.ctx
	borKeeper := s.borKeeper
	stakeKeeper := s.stakeKeeper

	// Add missing mock for GetLastMilestone
	s.milestoneKeeper.EXPECT().GetLastMilestone(ctx).Return(&milestoneTypes.Milestone{
		EndBlock: 1000,
	}, nil).AnyTimes()

	err := s.borKeeper.AddNewSpan(ctx, &types.Span{
		Id:         0,
		StartBlock: 0,
		EndBlock:   1001,
		SelectedProducers: []staketypes.Validator{
			{ValId: 1, VotingPower: 100},
		},
	})
	require.NoError(err)

	// Test validators
	val1 := staketypes.Validator{ValId: 1, VotingPower: 100}
	val2 := staketypes.Validator{ValId: 2, VotingPower: 90}
	val3 := staketypes.Validator{ValId: 3, VotingPower: 80}

	testCases := []struct {
		name               string
		setupSpan          func()
		setupProducerVotes func()
		producerSetLimit   uint64
		setupValidatorSet  func()
		activeValidatorIDs map[uint64]struct{}
		expectedProducer   uint64
		expectedError      bool
		errorContains      string
	}{
		{
			name: "No last span found",
			setupSpan: func() {
				// Don't set up any span
			},
			setupProducerVotes: func() {},
			producerSetLimit:   3,
			setupValidatorSet: func() {
				valSet := staketypes.ValidatorSet{
					Validators: []*staketypes.Validator{
						{ValId: 1, VotingPower: 100},
					},
				}
				stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).AnyTimes()
				// Updated mock to handle both expected IDs
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(1)).Return(val1, nil).AnyTimes()
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, gomock.Any()).Return(staketypes.Validator{VotingPower: 0}, nil).AnyTimes()
			},
			activeValidatorIDs: map[uint64]struct{}{},
			expectedProducer:   2,
			expectedError:      false,
		},
		{
			name: "Last span has no selected producers",
			setupSpan: func() {
				span := types.Span{
					Id:                1,
					StartBlock:        1,
					EndBlock:          100,
					SelectedProducers: []staketypes.Validator{}, // Empty
				}
				require.NoError(borKeeper.AddNewSpan(ctx, &span))
			},
			setupProducerVotes: func() {},
			producerSetLimit:   3,
			setupValidatorSet:  func() {},
			activeValidatorIDs: map[uint64]struct{}{},
			expectedProducer:   2,
			expectedError:      false,
		},
		{
			name: "Fail to find any candidates",
			setupSpan: func() {
				span := types.Span{
					Id:         1,
					StartBlock: 1,
					EndBlock:   100,
					SelectedProducers: []staketypes.Validator{
						{ValId: 1, VotingPower: 100}, // Current producer
					},
				}
				require.NoError(borKeeper.AddNewSpan(ctx, &span))
			},
			setupProducerVotes: func() {
			},
			producerSetLimit: 3,
			setupValidatorSet: func() {
				// Scenario: No validators in the set, or they have no voting power / no votes
				// This will make CalculateProducerSet return an empty list, triggering fallback.
				valSet := staketypes.ValidatorSet{
					Validators: []*staketypes.Validator{
						// Option 1: Empty validator set
						// Option 2: Validators with 0 voting power
						// {ValId: 100, VotingPower: 0},
					},
				}
				stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).AnyTimes()
				// No GetValidatorFromValID needed if GetValidatorSet is empty or all have 0 power with no votes
			},
			activeValidatorIDs: map[uint64]struct{}{
				1: {}, // Active fallback producer
				2: {}, // Active fallback producer
				3: {}, // Active fallback producer
			},
			expectedProducer: 2,
			expectedError:    false,
		},
		{
			name: "Current producer not in candidate list - selects first candidate",
			setupSpan: func() {
				span := types.Span{
					Id:         1,
					StartBlock: 1,
					EndBlock:   100,
					SelectedProducers: []staketypes.Validator{
						{ValId: 5, VotingPower: 100}, // Current producer not in the candidate list [2,3]
					},
				}
				require.NoError(borKeeper.AddNewSpan(ctx, &span))
			},
			setupProducerVotes: func() {
				// Set specific votes for this test (val1, val2, val3 vote for candidates 2, 3)
				require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, val2.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, val3.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
			},
			producerSetLimit: 2,
			setupValidatorSet: func() {
				valSet := staketypes.ValidatorSet{
					Validators: []*staketypes.Validator{&val1, &val2, &val3}, // These are actual voters
				}
				stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).AnyTimes()
				// Specific expectations first
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val1.ValId).Return(val1, nil).AnyTimes()
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val2.ValId).Return(val2, nil).AnyTimes()
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val3.ValId).Return(val3, nil).AnyTimes()
				// Catch-all for any other ID will be checked after specific ones
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, gomock.Any()).
					Return(staketypes.Validator{VotingPower: 0}, nil).AnyTimes()
			},
			activeValidatorIDs: map[uint64]struct{}{
				2: {},
				3: {},
			},
			expectedProducer: 2, // Candidates [2,3] (due to ProducerCount=2 and votes), current 5 not in the list, so selects 2
			expectedError:    false,
		},
		{
			name: "Wrapping around - last candidate selects first",
			setupSpan: func() {
				span := types.Span{
					Id:         1,
					StartBlock: 1,
					EndBlock:   100,
					SelectedProducers: []staketypes.Validator{
						{ValId: 3, VotingPower: 100}, // Current producer is last in the candidate list [2,3]
					},
				}
				require.NoError(borKeeper.AddNewSpan(ctx, &span))
			},
			setupProducerVotes: func() {
				// Set specific votes for this test (val1, val2, val3 vote for candidates 2, 3)
				require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, val2.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, val3.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
			},
			producerSetLimit: 2,
			setupValidatorSet: func() {
				valSet := staketypes.ValidatorSet{
					Validators: []*staketypes.Validator{&val1, &val2, &val3}, // Actual voters
				}
				stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).AnyTimes()
				// Specific expectations first
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val1.ValId).Return(val1, nil).AnyTimes()
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val2.ValId).Return(val2, nil).AnyTimes()
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val3.ValId).Return(val3, nil).AnyTimes()
				// Catch-all for any other ID will be checked after specific ones
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, gomock.Any()).
					Return(staketypes.Validator{VotingPower: 0}, nil).AnyTimes()
			},
			activeValidatorIDs: map[uint64]struct{}{
				2: {},
				3: {},
			},
			expectedProducer: 2, // Candidates [2,3], current 3 is last, wraps to 2
			expectedError:    false,
		},
		{
			name: "Even fallback has no active candidates - selects next candidate",
			setupSpan: func() {
				span := types.Span{
					Id:         1,
					StartBlock: 1,
					EndBlock:   100,
					SelectedProducers: []staketypes.Validator{
						{ValId: 1, VotingPower: 100},
					},
				}
				require.NoError(borKeeper.AddNewSpan(ctx, &span))
			},
			setupProducerVotes: func() {},
			producerSetLimit:   3,
			setupValidatorSet: func() {
				// Ensure CalculateProducerSet returns an empty set initially, triggering fallback
				valSet := staketypes.ValidatorSet{
					Validators: []*staketypes.Validator{
						{ValId: 100, VotingPower: 10},
					},
				}
				stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).AnyTimes()
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(100)).Return(staketypes.Validator{ValId: 100, VotingPower: 10}, nil).AnyTimes()
			},
			activeValidatorIDs: map[uint64]struct{}{
				// No validator IDs are active, so even fallback [1,2,3] becomes empty
			},
			expectedProducer: 2,
			expectedError:    false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err = borKeeper.ClearProducerVotes(ctx)
			require.NoError(err)

			tc.setupSpan()
			if tc.setupProducerVotes != nil {
				tc.setupProducerVotes()
			}
			tc.setupValidatorSet()
			lastSpan, err := borKeeper.GetLastSpan(ctx)
			require.NoError(err)

			params, err := borKeeper.GetParams(ctx)
			require.NoError(err)

			startBlock := lastSpan.EndBlock + 1
			endBlock := lastSpan.EndBlock + params.SpanDuration

			result, err := borKeeper.SelectNextSpanProducer(ctx, 1, tc.activeValidatorIDs, tc.producerSetLimit, startBlock, endBlock, types.RoundRobinDefault)

			if tc.expectedError {
				require.Error(err)
				if tc.errorContains != "" {
					require.Contains(err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(err)
				if tc.expectedProducer != 0 {
					require.Equal(tc.expectedProducer, result)
				}
			}
		})
	}
}

func (s *KeeperTestSuite) TestCalculateProducerSet_TotalPotentialProducersVoteCap() {
	require := s.Require()
	ctx := s.ctx
	borKeeper := s.borKeeper
	stakeKeeper := s.stakeKeeper

	val1 := staketypes.Validator{ValId: 1, VotingPower: 100}
	prodA := uint64(101)
	prodB := uint64(102)

	// Clear any existing votes
	err := borKeeper.ClearProducerVotes(ctx)
	require.NoError(err)

	// Set up validator set with only 1 validator (totalPotentialProducers will be 1)
	valSet := staketypes.ValidatorSet{
		Validators: []*staketypes.Validator{&val1},
	}
	stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).AnyTimes()
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(1)).Return(val1, nil).AnyTimes()

	// Set up votes: val1 votes for [prodA, prodB]
	require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{prodA, prodB}}))

	// Call CalculateProducerSet
	candidates, err := borKeeper.CalculateProducerSet(ctx, 2)

	// Verify results
	require.NoError(err)
	require.ElementsMatch([]uint64{prodA}, candidates, "Expected: [%d], Got: %v", prodA, candidates)
}

func (s *KeeperTestSuite) TestAddNewVeBlopSpan() {
	require := s.Require()
	ctx := s.ctx
	borKeeper := s.borKeeper
	stakeKeeper := s.stakeKeeper

	require.NoError(borKeeper.AddNewSpan(ctx, &types.Span{
		Id:         0,
		StartBlock: 0,
		EndBlock:   199,
		SelectedProducers: []staketypes.Validator{
			{ValId: 1, VotingPower: 100},
		},
	}))

	val1 := staketypes.Validator{ValId: 1, VotingPower: 100}
	val2 := staketypes.Validator{ValId: 2, VotingPower: 180}
	val3 := staketypes.Validator{ValId: 3, VotingPower: 70}
	valSet := staketypes.ValidatorSet{Validators: []*staketypes.Validator{&val1, &val2, &val3}}

	stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).AnyTimes()
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(1)).Return(val1, nil).AnyTimes()
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(2)).Return(val2, nil).AnyTimes()
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(3)).Return(val3, nil).AnyTimes()

	// Set producer votes so that candidate 2 qualifies as the next producer
	require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
	require.NoError(borKeeper.SetProducerVotes(ctx, val2.ValId, types.ProducerVotes{Votes: []uint64{1, 3}}))
	require.NoError(borKeeper.SetProducerVotes(ctx, val3.ValId, types.ProducerVotes{Votes: []uint64{3, 1}}))

	active := map[uint64]struct{}{1: {}, 2: {}, 3: {}}

	startBlock := uint64(200)
	endBlock := uint64(300)
	borChainID := "137"
	heimdallBlock := uint64(5000)

	err := borKeeper.AddNewVeBlopSpan(ctx, 1, startBlock, endBlock, borChainID, active, heimdallBlock, types.RoundRobinDefault)
	require.NoError(err)

	span, err := borKeeper.GetSpan(ctx, 1)
	require.NoError(err)
	require.Equal(uint64(1), span.Id)
	require.Equal(startBlock, span.StartBlock)
	require.Equal(endBlock, span.EndBlock)
	require.Equal(borChainID, span.BorChainId)
	require.Equal(valSet, span.ValidatorSet)

	// Selected producer is validator 2
	require.Len(span.SelectedProducers, 1)
	require.Equal(uint64(2), span.SelectedProducers[0].ValId)

	lastBlock, err := borKeeper.GetLastSpanBlock(ctx)
	require.NoError(err)
	require.Equal(heimdallBlock, lastBlock)
}

func (s *KeeperTestSuite) TestSelectNextSpanProducerWithTarget() {
	require := s.Require()
	ctx := s.ctx
	borKeeper := s.borKeeper
	stakeKeeper := s.stakeKeeper

	s.milestoneKeeper.EXPECT().GetLastMilestone(ctx).Return(&milestoneTypes.Milestone{
		EndBlock: 1000,
	}, nil).AnyTimes()

	require.NoError(borKeeper.AddNewSpan(ctx, &types.Span{
		Id:         0,
		StartBlock: 0,
		EndBlock:   1001,
		SelectedProducers: []staketypes.Validator{
			{ValId: 1, VotingPower: 100},
		},
	}))

	val1 := staketypes.Validator{ValId: 1, VotingPower: 100}
	val2 := staketypes.Validator{ValId: 2, VotingPower: 90}
	val3 := staketypes.Validator{ValId: 3, VotingPower: 80}

	setupMocks := func() {
		valSet := staketypes.ValidatorSet{
			Validators: []*staketypes.Validator{&val1, &val2, &val3},
		}
		stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).AnyTimes()
		stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val1.ValId).Return(val1, nil).AnyTimes()
		stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val2.ValId).Return(val2, nil).AnyTimes()
		stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val3.ValId).Return(val3, nil).AnyTimes()
		stakeKeeper.EXPECT().GetValidatorFromValID(ctx, gomock.Any()).
			Return(staketypes.Validator{VotingPower: 0}, nil).AnyTimes()

		require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
		require.NoError(borKeeper.SetProducerVotes(ctx, val2.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
		require.NoError(borKeeper.SetProducerVotes(ctx, val3.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
	}

	// Happy path: valid target, active, not down -> selected directly
	s.Run("target in active candidates and not down", func() {
		require.NoError(borKeeper.ClearProducerVotes(ctx))
		setupMocks()

		active := map[uint64]struct{}{2: {}, 3: {}}
		result, err := borKeeper.SelectNextSpanProducer(ctx, 1, active, 2, 1002, 2001, 3)
		require.NoError(err)
		require.Equal(uint64(3), result)
	})

	// Target has planned downtime overlapping the range -> round-robin
	s.Run("target down for range falls through to round-robin", func() {
		require.NoError(borKeeper.ClearProducerVotes(ctx))
		setupMocks()

		require.NoError(borKeeper.ProducerPlannedDowntime.Set(ctx, 3, types.BlockRange{StartBlock: 1000, EndBlock: 2500}))

		active := map[uint64]struct{}{2: {}, 3: {}}
		result, err := borKeeper.SelectNextSpanProducer(ctx, 1, active, 2, 1002, 2001, 3)
		require.NoError(err)
		require.Equal(uint64(2), result)

		require.NoError(borKeeper.ProducerPlannedDowntime.Remove(ctx, 3))
	})

	// Target exists in candidate set but not active -> round-robin
	s.Run("target not in active candidates falls through", func() {
		require.NoError(borKeeper.ClearProducerVotes(ctx))
		setupMocks()

		active := map[uint64]struct{}{2: {}}
		result, err := borKeeper.SelectNextSpanProducer(ctx, 1, active, 2, 1002, 2001, 3)
		require.NoError(err)
		require.Equal(uint64(2), result)
	})

	// target=0 is the no-preference default -> always round-robin
	s.Run("target zero uses round-robin", func() {
		require.NoError(borKeeper.ClearProducerVotes(ctx))
		setupMocks()

		active := map[uint64]struct{}{2: {}, 3: {}}
		result, err := borKeeper.SelectNextSpanProducer(ctx, 1, active, 2, 1002, 2001, types.RoundRobinDefault)
		require.NoError(err)
		require.Equal(uint64(2), result)
	})

	// Non-existent validator ID as target -> not in candidates -> round-robin
	s.Run("target is non-existent validator falls through", func() {
		require.NoError(borKeeper.ClearProducerVotes(ctx))
		setupMocks()

		active := map[uint64]struct{}{2: {}, 3: {}}
		result, err := borKeeper.SelectNextSpanProducer(ctx, 1, active, 2, 1002, 2001, 999)
		require.NoError(err)
		require.Equal(uint64(2), result)
	})

	// Target equals current producer — SelectNextSpanProducer now guards against
	// self-targeting as defense-in-depth and falls through to round-robin.
	s.Run("target equals current producer falls through to round-robin", func() {
		require.NoError(borKeeper.ClearProducerVotes(ctx))
		setupMocks()

		active := map[uint64]struct{}{2: {}, 3: {}}
		result, err := borKeeper.SelectNextSpanProducer(ctx, 2, active, 2, 1002, 2001, 2)
		require.NoError(err)
		require.Equal(uint64(3), result) // round-robin: next after current=2 in [2,3] is 3
	})

	// Downtime overlaps only partially (start inside range) -> still considered down
	s.Run("target with partial downtime overlap falls through", func() {
		require.NoError(borKeeper.ClearProducerVotes(ctx))
		setupMocks()

		// Downtime [1500, 2500] overlaps range [1002, 2001]
		require.NoError(borKeeper.ProducerPlannedDowntime.Set(ctx, 3, types.BlockRange{StartBlock: 1500, EndBlock: 2500}))

		active := map[uint64]struct{}{2: {}, 3: {}}
		result, err := borKeeper.SelectNextSpanProducer(ctx, 1, active, 2, 1002, 2001, 3)
		require.NoError(err)
		require.Equal(uint64(2), result)

		require.NoError(borKeeper.ProducerPlannedDowntime.Remove(ctx, 3))
	})

	// Downtime range is entirely before the span range -> target NOT down -> selected
	s.Run("target with non-overlapping downtime is selected", func() {
		require.NoError(borKeeper.ClearProducerVotes(ctx))
		setupMocks()

		// Downtime [500, 900] does not overlap range [1002, 2001]
		require.NoError(borKeeper.ProducerPlannedDowntime.Set(ctx, 3, types.BlockRange{StartBlock: 500, EndBlock: 900}))

		active := map[uint64]struct{}{2: {}, 3: {}}
		result, err := borKeeper.SelectNextSpanProducer(ctx, 1, active, 2, 1002, 2001, 3)
		require.NoError(err)
		require.Equal(uint64(3), result)

		require.NoError(borKeeper.ProducerPlannedDowntime.Remove(ctx, 3))
	})

	// Round-robin result is deterministic regardless of whether a target was attempted
	s.Run("fallthrough produces same result as no target", func() {
		require.NoError(borKeeper.ClearProducerVotes(ctx))
		setupMocks()

		active := map[uint64]struct{}{2: {}, 3: {}}

		// With invalid target (falls through)
		resultWithTarget, err := borKeeper.SelectNextSpanProducer(ctx, 1, active, 2, 1002, 2001, 999)
		require.NoError(err)

		require.NoError(borKeeper.ClearProducerVotes(ctx))
		setupMocks()

		// Without target
		resultNoTarget, err := borKeeper.SelectNextSpanProducer(ctx, 1, active, 2, 1002, 2001, 0)
		require.NoError(err)

		require.Equal(resultNoTarget, resultWithTarget)
	})
}

func (s *KeeperTestSuite) TestAddNewVeBlopSpanWithTarget() {
	require := s.Require()
	ctx := s.ctx
	borKeeper := s.borKeeper
	stakeKeeper := s.stakeKeeper

	require.NoError(borKeeper.AddNewSpan(ctx, &types.Span{
		Id:         0,
		StartBlock: 0,
		EndBlock:   199,
		SelectedProducers: []staketypes.Validator{
			{ValId: 1, VotingPower: 100},
		},
	}))

	val1 := staketypes.Validator{ValId: 1, VotingPower: 100}
	val2 := staketypes.Validator{ValId: 2, VotingPower: 180}
	val3 := staketypes.Validator{ValId: 3, VotingPower: 70}
	valSet := staketypes.ValidatorSet{Validators: []*staketypes.Validator{&val1, &val2, &val3}}

	stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).AnyTimes()
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(1)).Return(val1, nil).AnyTimes()
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(2)).Return(val2, nil).AnyTimes()
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(3)).Return(val3, nil).AnyTimes()

	require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
	require.NoError(borKeeper.SetProducerVotes(ctx, val2.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
	require.NoError(borKeeper.SetProducerVotes(ctx, val3.ValId, types.ProducerVotes{Votes: []uint64{3, 2}}))

	active := map[uint64]struct{}{1: {}, 2: {}, 3: {}}

	// Valid target=3 should be selected over round-robin
	err := borKeeper.AddNewVeBlopSpan(ctx, 1, 200, 300, "137", active, 5000, 3)
	require.NoError(err)

	span, err := borKeeper.GetSpan(ctx, 1)
	require.NoError(err)
	require.Len(span.SelectedProducers, 1)
	require.Equal(uint64(3), span.SelectedProducers[0].ValId)
}

func (s *KeeperTestSuite) TestAddNewVeBlopSpanTargetFallthrough() {
	require := s.Require()
	ctx := s.ctx
	borKeeper := s.borKeeper
	stakeKeeper := s.stakeKeeper

	require.NoError(borKeeper.AddNewSpan(ctx, &types.Span{
		Id:         0,
		StartBlock: 0,
		EndBlock:   199,
		SelectedProducers: []staketypes.Validator{
			{ValId: 1, VotingPower: 100},
		},
	}))

	val1 := staketypes.Validator{ValId: 1, VotingPower: 100}
	val2 := staketypes.Validator{ValId: 2, VotingPower: 180}
	val3 := staketypes.Validator{ValId: 3, VotingPower: 70}
	valSet := staketypes.ValidatorSet{Validators: []*staketypes.Validator{&val1, &val2, &val3}}

	stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).AnyTimes()
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(1)).Return(val1, nil).AnyTimes()
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(2)).Return(val2, nil).AnyTimes()
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(3)).Return(val3, nil).AnyTimes()

	require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
	require.NoError(borKeeper.SetProducerVotes(ctx, val2.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
	require.NoError(borKeeper.SetProducerVotes(ctx, val3.ValId, types.ProducerVotes{Votes: []uint64{3, 2}}))

	// Target=3 is down for the range -> should fall through to round-robin (val2)
	require.NoError(borKeeper.ProducerPlannedDowntime.Set(ctx, 3, types.BlockRange{StartBlock: 150, EndBlock: 350}))

	active := map[uint64]struct{}{1: {}, 2: {}, 3: {}}

	err := borKeeper.AddNewVeBlopSpan(ctx, 1, 200, 300, "137", active, 5000, 3)
	require.NoError(err)

	span, err := borKeeper.GetSpan(ctx, 1)
	require.NoError(err)
	require.Len(span.SelectedProducers, 1)
	require.Equal(uint64(2), span.SelectedProducers[0].ValId)
}

func (s *KeeperTestSuite) TestAddNewVeBlopSpanCalculateProducerSetError() {
	require := s.Require()
	ctx := s.ctx
	borKeeper := s.borKeeper
	stakeKeeper := s.stakeKeeper

	require.NoError(borKeeper.AddNewSpan(ctx, &types.Span{
		Id:         0,
		StartBlock: 0,
		EndBlock:   199,
		SelectedProducers: []staketypes.Validator{
			{ValId: 1, VotingPower: 100},
		},
	}))

	// GetValidatorSet returns an error -> CalculateProducerSet fails -> SelectNextSpanProducer fails.
	stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(staketypes.ValidatorSet{}, fmt.Errorf("validator set error")).AnyTimes()

	active := map[uint64]struct{}{}

	err := borKeeper.AddNewVeBlopSpan(ctx, 1, 200, 300, "137", active, 5000, types.RoundRobinDefault)
	require.Error(err)
	require.Contains(err.Error(), "validator set error")
}

func (s *KeeperTestSuite) TestAddNewVeBlopSpanGetValidatorSetError() {
	require := s.Require()
	ctx := s.ctx
	borKeeper := s.borKeeper
	stakeKeeper := s.stakeKeeper

	require.NoError(borKeeper.AddNewSpan(ctx, &types.Span{
		Id:         0,
		StartBlock: 0,
		EndBlock:   199,
		SelectedProducers: []staketypes.Validator{
			{ValId: 1, VotingPower: 100},
		},
	}))

	val1 := staketypes.Validator{ValId: 1, VotingPower: 100}
	val2 := staketypes.Validator{ValId: 2, VotingPower: 180}
	valSet := staketypes.ValidatorSet{Validators: []*staketypes.Validator{&val1, &val2}}

	// First call to GetValidatorSet (inside CalculateProducerSet) succeeds;
	// second call (inside AddNewVeBlopSpan after SelectNextSpanProducer) fails.
	callCount := 0
	stakeKeeper.EXPECT().GetValidatorSet(ctx).DoAndReturn(func(_ interface{}) (staketypes.ValidatorSet, error) {
		callCount++
		if callCount <= 1 {
			return valSet, nil
		}
		return staketypes.ValidatorSet{}, fmt.Errorf("validator set unavailable")
	}).AnyTimes()

	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(1)).Return(val1, nil).AnyTimes()
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(2)).Return(val2, nil).AnyTimes()

	require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{2}}))
	require.NoError(borKeeper.SetProducerVotes(ctx, val2.ValId, types.ProducerVotes{Votes: []uint64{2}}))

	active := map[uint64]struct{}{1: {}, 2: {}}

	err := borKeeper.AddNewVeBlopSpan(ctx, 1, 200, 300, "137", active, 5000, types.RoundRobinDefault)
	require.Error(err)
	require.Contains(err.Error(), "validator set unavailable")
}
