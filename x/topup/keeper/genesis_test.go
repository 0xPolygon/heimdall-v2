package keeper_test

import (
	"github.com/0xPolygon/heimdall-v2/types"
	"math/rand"
	"strconv"
	"time"

	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
)

// TestInitExportGenesis tests import and export of genesis state
func (suite *KeeperTestSuite) TestInitExportGenesis() {
	keeper, ctx, require := suite.keeper, suite.ctx, suite.Require()
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	topupSequences := make([]string, 5)
	dividendAccounts := make([]types.DividendAccount, 5)
	accounts := simulation.RandomAccounts(r1, 5)

	for i := range topupSequences {
		topupSequences[i] = strconv.Itoa(simulation.RandIntBetween(r1, 1000, 100000))
	}
	for i := range dividendAccounts {
		dividendAccounts[i].User = accounts[i].Address.String()
		dividendAccounts[i].FeeAmount = strconv.Itoa(simulation.RandIntBetween(r1, 1000, 100000))
	}

	genesisState := topupTypes.GenesisState{
		TopupSequences:   topupSequences,
		DividendAccounts: dividendAccounts,
	}

	keeper.InitGenesis(ctx, &genesisState)

	actualParams := keeper.ExportGenesis(ctx)

	require.LessOrEqual(len(topupSequences), len(actualParams.TopupSequences))
	require.LessOrEqual(len(dividendAccounts), len(actualParams.DividendAccounts))
}
