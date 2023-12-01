package stake_test

import (
	"testing"

	keepertest "github.com/0xPolygon/heimdall-v2/testutil/keeper"
	"github.com/0xPolygon/heimdall-v2/testutil/nullify"
	"github.com/0xPolygon/heimdall-v2/x/stake"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx := keepertest.StakeKeeper(t)
	stake.InitGenesis(ctx, *k, genesisState)
	got := stake.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

}
