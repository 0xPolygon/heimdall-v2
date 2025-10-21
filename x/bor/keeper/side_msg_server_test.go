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
			name:       "start too soon - boundary (start+min == current) returns NO",
			current:    minFuture + 50, // ensure no underflow in start calculation
			msg:        newMsg((minFuture+50)-minFuture, (minFuture+50)-minFuture+10),
			expectVote: sidetxs.Vote_VOTE_NO,
		},
		{
			name:       "start too soon - strict (start+min < current) returns NO",
			current:    minFuture + 50,
			msg:        newMsg((minFuture+50)-minFuture-1, (minFuture+50)-minFuture+10),
			expectVote: sidetxs.Vote_VOTE_NO,
		},
		{
			name:       "end too far - boundary (current+max == end) returns NO",
			current:    2_000_000,
			msg:        newMsg(2_000_000+1, 2_000_000+maxFuture),
			expectVote: sidetxs.Vote_VOTE_NO,
		},
		{
			name:       "end too far - strict (current+max < end) returns NO",
			current:    2_000_000,
			msg:        newMsg(2_000_000+1, 2_000_000+maxFuture+1),
			expectVote: sidetxs.Vote_VOTE_NO,
		},
		{
			name:       "passes both checks - boundary just passing returns YES",
			current:    3_000_000,
			msg:        newMsg((3_000_000-minFuture)+1, (3_000_000+maxFuture)-1),
			expectVote: sidetxs.Vote_VOTE_YES,
		},
		{
			name:       "passes both checks - start well in future, end well within max returns YES",
			current:    4_000_000,
			msg:        newMsg(4_000_000+100, 4_000_000+maxFuture-100),
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

	// Helpers
	newMsg := func(prod string, start, end uint64) *types.MsgSetProducerDowntime {
		return &types.MsgSetProducerDowntime{
			Producer:      prod,
			DowntimeRange: types.BlockRange{StartBlock: start, EndBlock: end},
		}
	}

	setVotes := func(ids ...uint64) {
		require.NoError(s.borKeeper.ClearProducerVotes(s.ctx))
		for _, id := range ids {
			require.NoError(s.borKeeper.SetProducerVotes(s.ctx, id, types.ProducerVotes{}))
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
	id1, id2, id3 := uint64(1), uint64(2), uint64(3)

	// Add baseline params and a few spans so GetLastSpan works
	require.NoError(s.borKeeper.SetParams(s.ctx, types.DefaultParams()))
	// Create three spans, with first span's SelectedProducers[0] != id1 so we can avoid replacement-gen
	valSet, vals := s.genTestValidators()
	spans := []types.Span{
		{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
		{Id: 1, StartBlock: 200, EndBlock: 299, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
		{Id: 2, StartBlock: 300, EndBlock: 399, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
	}
	for i := range spans {
		require.NoError(s.borKeeper.AddNewSpan(s.ctx, &spans[i]))
	}

	tests := []struct {
		name          string
		sideVote      sidetxs.Vote
		msg           sdk.Msg
		setup         func()
		expectErr     bool
		errContains   string
		expectPDSet   bool // whether new PD for id1 should be stored
		expectPDRange *types.BlockRange
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
			name:     "side vote not YES",
			sideVote: sidetxs.Vote_VOTE_NO,
			msg:      newMsg(addr1, 1000, 1100),
			setup: func() {
				// still expect address lookup to be unused when vote != YES
			},
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
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				setVotes(id2, id3) // id1 missing
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
				setVotes(id1) // only one
			},
			expectErr:   true,
			errContains: "only one registered producer",
		},
		{
			name:     "overlaps with all other producers -> error",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 1000, 1100),
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				setVotes(id1, id2, id3)
				// Both other producers have overlapping planned downtimes
				setPD(id2, 1000, 1100)
				setPD(id3, 995, 1105)
			},
			expectErr:   true,
			errContains: "overlapping planned downtime with all other producers",
		},
		{
			name:     "success: no overlaps present",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 1200, 1300),
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				setVotes(id1, id2, id3)
				// No PDs for others -> no way to overlap with all
			},
			expectErr:     false,
			expectPDSet:   true,
			expectPDRange: &types.BlockRange{StartBlock: 1200, EndBlock: 1300},
		},
		{
			name:     "success: overlaps exist but not with all others (one other has non-overlapping PD)",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 1400, 1500),
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				setVotes(id1, id2, id3)
				// id2 overlaps, id3 does not -> should pass
				setPD(id2, 1450, 1550) // overlaps requested [1400,1500]
				setPD(id3, 2000, 2100) // no overlap
				// Also ensure spans overlap with the downtime window but producer for spans is not id1,
				// so replacement span generation path is not exercised here.
				// Our pre-seeded spans have SelectedProducers[0] from genTestValidators, not tied to id1.
			},
			expectErr:     false,
			expectPDSet:   true,
			expectPDRange: &types.BlockRange{StartBlock: 1400, EndBlock: 1500},
		},
		{
			name:     "success: downtime does not overlap any span (no replacement span generated)",
			sideVote: sidetxs.Vote_VOTE_YES,
			msg:      newMsg(addr1, 10, 20), // well before any span's StartBlock
			setup: func() {
				s.stakeKeeper.EXPECT().
					GetValIdFromAddress(gomock.Any(), addr1).
					Return(id1, nil).
					Times(1)
				setVotes(id1, id2)
				// No other PDs set; second producer ensures >1 registered
			},
			expectErr:     false,
			expectPDSet:   true,
			expectPDRange: &types.BlockRange{StartBlock: 10, EndBlock: 20},
		},
	}

	for _, tc := range tests {
		s.T().Run(tc.name, func(t *testing.T) {
			// Fresh state per subtest
			s.SetupTest()
			require.NoError(s.borKeeper.SetParams(s.ctx, types.DefaultParams()))

			// Seed a few spans again so GetLastSpan works
			valSet, vals := s.genTestValidators()
			for i, span := range []types.Span{
				{Id: 0, StartBlock: 100, EndBlock: 199, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
				{Id: 1, StartBlock: 200, EndBlock: 299, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
				{Id: 2, StartBlock: 300, EndBlock: 399, ValidatorSet: valSet, SelectedProducers: vals, BorChainId: "bor"},
			} {
				require.NoError(s.borKeeper.AddNewSpan(s.ctx, &span), "seed span %d", i)
			}

			if tc.setup != nil {
				tc.setup()
			}

			postHandler := s.sideMsgServer.PostTxHandler(sdk.MsgTypeURL(&types.MsgSetProducerDowntime{}))
			err := postHandler(s.ctx, tc.msg, tc.sideVote)

			if tc.expectErr {
				require.Error(err)
				if tc.errContains != "" {
					require.Contains(err.Error(), tc.errContains)
				}
			} else {
				require.NoError(err)
			}

			// Validate PD persistence if expected
			if tc.expectPDSet {
				br := getPD(id1)
				require.NotNil(br)
				require.Equal(tc.expectPDRange.StartBlock, br.StartBlock)
				require.Equal(tc.expectPDRange.EndBlock, br.EndBlock)
			}
		})
	}
}
