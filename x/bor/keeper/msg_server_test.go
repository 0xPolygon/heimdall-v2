package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
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

func (s *KeeperTestSuite) TestBackfillSpans() {
	require, ctx, borKeeper, cmKeeper, msgServer := s.Require(), s.ctx, s.borKeeper, s.chainManagerKeeper, s.msgServer

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
				LatestBorBlock:  10000,
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
				LatestBorBlock:  10000,
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
				LatestBorBlock:  10000,
				LatestBorSpanId: 7,
			},
			expRes: nil,
			expErr: "span not found for id: 2",
		},
		{
			name: "invalid last bor span id",
			backfillSpans: types.MsgBackfillSpans{
				Proposer:        common.HexToAddress("someProposer").String(),
				ChainId:         testChainParams.ChainParams.BorChainId,
				LatestSpanId:    1,
				LatestBorBlock:  10000,
				LatestBorSpanId: 0,
			},
			expErr: "invalid last bor span id",
		},
		{
			name: "invalid last bor block",
			backfillSpans: types.MsgBackfillSpans{
				Proposer:        common.HexToAddress("someProposer").String(),
				ChainId:         testChainParams.ChainParams.BorChainId,
				LatestSpanId:    1,
				LatestBorBlock:  1,
				LatestBorSpanId: 2,
			},
			expRes: nil,
			expErr: "invalid last bor block",
		},
		{
			name: "mismatch between calculated and provided last span id",
			backfillSpans: types.MsgBackfillSpans{

				Proposer:        common.HexToAddress("someProposer").String(),
				ChainId:         testChainParams.ChainParams.BorChainId,
				LatestSpanId:    1,
				LatestBorBlock:  1000,
				LatestBorSpanId: 3,
			},
			expRes: nil,
			expErr: "invalid span",
		},
	}

	cmKeeper.EXPECT().GetParams(ctx).Return(testChainParams, nil).AnyTimes()

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
