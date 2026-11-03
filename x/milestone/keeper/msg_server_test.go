package keeper_test

import (
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	borgrpc "github.com/0xPolygon/heimdall-v2/x/bor/grpc"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
)

func (s *KeeperTestSuite) TestMsgUpdateParams() {
	ctx, require, keeper, queryClient, msgServer, params := s.ctx, s.Require(), s.milestoneKeeper, s.queryClient, s.msgServer, types.DefaultParams()

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
			name: "invalid max milestone proposition length",
			input: &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params: types.Params{
					MaxMilestonePropositionLength: 0,
				},
			},
			expErr:    true,
			expErrMsg: "max milestone proposition length should not be zero",
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
		s.Run(tc.name, func() {
			_, err := msgServer.UpdateParams(ctx, tc.input)

			if tc.expErr {
				require.Error(err)
				require.Contains(err.Error(), tc.expErrMsg)
			} else {
				require.Equal(authtypes.NewModuleAddress(govtypes.ModuleName).String(), keeper.GetAuthority())
				require.NoError(err)

				res, err := queryClient.GetMilestoneParams(ctx, &types.QueryParamsRequest{})
				require.NoError(err)
				require.Equal(params, res.Params)
			}
		})
	}
}

func (s *KeeperTestSuite) TestMsgUpdateParamsPropositionLengthCap() {
	require, keeper, msgServer := s.Require(), s.milestoneKeeper, s.msgServer

	orig := helper.GetKarwarHeight()
	s.T().Cleanup(func() { helper.SetKarwarHeight(orig) })
	helper.SetKarwarHeight(100)

	testCases := []struct {
		name   string
		height int64
		length uint64
		expErr bool
	}{
		{name: "above cap before Karwar", height: 99, length: types.MilestonePropositionLengthCap + 1},
		{name: "at cap at Karwar", height: 100, length: types.MilestonePropositionLengthCap},
		{name: "above cap at Karwar", height: 100, length: types.MilestonePropositionLengthCap + 1, expErr: true},
		{name: "bor batch cap after Karwar", height: 101, length: uint64(borgrpc.MaxBlockInfoBatchSize), expErr: true},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			before, err := keeper.GetParams(s.ctx)
			require.NoError(err)

			params := types.DefaultParams()
			params.MaxMilestonePropositionLength = tc.length
			_, err = msgServer.UpdateParams(s.ctx.WithBlockHeight(tc.height), &types.MsgUpdateParams{
				Authority: keeper.GetAuthority(),
				Params:    params,
			})

			got, getErr := keeper.GetParams(s.ctx)
			require.NoError(getErr)
			if tc.expErr {
				require.ErrorContains(err, "exceeds cap")
				require.Equal(before, got)
				return
			}
			require.NoError(err)
			require.Equal(tc.length, got.MaxMilestonePropositionLength)
		})
	}
}
