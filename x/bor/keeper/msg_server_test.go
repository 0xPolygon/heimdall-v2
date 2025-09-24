package keeper_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/mock/gomock"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func (s *KeeperTestSuite) TestProposeSpan() {
	require, ctx, borKeeper, cmKeeper, msgServer := s.Require(), s.ctx, s.borKeeper, s.chainManagerKeeper, s.msgServer

	testChainParams := chainmanagertypes.DefaultParams()
	testSpan := s.genTestSpans(1)[0]
	err := borKeeper.AddNewSpan(ctx, testSpan)
	require.NoError(err)

	testcases := []struct {
		name   string
		span   types.MsgProposeSpan
		expRes *types.MsgProposeSpanResponse
		expErr string
	}{
		{
			name: "correct span gets proposed",
			span: types.MsgProposeSpan{
				SpanId:     2,
				Proposer:   common.HexToAddress("someProposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testSeed1").Bytes(),
			},
			expRes: &types.MsgProposeSpanResponse{},
		},
		{
			name: "incorrect validator address",
			span: types.MsgProposeSpan{
				SpanId:     2,
				Proposer:   "0x91b54cD48FD796A5d0A120A4C5298a7fAEA59B",
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testSeed1").Bytes(),
			},
			expRes: nil,
			expErr: "invalid proposer address",
		},
		{
			name: "incorrect chain id",
			span: types.MsgProposeSpan{
				SpanId:     2,
				Proposer:   common.HexToAddress("someProposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    "invalidChainId",
				Seed:       common.HexToHash("testSeed1").Bytes(),
			},
			expRes: nil,
			expErr: "invalid bor chain id",
		},
		{
			name: "span id not in continuity",
			span: types.MsgProposeSpan{
				SpanId:     3,
				Proposer:   common.HexToAddress("someProposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testSeed1").Bytes(),
			},
			expRes: nil,
			expErr: "invalid span",
		},
		{
			name: "start block not in continuity",
			span: types.MsgProposeSpan{
				SpanId:     2,
				Proposer:   common.HexToAddress("someProposer").String(),
				StartBlock: 105,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testSeed1").Bytes(),
			},
			expRes: nil,
			expErr: "invalid span",
		},
		{
			name: "end block less than start block",
			span: types.MsgProposeSpan{
				SpanId:     2,
				Proposer:   common.HexToAddress("someProposer").String(),
				StartBlock: 102,
				EndBlock:   100,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testSeed1").Bytes(),
			},
			expRes: nil,
			expErr: "invalid span",
		},
		{
			name: "end block equal to start block",
			span: types.MsgProposeSpan{
				SpanId:     2,
				Proposer:   common.HexToAddress("someProposer").String(),
				StartBlock: 102,
				EndBlock:   102,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testSeed1").Bytes(),
			},
			expRes: nil,
			expErr: "invalid span",
		},
	}

	cmKeeper.EXPECT().GetParams(ctx).Return(testChainParams, nil).AnyTimes()

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			res, err := msgServer.ProposeSpan(ctx, &tc.span)
			require.Equal(tc.expRes, res)
			if tc.expErr == "" {
				require.NoError(err)
			} else {
				require.ErrorContains(err, tc.expErr)
			}
		})
	}
}

func (s *KeeperTestSuite) TestMsgUpdateParams() {
	ctx, require, keeper, queryClient, msgServer, params := s.ctx, s.Require(), s.borKeeper, s.queryClient, s.msgServer, types.DefaultParams()

	testCases := []struct {
		name      string
		input     *types.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "invalid authority",
			input: &types.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "invalid sprint duration",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params: types.Params{
					SprintDuration: 0,
					SpanDuration:   params.SpanDuration,
					ProducerCount:  params.ProducerCount,
				},
			},
			expErr:    true,
			expErrMsg: "invalid value provided 0 for bor param sprint duration",
		},
		{
			name: "invalid span duration",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params: types.Params{
					SprintDuration: params.SprintDuration,
					SpanDuration:   0,
					ProducerCount:  params.ProducerCount,
				},
			},
			expErr:    true,
			expErrMsg: "invalid value provided 0 for bor param span duration",
		},
		{
			name: "invalid producer count",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params: types.Params{
					SprintDuration: params.SprintDuration,
					SpanDuration:   params.SpanDuration,
					ProducerCount:  0,
				},
			},
			expErr:    true,
			expErrMsg: "invalid value provided 0 for bor param producer count",
		},
		{
			name: "all good",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params:    params,
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			_, err := msgServer.UpdateParams(ctx, tc.input)

			if tc.expErr {
				require.Error(err)
				require.Contains(err.Error(), tc.expErrMsg)
			} else {
				require.Equal(authtypes.NewModuleAddress(govtypes.ModuleName).String(), keeper.GetAuthority())
				require.NoError(err)

				res, err := queryClient.GetBorParams(ctx, &types.QueryParamsRequest{})
				require.NoError(err)
				require.Equal(params, res.Params)
			}
		})
	}
}

