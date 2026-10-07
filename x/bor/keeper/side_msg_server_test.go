package keeper_test

import (
	"fmt"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func (s *KeeperTestSuite) TestSideHandleMsgSpan() {
	ctx, require, borKeeper, milestoneKeeper, cmKeeper, sideMsgServer := s.ctx, s.Require(), s.borKeeper, s.milestoneKeeper, s.chainManagerKeeper, s.sideMsgServer
	testChainParams, contractCaller := chainmanagertypes.DefaultParams(), &s.contractCaller

	borParams := types.DefaultParams()
	err := borKeeper.SetParams(ctx, borParams)
	require.NoError(err)

	cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(testChainParams, nil)

	valSet, vals := s.genTestValidators()

	spans := []types.Span{
		{
			Id:                0,
			StartBlock:        0,
			EndBlock:          256,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        "test-chain",
		},
		{
			Id:                1,
			StartBlock:        257,
			EndBlock:          6656,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        "test-chain",
		},
		{
			Id:                2,
			StartBlock:        6657,
			EndBlock:          16656,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        "test-chain",
		},
		{
			Id:                3,
			StartBlock:        16657,
			EndBlock:          26656,
			ValidatorSet:      valSet,
			SelectedProducers: vals,
			BorChainId:        "test-chain",
		},
	}

	seedBlock1 := spans[3].EndBlock
	val1Addr := common.HexToAddress(vals[0].GetOperator())
	blockHeader1 := ethTypes.Header{Number: big.NewInt(int64(seedBlock1))}
	blockHash1 := blockHeader1.Hash()
	blockHeader2 := ethTypes.Header{Number: big.NewInt(int64(seedBlock1 - 200))}
	blockHash2 := blockHeader2.Hash()

	for _, span := range spans {
		err := borKeeper.AddNewSpan(ctx, &span)
		require.NoError(err)
		err = borKeeper.StoreSeedProducer(ctx, span.Id, &val1Addr)
	}

	startBlock := uint64(26657)
	correctEndBlock := startBlock + borParams.SpanDuration - 1
	incorrectEndBlock := correctEndBlock - 100

	testcases := []struct {
		lastSpanId       uint64
		lastSeedProducer *common.Address
		expSeed          common.Hash
		name             string
		msg              sdk.Msg
		expVote          sidetxs.Vote
		mockFn           func()
	}{
		{
			name: "seed mismatch",
			msg: &types.MsgProposeSpan{
				SpanId:     4,
				Proposer:   val1Addr.String(),
				StartBlock: startBlock,
				EndBlock:   correctEndBlock,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       []byte("someWrongSeed"),
			},
			lastSeedProducer: &val1Addr,
			lastSpanId:       3,
			expSeed:          blockHash1,
			expVote:          sidetxs.Vote_VOTE_NO,
			mockFn: func() {
				contractCaller.On("GetBorChainBlockAuthor", mock.Anything, mock.Anything).Return(&val1Addr, nil)
				contractCaller.On("GetBorChainBlock", mock.Anything, mock.Anything).Return(&blockHeader1, nil)
			},
		},
		{
			name: "span duration mismatch",
			msg: &types.MsgProposeSpan{
				SpanId:     4,
				Proposer:   val1Addr.String(),
				StartBlock: startBlock,
				EndBlock:   incorrectEndBlock,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       blockHash2.Bytes(),
				SeedAuthor: val1Addr.Hex(),
			},
			lastSeedProducer: &val1Addr,
			lastSpanId:       3,
			expSeed:          blockHash2,
			expVote:          sidetxs.Vote_VOTE_NO,
			mockFn:           nil, // early return before any contract calls
		},
		{
			name: "span is not in turn",
			msg: &types.MsgProposeSpan{
				SpanId:     4,
				Proposer:   val1Addr.String(),
				StartBlock: startBlock,
				EndBlock:   correctEndBlock,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       blockHash1.Bytes(),
			},
			lastSeedProducer: &val1Addr,
			lastSpanId:       3,
			expSeed:          blockHash1,
			expVote:          sidetxs.Vote_VOTE_NO,
			mockFn: func() {
				contractCaller.On("GetBorChainBlockAuthor", mock.Anything, mock.Anything).Return(&val1Addr, nil)
				contractCaller.On("GetBorChainBlock", mock.Anything, big.NewInt(16656)).Return(&blockHeader1, nil).Times(1)
			},
		},
		{
			name: "correct span is proposed",
			msg: &types.MsgProposeSpan{
				SpanId:     4,
				Proposer:   val1Addr.String(),
				StartBlock: startBlock,
				EndBlock:   correctEndBlock,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       blockHash2.Bytes(),
				SeedAuthor: val1Addr.Hex(),
			},
			lastSeedProducer: &val1Addr,
			lastSpanId:       3,
			expSeed:          blockHash2,
			expVote:          sidetxs.Vote_VOTE_YES,
			mockFn: func() {
				contractCaller.On("GetBorChainBlockAuthor", mock.Anything, mock.Anything).Return(&val1Addr, nil)
				contractCaller.On("GetBorChainBlock", mock.Anything, mock.Anything).Return(&blockHeader2, nil)
			},
		},
	}

	milestoneKeeper.EXPECT().GetLastMilestone(ctx).Return(&milestoneTypes.Milestone{
		EndBlock: 1000,
	}, nil).AnyTimes()

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			if tc.mockFn != nil {
				tc.mockFn()
			}
			sideHandler := sideMsgServer.SideTxHandler(sdk.MsgTypeURL(&types.MsgProposeSpan{}))
			res := sideHandler(s.ctx, tc.msg)
			require.Equal(tc.expVote, res)
		})
		// clean up the contract caller to update the mocked expected calls
		s.contractCaller = mocks.IContractCaller{}
	}
}

