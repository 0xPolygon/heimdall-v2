package keeper_test

import (
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

const (
	MaticTokenAddress     = "0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0"
	StakingManagerAddress = "0x5e3ef299fddf15eaa0432e6e66473ace8c13d908"
	SlashManagerAddress   = "0x01f645dcd6c796f6bc6c982159b32faaaebdc96a"
	RootChainAddress      = "0x86e4dc95c7fbdbf52e33d563bbdb00823894c287"
	StakingInfoAddress    = "0xa59c847bd5ac0172ff4fe912c5d29e5a71a7512b"
	StateSenderAddress    = "0x28e4f3a7f651294b9564800b2d01f35189a5bfbe"
)

func (s *KeeperTestSuite) TestMsgUpdateParams() {
	ctx, require, cmKeeper, queryClient, msgServer, params := s.ctx, s.Require(), s.cmKeeper, s.queryClient, s.msgServer, s.getParams()

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
			name: "invalid params",
			input: &types.MsgUpdateParams{
				Authority: cmKeeper.GetAuthority(),
				Params: types.Params{
					ChainParams: types.ChainParams{
						MaticTokenAddress: "def",
					},
				},
			},
			expErr:    true,
			expErrMsg: "invalid address for value def for matic_token_address in chain_params",
		},
		{
			name: "all good",
			input: &types.MsgUpdateParams{
				Authority: cmKeeper.GetAuthority(),
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
				require.Equal(authtypes.NewModuleAddress(govtypes.ModuleName).String(), cmKeeper.GetAuthority())
				require.NoError(err)

				res, err := queryClient.GetParams(ctx, &types.QueryParamsRequest{})
				require.NoError(err)
				require.Equal(params, res.Params)
			}
		})
	}
}

func (s *KeeperTestSuite) getParams() types.Params {
	s.T().Helper()

	params := types.DefaultParams()
	params.ChainParams.MaticTokenAddress = MaticTokenAddress
	params.ChainParams.StakingManagerAddress = StakingManagerAddress
	params.ChainParams.SlashManagerAddress = SlashManagerAddress
	params.ChainParams.RootChainAddress = RootChainAddress
	params.ChainParams.StakingInfoAddress = StakingInfoAddress
	params.ChainParams.StateSenderAddress = StateSenderAddress

	return params
}
