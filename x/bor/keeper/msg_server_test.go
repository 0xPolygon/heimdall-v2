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
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
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
