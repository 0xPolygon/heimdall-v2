package keeper_test

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

func (s *KeeperTestSuite) TestSideHandleMsgSpan() {
	ctx, require, borKeeper, sideMsgServer := s.ctx, s.Require(), s.borKeeper, s.sideMsgServer
	testChainParams, contractCaller := chainmanagertypes.DefaultParams(), &s.contractCaller

	spans := s.genTestSpans(1)
	err := borKeeper.AddNewSpan(ctx, spans[0])
	require.NoError(err)

	lastEthBlock := big.NewInt(100)
	err = borKeeper.SetLastEthBlock(ctx, lastEthBlock)
	require.NoError(err)

	nextEthBlock := lastEthBlock.Add(lastEthBlock, big.NewInt(1))
	nextEthBlockHeader := &ethTypes.Header{Number: nextEthBlock}

	testcases := []struct {
		name    string
		msg     sdk.Msg
		expVote sidetxs.Vote
		mockFn  func()
	}{
		{
			name: "seed mismatch",
			msg: &types.MsgProposeSpanRequest{
				SpanId:     2,
				Proposer:   "0x91b54cD48FD796A5d0A120A4C5298a7fAEA59B",
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("invalidSeed").Bytes(),
			},
			expVote: sidetxs.Vote_VOTE_NO,
		},
		{
			name: "span is not in turn (current child block is less than last span start block)",
			msg: &types.MsgProposeSpanRequest{
				SpanId:     2,
				Proposer:   common.HexToAddress("someProposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       nextEthBlockHeader.Hash().Bytes(),
			},
			expVote: sidetxs.Vote_VOTE_NO,
			mockFn: func() {
				contractCaller.On("GetPolygonPosChainBlock", (*big.Int)(nil)).Return(&ethTypes.Header{Number: big.NewInt(0)}, nil).Times(1)
			},
		},
		{
			name: "span is not in turn (current child block is greater than last span end block)",
			msg: &types.MsgProposeSpanRequest{
				SpanId:     2,
				Proposer:   common.HexToAddress("someProposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       nextEthBlockHeader.Hash().Bytes(),
			},
			expVote: sidetxs.Vote_VOTE_NO,
			mockFn: func() {
				contractCaller.On("GetPolygonPosChainBlock", (*big.Int)(nil)).Return(&ethTypes.Header{Number: big.NewInt(103)}, nil).Times(1)
			},
		},
		{
			name: "correct span is proposed",
			msg: &types.MsgProposeSpanRequest{
				SpanId:     2,
				Proposer:   common.HexToAddress("someProposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       nextEthBlockHeader.Hash().Bytes(),
			},
			expVote: sidetxs.Vote_VOTE_YES,
			mockFn: func() {
				contractCaller.On("GetPolygonPosChainBlock", (*big.Int)(nil)).Return(&ethTypes.Header{Number: big.NewInt(50)}, nil).Times(1)
			},
		},
	}

	contractCaller.On("GetMainChainBlock", nextEthBlock).Return(nextEthBlockHeader, nil).Times(len(testcases))
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {

			if tc.mockFn != nil {
				tc.mockFn()
			}
			sideHandler := sideMsgServer.SideTxHandler(sdk.MsgTypeURL(&types.MsgProposeSpanRequest{}))
			res := sideHandler(s.ctx, tc.msg)
			require.Equal(tc.expVote, res)
		})
	}
}

func (s *KeeperTestSuite) TestPostHandleMsgEventSpan() {
	require, ctx, stakeKeeper, borKeeper, sideMsgServer := s.Require(), s.ctx, s.stakeKeeper, s.borKeeper, s.sideMsgServer

	stakeKeeper.EXPECT().GetSpanEligibleValidators(ctx).Times(1)
	stakeKeeper.EXPECT().GetValidatorSet(ctx).Times(1)
	stakeKeeper.EXPECT().GetValidatorFromValID(ctx, gomock.Any()).AnyTimes()

	borParams := types.DefaultParams()
	err := borKeeper.SetParams(ctx, borParams)
	require.NoError(err)

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
			msg: &types.MsgProposeSpanRequest{
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
			msg: &types.MsgProposeSpanRequest{
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
			msg: &types.MsgProposeSpanRequest{
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
			postHandler := sideMsgServer.PostTxHandler(sdk.MsgTypeURL(&types.MsgProposeSpanRequest{}))
			postHandler(ctx, tc.msg, tc.vote)

			lastSpan, err := borKeeper.GetLastSpan(ctx)
			require.NoError(err)
			require.Equal(tc.expLastSpanId, lastSpan.Id)
		})
	}
}
