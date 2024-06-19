package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"cosmossdk.io/log"
	confixcmd "cosmossdk.io/tools/confix/cmd"
	"github.com/0xPolygon/heimdall-v2/app"
	cmdhelper "github.com/0xPolygon/heimdall-v2/cmd"
	"github.com/0xPolygon/heimdall-v2/file"
	"github.com/0xPolygon/heimdall-v2/helper"
	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/cli"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	// TODO HV2 - uncomment when we have the client package
	// hmTxCli "github.com/0xPolygon/heimdall-v2/client/tx"
)

const (
	FlagLoadLatest  = "load-latest"
	DumpGenesosFile = "config/dump-genesis.json"
)

// initCometBFTConfig helps to override default CometBFT Config values.
// It return cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initCometBFTConfig() *cmtcfg.Config {
	return cmtcfg.DefaultConfig()
}

// initAppConfig helps to override default appConfig template and configs.
// It returns '"", nil” if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	return "", nil
}

// initRootCmd adds the necessary commands to the root command
func initRootCmd(
	rootCmd *cobra.Command,
	cliCtx client.Context,
	basicManager module.BasicManager,
) {
	cfg := sdk.GetConfig()
	cfg.Seal()

	rootCmd.AddCommand(
		genutilcli.InitCmd(basicManager, app.DefaultNodeHome),
		debug.Cmd(),
		confixcmd.ConfigCommand(),
		// TODO HV2 - should we have these?
		/*
			pruning.Cmd(newApp, app.DefaultNodeHome),
			snapshot.Cmd(newApp),
		*/
	)

	// TODO HV2 - should we have these?
	// server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport, addModuleInitFlags)

	// add keybase, auxiliary RPC, query, genesis, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		queryCommand(),
		txCommand(),
		keys.Commands(),
		exportCmd(),
		// TODO HV2 - do we need this? Why?
		// generateKeystore(),
		// PSP - TODO HV2 - uncomment this
		// generateValidatorKey(),

		StakeCmd(cliCtx),
		ApproveCmd(cliCtx),
	)
}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.QueryEventForTxCmd(),
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		server.QueryBlocksCmd(),
		authcmd.QueryTxCmd(),
		server.QueryBlockResultsCmd(),
		rpc.ValidatorCommand(),
		// TODO HV2 - uncomment when we have the client package
		/*
			hmTxCli.QueryTxsByEventsCmd(),
			hmTxCli.QueryTxCmd(),
		*/
	)

	return cmd
}

func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		authcmd.GetSimulateCmd(),
		authcmd.GetSignCommand(),
		// TODO HV2 - uncomment when we have the client package
		/*
			hmTxCli.GetBroadcastCommand(),
			hmTxCli.GetEncodeCommand(),
		*/
	)

	return cmd
}

// exportCmd exports genesis file with state-dump
func exportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-heimdall",
		Short: "Export genesis file with state-dump",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config
			config.SetRoot(viper.GetString(cli.HomeFlag))

			// create chain id
			chainID := viper.GetString(flags.FlagChainID)
			if chainID == "" {
				// TODO HV2 - check the random chain id generation (RandStringRunes)
				chainID = fmt.Sprintf("heimdall-%v", cmdhelper.RandStringRunes(6))
			}

			dataDir := path.Join(viper.GetString(cli.HomeFlag), "data")
			logger := log.NewLogger(os.Stderr)
			db, err := dbm.NewDB("application", dbm.GoLevelDBBackend, dataDir)
			if err != nil {
				panic(err)
			}

			forZeroHeight, err := cmd.Flags().GetBool(server.FlagForZeroHeight)
			if err != nil {
				panic(err)
			}

			jailAllowedAddrs, err := cmd.Flags().GetStringSlice(server.FlagJailAllowedAddrs)
			if err != nil {
				panic(err)
			}

			modulesToExport, err := cmd.Flags().GetStringSlice(server.FlagModulesToExport)
			if err != nil {
				panic(err)
			}

			loadLatest, err := cmd.Flags().GetBool(FlagLoadLatest)
			if err != nil {
				panic(err)
			}

			// TODO HV2 - what app options should we pass?
			// or should we pass nil? `app.EmptyAppOptions{}`
			appOptions := make(simtestutil.AppOptionsMap, 0)

			happ := app.NewHeimdallApp(logger, db, nil, loadLatest, appOptions)
			exportedApp, err := happ.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
			if err != nil {
				panic(err)
			}

			err = writeGenesisFile(file.Rootify(DumpGenesosFile, config.RootDir), chainID, exportedApp.AppState)
			if err == nil {
				fmt.Println("New genesis json file created:", file.Rootify(DumpGenesosFile, config.RootDir))
			}
			return err
		},
	}
	cmd.Flags().String(cli.HomeFlag, helper.DefaultNodeHome, "node's home directory")
	cmd.Flags().String(helper.FlagClientHome, helper.DefaultCLIHome, "client's home directory")
	cmd.Flags().String(flags.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")

	return cmd
}

/*
// generateKeystore generate keystore file from private key
func generateKeystore(_ *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-keystore <private-key>",
		Short: "Generates keystore file using private key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := strings.ReplaceAll(args[0], "0x", "")
			pk, err := crypto.HexToECDSA(s)
			if err != nil {
				return err
			}

			id, err := uuid.NewRandom()
			if err != nil {
				return err
			}
			key := &keystore.Key{
				Id:         id,
				Address:    crypto.PubkeyToAddress(pk.PublicKey),
				PrivateKey: pk,
			}

			passphrase, err := promptPassphrase(true)
			if err != nil {
				return err
			}

			keyjson, err := keystore.EncryptKey(key, passphrase, keystore.StandardScryptN, keystore.StandardScryptP)
			if err != nil {
				return err
			}

			// Then write the new keyfile in place of the old one.
			if err := os.WriteFile(keyFileName(key.Address), keyjson, 0600); err != nil {
				return err
			}
			return nil
		},
	}

	return client.GetCommands(cmd)[0]
}
*/

/*
// generateValidatorKey generate validator key
func generateValidatorKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-validatorkey <private-key>",
		Short: "Generate validator key file using private key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := strings.ReplaceAll(args[0], "0x", "")
			ds, err := hex.DecodeString(s)
			if err != nil {
				return err
			}

			// set private object
			var privObject secp256k1.PrivKey
			copy(privObject.Key[:], ds)

			// node key
			nodeKey := privval.FilePVKey{
				Address: privObject.PubKey().Address(),
				PubKey:  privObject.PubKey(),
				PrivKey: privObject,
			}

			jsonBytes, err := cdc.MarshalJSONIndent(nodeKey, "", "  ")
			if err != nil {
				return err
			}

			err = os.WriteFile("priv_validator_key.json", jsonBytes, 0600)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return client.GetCommands(cmd)[0]
}
*/

func writeGenesisFile(genesisFile, chainID string, appState json.RawMessage) error {
	genDoc := cmttypes.GenesisDoc{
		ChainID:  chainID,
		AppState: appState,
	}

	if err := genDoc.ValidateAndComplete(); err != nil {
		return err
	}

	return genDoc.SaveAs(genesisFile)
}
