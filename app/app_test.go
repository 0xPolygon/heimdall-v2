package app

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/core/appmodule"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func TestHeimdallAppExport(t *testing.T) {
	// TODO HV2: enable this test once modules implementation is completed
	//  See https://polygon.atlassian.net/browse/POS-2626
	t.Skip("to be enabled")
	t.Parallel()
	app, db, logger := SetupApp(t, 1)

	// finalize block so we have CheckTx state set
	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 1,
	})

	require.NoError(t, err)

	_, err = app.Commit()
	require.NoError(t, err)

	// Making a new app object with the db, so that initchain hasn't been called
	app2 := NewHeimdallApp(logger, db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
	_, err = app2.ExportAppStateAndValidators(false, []string{}, []string{})
	require.NoError(t, err)
}

//nolint:tparallel
func TestRunMigrations(t *testing.T) {
	// TODO HV2: enable this test once modules implementation is completed
	//  See https://polygon.atlassian.net/browse/POS-2626
	t.Skip("to be enabled")
	t.Parallel()
	app, db, logger := SetupApp(t, 1)

	// Create a new baseapp and configurator for the purpose of this test.
	bApp := baseapp.NewBaseApp(app.Name(), logger.With("instance", "baseapp"), db, app.GetTxConfig().TxDecoder())
	bApp.SetCommitMultiStoreTracer(nil)
	bApp.SetInterfaceRegistry(app.InterfaceRegistry())
	app.BaseApp = bApp
	configurator := module.NewConfigurator(app.appCodec, bApp.MsgServiceRouter(), app.GRPCQueryRouter())

	// We register all modules on the Configurator, except x/bank. x/bank will
	// serve as the test subject on which we run the migration tests.
	//
	// The loop below is the same as calling `RegisterServices` on
	// ModuleManager, except that we skip x/bank.
	for name, mod := range app.mm.Modules {
		if name == banktypes.ModuleName {
			continue
		}

		if mod, ok := mod.(module.HasServices); ok {
			mod.RegisterServices(configurator)
		}

		if mod, ok := mod.(appmodule.HasServices); ok {
			err := mod.RegisterServices(configurator)
			require.NoError(t, err)
		}

		require.NoError(t, configurator.Error())
	}

	// Initialize the chain
	_, err := app.InitChain(&abci.RequestInitChain{})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)

	testCases := []struct {
		name         string
		moduleName   string
		fromVersion  uint64
		toVersion    uint64
		expRegErr    bool // errors while registering migration
		expRegErrMsg string
		expRunErr    bool // errors while running migration
		expRunErrMsg string
		expCalled    int
	}{
		{
			"cannot register migration for version 0",
			"bank", 0, 1,
			true, "module migration versions should start at 1: invalid version", false, "", 0,
		},
		{
			"throws error on RunMigrations if no migration registered for bank",
			"", 1, 2,
			false, "", true, "no migrations found for module bank: not found", 0,
		},
		{
			"can register 1->2 migration handler for x/bank, cannot run migration",
			"bank", 1, 2,
			false, "", true, "no migration found for module bank from version 2 to version 3: not found", 0,
		},
		{
			"can register 2->3 migration handler for x/bank, can run migration",
			"bank", 2, bank.AppModule{}.ConsensusVersion(),
			false, "", false, "", int(bank.AppModule{}.ConsensusVersion() - 2), // minus 2 because 1-2 is run in the previous test case.
		},
		{
			"cannot register migration handler for same module & fromVersion",
			"bank", 1, 2,
			true, "another migration for module bank and version 1 already exists: internal logic error", false, "", 0,
		},
	}

	//nolint:paralleltest
	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			var err error

			// Since it's very hard to test actual in-place store migrations in
			// tests (due to the difficulty of maintaining multiple versions of a
			// module), we're just testing here that the migration logic is
			// called.
			called := 0

			if tc.moduleName != "" {
				for i := tc.fromVersion; i < tc.toVersion; i++ {
					// Register migration for module from version `fromVersion` to `fromVersion+1`.
					tt.Logf("Registering migration for %q v%d", tc.moduleName, i)
					err = configurator.RegisterMigration(tc.moduleName, i, func(sdk.Context) error {
						called++

						return nil
					})

					if tc.expRegErr {
						require.EqualError(tt, err, tc.expRegErrMsg)

						return
					}
					require.NoError(tt, err, "registering migration")
				}
			}

			// Run migrations only for bank. That's why we put the initial
			// version for bank as 1, and for all other modules, we put as
			// their latest ConsensusVersion.
			_, err = app.mm.RunMigrations(
				app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()}), configurator,
				module.VersionMap{
					"bank": 1,
					"auth": auth.AppModule{}.ConsensusVersion(),
					"gov":  gov.AppModule{}.ConsensusVersion(),
					// TODO HV2: do we need to add ConsensusVersion for all custom modules?
					// "stake":      stake.AppModule{}.ConsensusVersion(),
					// "bor": bor.AppModule{}.ConsensusVersion(),
					// "clerk": clerk.AppModule{}.ConsensusVersion(),
					// "checkpoint": checkpoint.AppModule{}.ConsensusVersion(),
					// "topup": topup.AppModule{}.ConsensusVersion(),
					// "chainmanager": chainmanager.AppModule{}.ConsensusVersion(),

				},
			)

			if tc.expRunErr {
				require.EqualError(tt, err, tc.expRunErrMsg, "running migration")
			} else {
				require.NoError(tt, err, "running migration")
				// Make sure bank's migration is called.
				require.Equal(tt, tc.expCalled, called)
			}
		})
	}
}

