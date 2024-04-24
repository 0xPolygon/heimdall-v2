package keeper_test

import (
	"errors"
	"testing"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/ethereum/go-ethereum/common"
)

func (suite *KeeperTestSuite) TestProposeSpan() {
	require := suite.Require()

	testChainParams := chainmanagertypes.DefaultParams()
	suite.chainManagerKeeper.EXPECT().GetParams(suite.ctx).Return(testChainParams, nil).AnyTimes()

	testSpan := suite.genTestSpans(1)[0]
	err := suite.borKeeper.AddNewSpan(suite.ctx, testSpan)
	require.NoError(err)

	testcases := []struct {
		name   string
		span   types.MsgProposeSpanRequest
		expRes *types.MsgProposeSpanResponse
		expErr error
	}{
		{
			name: "correct span gets proposed",
			span: types.MsgProposeSpanRequest{
				SpanId:     2,
				Proposer:   common.HexToAddress("someproposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testseed1").Bytes(),
			},
			expRes: &types.MsgProposeSpanResponse{},
			expErr: nil,
		},
		{
			name: "incorrect validator address",
			span: types.MsgProposeSpanRequest{
				SpanId:     2,
				Proposer:   "0x91b54cD48FD796A5d0A120A4C5298a7fAEA59B",
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testseed1").Bytes(),
			},
			expRes: nil,
			expErr: errors.New("decoding address from hex string failed: not valid address"),
		},
		{
			name: "incorrect chain id",
			span: types.MsgProposeSpanRequest{
				SpanId:     2,
				Proposer:   common.HexToAddress("someproposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    "invalidchainid",
				Seed:       common.HexToHash("testseed1").Bytes(),
			},
			expRes: nil,
			expErr: types.ErrInvalidChainID,
		},
		{
			name: "span id not in continuity",
			span: types.MsgProposeSpanRequest{
				SpanId:     3,
				Proposer:   common.HexToAddress("someproposer").String(),
				StartBlock: 102,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testseed1").Bytes(),
			},
			expRes: nil,
			expErr: types.ErrInvalidSpan,
		},
		{
			name: "start block not in continuity",
			span: types.MsgProposeSpanRequest{
				SpanId:     2,
				Proposer:   common.HexToAddress("someproposer").String(),
				StartBlock: 105,
				EndBlock:   202,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testseed1").Bytes(),
			},
			expRes: nil,
			expErr: types.ErrInvalidSpan,
		},
		{
			name: "end block less than start block",
			span: types.MsgProposeSpanRequest{
				SpanId:     2,
				Proposer:   common.HexToAddress("someproposer").String(),
				StartBlock: 102,
				EndBlock:   100,
				ChainId:    testChainParams.ChainParams.BorChainId,
				Seed:       common.HexToHash("testseed1").Bytes(),
			},
			expRes: nil,
			expErr: types.ErrInvalidSpan,
		},
	}

	for _, tc := range testcases {
		suite.T().Run(tc.name, func(t *testing.T) {
			res, err := suite.msgServer.ProposeSpan(suite.ctx, &tc.span)
			require.Equal(tc.expRes, res)
			require.Equal(tc.expErr, err)
		})
	}
}