func (s *KeeperTestSuite) TestPostHandleMsgEventSpan() {
	require, ctx, stakeKeeper, borKeeper, sideMsgServer, contractCaller := s.Require(), s.ctx, s.stakeKeeper, s.borKeeper, s.sideMsgServer, &s.contractCaller

	stakeKeeper.EXPECT().GetSpanEligibleValidators(ctx).Times(1)
	stakeKeeper.EXPECT().GetValidatorSet(ctx).Times(1)
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, gomock.Any()).AnyTimes()

	borParams := types.DefaultParams()
	err := borKeeper.SetParams(ctx, borParams)
	require.NoError(err)

	// add genesis span
	err = borKeeper.AddNewSpan(ctx, &types.Span{
		Id:         0,
		StartBlock: 0,
		EndBlock:   100,
	})
	require.NoError(err)

	producer1 := common.HexToAddress("0xc0ffee254729296a45a3885639AC7E10F9d54979")
	producer2 := common.HexToAddress("0xd0ffee254729296a45a3885639AC7E10F9d54979")
	err = borKeeper.StoreSeedProducer(s.ctx, 1, &producer1)
	s.Require().NoError(err)

	lastBorBlockHeader := &ethTypes.Header{Number: big.NewInt(0)}
	contractCaller.On("GetBorChainBlock", mock.Anything, big.NewInt(0)).Return(lastBorBlockHeader, nil).Times(1)
	contractCaller.On("GetBorChainBlockAuthor", mock.Anything, big.NewInt(0)).Return(&producer1, nil).Times(1)
	contractCaller.On("GetBorChainBlockAuthor", mock.Anything, big.NewInt(100)).Return(&producer2, nil).Times(1)

	testChainParams := chainmanagertypes.DefaultParams()
	spans := s.genTestSpans(1)
	err = borKeeper.AddNewSpan(ctx, spans[0])
	require.NoError(err)

	testcases := []struct {
		name          string
		msg           sdk.Msg
		vote          sidetxs.Vote
		expLastSpanId uint64
	}{
		{
			name: "doesn't have yes vote",
			msg: &types.MsgProposeSpan{
				SpanId:     2,
				Proposer:   "0x91b54cD48FD796A5d0A120A4C5298a7fAEA59B",
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testSeed1").Bytes(),
			},
			vote:          sidetxs.Vote_VOTE_NO,
			expLastSpanId: spans[0].Id,
		},
		{
			name: "span replayed",
			msg: &types.MsgProposeSpan{
				SpanId:     1,
				Proposer:   common.HexToAddress("someProposer").String(),
				StartBlock: 1,
				EndBlock:   101,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testSeed1").Bytes(),
			},
			vote:          sidetxs.Vote_VOTE_YES,
			expLastSpanId: spans[0].Id,
		},
		{
			name: "correct span is proposed",
			msg: &types.MsgProposeSpan{
				SpanId:     2,
				Proposer:   common.HexToAddress("someProposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testSeed1").Bytes(),
			},
			vote:          sidetxs.Vote_VOTE_YES,
			expLastSpanId: 2,
		},
	}

	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			postHandler := sideMsgServer.PostTxHandler(sdk.MsgTypeURL(&types.MsgProposeSpan{}))
			_ = postHandler(ctx, tc.msg, tc.vote)

			lastSpan, err := borKeeper.GetLastSpan(ctx)
			require.NoError(err)
			require.Equal(tc.expLastSpanId, lastSpan.Id)
		})
	}
}

func (s *KeeperTestSuite) TestSideHandleSetProducerDowntime() {
	require := s.Require()

	minFuture := uint64(types.PlannedDowntimeMinimumTimeInFuture)
	maxFuture := types.PlannedDowntimeMaximumTimeInFuture
	producerAddr := common.HexToAddress("0x0000000000000000000000000000000000000001").Hex()
	otherProducerAddr := common.HexToAddress("0x0000000000000000000000000000000000000002").Hex()

	newMsg := func(producer string, start, end uint64) *types.MsgSetProducerDowntime {
		return &types.MsgSetProducerDowntime{
			Producer:      producer,
			DowntimeRange: types.BlockRange{StartBlock: start, EndBlock: end},
		}
	}

	type testCase struct {
		name                string
		typeMismatch        bool
		current             uint64
		msg                 *types.MsgSetProducerDowntime
		getBlockErr         error
		activeProducerAddrs []string
		expectVote          sidetxs.Vote
	}

	tests := []testCase{
		{
			name:         "type mismatch returns NO",
			typeMismatch: true,
			expectVote:   sidetxs.Vote_VOTE_NO,
		},
		{
			name:        "GetBorChainBlock error returns NO",
			current:     1_000_000,
			msg:         newMsg(producerAddr, 1_000_100, 1_000_200),
			getBlockErr: fmt.Errorf("rpc error"),
			expectVote:  sidetxs.Vote_VOTE_NO,
		},
		{
			name:                "producer not in active producers set returns NO",
			current:             1_000_000,
			msg:                 newMsg(producerAddr, 1_000_200, 1_000_400),
			activeProducerAddrs: []string{otherProducerAddr},
			expectVote:          sidetxs.Vote_VOTE_NO,
		},
		{
			name:       "start too soon - boundary (start == current+min-1) returns NO",
			current:    5_000_000,
			msg:        newMsg(producerAddr, (5_000_000+minFuture)-1, (5_000_000+minFuture)+10),
			expectVote: sidetxs.Vote_VOTE_NO,
		},
		{
			name:       "start too soon - strict (start < current+min-1) returns NO",
			current:    5_000_000,
			msg:        newMsg(producerAddr, (5_000_000+minFuture)-2, (5_000_000+minFuture)+10),
			expectVote: sidetxs.Vote_VOTE_NO,
		},
		{
			// handler rejects only if the end > current+maxFuture; equality is allowed
			name:       "end boundary (end == current+max) returns YES",
			current:    2_000_000,
			msg:        newMsg(producerAddr, 2_000_000+minFuture, 2_000_000+maxFuture),
			expectVote: sidetxs.Vote_VOTE_YES,
		},
		{
			name:       "end too far - strict (end > current+max) returns NO",
			current:    2_000_000,
			msg:        newMsg(producerAddr, 2_000_000+minFuture, 2_000_000+maxFuture+1),
			expectVote: sidetxs.Vote_VOTE_NO,
		},
		{
			name:    "passes both checks - boundary just passing returns YES",
			current: 3_000_000,
			// start == current+min, end == current+max-1
			msg:        newMsg(producerAddr, 3_000_000+minFuture, (3_000_000+maxFuture)-1),
			expectVote: sidetxs.Vote_VOTE_YES,
		},
		{
			name:    "passes both checks - start well in future, end well within max returns YES",
			current: 4_000_000,
			// start >= current+min, end < current+max
			msg:        newMsg(producerAddr, 4_000_000+minFuture+100, 4_000_000+maxFuture-100),
			expectVote: sidetxs.Vote_VOTE_YES,
		},
	}

	// range too small (end - start < PlannedDowntimeMinRange) -> VOTE_NO
	tests = append(tests, testCase{
		name:       "range too small returns NO",
		current:    1_000_000,
		msg:        newMsg(producerAddr, 1_000_000+minFuture, 1_000_000+minFuture+10),
		expectVote: sidetxs.Vote_VOTE_NO,
	})

	// start exactly at minFuture with valid range -> YES
	tests = append(tests, testCase{
		name:       "start at exact minFuture boundary with valid range returns YES",
		current:    6_000_000,
		msg:        newMsg(producerAddr, 6_000_000+minFuture, 6_000_000+minFuture+uint64(types.PlannedDowntimeMinRange)),
		expectVote: sidetxs.Vote_VOTE_YES,
	})

	for _, tc := range tests {
		s.T().Run(tc.name, func(t *testing.T) {
			// fresh state and mocks per subtest
			s.SetupTest()
			ctx := s.ctx

			var msgI sdk.Msg
			if tc.typeMismatch {
				// Any other message type should lead to NO
				msgI = &types.MsgProposeSpan{}
			} else {
				msgI = tc.msg
				if tc.getBlockErr != nil {
					s.contractCaller.On("GetBorChainBlock", mock.Anything, (*big.Int)(nil)).
						Return((*ethTypes.Header)(nil), tc.getBlockErr).Once()
				} else {
					producers := tc.activeProducerAddrs
					if len(producers) == 0 {
						producers = []string{producerAddr}
					}

					selectedProducers := make([]stakeTypes.Validator, 0, len(producers))
					for i, producer := range producers {
						selectedProducers = append(selectedProducers, stakeTypes.Validator{
							ValId:  uint64(i + 1),
							Signer: producer,
						})
					}

					require.NoError(s.borKeeper.AddNewSpan(ctx, &types.Span{
						Id:                1,
						StartBlock:        1,
						EndBlock:          tc.current + maxFuture + 10_000,
						SelectedProducers: selectedProducers,
						BorChainId:        "bor",
					}))

					s.contractCaller.On("GetBorChainBlock", mock.Anything, (*big.Int)(nil)).
						Return(&ethTypes.Header{Number: big.NewInt(int64(tc.current))}, nil).Once()
				}
			}

			sideHandler := s.sideMsgServer.SideTxHandler(sdk.MsgTypeURL(&types.MsgSetProducerDowntime{}))
			v := sideHandler(ctx, msgI)
			require.Equal(tc.expectVote, v)

			// verify expectations for contract caller when applicable
			s.contractCaller.AssertExpectations(s.T())
		})
	}
}

