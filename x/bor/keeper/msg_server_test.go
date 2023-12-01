package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/0xPolygon/heimdall-v2/testutil/keeper"
	"github.com/0xPolygon/heimdall-v2/x/bor/keeper"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func setupMsgServer(tb testing.TB) (types.MsgServer, context.Context) {
	tb.Helper()
	k, ctx := keepertest.BorKeeper(tb)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}

func TestMsgServer(t *testing.T) {
	t.Parallel()
	ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
}
