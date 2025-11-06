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

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func (s *KeeperTestSuite) TestSideHandleMsgSpan() {
	ctx, require, borKeeper, milestoneKeeper, sideMsgServer := s.ctx, s.Require(), s.borKeeper, s.milestoneKeeper, s.sideMsgServer
	testChainParams, contractCaller := chainmanagertypes.DefaultParams(), &s.contractCaller

	borParams := types.DefaultParams()
	err := borKeeper.SetParams(ctx, borParams)
	require.NoError(err)

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
				contractCaller.On("GetBorChainBlockAuthor", mock.Anything).Return(&val1Addr, nil)
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
				contractCaller.On("GetBorChainBlockAuthor", mock.Anything).Return(&val1Addr, nil)
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
				contractCaller.On("GetBorChainBlockAuthor", mock.Anything).Return(&val1Addr, nil)
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
	contractCaller.On("GetBorChainBlockAuthor", big.NewInt(0)).Return(&producer1, nil).Times(1)
	contractCaller.On("GetBorChainBlockAuthor", big.NewInt(100)).Return(&producer2, nil).Times(1)

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
	maxFuture := uint64(types.PlannedDowntimeMaximumTimeInFuture)

	newMsg := func(start, end uint64) *types.MsgSetProducerDowntime {
		return &types.MsgSetProducerDowntime{
			Producer:      common.HexToAddress("0x0000000000000000000000000000000000000001").Hex(),
			DowntimeRange: types.BlockRange{StartBlock: start, EndBlock: end},
		}
	}

	type testCase struct {
		name         string
		typeMismatch bool
		current      uint64
		msg          *types.MsgSetProducerDowntime
		getBlockErr  error
		expectVote   sidetxs.Vote
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
			msg:         newMsg(1_000_100, 1_000_200),
			getBlockErr: fmt.Errorf("rpc error"),
			expectVote:  sidetxs.Vote_VOTE_NO,
		},
		{
			name:       "start too soon - boundary (start == current+min-1) returns NO",
			current:    5_000_000,
			msg:        newMsg((5_000_000+minFuture)-1, (5_000_000+minFuture)+10),
			expectVote: sidetxs.Vote_VOTE_NO,
		},
		{
			name:       "start too soon - strict (start < current+min-1) returns NO",
			current:    5_000_000,
			msg:        newMsg((5_000_000+minFuture)-2, (5_000_000+minFuture)+10),
			expectVote: sidetxs.Vote_VOTE_NO,
		},
		{
			name:       "end too far - boundary (end == current+max) returns NO",
			current:    2_000_000,
			msg:        newMsg(2_000_000+minFuture, 2_000_000+maxFuture),
			expectVote: sidetxs.Vote_VOTE_NO,
		},
		{
			name:       "end too far - strict (end > current+max) returns NO",
			current:    2_000_000,
			msg:        newMsg(2_000_000+minFuture, 2_000_000+maxFuture+1),
			expectVote: sidetxs.Vote_VOTE_NO,
		},
		{
			name:    "passes both checks - boundary just passing returns YES",
			current: 3_000_000,
			// start == current+min, end == current+max-1
			msg:        newMsg(3_000_000+minFuture, (3_000_000+maxFuture)-1),
			expectVote: sidetxs.Vote_VOTE_YES,
		},
		{
			name:    "passes both checks - start well in future, end well within max returns YES",
			current: 4_000_000,
			// start >= current+min, end < current+max
			msg:        newMsg(4_000_000+minFuture+100, 4_000_000+maxFuture-100),
			expectVote: sidetxs.Vote_VOTE_YES,
		},
	}

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
	// This drives CalculateProducerSet to return the given candidates (subject to threshold).
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
				// id1 resolves but is not in producer set
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
				// all voters rank only id1, so only id1 gets a score and passes threshold.
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

				// Seed latest active producers (optional; safe even if AddNewVeblopSpan can handle nil)
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
			// New PostHandler adds exactly one veblop span when any overlap exists.
			expectSpanDelta: 1,
		},
	}

	for _, tc := range tests {
		s.T().Run(tc.name, func(t *testing.T) {
			// Fresh state
			s.SetupTest()

			require.NoError(s.borKeeper.SetParams(s.ctx, types.DefaultParams()))
			primeStakeMocks()

			// Seed minimal spans so GetLastSpan works, unless test seeds its own
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