func (s *KeeperTestSuite) TestSideHandleSetProducerDowntimeStartGteEnd() {
	s.SetupTest()
	msg := &types.MsgSetProducerDowntime{
		Producer:      common.HexToAddress("0x0000000000000000000000000000000000000001").Hex(),
		DowntimeRange: types.BlockRange{StartBlock: 500, EndBlock: 500},
	}
	sideHandler := s.sideMsgServer.SideTxHandler(sdk.MsgTypeURL(&types.MsgSetProducerDowntime{}))
	s.Require().Equal(sidetxs.Vote_VOTE_NO, sideHandler(s.ctx, msg))

	msg.DowntimeRange.StartBlock = 600
	s.Require().Equal(sidetxs.Vote_VOTE_NO, sideHandler(s.ctx, msg))
}

func (s *KeeperTestSuite) TestSideHandleSetProducerDowntimeTargetProducer() {
	require := s.Require()

	minFuture := uint64(types.PlannedDowntimeMinimumTimeInFuture)
	addr1 := common.HexToAddress("0x0000000000000000000000000000000000000001").Hex()
	addr2 := common.HexToAddress("0x0000000000000000000000000000000000000002").Hex()
	addr3 := common.HexToAddress("0x0000000000000000000000000000000000000003").Hex()
	id1, id2, id3 := uint64(1), uint64(2), uint64(3)

	current := uint64(3_000_000)
	start := current + minFuture
	end := start + uint64(types.PlannedDowntimeMinRange)

	type testCase struct {
		name       string
		msg        *types.MsgSetProducerDowntime
		expectVote sidetxs.Vote
	}

	tests := []testCase{
		{
			// target=0 -> skip target validation entirely -> YES
			name: "target zero accepted",
			msg: &types.MsgSetProducerDowntime{
				Producer:         addr1,
				DowntimeRange:    types.BlockRange{StartBlock: start, EndBlock: end},
				TargetProducerId: types.RoundRobinDefault,
			},
			expectVote: sidetxs.Vote_VOTE_YES,
		},
		{
			// valid target: exists, not self, in producer set -> YES
			name: "valid target in producer set accepted",
			msg: &types.MsgSetProducerDowntime{
				Producer:         addr1,
				DowntimeRange:    types.BlockRange{StartBlock: start, EndBlock: end},
				TargetProducerId: id2,
			},
			expectVote: sidetxs.Vote_VOTE_YES,
		},
		{
			// target is the declaring producer -> NO
			name: "target is self rejected",
			msg: &types.MsgSetProducerDowntime{
				Producer:         addr1,
				DowntimeRange:    types.BlockRange{StartBlock: start, EndBlock: end},
				TargetProducerId: id1,
			},
			expectVote: sidetxs.Vote_VOTE_NO,
		},
		{
			// target is a valid validator but not in the computed producer set -> NO
			name: "target not in producer set rejected",
			msg: &types.MsgSetProducerDowntime{
				Producer:         addr1,
				DowntimeRange:    types.BlockRange{StartBlock: start, EndBlock: end},
				TargetProducerId: 999,
			},
			expectVote: sidetxs.Vote_VOTE_NO,
		},
	}

	// Pre-fork tests: target != 0 before fork height -> NO
	preForkTests := []testCase{
		{
			name: "target rejected before fork height",
			msg: &types.MsgSetProducerDowntime{
				Producer:         addr1,
				DowntimeRange:    types.BlockRange{StartBlock: start, EndBlock: end},
				TargetProducerId: id2,
			},
			expectVote: sidetxs.Vote_VOTE_NO,
		},
		{
			name: "round-robin default accepted before fork height",
			msg: &types.MsgSetProducerDowntime{
				Producer:         addr1,
				DowntimeRange:    types.BlockRange{StartBlock: start, EndBlock: end},
				TargetProducerId: types.RoundRobinDefault,
			},
			expectVote: sidetxs.Vote_VOTE_YES,
		},
	}
	tests = append(tests, preForkTests...)

	for _, tc := range tests {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			ctx := s.ctx

			// Stake mocks: 3 validators, all vote for [id1, id2, id3] so producer set = [id1, id2, id3]
			s.stakeKeeper.EXPECT().
				GetValidatorSet(gomock.Any()).
				Return(stakeTypes.ValidatorSet{
					Validators: []*stakeTypes.Validator{
						{ValId: id1, Signer: addr1, VotingPower: 100},
						{ValId: id2, Signer: addr2, VotingPower: 100},
						{ValId: id3, Signer: addr3, VotingPower: 100},
					},
				}, nil).
				AnyTimes()

			s.stakeKeeper.EXPECT().
				GetValidatorFromValID(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ sdk.Context, vid uint64) (stakeTypes.Validator, error) {
					switch vid {
					case id1:
						return stakeTypes.Validator{ValId: id1, Signer: addr1, VotingPower: 100}, nil
					case id2:
						return stakeTypes.Validator{ValId: id2, Signer: addr2, VotingPower: 100}, nil
					case id3:
						return stakeTypes.Validator{ValId: id3, Signer: addr3, VotingPower: 100}, nil
					default:
						return stakeTypes.Validator{}, fmt.Errorf("unknown validator id %d", vid)
					}
				}).
				AnyTimes()

			// GetValIdFromAddress mock for ID-based self-targeting check
			s.stakeKeeper.EXPECT().
				GetValIdFromAddress(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ sdk.Context, addr string) (uint64, error) {
					switch addr {
					case addr1:
						return id1, nil
					case addr2:
						return id2, nil
					case addr3:
						return id3, nil
					default:
						return 0, fmt.Errorf("unknown address %s", addr)
					}
				}).
				AnyTimes()

			// All vote for [id1, id2, id3] -> CalculateProducerSet returns [id1, id2, id3]
			require.NoError(s.borKeeper.ClearProducerVotes(ctx))
			for _, voter := range []uint64{id1, id2, id3} {
				require.NoError(s.borKeeper.SetProducerVotes(ctx, voter, types.ProducerVotes{Votes: []uint64{id1, id2, id3}}))
			}

			// Span covering the downtime range with addr1 as active producer
			maxFuture := types.PlannedDowntimeMaximumTimeInFuture
			require.NoError(s.borKeeper.AddNewSpan(ctx, &types.Span{
				Id:         1,
				StartBlock: 1,
				EndBlock:   current + maxFuture + 10_000,
				SelectedProducers: []stakeTypes.Validator{
					{ValId: id1, Signer: addr1},
					{ValId: id2, Signer: addr2},
					{ValId: id3, Signer: addr3},
				},
				BorChainId: "bor",
			}))

			// Set fork height high for pre-fork tests, restore after.
			isPreFork := false
			for _, pf := range preForkTests {
				if pf.name == tc.name {
					isPreFork = true
					break
				}
			}
			if isPreFork {
				helper.SetZurichHardforkHeight(999999)
				defer helper.SetZurichHardforkHeight(0)
			}

			s.contractCaller.On("GetBorChainBlock", mock.Anything, (*big.Int)(nil)).
				Return(&ethTypes.Header{Number: big.NewInt(int64(current))}, nil).Once()

			sideHandler := s.sideMsgServer.SideTxHandler(sdk.MsgTypeURL(&types.MsgSetProducerDowntime{}))
			v := sideHandler(ctx, tc.msg)
			require.Equal(tc.expectVote, v)

			s.contractCaller.AssertExpectations(s.T())
		})
	}
}

