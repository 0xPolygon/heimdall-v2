package keeper_test

import (
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/0xPolygon/heimdall-v2/types"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
)

// TestInitExportGenesis tests import and export of genesis state
func (s *KeeperTestSuite) TestInitExportGenesis() {
	keeper, ctx, require := s.keeper, s.ctx, s.Require()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	num := 5

	topupSequences := make([]string, num)
	dividendAccounts := make([]types.DividendAccount, num)
	accounts := simulation.RandomAccounts(r, num)

	for i := range topupSequences {
		topupSequences[i] = strconv.Itoa(simulation.RandIntBetween(r, 1000, 100000))
	}
	for i := range dividendAccounts {
		dividendAccounts[i].User = accounts[i].Address.String()
		dividendAccounts[i].FeeAmount = strconv.Itoa(simulation.RandIntBetween(r, 1000, 100000))
	}

	genesisState := topupTypes.GenesisState{
		TopupSequences:   topupSequences,
		DividendAccounts: dividendAccounts,
	}

	keeper.InitGenesis(ctx, &genesisState)

	actualParams := keeper.ExportGenesis(ctx)

	require.LessOrEqual(len(topupSequences), len(actualParams.TopupSequences))
	require.LessOrEqual(len(dividendAccounts), len(actualParams.DividendAccounts))

	sort.Strings(actualParams.TopupSequences)
	sort.Strings(topupSequences)

	sort.Slice(actualParams.DividendAccounts, func(i, j int) bool {
		return actualParams.DividendAccounts[i].User < actualParams.DividendAccounts[j].User
	})

	sort.Slice(dividendAccounts, func(i, j int) bool {
		return dividendAccounts[i].User < dividendAccounts[j].User
	})

	for i := range topupSequences {
		require.Equal(topupSequences[i], actualParams.TopupSequences[i])
	}
	for i := range dividendAccounts {
		require.Equal(dividendAccounts[i], actualParams.DividendAccounts[i])
		require.Equal(dividendAccounts[i].User, actualParams.DividendAccounts[i].User)
	}
}
