package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/0xPolygon/heimdall-v2/x/engine/testutil"
	"github.com/0xPolygon/heimdall-v2/x/engine/types"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := keepertest.EngineKeeper(t)
	params := types.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))

	response, err := keeper.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