func (s *KeeperTestSuite) TestVoteProducers() {
	require, ctx, borKeeper, skMock, msgServer := s.Require(), s.ctx, s.borKeeper, s.stakeKeeper, s.msgServer

	// Generate a key pair for the voter and validator
	voterPrivKey := secp256k1.GenPrivKey()
	voterPubKey := voterPrivKey.PubKey()
	voterAccAddress := sdk.AccAddress(voterPubKey.Address())
	voterAccAddressHex := hex.EncodeToString(voterAccAddress.Bytes()) // Expected format for msg.Voter

	err := borKeeper.AddNewSpan(ctx, &types.Span{
		Id:         1,
		StartBlock: 1,
		EndBlock:   1000,
		BorChainId: "1",
	})

	helper.SetRioHeight(1000)

	require.NoError(err)

	// Validator whose PubKey matches voterAccAddress
	matchingVal := staketypes.Validator{
		ValId:  1,
		Signer: voterAccAddress.String(), // Bech32 representation
		PubKey: voterPubKey.Bytes(),      // Raw public key bytes
	}

	// Validator with a different PubKey
	differentPrivKey := secp256k1.GenPrivKey()
	differentPubKey := differentPrivKey.PubKey()
	differentVal := staketypes.Validator{
		ValId:  2,
		Signer: sdk.AccAddress(differentPubKey.Address()).String(),
		PubKey: differentPubKey.Bytes(),
	}

	sampleVotes := types.ProducerVotes{
		Votes: []uint64{10, 20, 30}, // Slice of uint64 as per bor.pb.go
	}

	testCases := []struct {
		name          string
		msg           types.MsgVoteProducers
		mockSetup     func(tc types.MsgVoteProducers)
		expectError   bool
		errorContains string
		verifyState   func(tc types.MsgVoteProducers)
	}{
		{
			name: "successful vote",
			msg: types.MsgVoteProducers{
				Voter:   voterAccAddressHex,
				VoterId: matchingVal.ValId,
				Votes:   sampleVotes,
			},
			mockSetup: func(tc types.MsgVoteProducers) {
				skMock.EXPECT().GetValidatorFromValID(ctx, matchingVal.ValId).Return(matchingVal, nil).Times(1)
			},
			expectError: false,
			verifyState: func(tc types.MsgVoteProducers) {
				storedVotes, err := borKeeper.GetProducerVotes(ctx, tc.VoterId)
				require.NoError(err)
				require.Equal(tc.Votes, storedVotes)
			},
		},
		{
			name: "invalid voter hex address string",
			msg: types.MsgVoteProducers{
				Voter:   "not-a-hex-string",
				VoterId: matchingVal.ValId,
				Votes:   sampleVotes,
			},
			mockSetup:     func(tc types.MsgVoteProducers) {},
			expectError:   true,
			errorContains: "invalid voter address: addresses cannot be empty: unknown address",
		},
		{
			name: "validator not found for VoterId",
			msg: types.MsgVoteProducers{
				Voter:   voterAccAddressHex,
				VoterId: 99,
				Votes:   sampleVotes,
			},
			mockSetup: func(tc types.MsgVoteProducers) {
				skMock.EXPECT().GetValidatorFromValID(ctx, uint64(99)).Return(staketypes.Validator{}, fmt.Errorf("validator with id 99 not found")).Times(1)
			},
			expectError:   true,
			errorContains: "invalid voter id: validator with id 99 not found",
		},
		{
			name: "voter address does not match validator's pubkey address",
			msg: types.MsgVoteProducers{
				Voter:   voterAccAddressHex, // Voter derived from voterPrivKey
				VoterId: differentVal.ValId, // Validator derived from differentPrivKey
				Votes:   sampleVotes,
			},
			mockSetup: func(tc types.MsgVoteProducers) {
				skMock.EXPECT().GetValidatorFromValID(ctx, differentVal.ValId).Return(differentVal, nil).Times(1)
			},
			expectError:   true,
			errorContains: "does not match validator address",
		},
		{
			name: "duplicate votes",
			msg: types.MsgVoteProducers{
				Voter:   voterAccAddressHex,
				VoterId: matchingVal.ValId,
				Votes: types.ProducerVotes{
					Votes: []uint64{10, 20, 10}, // Duplicate vote for 10
				},
			},
			mockSetup: func(tc types.MsgVoteProducers) {
				skMock.EXPECT().GetValidatorFromValID(ctx, matchingVal.ValId).Return(matchingVal, nil).Times(1)
			},
			expectError:   true,
			errorContains: fmt.Sprintf("duplicate vote for validator id %d", 10),
		},
		{
			name: "VEBLOP height not reached - should reject vote",
			msg: types.MsgVoteProducers{
				Voter:   voterAccAddressHex,
				VoterId: matchingVal.ValId,
				Votes:   sampleVotes,
			},
			mockSetup: func(tc types.MsgVoteProducers) {
				// Set VEBLOP height to be higher than next span start (1001)
				// This simulates the case where we haven't reached VEBLOP phase yet
				helper.SetRioHeight(1002) // Much higher than span end (1000) + 1
			},
			expectError:   true,
			errorContains: "span is not in VEBLOP phase",
		},
		{
			name: "VEBLOP height exactly at boundary - should accept vote",
			msg: types.MsgVoteProducers{
				Voter:   voterAccAddressHex,
				VoterId: matchingVal.ValId,
				Votes:   sampleVotes,
			},
			mockSetup: func(tc types.MsgVoteProducers) {
				// Set VEBLOP height exactly at next span start (1001)
				// This should be in VEBLOP phase and allow the vote
				helper.SetRioHeight(1001) // Exactly at span end (1000) + 1
				skMock.EXPECT().GetValidatorFromValID(ctx, matchingVal.ValId).Return(matchingVal, nil).Times(1)
			},
			expectError: false,
			verifyState: func(tc types.MsgVoteProducers) {
				storedVotes, err := borKeeper.GetProducerVotes(ctx, tc.VoterId)
				require.NoError(err)
				require.Equal(tc.Votes, storedVotes)
			},
		},
		{
			name: "VEBLOP height just below boundary - should reject vote",
			msg: types.MsgVoteProducers{
				Voter:   voterAccAddressHex,
				VoterId: matchingVal.ValId,
				Votes:   sampleVotes,
			},
			mockSetup: func(tc types.MsgVoteProducers) {
				// Set VEBLOP height just below next span start (1001)
				// This should NOT be in VEBLOP phase and reject the vote
				helper.SetRioHeight(1002) // Above span end (1000) + 1, so 1001 < 1002 = not in VEBLOP phase
				// Note: No mock setup needed since this should fail at VEBLOP validation before reaching validator lookup
			},
			expectError:   true,
			errorContains: "span is not in VEBLOP phase",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// Reset VEBLOP height to default for proper test isolation
			helper.SetRioHeight(1000) // Reset to span EndBlock for VEBLOP phase

			_ = borKeeper.SetProducerVotes(ctx, tc.msg.VoterId, types.ProducerVotes{})

			tc.mockSetup(tc.msg)

			res, err := msgServer.VoteProducers(ctx, &tc.msg)

			if tc.expectError {
				require.Error(err)
				require.Contains(err.Error(), tc.errorContains)
				require.Nil(res)
			} else {
				require.NoError(err)
				require.NotNil(res)
				require.Equal(&types.MsgVoteProducersResponse{}, res)
				if tc.verifyState != nil {
					tc.verifyState(tc.msg)
				}
			}
		})
	}
}

