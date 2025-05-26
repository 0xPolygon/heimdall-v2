package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/log"
	rpcserver "github.com/cometbft/cometbft/rpc/jsonrpc/server"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/0xPolygon/heimdall-v2/app"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/version"
)

const (
	bridgeDBFlag   = "bridge-db"
	borChainIDFlag = "bor-chain-id"
	logsTypeFlag   = "logs-type"
)

var (
	logger = helper.Logger.With("module", "bridge/cmd/")

	metricsServer http.Server
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "heimdall-bridge",
	Aliases: []string{"bridge"},
	Short:   "Heimdall bridge daemon",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Use != version.Cmd.Use {
			// initialize cometbft viper config
			initCometBFTViperConfig(cmd)

			// init metrics server
			initMetrics()
		}
	},
	PostRunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		return metricsServer.Shutdown(ctx)
	},
}

// BridgeCommands returns command for bridge service
func BridgeCommands(v *viper.Viper, loggerInstance log.Logger, caller string) *cobra.Command {
	DecorateWithBridgeRootFlags(rootCmd, v, loggerInstance, caller)
	return rootCmd
}

// DecorateWithBridgeRootFlags is called when bridge flags needs to be added to command
func DecorateWithBridgeRootFlags(cmd *cobra.Command, v *viper.Viper, loggerInstance log.Logger, caller string) {
	cmd.PersistentFlags().StringP(helper.CometBFTNodeFlag, "n", helper.DefaultCometBFTNode, "Node to connect to")

	if err := v.BindPFlag(helper.CometBFTNodeFlag, cmd.PersistentFlags().Lookup(helper.CometBFTNodeFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, helper.CometBFTNodeFlag), "Error", err)
	}

	cmd.PersistentFlags().String(flags.FlagHome, app.DefaultNodeHome, "directory for config and data")

	if err := v.BindPFlag(flags.FlagHome, cmd.PersistentFlags().Lookup(flags.FlagHome)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, flags.FlagHome), "Error", err)
	}

	// bridge storage db
	cmd.PersistentFlags().String(
		bridgeDBFlag,
		"",
		"Bridge db path (default <home>/bridge/storage)",
	)

	if err := v.BindPFlag(bridgeDBFlag, cmd.PersistentFlags().Lookup(bridgeDBFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, bridgeDBFlag), "Error", err)
	}

	// bridge chain id
	cmd.PersistentFlags().String(
		borChainIDFlag,
		helper.DefaultBorChainID,
		"Bor chain id",
	)

	// bridge logging type
	cmd.PersistentFlags().String(
		logsTypeFlag,
		helper.DefaultLogsType,
		"Use json logger",
	)

	if err := v.BindPFlag(borChainIDFlag, cmd.PersistentFlags().Lookup(borChainIDFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, borChainIDFlag), "Error", err)
	}
}

// initMetrics initializes metrics server with the default handler
func initMetrics() {
	cfg := rpcserver.DefaultConfig()

	metricsServer = http.Server{
		Addr:              ":2112",
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		if err := metricsServer.ListenAndServe(); err != nil {
			logger.Error("failed to start metrics server", "error", err)
			os.Exit(1)
		}
	}()
}

// AdjustBridgeDBValue function is called to set appropriate bridge db path
func AdjustBridgeDBValue(cmd *cobra.Command) {
	cometbftNode, _ := cmd.Flags().GetString(helper.CometBFTNodeFlag)
	withHeimdallConfigValue, _ := cmd.Flags().GetString(helper.WithHeimdallConfigFlag)
	bridgeDBValue, _ := cmd.Flags().GetString(bridgeDBFlag)
	borChainIDValue, _ := cmd.Flags().GetString(borChainIDFlag)
	logsTypeValue, _ := cmd.Flags().GetString(logsTypeFlag)

	// bridge-db directory (default storage)
	if bridgeDBValue == "" {
		bridgeDBValue = filepath.Join(viper.GetString(flags.FlagHome), "bridge", "storage")
	}

	// set to viper
	viper.Set(helper.CometBFTNodeFlag, cometbftNode)
	viper.Set(helper.WithHeimdallConfigFlag, withHeimdallConfigValue)
	viper.Set(bridgeDBFlag, bridgeDBValue)
	viper.Set(borChainIDFlag, borChainIDValue)
	viper.Set(logsTypeFlag, logsTypeValue)
}

// initCometBFTViperConfig sets global viper configuration needed to heimdall
func initCometBFTViperConfig(cmd *cobra.Command) {
	// set appropriate bridge DB
	AdjustBridgeDBValue(cmd)

	// start heimdall config
	helper.InitHeimdallConfig("")
}
