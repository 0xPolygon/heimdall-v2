package heimdalld

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cosmossdk.io/log"
	confixcmd "cosmossdk.io/tools/confix/cmd"
	cmtcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/libs/cli"
	cmtos "github.com/cometbft/cometbft/libs/os"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/cosmos/cosmos-sdk/server/types"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console/prompt"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/0xPolygon/heimdall-v2/app"
	bridgeCmd "github.com/0xPolygon/heimdall-v2/bridge/cmd"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/version"
)

const (
	flagNodeDirPrefix    = "node-dir-prefix"
	flagNumValidators    = "v"
	flagNumNonValidators = "n"
	flagOutputDir        = "output-dir"
	flagNodeDaemonHome   = "node-daemon-home"
	flagNodeCliHome      = "node-cli-home"
	flagNodeHostPrefix   = "node-host-prefix"
)

// CometBFT full-node start flags
const (
	flagAddress      = "address"
	flagTraceStore   = "trace-store"
	flagPruning      = "pruning"
	flagCPUProfile   = "cpu-profile"
	FlagMinGasPrices = "minimum-gas-prices"
	FlagHaltHeight   = "halt-height"
	FlagHaltTime     = "halt-time"
)

// Open Collector Flags
var (
	FlagOpenTracing           = "open-tracing"
	FlagOpenCollectorEndpoint = "open-collector-endpoint"
)

const (
	nodeDirPerm = 0755
)

var tempDir = func() string {
	dir, err := os.MkdirTemp("", "heimdall")
	if err != nil {
		dir = app.DefaultNodeHome
	}
	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			fmt.Printf("Failed to remove directory %s: %v\n", dir, err)
		}
	}()

	return dir
}

// ValidatorAccountFormatter helps to print local validator account information
type ValidatorAccountFormatter struct {
	Address string `json:"address,omitempty" yaml:"address"`
	PrivKey string `json:"priv_key,omitempty" yaml:"priv_key"`
	PubKey  string `json:"pub_key,omitempty" yaml:"pub_key"`
}

func CryptoKeyToPubKey(key crypto.PubKey) secp256k1.PubKey {
	return helper.GetPubObjects(key)
}

// GetSignerInfo returns signer information
func GetSignerInfo(pub crypto.PubKey, privKey []byte, cdc *codec.LegacyAmino) ValidatorAccountFormatter {
	privKeyObject := secp256k1.PrivKey(privKey)
	pubKeyObject := secp256k1.PubKey(pub.Bytes())

	return ValidatorAccountFormatter{
		Address: ethCommon.BytesToAddress(pub.Address().Bytes()).String(),
		PubKey:  pubKeyObject.String(),
		PrivKey: "0x" + hex.EncodeToString(privKeyObject[:]),
	}
}

// initCometBFTConfig helps to override default CometBFT Config values.
// It returns cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initCometBFTConfig() *cmtcfg.Config {
	return cmtcfg.DefaultConfig()
}

// initAppConfig helps to override default appConfig template and configs.
// It returns "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	srvConf := serverconfig.DefaultConfig()
	srvConf.API.Enable = true // enable REST server by default
	srvConf.API.Address = helper.DefaultHeimdallServerURL
	customAppConfig := helper.CustomAppConfig{
		Config: *srvConf,
		Custom: helper.GetDefaultHeimdallConfig(),
	}

	customAppTemplate := serverconfig.DefaultConfigTemplate + helper.DefaultConfigTemplate

	return customAppTemplate, customAppConfig
}

