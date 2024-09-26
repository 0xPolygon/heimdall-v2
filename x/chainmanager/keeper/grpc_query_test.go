package keeper_test

import (
	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

func (s *KeeperTestSuite) TestGRPCQueryParams() {
	queryClient, require, ctx := s.queryClient, s.Require(), s.ctx

	expParams := types.DefaultParams()
	res, err := queryClient.GetParams(ctx, &types.QueryParamsRequest{})
	require.NoError(err)
	require.NotNil(res)
	require.Equal(expParams, res.Params)
}
