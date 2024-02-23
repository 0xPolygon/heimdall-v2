package keeper

import (
	"testing"

	"github.com/0xPolygon/heimdall-v2/app"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

// GenesisTestSuite integrate test suite context object
type GenesisTestSuite struct {
	suite.Suite

	app *app.App
	ctx sdk.Context
}

// SetupTest setup necessary things for genesis test
func (suite *GenesisTestSuite) SetupTest() {
	// TODO HV2: uncomment when the app test utils are implemented:https://github.com/0xPolygon/heimdall-v2/pull/12
	// suite.app, suite.ctx = createTestApp(true)
}

// TestGenesisTestSuite
func TestGenesisTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(GenesisTestSuite))
}

// TestInitExportGenesis test import and export genesis state
// // TODO HV2: uncomment when the heimdall app is implemented:https://github.com/0xPolygon/heimdall-v2/pull/9
// func (suite *GenesisTestSuite) TestInitExportGenesis() {
// 	t, app, ctx := suite.T(), suite.app, suite.ctx
// 	params := types.DefaultParams()

// 	genesisState := types.GenesisState{
// 		Params: params,
// 	}
// 	InitGenesis(ctx, app.ChainKeeper, genesisState)

// 	actualParams := ExportGenesis(ctx, app.ChainKeeper)
// 	require.Equal(t, genesisState, actualParams)
// }
