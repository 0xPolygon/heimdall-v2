package clerk_test

import (
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
func createTestApp(isCheckTx bool) (*app.HeimdallApp, sdk.Context) {
	app := &app.HeimdallApp{}
	ctx := app.BaseApp.NewContext(isCheckTx)

	return app, ctx
}

// setupClerkGenesis initializes a new Heimdall with the default genesis data.
func setupClerkGenesis() *app.HeimdallApp {
	happ := &app.HeimdallApp{}

	ctx := happ.BaseApp.NewContext(false)

	// initialize the chain with the default genesis state
	genesisState := happ.BasicManager.DefaultGenesis(happ.AppCodec())

	clerkGenesis := types.NewGenesisState(types.DefaultGenesisState().EventRecords, types.DefaultGenesisState().RecordSequences)
	genesisState[types.ModuleName] = happ.AppCodec().MustMarshalJSON(&clerkGenesis)

	// TODO HV2 - what marshiling are we using here? Update after the heimdall app PR is merged
	// stateBytes, err := codec.MarshalJSONIndent(happ.AppCodec(), genesisState)
	stateBytes, err := codec.MarshalJSONIndent(nil, genesisState)
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