func initRootCmd(
	rootCmd *cobra.Command,
	_ client.TxConfig,
	basicManager module.BasicManager,
	hApp *app.HeimdallApp,
) {
	cdc := codec.NewLegacyAmino()
	ctx := server.NewDefaultContext()

	cfg := sdk.GetConfig()
	cfg.Seal()

	rootCmd.AddCommand(
		genutilcli.InitCmd(basicManager, app.DefaultNodeHome),
		// TODO HV2 - check this (Testnet Command)
		// NewTestnetCmd(basicManager, banktypes.GenesisBalancesIterator{}),
		debug.Cmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(newApp, app.DefaultNodeHome),
		snapshot.Cmd(newApp),
		MigrateCommand(),
	)

	AddCommandsWithStartCmdOptions(rootCmd, app.DefaultNodeHome, newApp, appExport, server.StartCmdOptions{
		AddFlags: func(startCmd *cobra.Command) {
			startCmd.Flags().Bool(helper.RestServerFlag, true, "Enable the REST server")
			startCmd.Flags().Bool(helper.BridgeFlag, false, "Enable the bridge server")
			startCmd.Flags().Bool(helper.AllProcessesFlag, false, "Enable all bridge processes")
			startCmd.Flags().Bool(helper.OnlyProcessesFlag, false, "Enable only the specified bridge process(es)")
		},
		PostSetup: func(svrCtx *server.Context, clientCtx client.Context, ctx context.Context, g *errgroup.Group) error {
			helper.InitHeimdallConfig("")

			// wait for rest server to start
			if err := g.Wait(); err != nil {
				return fmt.Errorf("error waiting for goroutines: %w", err)
			}

			// start bridge
			if viper.GetBool(helper.BridgeFlag) {
				bridgeCmd.AdjustBridgeDBValue(rootCmd)
				g.Go(func() error {
					return bridgeCmd.StartBridgeWithCtx(ctx)
				})
			}

			return nil
		},
	})

	// add keybase, auxiliary RPC, query, genesis, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		// TODO HV2: enable this? Removed from app and not present in v1
		// genesisCommand(txConfig, basicManager),
		queryCommand(),
		txCommand(),
		keys.Commands(),
	)

	// add custom commands
	rootCmd.AddCommand(
		testnetCmd(ctx, cdc, hApp.BasicManager),
		generateKeystore(),
		importKeyStore(),
		generateValidatorKey(),
		importValidatorKey(),
		StakeCmd(),
		ApproveCmd(),
		version.Cmd,
	)

	rootCmd.AddCommand(showPrivateKeyCmd())
	rootCmd.AddCommand(bridgeCmd.BridgeCommands(viper.GetViper(), logger, "main"))
	rootCmd.AddCommand(VerifyGenesis(ctx, hApp))

	// TODO HV2 - I guess we are safe to remove this, as `genutilcli.InitCmd(basicManager, app.DefaultNodeHome)`
	// already does the same thing
	// commenting it out for now, will remove it later (after testing)
	// rootCmd.AddCommand(initCmd(ctx, cdc, hApp.BasicManager))

}

// AddCommandsWithStartCmdOptions adds server commands with the provided StartCmdOptions.
// HV2 - This function is taken from cosmos-sdk
func AddCommandsWithStartCmdOptions(rootCmd *cobra.Command, defaultNodeHome string, appCreator types.AppCreator, appExport types.AppExporter, opts server.StartCmdOptions) {
	cometCmd := &cobra.Command{
		Use:     "comet",
		Aliases: []string{"cometbft", "tendermint"},
		Short:   "CometBFT subcommands",
	}

	cometCmd.AddCommand(
		server.ShowNodeIDCmd(),
		server.ShowValidatorCmd(),
		server.ShowAddressCmd(),
		server.VersionCmd(),
		cmtcmd.ResetAllCmd,
		cmtcmd.ResetStateCmd,
		server.BootstrapStateCmd(appCreator),
	)

	startCmd := server.StartCmdWithOptions(appCreator, defaultNodeHome, opts)

	rootCmd.AddCommand(
		startCmd,
		cometCmd,
		server.ExportCmd(appExport, defaultNodeHome),
		server.NewRollbackCmd(appCreator, defaultNodeHome),
	)
}

// genesisCommand builds genesis-related `heimdalld genesis` command. Users may provide application specific commands as a parameter
// nolint:unused // TODO - remove this once the function is being used
func genesisCommand(txConfig client.TxConfig, basicManager module.BasicManager, cmds ...*cobra.Command) *cobra.Command {
	cmd := genutilcli.Commands(txConfig, basicManager, app.DefaultNodeHome)

	for _, subCmd := range cmds {
		cmd.AddCommand(subCmd)
	}
	return cmd
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
	)

	return cmd
}

// newApp creates the application
func newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	baseappOptions := server.DefaultBaseappOptions(appOpts)

	return app.NewHeimdallApp(
		logger, db, traceStore, true,
		appOpts,
		baseappOptions...,
	)
}

