package keeper_test

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

func (s *KeeperTestSuite) TestGRPCQueryParams() {
	queryClient := s.queryClient
	require := s.Require()

	expParams := types.DefaultParams()
	res, err := queryClient.Params(context.Background(), &types.QueryParamsRequest{})
	require.NoError(err)
	require.NotNil(res)
	require.Equal(expParams, res.Params)
}
