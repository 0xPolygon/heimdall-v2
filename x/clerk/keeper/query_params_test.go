package keeper_test

import (
	"testing"

	testkeeper "github.com/0xPolygon/heimdall-v2/testutil/keeper"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestParamsQuery(t *testing.T) {
	t.Parallel()
	keeper, ctx := testkeeper.ClerkKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	params := types.DefaultParams()
	keeper.SetParams(ctx, params)

	response, err := keeper.Params(wctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
