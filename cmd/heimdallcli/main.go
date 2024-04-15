package main

import (
	"fmt"
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func main() {
	rootCmd := NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "HD", os.ExpandEnv("/var/lib/heimdall")); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}

/*
func queryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:     "query",
		Aliases: []string{"q"},
		Short:   "Querying subcommands",
	}

	queryCmd.AddCommand(
		rpc.ValidatorCommand(),

		hmTxCli.QueryTxsByEventsCmd(cdc),
		hmTxCli.QueryTxCmd(cdc),
	)

	// add modules' query commands
	app.ModuleBasics.AddQueryCommands(queryCmd)

	return queryCmd
}

func txCmd(cdc *amino.Codec) *cobra.Command {
	txCmd := &cobra.Command{
		Use:   "tx",
		Short: "Transactions subcommands",
	}

	txCmd.AddCommand(
		authCli.GetSignCommand(),
		hmTxCli.GetBroadcastCommand(cdc),
		hmTxCli.GetEncodeCommand(cdc),
		flags.LineBreak,
	)

	// add modules' tx commands
	app.ModuleBasics.AddTxCommands(txCmd, cdc)

	return txCmd
}

func convertAddressToHexCmd(_ *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "address-to-hex [address]",
		Short: "Convert address to hex",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			fmt.Println("Hex:", ethCommon.BytesToAddress(key).String())
			return nil
		},
	}

	return flags.GetCommands(cmd)[0]
}

func convertHexToAddressCmd(_ *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hex-to-address [hex]",
		Short: "Convert hex to address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := ethCommon.HexToAddress(args[0])
			fmt.Println("Address:", sdk.AccAddress(address.Bytes()).String())
			return nil
		},
	}

	return client.GetCommands(cmd)[0]
}

// exportCmd a state dump file
func exportCmd(ctx *server.Context, _ *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-heimdall",
		Short: "Export genesis file with state-dump",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {

			// cliCtx := context.NewCLIContext().WithCodec(cdc)
			config := ctx.Config
			config.SetRoot(viper.GetString(cli.HomeFlag))

			// create chain id
			chainID := viper.GetString(client.FlagChainID)
			if chainID == "" {
				chainID = fmt.Sprintf("heimdall-%v", common.RandStr(6))
			}

			dataDir := path.Join(viper.GetString(cli.HomeFlag), "data")
			logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
			db, err := sdk.NewLevelDB("application", dataDir)
			if err != nil {
				panic(err)
			}

			happ := app.NewHeimdallApp(logger, db)
			appState, _, err := happ.ExportAppStateAndValidators()
			if err != nil {
				panic(err)
			}

			err = writeGenesisFile(file.Rootify("config/dump-genesis.json", config.RootDir), chainID, appState)
			if err == nil {
				fmt.Println("New genesis json file created:", file.Rootify("config/dump-genesis.json", config.RootDir))
			}
			return err
		},
	}
	cmd.Flags().String(cli.HomeFlag, helper.DefaultNodeHome, "node's home directory")
	cmd.Flags().String(helper.FlagClientHome, helper.DefaultCLIHome, "client's home directory")
	cmd.Flags().String(client.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")

	return cmd
}

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

// generateValidatorKey generate validator key
func generateValidatorKey(cdc *codec.Codec) *cobra.Command {
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
			var privObject secp256k1.PrivKeySecp256k1
			copy(privObject[:], ds)

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

//
// Internal functions
//

func writeGenesisFile(genesisFile, chainID string, appState json.RawMessage) error {
	genDoc := tmTypes.GenesisDoc{
		ChainID:  chainID,
		AppState: appState,
	}

	if err := genDoc.ValidateAndComplete(); err != nil {
		return err
	}

	return genDoc.SaveAs(genesisFile)
}

// keyFileName implements the naming convention for keyfiles:
// UTC--<created_at UTC ISO8601>-<address hex>
func keyFileName(keyAddr ethCommon.Address) string {
	ts := time.Now().UTC()
	return fmt.Sprintf("UTC--%s--%s", toISO8601(ts), hex.EncodeToString(keyAddr[:]))
}

func toISO8601(t time.Time) string {
	var tz string

	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}

	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}

// promptPassphrase prompts the user for a passphrase.  Set confirmation to true
// to require the user to confirm the passphrase.
func promptPassphrase(confirmation bool) (string, error) {
	passphrase, err := prompt.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		return "", err
	}

	if confirmation {
		confirm, err := prompt.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			return "", err
		}

		if passphrase != confirm {
			return "", errors.New("Passphrases do not match")
		}
	}

	return passphrase, nil
}
*/
