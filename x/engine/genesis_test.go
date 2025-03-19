package engine_test

import (
	"testing"

	"github.com/0xPolygon/heimdall-v2/x/engine"
	keepertest "github.com/0xPolygon/heimdall-v2/x/engine/testutil"
	"github.com/0xPolygon/heimdall-v2/x/engine/testutil/nullify"
	"github.com/0xPolygon/heimdall-v2/x/engine/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.EngineKeeper(t)
	engine.InitGenesis(ctx, k, genesisState)
	got := engine.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
