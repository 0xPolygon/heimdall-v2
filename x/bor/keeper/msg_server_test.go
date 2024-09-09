package keeper_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

func (suite *KeeperTestSuite) TestProposeSpan() {
	require := suite.Require()

	testChainParams := chainmanagertypes.DefaultParams()
	testSpan := suite.genTestSpans(1)[0]
	err := suite.borKeeper.AddNewSpan(suite.ctx, testSpan)
	require.NoError(err)

	testcases := []struct {
		name   string
		span   types.MsgProposeSpanRequest
		expRes *types.MsgProposeSpanResponse
		expErr string
	}{
		{
			name: "correct span gets proposed",
			span: types.MsgProposeSpanRequest{
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
			span: types.MsgProposeSpanRequest{
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
			span: types.MsgProposeSpanRequest{
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
			span: types.MsgProposeSpanRequest{
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
			span: types.MsgProposeSpanRequest{
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
			span: types.MsgProposeSpanRequest{
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
			span: types.MsgProposeSpanRequest{
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

	suite.chainManagerKeeper.EXPECT().GetParams(suite.ctx).Return(testChainParams, nil).AnyTimes()

	for _, tc := range testcases {
		suite.T().Run(tc.name, func(t *testing.T) {
			res, err := suite.msgServer.ProposeSpan(suite.ctx, &tc.span)
			require.Equal(tc.expRes, res)
			if tc.expErr == "" {
				require.NoError(err)
			} else {
				require.ErrorContains(err, tc.expErr)
			}
		})
	}
}