func (s *KeeperTestSuite) TestSideHandleSetProducerDowntimeGetValIdError() {
	require := s.Require()

	minFuture := uint64(types.PlannedDowntimeMinimumTimeInFuture)
	addr1 := common.HexToAddress("0x0000000000000000000000000000000000000001").Hex()
	addr2 := common.HexToAddress("0x0000000000000000000000000000000000000002").Hex()
	addr3 := common.HexToAddress("0x0000000000000000000000000000000000000003").Hex()
	id1, id2, id3 := uint64(1), uint64(2), uint64(3)

	current := uint64(3_000_000)
	start := current + minFuture
	end := start + uint64(types.PlannedDowntimeMinRange)

	s.SetupTest()
	ctx := s.ctx

	s.stakeKeeper.EXPECT().
		GetValidatorSet(gomock.Any()).
		Return(stakeTypes.ValidatorSet{
			Validators: []*stakeTypes.Validator{
				{ValId: id1, Signer: addr1, VotingPower: 100},
				{ValId: id2, Signer: addr2, VotingPower: 100},
				{ValId: id3, Signer: addr3, VotingPower: 100},
			},
		}, nil).
		AnyTimes()

	s.stakeKeeper.EXPECT().
		GetValidatorFromValID(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ sdk.Context, vid uint64) (stakeTypes.Validator, error) {
			switch vid {
			case id1:
				return stakeTypes.Validator{ValId: id1, Signer: addr1, VotingPower: 100}, nil
			case id2:
				return stakeTypes.Validator{ValId: id2, Signer: addr2, VotingPower: 100}, nil
			case id3:
				return stakeTypes.Validator{ValId: id3, Signer: addr3, VotingPower: 100}, nil
			default:
				return stakeTypes.Validator{}, fmt.Errorf("unknown validator id %d", vid)
			}
		}).
		AnyTimes()

	// GetValIdFromAddress returns an error -> should vote NO
	s.stakeKeeper.EXPECT().
		GetValIdFromAddress(gomock.Any(), addr1).
		Return(uint64(0), fmt.Errorf("address lookup failed")).
		Times(1)

	require.NoError(s.borKeeper.ClearProducerVotes(ctx))
	for _, voter := range []uint64{id1, id2, id3} {
		require.NoError(s.borKeeper.SetProducerVotes(ctx, voter, types.ProducerVotes{Votes: []uint64{id1, id2, id3}}))
	}

	maxFuture := types.PlannedDowntimeMaximumTimeInFuture
	require.NoError(s.borKeeper.AddNewSpan(ctx, &types.Span{
		Id:         1,
		StartBlock: 1,
		EndBlock:   current + maxFuture + 10_000,
		SelectedProducers: []stakeTypes.Validator{
			{ValId: id1, Signer: addr1},
			{ValId: id2, Signer: addr2},
			{ValId: id3, Signer: addr3},
		},
		BorChainId: "bor",
	}))

	s.contractCaller.On("GetBorChainBlock", mock.Anything, (*big.Int)(nil)).
		Return(&ethTypes.Header{Number: big.NewInt(int64(current))}, nil).Once()

	msg := &types.MsgSetProducerDowntime{
		Producer:         addr1,
		DowntimeRange:    types.BlockRange{StartBlock: start, EndBlock: end},
		TargetProducerId: id2,
	}
	sideHandler := s.sideMsgServer.SideTxHandler(sdk.MsgTypeURL(&types.MsgSetProducerDowntime{}))
	v := sideHandler(ctx, msg)
	require.Equal(sidetxs.Vote_VOTE_NO, v)
}

