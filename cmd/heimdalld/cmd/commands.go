package heimdalld

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cosmossdk.io/log"
	confixcmd "cosmossdk.io/tools/confix/cmd"
	"github.com/0xPolygon/heimdall-v2/app"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/version"
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
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console/prompt"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	// TODO HV2 - uncomment when we have the client package
	// hmTxCli "github.com/0xPolygon/heimdall-v2/client/tx"
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

var ZeroIntString = big.NewInt(0).String()

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
	var privKeyObject secp256k1.PrivKey

	cdc.MustUnmarshal(privKey, &privKeyObject)

	return ValidatorAccountFormatter{
		Address: ethCommon.BytesToAddress(pub.Address().Bytes()).String(),
		PubKey:  CryptoKeyToPubKey(pub).String(),
		PrivKey: "0x" + hex.EncodeToString(privKeyObject[:]),
	}
}

// initCometBFTConfig helps to override default CometBFT Config values.
// It return cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initCometBFTConfig() *cmtcfg.Config {
	return cmtcfg.DefaultConfig()
}

// initAppConfig helps to override default appConfig template and configs.
// It returns "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	return "", nil
}

func initRootCmd(
	rootCmd *cobra.Command,
	txConfig client.TxConfig,
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
	)

	server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport, addModuleInitFlags)

	cometbftCmd := &cobra.Command{
		Use:   "cometbft",
		Short: "CometBFT subcommands",
	}

	cometbftCmd.AddCommand(
		server.ShowNodeIDCmd(),
		server.ShowValidatorCmd(),
		server.ShowAddressCmd(),
		server.VersionCmd(),
	)

	// add keybase, auxiliary RPC, query, genesis, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		genesisCommand(txConfig, basicManager),
		queryCommand(),
		txCommand(),
		keys.Commands(),
		cometbftCmd,
	)

	// add custom commands
	rootCmd.AddCommand(
		generateKeystore(),
		generateValidatorKey(),
		StakeCmd(),
		ApproveCmd(),
		version.Cmd,
	)

	rootCmd.AddCommand(showPrivateKeyCmd())
	// TODO HV2 - uncomment when we have server implemented
	// rootCmd.AddCommand(restServer.ServeCommands(shutdownCtx, cdc, restServer.RegisterRoutes))
	// TODO HV2 - uncomment when we have bridge implemented
	// rootCmd.AddCommand(bridgeCmd.BridgeCommands(viper.GetViper(), logger, "main"))
	rootCmd.AddCommand(VerifyGenesis(ctx, hApp))
	rootCmd.AddCommand(initCmd(ctx, cdc, hApp.BasicManager))
	rootCmd.AddCommand(testnetCmd(ctx, cdc, hApp.BasicManager))

	// rollback cmd
	rootCmd.AddCommand(rollbackCmd(newApp))

	// pruning cmd
	pruning.Cmd(newApp, app.DefaultNodeHome)

	// snapshot cmd
	snapshot.Cmd(newApp)
}

// genesisCommand builds genesis-related `heimdalld genesis` command. Users may provide application specific commands as a parameter
func genesisCommand(txConfig client.TxConfig, basicManager module.BasicManager, cmds ...*cobra.Command) *cobra.Command {
	cmd := genutilcli.Commands(txConfig, basicManager, app.DefaultNodeHome)

	for _, subCmd := range cmds {
		cmd.AddCommand(subCmd)
	}
	return cmd
}

func addModuleInitFlags(startCmd *cobra.Command) {
	crisis.AddModuleInitFlags(startCmd)
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
		Use:   "generate-keystore <private-key>",
		Short: "Generates keystore file using private key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := strings.ReplaceAll(args[0], "0x", "")
			pk, err := ethcrypto.HexToECDSA(s)
			if err != nil {
				return err
			}

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
		},
	}

	return cmd
}

// generateValidatorKey generate validator key
func generateValidatorKey() *cobra.Command {
	cdc := codec.NewLegacyAmino()
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
			var privKeyObject secp256k1.PrivKey
			copy(privKeyObject[:], ds)

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

func showPrivateKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show-privatekey",
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

func openDB(rootDir string) (dbm.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return dbm.NewDB("application", dbm.GoLevelDBBackend, dataDir)
}

func openTraceWriter(traceWriterFile string) (io.Writer, error) {
	if traceWriterFile == "" {
		return nil, nil
	}

	return os.OpenFile(
		traceWriterFile,
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0666,
	)
}

// Total Validators to be included in the testnet
func totalValidators() int {
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
	persistentPeers := make([]string, totalValidators())

	for i := 0; i < totalValidators(); i++ {
		config.SetRoot(nodeDir(i))

		nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
		if err != nil {
			return
		}

		persistentPeers[i] = p2p.IDAddressString(nodeKey.ID(), fmt.Sprintf("%s:%d", hostnameOrIP(i), 26656))
	}

	persistentPeersList := strings.Join(persistentPeers, ",")

	for i := 0; i < totalValidators(); i++ {
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
func WriteDefaultHeimdallConfig(path string, conf helper.Configuration) {
	// Don't write if config file in path already exists
	if _, err := os.Stat(path); err == nil {
		logger.Info(fmt.Sprintf("Config file %s already exists. Skip writing default heimdall config.", path))
	} else if errors.Is(err, os.ErrNotExist) {
		helper.WriteConfigFile(path, &conf)
	} else {
		logger.Error("error while checking for config file", "error", err)
	}
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