// appExport creates a new heimdall app (optionally at a given height) and exports state.
func appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	// this check is necessary as we use the flag in x/upgrade.
	// we can exit more gracefully by checking the flag here.
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home not set")
	}

	viperAppOpts, ok := appOpts.(*viper.Viper)
	if !ok {
		return servertypes.ExportedApp{}, errors.New("appOpts is not viper.Viper")
	}

	// overwrite the FlagInvCheckPeriod
	viperAppOpts.Set(server.FlagInvCheckPeriod, 1)
	appOpts = viperAppOpts

	var hApp *app.HeimdallApp
	if height != -1 {
		hApp = app.NewHeimdallApp(logger, db, traceStore, false, appOpts)

		if err := hApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	} else {
		hApp = app.NewHeimdallApp(logger, db, traceStore, true, appOpts)
	}

	return hApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

// generateKeystore generate keystore file from private key
func generateKeystore() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-keystore",
		Short: "Generates keystore file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pk, err := ethcrypto.GenerateKey()
			if err != nil {
				return err
			}

			if err = createKeyStore(pk); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

// importKeyStore imports keystore from private key in the given file path
func importKeyStore() *cobra.Command {
	return &cobra.Command{
		Use:   "import-keystore <keystore-file>",
		Short: "Import keystore from a private key stored in file (without 0x prefix)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pk, err := ethcrypto.LoadECDSA(args[0])
			if err != nil {
				return err
			}

			if err = createKeyStore(pk); err != nil {
				return err
			}

			return nil
		},
	}
}

// generateValidatorKey generate validator key
func generateValidatorKey() *cobra.Command {
	cdc := codec.NewLegacyAmino()
	cmd := &cobra.Command{
		Use:   "generate-validator-key",
		Short: "Generate validator key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			// generate private key
			privKeyObject := secp256k1.GenPrivKey()

			// node key
			nodeKey := privval.FilePVKey{
				Address: privKeyObject.PubKey().Address(),
				PubKey:  privKeyObject.PubKey(),
				PrivKey: privKeyObject,
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

	return cmd
}

// importValidatorKey imports validator private key from the given file path
func importValidatorKey() *cobra.Command {
	cdc := codec.NewLegacyAmino()
	return &cobra.Command{
		Use:   "import-validator-key <private-key-file>",
		Short: "Import private key from a private key stored in file (without 0x prefix)",
		RunE: func(cmd *cobra.Command, args []string) error {

			pk, err := ethcrypto.LoadECDSA(args[0])
			if err != nil {
				return err
			}

			bz := ethcrypto.FromECDSA(pk)

			// set private object
			var privKeyObject secp256k1.PrivKey
			copy(privKeyObject[:], bz)

			// node key
			nodeKey := privval.FilePVKey{
				Address: privKeyObject.PubKey().Address(),
				PubKey:  privKeyObject.PubKey(),
				PrivKey: privKeyObject,
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
}

func showPrivateKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show-private-key",
		Short: "Print the account's private key",
		Run: func(cmd *cobra.Command, args []string) {
			// init heimdall config
			helper.InitHeimdallConfig("")

			// get private and public keys
			privKeyObject := helper.GetPrivKey()

			account := &ValidatorAccountFormatter{
				PrivKey: "0x" + hex.EncodeToString(privKeyObject[:]),
			}

			b, err := json.MarshalIndent(account, "", "    ")
			if err != nil {
				panic(err)
			}

			// prints json info
			fmt.Printf("%s", string(b))
		},
	}
}

// VerifyGenesis verifies the genesis file and brings it in sync with on-chain contract
func VerifyGenesis(ctx *server.Context, hApp *app.HeimdallApp) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-genesis",
		Short: "Verify if the genesis matches",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			config := ctx.Config
			config.SetRoot(viper.GetString(cli.HomeFlag))
			helper.InitHeimdallConfig("")

			// Loading genesis doc
			genDoc, err := cmttypes.GenesisDocFromFile(filepath.Join(config.RootDir, "config/genesis.json"))
			if err != nil {
				return err
			}

			// get genesis state
			var genesisState app.GenesisState
			err = json.Unmarshal(genDoc.AppState, &genesisState)
			if err != nil {
				return err
			}

			// TODO HV2 - verify if this is correct to comment and use `hApp.BasicManager.ValidateGenesis` instead
			/*
				// verify genesis
				for _, b := range hApp.ModuleBasics {
					m := b.(hmModule.HeimdallModuleBasic)
					if err := m.VerifyGenesis(genesisState); err != nil {
						return err
					}
				}
			*/
			clientCtx := client.GetClientContextFromCmd(cmd)
			cliCdc := clientCtx.Codec

			return hApp.BasicManager.ValidateGenesis(cliCdc, hApp.GetTxConfig(), genesisState)
		},
	}

	return cmd
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
			return "", errors.New("passphrases do not match")
		}
	}

	return passphrase, nil
}

