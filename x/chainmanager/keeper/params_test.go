package keeper_test

import (
	"testing"

	testkeeper "github.com/0xPolygon/heimdall-v2/testutil/keeper"
	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	t.Parallel()
	k, ctx := testkeeper.ChainmanagerKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