func (s *KeeperTestSuite) TestBackfillSpans() {
	require, ctx, borKeeper, milestoneKeeper, cmKeeper, msgServer := s.Require(), s.ctx, s.borKeeper, s.milestoneKeeper, s.chainManagerKeeper, s.msgServer

	testChainParams := chainmanagertypes.DefaultParams()
	testSpan := s.genTestSpans(1)[0]
	err := borKeeper.AddNewSpan(ctx, testSpan)
	require.NoError(err)

	testcases := []struct {
		name          string
		backfillSpans types.MsgBackfillSpans
		expRes        *types.MsgBackfillSpansResponse
		expErr        string
	}{
		{
			name: "incorrect validator address",
			backfillSpans: types.MsgBackfillSpans{
				Proposer:        "ValidatorAddress",
				ChainId:         testChainParams.ChainParams.BorChainId,
				LatestSpanId:    1,
				LatestBorSpanId: 7,
			},
			expRes: nil,
			expErr: "invalid proposer address",
		},
		{
			name: "incorrect chain id",
			backfillSpans: types.MsgBackfillSpans{
				Proposer:        common.HexToAddress("someProposer").String(),
				ChainId:         "invalidChainId",
				LatestSpanId:    1,
				LatestBorSpanId: 7,
			},
			expRes: nil,
			expErr: "invalid bor chain id",
		},
		{
			name: "invalid last heimdall span id",
			backfillSpans: types.MsgBackfillSpans{
				Proposer:        common.HexToAddress("someProposer").String(),
				ChainId:         testChainParams.ChainParams.BorChainId,
				LatestSpanId:    2,
				LatestBorSpanId: 7,
			},
			expRes: nil,
			expErr: "invalid span",
		},
		{
			name: "invalid last bor span id",
			backfillSpans: types.MsgBackfillSpans{
				Proposer:        common.HexToAddress("someProposer").String(),
				ChainId:         testChainParams.ChainParams.BorChainId,
				LatestSpanId:    1,
				LatestBorSpanId: 0,
			},
			expErr: "invalid last bor span id",
		},
		{
			name: "mismatch between calculated and provided last span id",
			backfillSpans: types.MsgBackfillSpans{

				Proposer:        common.HexToAddress("someProposer").String(),
				ChainId:         testChainParams.ChainParams.BorChainId,
				LatestSpanId:    1,
				LatestBorSpanId: 3,
			},
			expRes: nil,
			expErr: "invalid span",
		},
	}

	cmKeeper.EXPECT().GetParams(ctx).Return(testChainParams, nil).AnyTimes()
	milestoneKeeper.EXPECT().GetLastMilestone(ctx).Return(&milestoneTypes.Milestone{
		EndBlock: 1000,
	}, nil).AnyTimes()

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			res, err := msgServer.BackfillSpans(ctx, &tc.backfillSpans)
			require.Equal(tc.expRes, res)
			if tc.expErr == "" {
				require.NoError(err)
			} else {
				require.ErrorContains(err, tc.expErr)
			}
		})
	}
}

