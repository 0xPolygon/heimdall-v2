package keeper_test

import (
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/golang/mock/gomock"
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
		{
			name:               "totalPotentialProducers vote cap in action",
			params:             types.Params{ProducerCount: 2},
			allValidatorsInSet: []staketypes.Validator{val1}, // totalPotentialProducers will be 1
			validatorDetails:   map[uint64]staketypes.Validator{1: val1},
			setupVotes: func() {
				require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{prodA, prodB}}))
			},
			expectedCandidates: []uint64{prodA},
			expectedError:      false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			valIDsToClear := []uint64{1, 2, 3}
			for _, valID := range valIDsToClear {
				if ctx.IsZero() {
					s.T().Fatalf("SDK context is zero before clearing votes, test setup issue.")
				}
				err := borKeeper.SetProducerVotes(ctx, valID, types.ProducerVotes{Votes: []uint64{}})
				require.NoError(err, "Failed to clear votes for validator %d", valID)
			}

			require.NoError(borKeeper.SetParams(ctx, tc.params))

			valSet := staketypes.ValidatorSet{Validators: make([]*staketypes.Validator, len(tc.allValidatorsInSet))}
			for i := range tc.allValidatorsInSet {
				valSet.Validators[i] = &tc.allValidatorsInSet[i]
			}
			stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).Times(1)

			for valIDInDetails, valDetailFromDetails := range tc.validatorDetails {
				localValDetail := valDetailFromDetails
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, valIDInDetails).Return(localValDetail, nil).AnyTimes()
			}

			tc.setupVotes()

			candidates, err := borKeeper.CalculateProducerSet(ctx)

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

	// Test validators
	val1 := staketypes.Validator{ValId: 1, VotingPower: 100}
	val2 := staketypes.Validator{ValId: 2, VotingPower: 90}
	val3 := staketypes.Validator{ValId: 3, VotingPower: 80}

	testCases := []struct {
		name               string
		setupSpan          func()
		setupProducerVotes func()
		setupParams        func()
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
			setupParams:        func() {},
			setupValidatorSet:  func() {},
			activeValidatorIDs: map[uint64]struct{}{},
			expectedError:      true,
			errorContains:      "failed to get last span",
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
			setupParams:        func() {},
			setupValidatorSet:  func() {},
			activeValidatorIDs: map[uint64]struct{}{},
			expectedError:      true,
			errorContains:      "no producers found in last span",
		},
		{
			name: "Fallback with active candidates - selects next from fallback",
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
				// Ensure global producer votes are cleared for this specific test
				// to guarantee CalculateProducerSet will not find any candidates from votes.
				allPossibleVoterIDs := []uint64{1, 2, 3, 4, 5, 100, 101, 102, 103}
				for _, id := range allPossibleVoterIDs {
					require.NoError(borKeeper.SetProducerVotes(ctx, id, types.ProducerVotes{Votes: []uint64{}}))
				}
			},
			setupParams: func() {
				params := types.DefaultParams()
				params.ProducerCount = 3
				require.NoError(borKeeper.SetParams(ctx, params))
			},
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
				stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).Times(1)
				// No GetValidatorFromValID needed if GetValidatorSet is empty or all have 0 power with no votes
			},
			activeValidatorIDs: map[uint64]struct{}{
				1: {}, // Active fallback producer
				2: {}, // Active fallback producer
				3: {}, // Active fallback producer
			},
			expectedProducer: 2, // Fallback is [1,2,3], current is 1, next is 2
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
						{ValId: 5, VotingPower: 100}, // Current producer not in candidate list [2,3]
					},
				}
				require.NoError(borKeeper.AddNewSpan(ctx, &span))
			},
			setupProducerVotes: func() {
				// Clear relevant votes first for this specific test
				idsToClear := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 100, 101, 102, 103} // Wider range
				for _, id := range idsToClear {
					require.NoError(borKeeper.SetProducerVotes(ctx, id, types.ProducerVotes{Votes: []uint64{}}))
				}
				// Set specific votes for this test (val1, val2, val3 vote for candidates 2, 3)
				require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, val2.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, val3.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
			},
			setupParams: func() {
				params := types.DefaultParams()
				params.ProducerCount = 2 // Limits candidate set size, so 2,3 should be the only ones
				require.NoError(borKeeper.SetParams(ctx, params))
			},
			setupValidatorSet: func() {
				valSet := staketypes.ValidatorSet{
					Validators: []*staketypes.Validator{&val1, &val2, &val3}, // These are actual voters
				}
				stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).Times(1)
				// Specific expectations first
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val1.ValId).Return(val1, nil).AnyTimes()
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val2.ValId).Return(val2, nil).AnyTimes()
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val3.ValId).Return(val3, nil).AnyTimes()
				// Catch-all for any other ID, will be checked after specific ones
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, gomock.Any()).
					Return(staketypes.Validator{VotingPower: 0}, nil).AnyTimes()
			},
			activeValidatorIDs: map[uint64]struct{}{
				2: {},
				3: {},
			},
			expectedProducer: 2, // Candidates [2,3] (due to ProducerCount=2 and votes), current 5 not in list, so selects 2
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
						{ValId: 3, VotingPower: 100}, // Current producer is last in candidate list [2,3]
					},
				}
				require.NoError(borKeeper.AddNewSpan(ctx, &span))
			},
			setupProducerVotes: func() {
				// Clear relevant votes first for this specific test
				idsToClear := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 100, 101, 102, 103} // Wider range
				for _, id := range idsToClear {
					require.NoError(borKeeper.SetProducerVotes(ctx, id, types.ProducerVotes{Votes: []uint64{}}))
				}
				// Set specific votes for this test (val1, val2, val3 vote for candidates 2, 3)
				require.NoError(borKeeper.SetProducerVotes(ctx, val1.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, val2.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
				require.NoError(borKeeper.SetProducerVotes(ctx, val3.ValId, types.ProducerVotes{Votes: []uint64{2, 3}}))
			},
			setupParams: func() {
				params := types.DefaultParams()
				params.ProducerCount = 2 // Limits candidate set size to 2,3
				require.NoError(borKeeper.SetParams(ctx, params))
			},
			setupValidatorSet: func() {
				valSet := staketypes.ValidatorSet{
					Validators: []*staketypes.Validator{&val1, &val2, &val3}, // Actual voters
				}
				stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).Times(1)
				// Specific expectations first
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val1.ValId).Return(val1, nil).AnyTimes()
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val2.ValId).Return(val2, nil).AnyTimes()
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, val3.ValId).Return(val3, nil).AnyTimes()
				// Catch-all for any other ID, will be checked after specific ones
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
			name: "Even fallback has no active candidates - returns error",
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
			setupParams: func() {
				params := types.DefaultParams()
				params.ProducerCount = 3
				require.NoError(borKeeper.SetParams(ctx, params))
			},
			setupValidatorSet: func() {
				// Ensure CalculateProducerSet returns an empty set initially, triggering fallback
				valSet := staketypes.ValidatorSet{
					Validators: []*staketypes.Validator{
						{ValId: 100, VotingPower: 10},
					},
				}
				stakeKeeper.EXPECT().GetValidatorSet(ctx).Return(valSet, nil).Times(1)
				stakeKeeper.EXPECT().GetValidatorFromValID(ctx, uint64(100)).Return(staketypes.Validator{ValId: 100, VotingPower: 10}, nil).AnyTimes()
			},
			activeValidatorIDs: map[uint64]struct{}{
				// No validator IDs are active, so even fallback [1,2,3] becomes empty
			},
			expectedError: true,
			errorContains: "no candidates found", // SelectProducer should fail with empty candidates
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Clear producer votes from previous tests
			valIDsToClear := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 99, 101, 102, 103}
			for _, valID := range valIDsToClear {
				err := borKeeper.SetProducerVotes(ctx, valID, types.ProducerVotes{Votes: []uint64{}})
				require.NoError(err, "Failed to clear votes for validator %d", valID)
			}

			tc.setupSpan()
			if tc.setupProducerVotes != nil {
				tc.setupProducerVotes()
			}
			tc.setupParams()
			tc.setupValidatorSet()

			lastMilestone, err := s.milestoneKeeper.GetLastMilestone(ctx)
			if err != nil {
				s.T().Fatalf("Failed to get last milestone block: %v", err)
			}

			currentProducer, err := borKeeper.FindCurrentProducerID(ctx, lastMilestone.EndBlock)
			if err != nil {
				s.T().Fatalf("Failed to find current producer: %v", err)
			}

			result, err := borKeeper.SelectNextSpanProducer(ctx, currentProducer, tc.activeValidatorIDs)

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
