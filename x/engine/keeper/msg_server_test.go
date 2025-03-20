package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/engine/keeper"
	keepertest "github.com/0xPolygon/heimdall-v2/x/engine/testutil"
	"github.com/0xPolygon/heimdall-v2/x/engine/types"
)

func setupMsgServer(t testing.TB) (keeper.Keeper, types.MsgServer, context.Context) {
	k, ctx := keepertest.EngineKeeper(t)

	return k, keeper.NewMsgServerImpl(k), ctx
}

func TestMsgServer(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}