// Total Validators to be included in the testnet
func getTotalNumberOfNodes() int {
	numValidators := viper.GetInt(flagNumValidators)
	numNonValidators := viper.GetInt(flagNumNonValidators)

	return numNonValidators + numValidators
}

// nodeDir gets the node directory path
func nodeDir(i int) string {
	outDir := viper.GetString(flagOutputDir)
	nodeDirName := fmt.Sprintf("%s%d", viper.GetString(flagNodeDirPrefix), i)
	nodeDaemonHomeName := viper.GetString(flagNodeDaemonHome)

	return filepath.Join(outDir, nodeDirName, nodeDaemonHomeName)
}

// hostnameOrIP returns the hostname of ip of nodes
func hostnameOrIP(i int) string {
	return fmt.Sprintf("%s%d", viper.GetString(flagNodeHostPrefix), i)
}

// populatePersistentPeersInConfigAndWriteIt populates persistent peers in config
func populatePersistentPeersInConfigAndWriteIt(config *cmtcfg.Config) {
	persistentPeers := make([]string, getTotalNumberOfNodes())

	for i := 0; i < getTotalNumberOfNodes(); i++ {
		config.SetRoot(nodeDir(i))

		nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
		if err != nil {
			return
		}

		persistentPeers[i] = p2p.IDAddressString(nodeKey.ID(), fmt.Sprintf("%s:%d", hostnameOrIP(i), 26656))
	}

	persistentPeersList := strings.Join(persistentPeers, ",")

	for i := 0; i < getTotalNumberOfNodes(); i++ {
		config.SetRoot(nodeDir(i))
		config.P2P.PersistentPeers = persistentPeersList
		config.P2P.AddrBookStrict = false

		// overwrite default config
		cmtcfg.WriteConfigFile(filepath.Join(nodeDir(i), "config", "config.toml"), config)
	}
}

// InitializeNodeValidatorFiles initializes node and priv validator files
func InitializeNodeValidatorFiles(
	config *cmtcfg.Config) (nodeID string, valPubKey crypto.PubKey, privKey crypto.PrivKey, err error,
) {
	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		return nodeID, valPubKey, privKey, err
	}

	nodeID = string(nodeKey.ID())

	pvKeyFile := config.PrivValidatorKeyFile()
	if err := cmtos.EnsureDir(filepath.Dir(pvKeyFile), 0777); err != nil {
		return nodeID, valPubKey, privKey, err
	}

	pvStateFile := config.PrivValidatorStateFile()
	if err := cmtos.EnsureDir(filepath.Dir(pvStateFile), 0777); err != nil {
		return nodeID, valPubKey, privKey, err
	}

	FilePv := privval.LoadOrGenFilePV(pvKeyFile, pvStateFile)
	valPubKey, err = FilePv.GetPubKey()
	if err != nil {
		return nodeID, valPubKey, privKey, err
	}

	return nodeID, valPubKey, FilePv.Key.PrivKey, nil
}

// WriteDefaultHeimdallConfig writes default heimdall config to the given path
func WriteDefaultHeimdallConfig(path string, conf helper.CustomConfig) {
	// Don't write if config file in path already exists
	if _, err := os.Stat(path); err == nil {
		logger.Info(fmt.Sprintf("Config file %s already exists. Skip writing default heimdall config.", path))
	} else if errors.Is(err, os.ErrNotExist) {
		helper.WriteConfigFile(path, &conf)
	} else {
		logger.Error("error while checking for config file", "error", err)
	}
}

func createKeyStore(pk *ecdsa.PrivateKey) error {
	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	key := &keystore.Key{
		Id:         id,
		Address:    ethcrypto.PubkeyToAddress(pk.PublicKey),
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

}

// TODO HV2 - check if we need this
/*
func getGenesisAccount(address []byte) authTypes.GenesisAccount {
	acc := authTypes.NewBaseAccountWithAddress(sdk.AccAddress(address))

	genesisBalance, _ := big.NewInt(0).SetString("1000000000000000000000", 10)

	if err := acc.SetCoins(sdk.Coins{sdk.Coin{Denom: authTypes.FeeToken, Amount: math.NewIntFromBigInt(genesisBalance)}}); err != nil {
		logger.Error("getgenesisaccount | setcoins", "error", err)
	}

	result, _ := authTypes.NewGenesisAccountI(&acc)

	return result
}
*/
