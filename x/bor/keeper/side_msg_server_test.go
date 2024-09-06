package keeper_test

import (
	"math/big"
	"testing"

	hModule "github.com/0xPolygon/heimdall-v2/module"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
)

func (suite *KeeperTestSuite) TestSideHandleMsgSpan() {
	ctx := suite.ctx
	require := suite.Require()

	testChainParams := chainmanagertypes.DefaultParams()

	spans := suite.genTestSpans(1)
	err := suite.borKeeper.AddNewSpan(suite.ctx, spans[0])
	require.NoError(err)

	lastEthBlock := big.NewInt(100)
	err = suite.borKeeper.SetLastEthBlock(ctx, lastEthBlock)
	require.NoError(err)

	nextEthBlock := lastEthBlock.Add(lastEthBlock, big.NewInt(1))
	nextEthBlockHeader := &ethTypes.Header{Number: nextEthBlock}

	testcases := []struct {
		name    string
		msg     sdk.Msg
		expVote hModule.Vote
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
				Seed:       common.HexToHash("invalidseed").Bytes(),
			},
			expVote: hModule.Vote_VOTE_NO,
		},
		{
			name: "span is not in turn (current child block is less than last span start block)",
			msg: &types.MsgProposeSpanRequest{
				SpanId:     2,
				Proposer:   common.HexToAddress("someproposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       nextEthBlockHeader.Hash().Bytes(),
			},
			expVote: hModule.Vote_VOTE_NO,
			mockFn: func() {
				suite.contractCaller.On("GetBorChainBlock", (*big.Int)(nil)).Return(&ethTypes.Header{Number: big.NewInt(0)}, nil).Times(1)
			},
		},
		{
			name: "span is not in turn (current child block is greater than last span end block)",
			msg: &types.MsgProposeSpanRequest{
				SpanId:     2,
				Proposer:   common.HexToAddress("someproposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       nextEthBlockHeader.Hash().Bytes(),
			},
			expVote: hModule.Vote_VOTE_NO,
			mockFn: func() {
				suite.contractCaller.On("GetBorChainBlock", (*big.Int)(nil)).Return(&ethTypes.Header{Number: big.NewInt(103)}, nil).Times(1)
			},
		},
		{
			name: "correct span is proposed",
			msg: &types.MsgProposeSpanRequest{
				SpanId:     2,
				Proposer:   common.HexToAddress("someproposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       nextEthBlockHeader.Hash().Bytes(),
			},
			expVote: hModule.Vote_VOTE_YES,
			mockFn: func() {
				suite.contractCaller.On("GetBorChainBlock", (*big.Int)(nil)).Return(&ethTypes.Header{Number: big.NewInt(50)}, nil).Times(1)
			},
		},
	}

	suite.contractCaller.On("GetMainChainBlock", nextEthBlock).Return(nextEthBlockHeader, nil).Times(len(testcases))
	for _, tc := range testcases {
		suite.T().Run(tc.name, func(t *testing.T) {

			if tc.mockFn != nil {
				tc.mockFn()
			}
			sideHandler := suite.sideMsgServer.SideTxHandler(sdk.MsgTypeURL(&types.MsgProposeSpanRequest{}))
			res := sideHandler(suite.ctx, tc.msg)
			require.Equal(tc.expVote, res)
		})
	}
}

func (suite *KeeperTestSuite) TestPostHandleMsgEventSpan() {
	require := suite.Require()

	suite.stakeKeeper.EXPECT().GetSpanEligibleValidators(suite.ctx).Times(1)
	suite.stakeKeeper.EXPECT().GetValidatorSet(suite.ctx).Times(1)
	suite.stakeKeeper.EXPECT().GetValidatorFromValID(suite.ctx, gomock.Any()).AnyTimes()

	borParams := types.DefaultParams()
	err := suite.borKeeper.SetParams(suite.ctx, borParams)
	require.NoError(err)

	testChainParams := chainmanagertypes.DefaultParams()
	spans := suite.genTestSpans(1)
	err = suite.borKeeper.AddNewSpan(suite.ctx, spans[0])
	require.NoError(err)

	testcases := []struct {
		name          string
		msg           sdk.Msg
		vote          hModule.Vote
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
				Seed:       common.HexToHash("testseed1").Bytes(),
			},
			vote:          hModule.Vote_VOTE_NO,
			expLastSpanId: spans[0].Id,
		},
		{
			name: "span replayed",
			msg: &types.MsgProposeSpanRequest{
				SpanId:     1,
				Proposer:   common.HexToAddress("someproposer").String(),
				StartBlock: 1,
				EndBlock:   101,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testseed1").Bytes(),
			},
			vote:          hModule.Vote_VOTE_YES,
			expLastSpanId: spans[0].Id,
		},
		{
			name: "correct span is proposed",
			msg: &types.MsgProposeSpanRequest{
				SpanId:     2,
				Proposer:   common.HexToAddress("someproposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testseed1").Bytes(),
			},
			vote:          hModule.Vote_VOTE_YES,
			expLastSpanId: 2,
		},
	}

	for _, tc := range testcases {
		suite.T().Run(tc.name, func(t *testing.T) {
			postHandler := suite.sideMsgServer.PostTxHandler(sdk.MsgTypeURL(&types.MsgProposeSpanRequest{}))
			postHandler(suite.ctx, tc.msg, tc.vote)

			lastSpan, err := suite.borKeeper.GetLastSpan(suite.ctx)
			require.NoError(err)
			require.Equal(tc.expLastSpanId, lastSpan.Id)
		})
	}
}
