package clerk_test

// TODO HV2 - uncomment after the app tests are fixed
/*
// createTestApp returns context and app on clerk keeper
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

	_, err = happ.InitChain(
		&abci.RequestInitChain{
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)
	if err != nil {
		panic(err)
	}

	_, err = happ.Commit()
	if err != nil {
		panic(err)
	}

	_, err = happ.BeginBlocker(ctx)
	if err != nil {
		panic(err)
	}

	return happ
}
*/