func (s *KeeperTestSuite) TestCanVoteProducers() {
	require, ctx, borKeeper := s.Require(), s.ctx, s.borKeeper

	// Add a test span
	err := borKeeper.AddNewSpan(ctx, &types.Span{
		Id:         1,
		StartBlock: 100,
		EndBlock:   200,
		BorChainId: "1",
	})
	require.NoError(err)

	testCases := []struct {
		name          string
		rioHeight     int64
		operation     string
		expectError   bool
		errorContains string
	}{
		{
			name:        "VEBLOP phase active - should pass",
			rioHeight:   201, // Next span starts at 201, VEBLOP at 201 = active
			operation:   "test",
			expectError: false,
		},
		{
			name:          "VEBLOP phase not active - should fail",
			rioHeight:     300, // Next span starts at 201, VEBLOP at 300 = not active yet
			operation:     "test",
			expectError:   true,
			errorContains: "span is not in VEBLOP phase",
		},
		{
			name:        "VEBLOP height exactly at boundary - should pass",
			rioHeight:   201, // Exactly at next span start
			operation:   "boundary_test",
			expectError: false,
		},
		{
			name:          "VEBLOP height below next span start - should fail",
			rioHeight:     202, // Above next span start (201), so 201 < 202 = not in VEBLOP phase
			operation:     "below_boundary_test",
			expectError:   true,
			errorContains: "span is not in VEBLOP phase",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			helper.SetRioHeight(tc.rioHeight)

			err := borKeeper.CanVoteProducers(ctx)

			if tc.expectError {
				require.Error(err)
				require.Contains(err.Error(), tc.errorContains)
			} else {
				require.NoError(err)
			}
		})
	}
}