func (s *KeeperTestSuite) TestPostHandleSetProducerDowntimeTargetProducer() {
	require := s.Require()

	addr1 := common.HexToAddress("0x0000000000000000000000000000000000000001").Hex()
	addr2 := common.HexToAddress("0x0000000000000000000000000000000000000002").Hex()
	addr3 := common.HexToAddress("0x0000000000000000000000000000000000000003").Hex()
	id1, id2, id3 := uint64(1), uint64(2), uint64(3)

	primeStakeMocks := func() {
		s.stakeKeeper.EXPECT().
			GetValidatorSet(gomock.Any()).
			Return(stakeTypes.ValidatorSet{
				Validators: []*stakeTypes.Validator{
					{ValId: id1, Signer: addr1, VotingPower: 100},
					{ValId: id2, Signer: addr2, VotingPower: 100},
					{ValId: id3, Signer: addr3, VotingPower: 100},
				},
			}, nil).
			AnyTimes()

		s.stakeKeeper.EXPECT().
			GetSpanEligibleValidators(gomock.Any()).
			Return([]stakeTypes.Validator{
				{ValId: id1, Signer: addr1, VotingPower: 100},
				{ValId: id2, Signer: addr2, VotingPower: 100},
				{ValId: id3, Signer: addr3, VotingPower: 100},
			}).
			AnyTimes()

		s.stakeKeeper.EXPECT().
			GetValidatorFromValID(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ sdk.Context, vid uint64) (stakeTypes.Validator, error) {
				switch vid {
				case id1:
					return stakeTypes.Validator{ValId: id1, Signer: addr1, VotingPower: 100}, nil
				case id2:
					return stakeTypes.Validator{ValId: id2, Signer: addr2, VotingPower: 100}, nil
				case id3:
					return stakeTypes.Validator{ValId: id3, Signer: addr3, VotingPower: 100}, nil
				default:
					return stakeTypes.Validator{}, fmt.Errorf("unknown validator id %d", vid)
				}
			}).
			AnyTimes()
	}

	setVotesForAll := func(voteList []uint64) {
		require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
		for _, voter := range []uint64{id1, id2, id3} {
			require.NoError(s.borKeeper.SetProducerVotes(s.ctx, voter, types.ProducerVotes{Votes: voteList}))
		}
	}

	type testCase struct {
		name                string
		targetProducerID    uint64
		targetDown          bool // set planned downtime for target
		expectSelectedValId uint64
	}

	tests := []testCase{
		{
			// Valid target, active, not down -> span gets the target as producer
			name:                "valid target selected as producer",
			targetProducerID:    id3,
			expectSelectedValId: id3,
		},
		{
			// Target has overlapping downtime -> falls through to round-robin
			name:                "target down falls through to round-robin",
			targetProducerID:    id3,
			targetDown:          true,
			expectSelectedValId: id2, // round-robin from current=id1
		},
		{
			// target=0 -> standard round-robin
			name:                "target zero uses round-robin",
			targetProducerID:    0,
			expectSelectedValId: id2,
		},
	}

	for _, tc := range tests {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			require.NoError(s.borKeeper.SetParams(s.ctx, types.DefaultParams()))
			primeStakeMocks()

			// Producer set = [id1, id2, id3]
			setVotesForAll([]uint64{id1, id2, id3})

			// Seed spans and active producers
			require.NoError(s.borKeeper.UpdateLatestActiveProducer(s.ctx, map[uint64]struct{}{id2: {}, id3: {}}))

			valSet, vals := s.genTestValidators()
			require.NotEmpty(vals)

			sp0Prods := make([]stakeTypes.Validator, len(vals))
			copy(sp0Prods, vals)
			sp0Prods[0].ValId = id1

			spans := []types.Span{
				{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
				{Id: 1, StartBlock: 200, EndBlock: 299, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
				{Id: 2, StartBlock: 300, EndBlock: 399, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
			}
			for i := range spans {
				require.NoError(s.borKeeper.AddNewSpan(s.ctx, &spans[i]))
			}

			if tc.targetDown {
				require.NoError(s.borKeeper.ProducerPlannedDowntime.Set(s.ctx, tc.targetProducerID,
					types.BlockRange{StartBlock: 100, EndBlock: 500}))
			}

			s.stakeKeeper.EXPECT().
				GetValIdFromAddress(gomock.Any(), addr1).
				Return(id1, nil).
				Times(1)

			msg := &types.MsgSetProducerDowntime{
				Producer:         addr1,
				DowntimeRange:    types.BlockRange{StartBlock: 150, EndBlock: 350},
				TargetProducerId: tc.targetProducerID,
			}

			initialLast, err := s.borKeeper.GetLastSpan(s.ctx)
			require.NoError(err)

			handler := s.sideMsgServer.PostTxHandler(sdk.MsgTypeURL(&types.MsgSetProducerDowntime{}))
			err = handler(s.ctx, msg, sidetxs.Vote_VOTE_YES)
			require.NoError(err)

			// A new span should have been created
			newLast, err := s.borKeeper.GetLastSpan(s.ctx)
			require.NoError(err)
			require.Equal(initialLast.Id+1, newLast.Id)
			require.Len(newLast.SelectedProducers, 1)
			require.Equal(tc.expectSelectedValId, newLast.SelectedProducers[0].ValId)
		})
	}
}

