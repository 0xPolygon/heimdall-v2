package app

import (
	"testing"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// HeimdallAppChainID hardcoded chainID for simulation
const HeimdallAppChainID = "simulation-app"

var FlagEnableStreamingValue bool

var FlagEnableBenchStreamingValue bool

// Get flags every time the simulator is run
func init() {
	flag.BoolVar(&FlagEnableBenchStreamingValue, "EnableStreaming", false, "Enable streaming service")
}

// Setup initializes a new App. A Nop logger is set in App.
func Setup(t *testing.T, isCheckTx bool) *HeimdallApp {
	db := dbm.NewMemDB()

	appOptions := viper.New()
	if FlagEnableStreamingValue {
		m := make(map[string]interface{})
		m["streaming.abci.keys"] = []string{"*"}
		m["streaming.abci.plugin"] = "abci_v1" //nolint:goconst
		m["streaming.abci.stop-node-on-err"] = true
		for key, value := range m {
			appOptions.SetDefault(key, value)
		}
	}
	appOptions.SetDefault(flags.FlagHome, DefaultNodeHome)
	appOptions.SetDefault(server.FlagInvCheckPeriod, simcli.FlagPeriodValue)

	app := NewHeimdallApp(log.NewNopLogger(), db, nil, true, appOptions, baseapp.SetChainID(HeimdallAppChainID))

	if !isCheckTx {
		// init chain must be called to stop deliverState from being nil
		genesisState := app.DefaultGenesis()

		stateBytes, err := codec.MarshalJSONIndent(app.LegacyAmino(), genesisState)
		if err != nil {
			panic(err)
		}

		// Initialize the chain
		app.InitChain(
			&abci.RequestInitChain{
				Validators:    []abci.ValidatorUpdate{},
				AppStateBytes: stateBytes,
			},
		)
	}

	return app
}