func (s *KeeperTestSuite) TestSetProducerDowntime() {
	require := s.Require()

	baseTs := uint64(1_700_000_000) // arbitrary baseline timestamp for tests

	// Define three validators for tests
	type valInfo struct {
		id      uint64
		hexAddr string
	}
	val1 := valInfo{id: 1, hexAddr: common.HexToAddress("0x0000000000000000000000000000000000000001").Hex()}
	val2 := valInfo{id: 2, hexAddr: common.HexToAddress("0x0000000000000000000000000000000000000002").Hex()}
	val3 := valInfo{id: 3, hexAddr: common.HexToAddress("0x0000000000000000000000000000000000000003").Hex()}

	// Default validator set returned by stake keeper (contains all)
	allValidators := []staketypes.Validator{
		{ValId: val1.id, Signer: val1.hexAddr},
		{ValId: val2.id, Signer: val2.hexAddr},
		{ValId: val3.id, Signer: val3.hexAddr},
	}

	// Default milestone that passes phase check and provides a stable Timestamp.
	okMilestone := &milestoneTypes.Milestone{
		EndBlock:  10,     // < setProducerDowntimeHeight -> phase allowed
		Timestamp: baseTs, // used for future/time-range checks
	}

	// Helper to build the message
	newMsg := func(addr string, start, end uint64) *types.MsgSetProducerDowntime {
		return types.NewMsgSetProducerDowntime(addr, start, end)
	}

	type testCase struct {
		name          string
		msg           *types.MsgSetProducerDowntime
		setup         func()
		expectErr     bool
		errContains   string
		fallbackFirst bool
	}

	tests := []testCase{
		{
			name: "tx should be rejected when last milestone EndBlock >= setProducerDowntimeHeight",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMinimumTimeInFuture+10, baseTs+types.PlannedDowntimeMinimumTimeInFuture+types.PlannedDowntimeMinRange+10),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(5)
				s.milestoneKeeper.EXPECT().
					GetLastMilestone(gomock.Any()).
					Return(&milestoneTypes.Milestone{
						EndBlock:  10, // >= 5 -> should reject
						Timestamp: baseTs,
					}, nil)
			},
			expectErr: true,
		},
		{
			name: "producer address not found in current validator set",
			msg:  newMsg("0x00000000000000000000000000000000000000aa", baseTs+types.PlannedDowntimeMinimumTimeInFuture+10, baseTs+types.PlannedDowntimeMinimumTimeInFuture+types.PlannedDowntimeMinRange+10),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
			},
			expectErr:   true,
			errContains: "producer with address",
		},
		{
			name: "start timestamp equal to end timestamp",
			msg:  newMsg(val1.hexAddr, 1000, 1000),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
			},
			expectErr:   true,
			errContains: "start timestamp must be less than end timestamp",
		},
		{
			name: "should fail on GetLastMilestone error",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMinimumTimeInFuture+10, baseTs+types.PlannedDowntimeMinimumTimeInFuture+types.PlannedDowntimeMinRange+10),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
				s.milestoneKeeper.EXPECT().
					GetLastMilestone(gomock.Any()).
					Return(nil, fmt.Errorf("db error"))
			},
			expectErr:   true,
			errContains: "failed to get latest milestone",
		},
		{
			name: "should fail on error on second call to GetLastMilestone",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMinimumTimeInFuture+10, baseTs+types.PlannedDowntimeMinimumTimeInFuture+types.PlannedDowntimeMinRange+10),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
				first := s.milestoneKeeper.EXPECT().
					GetLastMilestone(gomock.Any()).
					Return(okMilestone, nil)
				s.milestoneKeeper.EXPECT().
					GetLastMilestone(gomock.Any()).
					Return(nil, fmt.Errorf("oops")).
					After(first)
			},
			expectErr:   true,
			errContains: "failed to get latest milestone",
		},
		{
			name: "GetLastMilestone returns nil milestone on second fetch",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMinimumTimeInFuture+10, baseTs+types.PlannedDowntimeMinimumTimeInFuture+types.PlannedDowntimeMinRange+10),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
				first := s.milestoneKeeper.EXPECT().
					GetLastMilestone(gomock.Any()).
					Return(okMilestone, nil)
				s.milestoneKeeper.EXPECT().
					GetLastMilestone(gomock.Any()).
					Return(nil, nil).
					After(first)
			},
			expectErr:   true,
			errContains: "latest milestone not found",
		},
		{
			name: "start too soon (< minimumTimeInFuture)",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMinimumTimeInFuture-1, baseTs+types.PlannedDowntimeMinimumTimeInFuture-1+types.PlannedDowntimeMinRange),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
			},
			expectErr: true,
		},
		{
			name: "start too far in future (> maxTimeInFuture)",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMaximumTimeInFuture+1, baseTs+types.PlannedDowntimeMaximumTimeInFuture+1+types.PlannedDowntimeMinRange),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
			},
			expectErr: true,
		},
		{
			name: "producer not registered in ProducerVotes",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMinimumTimeInFuture+10, baseTs+types.PlannedDowntimeMinimumTimeInFuture+10+types.PlannedDowntimeMinRange),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
				require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
				require.NoError(s.borKeeper.SetProducerVotes(s.ctx, val2.id, types.ProducerVotes{}))
				require.NoError(s.borKeeper.SetProducerVotes(s.ctx, val3.id, types.ProducerVotes{}))
			},
			expectErr: true,
		},
		{
			name: "success: only this producer registered (no other producers)",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMinimumTimeInFuture+100, baseTs+types.PlannedDowntimeMinimumTimeInFuture+100+types.PlannedDowntimeMinRange),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
				require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
				require.NoError(s.borKeeper.SetProducerVotes(s.ctx, val1.id, types.ProducerVotes{}))
			},
			expectErr: true,
		},
		{
			name: "success: other producers exist but none have planned downtime",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMinimumTimeInFuture+200, baseTs+types.PlannedDowntimeMinimumTimeInFuture+200+types.PlannedDowntimeMinRange),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
				require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
				for _, id := range []uint64{val1.id, val2.id, val3.id} {
					require.NoError(s.borKeeper.SetProducerVotes(s.ctx, id, types.ProducerVotes{}))
				}
			},
			expectErr: false,
		},
		{
			name: "success: some overlap but not with all other producers",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMinimumTimeInFuture+500, baseTs+types.PlannedDowntimeMinimumTimeInFuture+500+types.PlannedDowntimeMinRange),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
				require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
				for _, id := range []uint64{val1.id, val2.id, val3.id} {
					require.NoError(s.borKeeper.SetProducerVotes(s.ctx, id, types.ProducerVotes{}))
				}
				_, err := s.msgServer.SetProducerDowntime(s.ctx, newMsg(
					val2.hexAddr,
					baseTs+types.PlannedDowntimeMinimumTimeInFuture+500-10,
					baseTs+types.PlannedDowntimeMinimumTimeInFuture+500+types.PlannedDowntimeMinRange+10,
				))
				require.NoError(err)
			},
			expectErr:     false,
			fallbackFirst: true,
		},
		{
			name: "error: overlaps with all other producers' planned downtimes",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMinimumTimeInFuture+800, baseTs+types.PlannedDowntimeMinimumTimeInFuture+800+types.PlannedDowntimeMinRange),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
				require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
				for _, id := range []uint64{val1.id, val2.id, val3.id} {
					require.NoError(s.borKeeper.SetProducerVotes(s.ctx, id, types.ProducerVotes{}))
				}
				_, err := s.msgServer.SetProducerDowntime(s.ctx, newMsg(
					val2.hexAddr,
					baseTs+types.PlannedDowntimeMinimumTimeInFuture+800-10,
					baseTs+types.PlannedDowntimeMinimumTimeInFuture+800+types.PlannedDowntimeMinRange+10,
				))
				require.NoError(err)

				_, err = s.msgServer.SetProducerDowntime(s.ctx, newMsg(
					val3.hexAddr,
					baseTs+types.PlannedDowntimeMinimumTimeInFuture+800-5,
					baseTs+types.PlannedDowntimeMinimumTimeInFuture+800+types.PlannedDowntimeMinRange+5,
				))
				require.NoError(err)
			},
			expectErr:     true,
			fallbackFirst: true,
		},
		// Boundary and storage-friendly tests (keep)
		{
			name: "boundary ok: start == minimumTimeInFuture and range == minRange",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMinimumTimeInFuture, baseTs+types.PlannedDowntimeMinimumTimeInFuture+types.PlannedDowntimeMinRange),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
				require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
				for _, id := range []uint64{val1.id, val2.id, val3.id} {
					require.NoError(s.borKeeper.SetProducerVotes(s.ctx, id, types.ProducerVotes{}))
				}
			},
			expectErr: false,
		},
		{
			name: "boundary ok: start == maxTimeInFuture and range == maxRange",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMaximumTimeInFuture, baseTs+types.PlannedDowntimeMaximumTimeInFuture+types.PlannedDowntimeMaxRange),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
				require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
				for _, id := range []uint64{val1.id, val2.id, val3.id} {
					require.NoError(s.borKeeper.SetProducerVotes(s.ctx, id, types.ProducerVotes{}))
				}
			},
			expectErr: false,
		},
		{
			name: "success: non-overlapping with other producers' planned downtime",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMinimumTimeInFuture+300, baseTs+types.PlannedDowntimeMinimumTimeInFuture+300+types.PlannedDowntimeMinRange),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
				require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
				for _, id := range []uint64{val1.id, val2.id, val3.id} {
					require.NoError(s.borKeeper.SetProducerVotes(s.ctx, id, types.ProducerVotes{}))
				}
				_, err := s.msgServer.SetProducerDowntime(s.ctx, newMsg(
					val2.hexAddr,
					baseTs+types.PlannedDowntimeMinimumTimeInFuture+100,
					baseTs+types.PlannedDowntimeMinimumTimeInFuture+100+types.PlannedDowntimeMinRange,
				))
				require.NoError(err)
			},
			expectErr:     false,
			fallbackFirst: true,
		},
		{
			name: "error: producer registered but has identical full-overlap with all others",
			msg:  newMsg(val1.hexAddr, baseTs+types.PlannedDowntimeMinimumTimeInFuture+900, baseTs+types.PlannedDowntimeMinimumTimeInFuture+900+types.PlannedDowntimeMinRange),
			setup: func() {
				helper.SetSetProducerDowntimeHeight(1_000_000)
				require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
				for _, id := range []uint64{val1.id, val2.id, val3.id} {
					require.NoError(s.borKeeper.SetProducerVotes(s.ctx, id, types.ProducerVotes{}))
				}
				start := baseTs + types.PlannedDowntimeMinimumTimeInFuture + 900
				end := start + types.PlannedDowntimeMinRange
				_, err := s.msgServer.SetProducerDowntime(s.ctx, newMsg(val2.hexAddr, start, end))
				require.NoError(err)
				_, err = s.msgServer.SetProducerDowntime(s.ctx, newMsg(val3.hexAddr, start, end))
				require.NoError(err)
			},
			expectErr:     true,
			fallbackFirst: true,
		},
	}

	for _, tc := range tests {
		s.T().Run(tc.name, func(t *testing.T) {
			// Reset the entire suite to get fresh context, keeper, and mocks per subtest.
			s.SetupTest()

			ctx := s.ctx
			msgServer := s.msgServer

			// Default to "allowed" phase unless overridden
			helper.SetSetProducerDowntimeHeight(1_000_000)

			addFallbacks := func() {
				// Use gomock.Any() for sdk.Context to avoid brittle equality
				s.milestoneKeeper.EXPECT().
					GetLastMilestone(gomock.Any()).
					Return(okMilestone, nil).
					AnyTimes()
				s.stakeKeeper.EXPECT().
					GetSpanEligibleValidators(gomock.Any()).
					Return(allValidators).
					AnyTimes()
			}

			if tc.fallbackFirst {
				addFallbacks()
			}

			tc.setup()

			if !tc.fallbackFirst {
				addFallbacks()
			}

			res, err := msgServer.SetProducerDowntime(ctx, tc.msg)

			if tc.expectErr {
				require.Error(err)
				if tc.errContains != "" {
					require.Contains(err.Error(), tc.errContains)
				}
				require.Nil(res)
			} else {
				require.NoError(err)
				require.NotNil(res)
				require.IsType(&types.MsgSetProducerDowntimeResponse{}, res)
			}
		})
	}
}
