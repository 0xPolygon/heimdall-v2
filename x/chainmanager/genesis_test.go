package chainmanager_test

import (
	"testing"

	keepertest "github.com/0xPolygon/heimdall-v2/testutil/keeper"
	"github.com/0xPolygon/heimdall-v2/testutil/nullify"
	"github.com/0xPolygon/heimdall-v2/x/chainmanager"
	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	t.Parallel()
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx := keepertest.ChainmanagerKeeper(t)
	chainmanager.InitGenesis(ctx, *k, genesisState)
	got := chainmanager.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