func TestInitGenesisOnMigration(t *testing.T) {
	// TODO HV2: enable this test once modules implementation is completed
	//  See https://polygon.atlassian.net/browse/POS-2626
	t.Skip("to be enabled")
	t.Parallel()
	app, _, _ := SetupApp(t, 1)
	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})

	// Create a mock module. This module will serve as the new module we're
	// adding during a migration.
	mockCtrl := gomock.NewController(t)
	t.Cleanup(mockCtrl.Finish)
	mockModule := mock.NewMockAppModuleWithAllExtensions(mockCtrl)
	mockDefaultGenesis := json.RawMessage(`{"key": "value"}`)
	mockModule.EXPECT().DefaultGenesis(gomock.Eq(app.appCodec)).Times(1).Return(mockDefaultGenesis)
	mockModule.EXPECT().InitGenesis(gomock.Eq(ctx), gomock.Eq(app.appCodec), gomock.Eq(mockDefaultGenesis)).Times(1)
	mockModule.EXPECT().ConsensusVersion().Times(1).Return(uint64(0))

	app.mm.Modules["mock"] = mockModule

	// Run migrations only for "mock" module. We exclude it from
	// the VersionMap to simulate upgrading with a new module.
	_, err := app.mm.RunMigrations(ctx, app.configurator,
		module.VersionMap{
			"bank": 1,
			"auth": auth.AppModule{}.ConsensusVersion(),
			"gov":  gov.AppModule{}.ConsensusVersion(),
			// TODO HV2: do we need to add ConsensusVersion for all custom modules?
			// "stake":      stake.AppModule{}.ConsensusVersion(),
			// "bor": bor.AppModule{}.ConsensusVersion(),
			// "clerk": clerk.AppModule{}.ConsensusVersion(),
			// "checkpoint": checkpoint.AppModule{}.ConsensusVersion(),
			// "topup": topup.AppModule{}.ConsensusVersion(),
			// "chainmanager": chainmanager.AppModule{}.ConsensusVersion(),
		},
	)
	require.NoError(t, err)
}

func TestValidateGenesis(t *testing.T) {
	// TODO HV2: enable this test once modules implementation is completed
	//  See https://polygon.atlassian.net/browse/POS-2626
	t.Skip("to be enabled")
	t.Parallel()

	happ, _, _ := SetupApp(t, 1)

	// not valid app state
	require.Panics(t, func() {
		_, err := happ.InitChain(
			&abci.RequestInitChain{
				Validators:    []abci.ValidatorUpdate{},
				AppStateBytes: []byte("{}"),
			},
		)
		require.Error(t, err)
	})
}

func TestGetMaccPerms(t *testing.T) {
	// TODO HV2: enable this test once modules implementation is completed
	//  See https://polygon.atlassian.net/browse/POS-2626
	t.Skip("to be enabled")
	t.Parallel()

	dup := GetMaccPerms()
	require.Equal(t, maccPerms, dup, "duplicated module account permissions differed from actual module account permissions")
}
