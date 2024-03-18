package clerk_test

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/app"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

//
// Create test app
//

// returns context and app on clerk keeper
// nolint: unparam
func createTestApp(t *testing.T, isCheckTx bool) (*app.HeimdallApp, sdk.Context) {
	app := app.Setup(t, isCheckTx)
	ctx := app.BaseApp.NewContext(isCheckTx)

	return app, ctx
}

// setupClerkGenesis initializes a new Heimdall with the default genesis data.
func setupClerkGenesis(t *testing.T) *app.HeimdallApp {
	happ, ctx := createTestApp(t, false)

	// initialize the chain with the default genesis state
	genesisState := happ.BasicManager.DefaultGenesis(happ.AppCodec())

	clerkGenesis := types.NewGenesisState(types.DefaultGenesisState().EventRecords, types.DefaultGenesisState().RecordSequences)
	genesisState[types.ModuleName] = happ.AppCodec().MustMarshalJSON(&clerkGenesis)

	// TODO HV2 - what marshiling are we using here? Update after the heimdall app PR is merged
	stateBytes, err := codec.MarshalJSONIndent(happ.LegacyAmino(), genesisState)
	if err != nil {
		panic(err)
	}

	happ.InitChain(
		&abci.RequestInitChain{
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)

	happ.Commit()
	happ.BeginBlocker(ctx)

	return happ
}
