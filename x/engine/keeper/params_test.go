package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/0xPolygon/heimdall-v2/x/engine/testutil"
	"github.com/0xPolygon/heimdall-v2/x/engine/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.EngineKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