func (s *KeeperTestSuite) TestPostHandleSetProducerDowntime() {
	require := s.Require()

	newMsg := func(prod string, start, end uint64) *types.MsgSetProducerDowntime {
		return &types.MsgSetProducerDowntime{
			Producer:      prod,
			DowntimeRange: types.BlockRange{StartBlock: start, EndBlock: end},
		}
	}

	// Helpers
	setVotes := func(ids ...uint64) {
		require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
		for _, id := range ids {
			// legacy helper kept for some tests; writes empty votes (produces no candidates)
			require.NoError(s.borKeeper.SetProducerVotes(s.ctx, id, types.ProducerVotes{}))
		}
	}

	id1, id2, id3 := uint64(1), uint64(2), uint64(3)

	// New helper: make every validator vote the same ordered candidate list.
	// This drives CalculateProducerSet to return the given candidates (subject to the threshold).
	setVotesForAll := func(voteList []uint64) {
		require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
		for _, voter := range []uint64{id1, id2, id3} {
			require.NoError(s.borKeeper.SetProducerVotes(s.ctx, voter, types.ProducerVotes{Votes: voteList}))
		}
	}

	setPD := func(id, start, end uint64) {
		require.NoError(s.borKeeper.ProducerPlannedDowntime.Set(s.ctx, id, types.BlockRange{
			StartBlock: start, EndBlock: end,
		}))
	}
	getPD := func(id uint64) *types.BlockRange {
		ok, err := s.borKeeper.ProducerPlannedDowntime.Has(s.ctx, id)
		require.NoError(err)
		if !ok {
			return nil
		}
		br, err := s.borKeeper.ProducerPlannedDowntime.Get(s.ctx, id)
		require.NoError(err)
		return &br
	}

	// Producer and ids
	addr1 := common.HexToAddress("0x0000000000000000000000000000000000000001").Hex()
	addr2 := common.HexToAddress("0x0000000000000000000000000000000000000002").Hex()
	addr3 := common.HexToAddress("0x0000000000000000000000000000000000000003").Hex()

	// Prime stake mocks commonly used by CalculateProducerSet/veblop paths.
	primeStakeMocks := func() {
		// Non-zero voting power so thresholds can be met.
		s.stakeKeeper.EXPECT().
			GetValidatorSet(gomock.Any()).
			Return(stakeTypes.ValidatorSet{
				Validators: []*stakeTypes.Validator{
					{ValId: id1, Signer: addr1, VotingPower: 100},
					{ValId: id2, Signer: addr2, VotingPower: 100},
					{ValId: id3, Signer: addr3, VotingPower: 100},
				},
			}, nil).
			AnyTimes()

		s.stakeKeeper.EXPECT().
			GetSpanEligibleValidators(gomock.Any()).
			Return([]stakeTypes.Validator{
				{ValId: id1, Signer: addr1, VotingPower: 100},
				{ValId: id2, Signer: addr2, VotingPower: 100},
				{ValId: id3, Signer: addr3, VotingPower: 100},
			}).
			AnyTimes()

		s.stakeKeeper.EXPECT().
			GetValidatorFromValID(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ sdk.Context, vid uint64) (stakeTypes.Validator, error) {
				switch vid {
				case id1:
					return stakeTypes.Validator{ValId: id1, Signer: addr1, VotingPower: 100}, nil
				case id2:
					return stakeTypes.Validator{ValId: id2, Signer: addr2, VotingPower: 100}, nil
				case id3:
					return stakeTypes.Validator{ValId: id3, Signer: addr3, VotingPower: 100}, nil
				default:
					return stakeTypes.Validator{}, fmt.Errorf("unknown validator id %d", vid)
				}
			}).
			AnyTimes()
	}

	tests := []struct {
		name            string
		sideVote        sidetxs.Vote
		msg             sdk.Msg
		setup           func()
		expectErr       bool
		errContains     string
		expectPDSet     bool
		expectPDRange   *types.BlockRange
		expectSpanDelta int
	}{
		{
			name:        "type mismatch",
			sideVote:    sidetxs.Vote_VOTE_YES,
			msg:         &types.MsgProposeSpan{},
			setup:       func() {},
			expectErr:   true,
			errContains: "MsgSetProducerDowntime type mismatch",
		},
		{
			name:        "side vote not YES",
			sideVote:    sidetxs.Vote_VOTE_NO,
			msg:         newMsg(addr1, 1000, 1100),
			setup:       func() {},
			expectErr:   true,
			errContains: "side-tx didn't get yes votes",
		},
		{
			name:     "GetValIdFromAddress error",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 1000, 1100),
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(uint64(0), fmt.Errorf("lookup failed")).
					Times(1)
			},
			expectErr:   true,
			errContains: "lookup failed",
		},
		{
			name:     "producer not registered",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 1000, 1100),
			setup: func() {
				// id1 resolves but is not in the producer set
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				setVotes(id2, id3)
			},
			expectErr:   true,
			errContains: "not a registered producer",
		},
		{
			name:     "only one registered producer",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 1000, 1100),
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				// Force producer set to exactly [id1]:
				// all voters rank only id1, so only id1 gets a score and passes the threshold.
				setVotesForAll([]uint64{id1})
			},
			expectErr:   true,
			errContains: "only one registered producer",
		},

		{
			name:     "reject when all other producers have overlapping PDs",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 1000, 1100),
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				// Producer set [id1,id2,id3] by ranking all three
				setVotesForAll([]uint64{id1, id2, id3})
				setPD(id2, 995, 1105)
				setPD(id3, 1000, 1100)
				require.NoError(s.borKeeper.SetParams(s.ctx, types.DefaultParams()))
				valSet, vals := s.genTestValidators()
				for _, sp := range []types.Span{
					{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
					{Id: 1, StartBlock: 200, EndBlock: 299, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
				} {
					require.NoError(s.borKeeper.AddNewSpan(s.ctx, &sp))
				}
			},
			expectErr:   true,
			errContains: "overlapping planned downtime with all other producers",
		},

		{
			name:     "success: no overlaps present -> PD persisted",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 1200, 1300),
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				// Producer set [id1,id2,id3]
				setVotesForAll([]uint64{id1, id2, id3})
				require.NoError(s.borKeeper.SetParams(s.ctx, types.DefaultParams()))
				valSet, vals := s.genTestValidators()
				if len(vals) > 0 {
					vals[0].ValId = id2 // avoid replacement generation
				}
				for _, sp := range []types.Span{
					{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
					{Id: 1, StartBlock: 200, EndBlock: 299, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
				} {
					require.NoError(s.borKeeper.AddNewSpan(s.ctx, &sp))
				}
			},
			expectErr:     false,
			expectPDSet:   true,
			expectPDRange: &types.BlockRange{StartBlock: 1200, EndBlock: 1300},
		},

		{
			name:     "success: overlaps exist but not with all others -> PD persisted",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 1400, 1500),
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				// Producer set [id1,id2,id3]
				setVotesForAll([]uint64{id1, id2, id3})
				setPD(id2, 1450, 1550) // overlaps
				setPD(id3, 2000, 2100) // no overlap
				require.NoError(s.borKeeper.SetParams(s.ctx, types.DefaultParams()))
				valSet, vals := s.genTestValidators()
				if len(vals) > 0 {
					vals[0].ValId = id2 // avoid replacement generation
				}
				for _, sp := range []types.Span{
					{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
					{Id: 1, StartBlock: 200, EndBlock: 299, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
				} {
					require.NoError(s.borKeeper.AddNewSpan(s.ctx, &sp))
				}
			},
			expectErr:     false,
			expectPDSet:   true,
			expectPDRange: &types.BlockRange{StartBlock: 1400, EndBlock: 1500},
		},

		{
			// Downtime range is far beyond all spans -> hasOverlappingSpan returns false -> no new span
			name:     "success: downtime far beyond all spans -> PD persisted, no replacement span",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 50000, 51000),
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				setVotesForAll([]uint64{id1, id2, id3})
			},
			expectErr:       false,
			expectPDSet:     true,
			expectPDRange:   &types.BlockRange{StartBlock: 50000, EndBlock: 51000},
			expectSpanDelta: 0, // no overlap -> no new span
		},

		{
			// Backward scan traverses multiple spans before finding overlap in an earlier one
			name:     "success: backward scan finds overlap in earlier span",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 120, 250), // overlaps span 0 [100-199] and span 1 [200-299]
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				setVotesForAll([]uint64{id1, id2, id3})

				require.NoError(s.borKeeper.SetParams(s.ctx, types.DefaultParams()))
				require.NoError(s.borKeeper.UpdateLatestActiveProducer(s.ctx, map[uint64]struct{}{id2: {}, id3: {}}))

				valSet, vals := s.genTestValidators()
				require.NotEmpty(vals)

				sp0Prods := make([]stakeTypes.Validator, len(vals))
				copy(sp0Prods, vals)
				sp0Prods[0].ValId = id1

				// 4 spans so backward scan must traverse 3->2->1->0
				spans := []types.Span{
					{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
					{Id: 1, StartBlock: 200, EndBlock: 299, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
					{Id: 2, StartBlock: 300, EndBlock: 399, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
					{Id: 3, StartBlock: 400, EndBlock: 499, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
				}
				for i := range spans {
					require.NoError(s.borKeeper.AddNewSpan(s.ctx, &spans[i]))
				}
			},
			expectErr:       false,
			expectPDSet:     true,
			expectPDRange:   &types.BlockRange{StartBlock: 120, EndBlock: 250},
			expectSpanDelta: 1,
		},

		{
			// Producer is selected in a span but has no downtime -> scan finds no overlap
			name:     "success: producer selected but no downtime overlap -> no replacement",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 5000, 6000),
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				setVotesForAll([]uint64{id1, id2, id3})

				require.NoError(s.borKeeper.SetParams(s.ctx, types.DefaultParams()))
				valSet, vals := s.genTestValidators()
				require.NotEmpty(vals)

				sp0Prods := make([]stakeTypes.Validator, len(vals))
				copy(sp0Prods, vals)
				sp0Prods[0].ValId = id1

				spans := []types.Span{
					{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
					{Id: 1, StartBlock: 200, EndBlock: 299, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
				}
				for i := range spans {
					require.NoError(s.borKeeper.AddNewSpan(s.ctx, &spans[i]))
				}
			},
			expectErr:       false,
			expectPDSet:     true,
			expectPDRange:   &types.BlockRange{StartBlock: 5000, EndBlock: 6000},
			expectSpanDelta: 0,
		},

		{
			// latestFailedProducer is non-empty, exercises the delete loop at line 631
			name:     "success: failed producer excluded from active set during replacement",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 150, 350),
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				setVotesForAll([]uint64{id1, id2, id3})

				require.NoError(s.borKeeper.SetParams(s.ctx, types.DefaultParams()))
				require.NoError(s.borKeeper.UpdateLatestActiveProducer(s.ctx, map[uint64]struct{}{id2: {}, id3: {}}))
				// Mark id3 as a failed producer — exercises the for loop at line 631
				require.NoError(s.borKeeper.AddLatestFailedProducer(s.ctx, id3))

				valSet, vals := s.genTestValidators()
				require.NotEmpty(vals)
				sp0Prods := make([]stakeTypes.Validator, len(vals))
				copy(sp0Prods, vals)
				sp0Prods[0].ValId = id1

				spans := []types.Span{
					{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
					{Id: 1, StartBlock: 200, EndBlock: 299, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
					{Id: 2, StartBlock: 300, EndBlock: 399, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
				}
				for i := range spans {
					require.NoError(s.borKeeper.AddNewSpan(s.ctx, &spans[i]))
				}
			},
			expectErr:       false,
			expectPDSet:     true,
			expectPDRange:   &types.BlockRange{StartBlock: 150, EndBlock: 350},
			expectSpanDelta: 1,
		},

		{
			// Span gap: lastSpan is id=3 but id=2 doesn't exist. Backward scan hits GetSpan error -> breaks.
			// Downtime [500, 600] doesn't overlap span 3 [300-399], so scan must try to go backward.
			name:     "success: backward scan stops at span gap",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 500, 600),
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				setVotesForAll([]uint64{id1, id2, id3})

				require.NoError(s.borKeeper.SetParams(s.ctx, types.DefaultParams()))

				valSet, vals := s.genTestValidators()
				require.NotEmpty(vals)
				sp0Prods := make([]stakeTypes.Validator, len(vals))
				copy(sp0Prods, vals)
				sp0Prods[0].ValId = id2 // not id1, so no overlap trigger

				// Add span 0 and span 3 (skip 1 and 2 to create a gap)
				spans := []types.Span{
					{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
					{Id: 3, StartBlock: 300, EndBlock: 399, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
				}
				for i := range spans {
					require.NoError(s.borKeeper.AddNewSpan(s.ctx, &spans[i]))
				}
			},
			expectErr:       false,
			expectPDSet:     true,
			expectPDRange:   &types.BlockRange{StartBlock: 500, EndBlock: 600},
			expectSpanDelta: 0, // scan stops at gap, no overlap found
		},

		{
			name:     "success: replacement spans generated when requester is selected and overlaps",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 150, 350), // overlaps spans [0],[1],[2]
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				// Producer set [id1,id2,id3]
				setVotesForAll([]uint64{id1, id2, id3})

				params := types.DefaultParams()
				require.NoError(s.borKeeper.SetParams(s.ctx, params))

				// Seed latest active producers (optional; safe even if AddNewVeBlopSpan can handle nil)
				require.NoError(s.borKeeper.UpdateLatestActiveProducer(s.ctx, map[uint64]struct{}{id2: {}, id3: {}}))

				valSet, vals := s.genTestValidators()
				require.NotEmpty(vals)

				// Build per-span SelectedProducers; selection no longer affects overlap trigger,
				// but keep two spans with id1 for clarity.
				sp0Prods := make([]stakeTypes.Validator, len(vals))
				copy(sp0Prods, vals)
				sp0Prods[0].ValId = id1

				sp1Prods := make([]stakeTypes.Validator, len(vals))
				copy(sp1Prods, vals)
				sp1Prods[0].ValId = id2 // different producer

				sp2Prods := make([]stakeTypes.Validator, len(vals))
				copy(sp2Prods, vals)
				sp2Prods[0].ValId = id1

				spans := []types.Span{
					{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
					{Id: 1, StartBlock: 200, EndBlock: 299, ValidatorSet: valSet, SelectedProducers: sp1Prods, BorChainId: "bor"},
					{Id: 2, StartBlock: 300, EndBlock: 399, ValidatorSet: valSet, SelectedProducers: sp2Prods, BorChainId: "bor"},
				}
				for i := range spans {
					require.NoError(s.borKeeper.AddNewSpan(s.ctx, &spans[i]))
				}
			},
			expectErr:     false,
			expectPDSet:   true,
			expectPDRange: &types.BlockRange{StartBlock: 150, EndBlock: 350},
			// New PostHandler adds exactly one veBlop span when any overlap exists.
			expectSpanDelta: 1,
		},
	}

	for _, tc := range tests {
		s.T().Run(tc.name, func(t *testing.T) {
			// Fresh state
			s.SetupTest()

			require.NoError(s.borKeeper.SetParams(s.ctx, types.DefaultParams()))
			primeStakeMocks()

			// Seed minimal spans, so GetLastSpan works, unless the test seeds its own
			if tc.expectSpanDelta == 0 && tc.errContains == "" {
				valSet, vals := s.genTestValidators()
				if len(vals) > 0 {
					vals[0].ValId = id2 // avoid replacements in generic tests
				}
				for _, sp := range []types.Span{
					{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
					{Id: 1, StartBlock: 200, EndBlock: 299, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
					{Id: 2, StartBlock: 300, EndBlock: 399, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
				} {
					require.NoError(s.borKeeper.AddNewSpan(s.ctx, &sp))
				}
			}

			if tc.setup != nil {
				tc.setup()
			}

			// Snapshot initial last span if present
			initialLastID := uint64(0)
			hadInitialSpan := false
			if last, err := s.borKeeper.GetLastSpan(s.ctx); err == nil {
				hadInitialSpan = true
				initialLastID = last.Id
			}

			handler := s.sideMsgServer.PostTxHandler(sdk.MsgTypeURL(&types.MsgSetProducerDowntime{}))
			err := handler(s.ctx, tc.msg, tc.sideVote)

			if tc.expectErr {
				require.Error(err)
				if tc.errContains != "" {
					require.Contains(err.Error(), tc.errContains)
				}
			} else {
				require.NoError(err)
			}

			if tc.expectPDSet {
				br := getPD(id1)
				require.NotNil(br)
				require.Equal(tc.expectPDRange.StartBlock, br.StartBlock)
				require.Equal(tc.expectPDRange.EndBlock, br.EndBlock)
			}

			if tc.expectSpanDelta > 0 && !tc.expectErr {
				require.True(hadInitialSpan, "expected an initial span to compare deltas")
				last, err := s.borKeeper.GetLastSpan(s.ctx)
				require.NoError(err)
				require.Equal(initialLastID+uint64(tc.expectSpanDelta), last.Id)
			}
		})
	}
}

func (s *KeeperTestSuite) TestPostHandleSetProducerDowntime_VeBlopSpanDuration() {
	helper.SetZurichHardforkHeight(100)
	defer helper.SetZurichHardforkHeight(0)

	fixHeight := helper.GetZurichHardforkHeight()

	tests := []struct {
		name             string
		blockHeight      int64
		expectedEndBlock func(start, spanDuration uint64) uint64
		expectedDuration func(spanDuration uint64) uint64
	}{
		{
			name:             "pre-fork: span is one block longer than SpanDuration",
			blockHeight:      fixHeight - 1,
			expectedEndBlock: func(start, dur uint64) uint64 { return start + dur },
			expectedDuration: func(dur uint64) uint64 { return dur + 1 },
		},
		{
			name:             "at fork height: span has exactly SpanDuration blocks",
			blockHeight:      fixHeight,
			expectedEndBlock: func(start, dur uint64) uint64 { return start + dur - 1 },
			expectedDuration: func(dur uint64) uint64 { return dur },
		},
		{
			name:             "post-fork: span has exactly SpanDuration blocks",
			blockHeight:      fixHeight + 1,
			expectedEndBlock: func(start, dur uint64) uint64 { return start + dur - 1 },
			expectedDuration: func(dur uint64) uint64 { return dur },
		},
	}

	for _, tc := range tests {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			require := s.Require()

			id1, id2, id3 := uint64(1), uint64(2), uint64(3)
			addr1 := common.HexToAddress("0x0000000000000000000000000000000000000001").Hex()
			addr2 := common.HexToAddress("0x0000000000000000000000000000000000000002").Hex()
			addr3 := common.HexToAddress("0x0000000000000000000000000000000000000003").Hex()

			s.stakeKeeper.EXPECT().
				GetValidatorSet(gomock.Any()).
				Return(stakeTypes.ValidatorSet{
					Validators: []*stakeTypes.Validator{
						{ValId: id1, Signer: addr1, VotingPower: 100},
						{ValId: id2, Signer: addr2, VotingPower: 100},
						{ValId: id3, Signer: addr3, VotingPower: 100},
					},
				}, nil).
				AnyTimes()

			s.stakeKeeper.EXPECT().
				GetSpanEligibleValidators(gomock.Any()).
				Return([]stakeTypes.Validator{
					{ValId: id1, Signer: addr1, VotingPower: 100},
					{ValId: id2, Signer: addr2, VotingPower: 100},
					{ValId: id3, Signer: addr3, VotingPower: 100},
				}).
				AnyTimes()

			s.stakeKeeper.EXPECT().
				GetValidatorFromValID(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ sdk.Context, vid uint64) (stakeTypes.Validator, error) {
					switch vid {
					case id1:
						return stakeTypes.Validator{ValId: id1, Signer: addr1, VotingPower: 100}, nil
					case id2:
						return stakeTypes.Validator{ValId: id2, Signer: addr2, VotingPower: 100}, nil
					case id3:
						return stakeTypes.Validator{ValId: id3, Signer: addr3, VotingPower: 100}, nil
					default:
						return stakeTypes.Validator{}, fmt.Errorf("unknown validator id %d", vid)
					}
				}).
				AnyTimes()

			s.stakeKeeper.EXPECT().
				GetValIdFromAddress(gomock.Any(), addr1).
				Return(id1, nil).
				AnyTimes()

			require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
			for _, voter := range []uint64{id1, id2, id3} {
				require.NoError(s.borKeeper.SetProducerVotes(s.ctx, voter, types.ProducerVotes{Votes: []uint64{id1, id2, id3}}))
			}

			params := types.DefaultParams()
			require.NoError(s.borKeeper.SetParams(s.ctx, params))
			require.NoError(s.borKeeper.UpdateLatestActiveProducer(s.ctx, map[uint64]struct{}{id2: {}, id3: {}}))

			valSet, vals := s.genTestValidators()
			require.NotEmpty(vals)

			sp0Prods := make([]stakeTypes.Validator, len(vals))
			copy(sp0Prods, vals)
			sp0Prods[0].ValId = id1

			for i, sp := range []types.Span{
				{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
				{Id: 1, StartBlock: 200, EndBlock: 299, ValidatorSet: valSet, SelectedProducers: sp0Prods, BorChainId: "bor"},
			} {
				require.NoError(s.borKeeper.AddNewSpan(s.ctx, &[]types.Span{sp}[0]), "failed to add span %d", i)
			}

			downtimeStart := uint64(150)
			msg := &types.MsgSetProducerDowntime{
				Producer:      addr1,
				DowntimeRange: types.BlockRange{StartBlock: downtimeStart, EndBlock: 350},
			}

			ctx := s.ctx.WithBlockHeight(tc.blockHeight)
			handler := s.sideMsgServer.PostTxHandler(sdk.MsgTypeURL(&types.MsgSetProducerDowntime{}))
			err := handler(ctx, msg, sidetxs.Vote_VOTE_YES)
			require.NoError(err)

			lastSpan, err := s.borKeeper.GetLastSpan(s.ctx)
			require.NoError(err)
			require.Equal(uint64(2), lastSpan.Id)
			require.Equal(downtimeStart, lastSpan.StartBlock)
			require.Equal(tc.expectedEndBlock(downtimeStart, params.SpanDuration), lastSpan.EndBlock)
			require.Equal(tc.expectedDuration(params.SpanDuration), lastSpan.EndBlock-lastSpan.StartBlock+1)
		})
	}
}
