package heimdalld

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/stretchr/testify/require"

	heimdallApp "github.com/0xPolygon/heimdall-v2/app"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
)

func TestValidateGenesisAfterMigration(t *testing.T) {
	db := dbm.NewMemDB()

	appOptions := make(simtestutil.AppOptionsMap)
	appOptions[flags.FlagHome] = heimdallApp.DefaultNodeHome

	logger := log.NewTestLogger(t)
	app := heimdallApp.NewHeimdallApp(logger, db, nil, true, appOptions)

	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})

	genDoc, err := cmttypes.GenesisDocFromFile("./testdata/migrated_dump-genesis.json")
	require.NoError(t, err)

	var genesisState heimdallApp.GenesisState
	err = json.Unmarshal(genDoc.AppState, &genesisState)
	require.NoError(t, err)

	appCodec := app.AppCodec()
	err = app.BasicManager.ValidateGenesis(appCodec, nil, genesisState)
	require.NoError(t, err)

	_, err = app.ModuleManager.InitGenesis(ctx, appCodec, genesisState)
	require.NoError(t, err)
}
