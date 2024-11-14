package app

import (
	"encoding/json"
	"testing"

	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/0xPolygon/heimdall-v2/x/bor"
	"github.com/0xPolygon/heimdall-v2/x/chainmanager"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint"
	"github.com/0xPolygon/heimdall-v2/x/clerk"
	"github.com/0xPolygon/heimdall-v2/x/milestone"
	"github.com/0xPolygon/heimdall-v2/x/stake"
	"github.com/0xPolygon/heimdall-v2/x/topup"
)

func TestHeimdallAppExport(t *testing.T) {
	app, _, _ := SetupApp(t, 1)

	_, err := app.ExportAppStateAndValidators(false, []string{}, []string{})
	require.NoError(t, err)
}

func TestRunMigrations(t *testing.T) {
	hApp, _, _ := SetupApp(t, 1)
	configurator := module.NewConfigurator(hApp.appCodec, hApp.MsgServiceRouter(), hApp.GRPCQueryRouter())

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
			_, err = hApp.ModuleManager.RunMigrations(
				hApp.NewContextLegacy(true, cmtproto.Header{Height: hApp.LastBlockHeight()}), configurator,
				module.VersionMap{
					"bank":         1,
					"auth":         auth.AppModule{}.ConsensusVersion(),
					"gov":          gov.AppModule{}.ConsensusVersion(),
					"stake":        stake.AppModule{}.ConsensusVersion(),
					"clerk":        clerk.AppModule{}.ConsensusVersion(),
					"checkpoint":   checkpoint.AppModule{}.ConsensusVersion(),
					"chainmanager": chainmanager.AppModule{}.ConsensusVersion(),
					"milestone":    milestone.AppModule{}.ConsensusVersion(),
					"topup":        topup.AppModule{}.ConsensusVersion(),
					"bor":          bor.AppModule{}.ConsensusVersion(),
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

	app.ModuleManager.Modules["mock"] = mockModule

	// Run migrations only for "mock" module. We exclude it from
	// the VersionMap to simulate upgrading with a new module.
	_, err := app.ModuleManager.RunMigrations(ctx, app.configurator,
		module.VersionMap{
			"bank":         bank.AppModule{}.ConsensusVersion(),
			"auth":         auth.AppModule{}.ConsensusVersion(),
			"gov":          gov.AppModule{}.ConsensusVersion(),
			"stake":        stake.AppModule{}.ConsensusVersion(),
			"clerk":        clerk.AppModule{}.ConsensusVersion(),
			"checkpoint":   checkpoint.AppModule{}.ConsensusVersion(),
			"chainmanager": chainmanager.AppModule{}.ConsensusVersion(),
			"milestone":    milestone.AppModule{}.ConsensusVersion(),
			"topup":        topup.AppModule{}.ConsensusVersion(),
			"bor":          bor.AppModule{}.ConsensusVersion(),
		},
	)
	require.NoError(t, err)
}

func TestValidateGenesis(t *testing.T) {
	t.Parallel()

	hApp, _, _ := SetupApp(t, 1)

	// not valid app state
	require.Panics(t, func() {
		_, err := hApp.InitChain(
			&abci.RequestInitChain{
				Validators:    []abci.ValidatorUpdate{},
				AppStateBytes: []byte("{}"),
			},
		)
		require.Error(t, err)
	})
}

func TestGetMaccPerms(t *testing.T) {
	t.Parallel()

	dup := GetMaccPerms()
	require.Equal(t, maccPerms, dup, "duplicated module account permissions differed from actual module account permissions")
}
