package test

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/0xPolygon/heimdall-v2/app"
	topupTypes "github.com/0xPolygon/heimdall-v2/x/topup/types"
)

// GenesisTestSuite integrate test suite context object
type GenesisTestSuite struct {
	suite.Suite

	app *app.HeimdallApp
	ctx sdk.Context
}

// SetupTest setup necessary things for genesis test
func (suite *GenesisTestSuite) SetupTest() {
	suite.app, _, _ = app.SetupApp(suite.T(), 1)
}

// TestGenesisTestSuite
func TestGenesisTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(GenesisTestSuite))
}

// TestInitExportGenesis test import and export genesis state
func (suite *GenesisTestSuite) TestInitExportGenesis() {
	t, heimdallApp, ctx := suite.T(), suite.app, suite.ctx
	k := heimdallApp.TopupKeeper
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	topupSequences := make([]string, 5)

	for i := range topupSequences {
		topupSequences[i] = strconv.Itoa(simulation.RandIntBetween(r1, 1000, 100000))
	}

	genesisState := topupTypes.GenesisState{
		TopupSequences: topupSequences,
	}
	k.InitGenesis(ctx, &genesisState)

	actualParams := k.ExportGenesis(ctx)

	require.LessOrEqual(t, len(topupSequences), len(actualParams.TopupSequences))
}
