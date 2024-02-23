package keeper

import (
	"testing"

	"github.com/0xPolygon/heimdall-v2/app"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

type KeeperTestSuite struct {
	suite.Suite

	app *app.App
	ctx sdk.Context
}

func (suite *KeeperTestSuite) SetupTest() {
	// TODO HV2: uncomment when the app test utils are implemented:https://github.com/0xPolygon/heimdall-v2/pull/12
	// suite.app, suite.ctx = createTestApp(false)
}

func TestKeeperTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(KeeperTestSuite))
}

// TODO HV2: uncomment when the heimdall app is implemented:https://github.com/0xPolygon/heimdall-v2/pull/9
// func (suite *KeeperTestSuite) TestParamsGetterSetter() {
// 	t, app, ctx := suite.T(), suite.app, suite.ctx
// 	params := types.DefaultParams()

// 	app.ChainKeeper.SetParams(ctx, params)

// 	actualParams := app.ChainKeeper.GetParams(ctx)

// 	require.Equal(t, params, actualParams)
// }
