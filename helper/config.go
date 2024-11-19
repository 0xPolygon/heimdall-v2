package helper

import (
	"crypto/ecdsa"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	logger "github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/privval"
	cmTypes "github.com/cometbft/cometbft/types"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/go-amino"

	"github.com/0xPolygon/heimdall-v2/file"
	borgrpc "github.com/0xPolygon/heimdall-v2/x/bor/grpc"
)

const (
	CometBFTNodeFlag       = "node"
	WithHeimdallConfigFlag = "app"
	HomeFlag               = "home"
	FlagClientHome         = "home-client"
	OverwriteGenesisFlag   = "overwrite-genesis"
	RestServerFlag         = "rest-server"
	GRPCServerFlag         = "grpc-server"
	BridgeFlag             = "bridge"
	LogLevel               = "log_level"
	LogsWriterFileFlag     = "logs_writer_file"
	SeedsFlag              = "seeds"

	MainChain   = "mainnet"
	MumbaiChain = "mumbai"
	AmoyChain   = "amoy"
	LocalChain  = "local"

	// heimdall-config flags

	MainRPCUrlFlag = "eth_rpc_url"
	BorRPCUrlFlag  = "bor_rpc_url"
	BorGRPCUrlFlag = "bor_grpc_url"
	BorGRPCFlag    = "bor_grpc_flag"

	CometBFTNodeURLFlag          = "comet_bft_rpc_url"
	HeimdallServerURLFlag        = "heimdall_rest_server"
	GRPCServerURLFlag            = "grpc_server"
	AmqpURLFlag                  = "amqp_url"
	CheckpointerPollIntervalFlag = "checkpoint_poll_interval"
	SyncerPollIntervalFlag       = "syncer_poll_interval"
	NoACKPollIntervalFlag        = "noack_poll_interval"
	ClerkPollIntervalFlag        = "clerk_poll_interval"
	SpanPollIntervalFlag         = "span_poll_interval"
	MilestonePollIntervalFlag    = "milestone_poll_interval"
	MainchainGasLimitFlag        = "main_chain_gas_limit"
	MainchainMaxGasPriceFlag     = "main_chain_max_gas_price"

	NoACKWaitTimeFlag = "no_ack_wait_time"
	ChainFlag         = "chain"

	// TODO HV2 Move these to common client flags

	// BroadcastBlock defines a tx broadcasting mode where the client waits for
	// the tx to be committed in a block.
	BroadcastBlock = "block"

	// BroadcastSync defines a tx broadcasting mode where the client waits for
	// a CheckTx execution response only.
	BroadcastSync = "sync"

	// BroadcastAsync defines a tx broadcasting mode where the client returns
	// immediately.
	BroadcastAsync = "async"
	// --

	// RPC Endpoints
	DefaultMainRPCUrl = "http://localhost:9545"
	DefaultBorRPCUrl  = "http://localhost:8545"
	DefaultBorGRPCUrl = "localhost:3131"

	// RPC Timeouts
	DefaultEthRPCTimeout = 5 * time.Second
	DefaultBorRPCTimeout = 5 * time.Second

	// Services

	// DefaultAmqpURL represents default AMQP url
	DefaultAmqpURL           = "amqp://guest:guest@localhost:5672/"
	DefaultHeimdallServerURL = "http://0.0.0.0:1317"
	DefaultGRPCServerURL     = "http://0.0.0.0:1318"

	DefaultCometBFTNodeURL = "http://0.0.0.0:26657"

	NoACKWaitTime = 1800 * time.Second // Time ack service waits to clear buffer and elect new proposer (1800 seconds ~ 30 mins)

	DefaultCheckpointPollInterval = 5 * time.Minute
	DefaultSyncerPollInterval     = 1 * time.Minute
	DefaultNoACKPollInterval      = 1010 * time.Second
	DefaultClerkPollInterval      = 10 * time.Second
	DefaultSpanPollInterval       = 1 * time.Minute

	DefaultMilestonePollInterval = 30 * time.Second

	DefaultEnableSH              = false
	DefaultSHStateSyncedInterval = 15 * time.Minute
	DefaultSHStakeUpdateInterval = 3 * time.Hour

	DefaultSHMaxDepthDuration = time.Hour

	DefaultMainchainGasLimit = uint64(5000000)

	DefaultMainchainMaxGasPrice = 400000000000 // 400 Gwei

	DefaultBorChainID = "15001"

	DefaultLogsType = "json"
	DefaultChain    = MainChain

	DefaultCometBFTNode = "tcp://localhost:26657"

	// TODO HV2: Check these values and eventually update with the correct ones. Also, add support for amoy.
	DefaultMainnetSeeds       = "1500161dd491b67fb1ac81868952be49e2509c9f@52.78.36.216:26656,dd4a3f1750af5765266231b9d8ac764599921736@3.36.224.80:26656,8ea4f592ad6cc38d7532aff418d1fb97052463af@34.240.245.39:26656,e772e1fb8c3492a9570a377a5eafdb1dc53cd778@54.194.245.5:26656"
	DefaultMumbaiTestnetSeeds = "9df7ae4bf9b996c0e3436ed4cd3050dbc5742a28@43.200.206.40:26656,d9275750bc877b0276c374307f0fd7eae1d71e35@54.216.248.9:26656,1a3258eb2b69b235d4749cf9266a94567d6c0199@52.214.83.78:26656"
	DefaultAmoyTestnetSeeds   = "eb57fffe96d74312963ced94a94cbaf8e0d8ec2e@54.217.171.196:26656,080dcdffcc453367684b61d8f3ce032f357b0f73@13.251.184.185:26656"

	secretFilePerm = 0600

	// MaxStateSyncSize is the new max state sync size after SpanOverrideHeight hardfork
	MaxStateSyncSize = 30000

	// MilestoneLength is minimum supported length of milestone
	MilestoneLength = uint64(12)

	MilestonePruneNumber = uint64(100)

	BorChainMilestoneConfirmation = uint64(16)

	// MilestoneBufferLength defines the condition to propose the
	// milestoneTimeout if this many bor blocks have passed since
	// the last milestone
	MilestoneBufferLength = MilestoneLength * 5
	MilestoneBufferTime   = 256 * time.Second

	// DefaultOpenCollectorEndpoint is the default port of Heimdall open collector endpoint
	DefaultOpenCollectorEndpoint = "localhost:4317"
)

var (
	DefaultCLIHome  = os.ExpandEnv("$HOME/var/lib/heimdall")
	DefaultNodeHome = os.ExpandEnv("$HOME/var/lib/heimdall")
	MinBalance      = big.NewInt(100000000000000000) // aka 0.1 Ether
)

var cdc = amino.NewCodec()

func init() {
	Logger = logger.NewTMLogger(logger.NewSyncWriter(os.Stdout))
}

// CustomConfig represents heimdall config
type CustomConfig struct {
	EthRPCUrl      string `mapstructure:"eth_rpc_url"`       // RPC endpoint for main chain
	BorRPCUrl      string `mapstructure:"bor_rpc_url"`       // RPC endpoint for bor chain
	BorGRPCUrl     string `mapstructure:"bor_grpc_url"`      // gRPC endpoint for bor chain
	BorGRPCFlag    bool   `mapstructure:"bor_grpc_flag"`     // gRPC flag for bor chain
	CometBFTRPCUrl string `mapstructure:"comet_bft_rpc_url"` // cometbft node url
	SubGraphUrl    string `mapstructure:"sub_graph_url"`     // sub graph url

	EthRPCTimeout time.Duration `mapstructure:"eth_rpc_timeout"` // timeout for eth rpc
	BorRPCTimeout time.Duration `mapstructure:"bor_rpc_timeout"` // timeout for bor rpc

	AmqpURL           string `mapstructure:"amqp_url"`             // amqp url
	HeimdallServerURL string `mapstructure:"heimdall_rest_server"` // heimdall server url
	GRPCServerURL     string `mapstructure:"grpc_server_url"`      // grpc server url

	MainchainGasLimit uint64 `mapstructure:"main_chain_gas_limit"` // gas limit to mainchain transaction. eg....submit checkpoint.

	MainchainMaxGasPrice int64 `mapstructure:"main_chain_max_gas_price"` // max gas price to mainchain transaction. eg....submit checkpoint.

	// config related to bridge
	CheckpointPollInterval time.Duration `mapstructure:"checkpoint_poll_interval"` // Poll interval for checkpointer service to send new checkpoints or missing ACK
	SyncerPollInterval     time.Duration `mapstructure:"syncer_poll_interval"`     // Poll interval for syncer service to sync for changes on main chain
	NoACKPollInterval      time.Duration `mapstructure:"noack_poll_interval"`      // Poll interval for ack service to send no-ack in case of no checkpoints
	ClerkPollInterval      time.Duration `mapstructure:"clerk_poll_interval"`
	SpanPollInterval       time.Duration `mapstructure:"span_poll_interval"`
	MilestonePollInterval  time.Duration `mapstructure:"milestone_poll_interval"`
	EnableSH               bool          `mapstructure:"enable_self_heal"`         // Enable self-healing
	SHStateSyncedInterval  time.Duration `mapstructure:"sh_state_synced_interval"` // Interval to self-heal StateSynced events if missing
	SHStakeUpdateInterval  time.Duration `mapstructure:"sh_stake_update_interval"` // Interval to self-heal StakeUpdate events if missing
	SHMaxDepthDuration     time.Duration `mapstructure:"sh_max_depth_duration"`    // Max duration that allows to suggest self-healing is not needed

	// wait time related options
	NoACKWaitTime time.Duration `mapstructure:"no_ack_wait_time"` // Time ack service waits to clear buffer and elect new proposer

	// Log related options
	LogsType       string `mapstructure:"logs_type"`        // if true, enable logging in json format
	LogsWriterFile string `mapstructure:"logs_writer_file"` // if given, Logs will be written to this file else os.Stdout

	Chain string `mapstructure:"chain"`
}

type CustomAppConfig struct {
	serverconfig.Config `mapstructure:",squash"`
	Custom              CustomConfig `mapstructure:"custom"`
}

var conf CustomAppConfig

// MainChainClient stores eth clie nt for Main chain Network
var mainChainClient *ethclient.Client
var mainRPCClient *rpc.Client

// borClient stores eth/rpc client for Polygon Pos Network
var borClient *ethclient.Client
var borRPCClient *rpc.Client
var borGRPCClient *borgrpc.BorGRPCClient

// private key object
var privKeyObject secp256k1.PrivKey

var pubKeyObject secp256k1.PubKey

// Logger stores global logger object
var Logger logger.Logger

// GenesisDoc contains the genesis file
var GenesisDoc cmTypes.GenesisDoc

var milestoneBorBlockHeight uint64 = 0

type ChainManagerAddressMigration struct {
	PolTokenAddress       string
	RootChainAddress      string
	StakingManagerAddress string
	SlashManagerAddress   string
	StakingInfoAddress    string
	StateSenderAddress    string
}

var chainManagerAddressMigrations = map[string]map[int64]ChainManagerAddressMigration{
	MainChain:   {},
	MumbaiChain: {},
	AmoyChain:   {},
	"default":   {},
}

// InitHeimdallConfig initializes with viper config (from heimdall configuration)
func InitHeimdallConfig(homeDir string) {
	if strings.Compare(homeDir, "") == 0 {
		// get home dir from viper
		homeDir = viper.GetString(HomeFlag)
	}

	// get heimdall config filepath from viper/cobra flag
	heimdallConfigFileFromFlag := viper.GetString(WithHeimdallConfigFlag)

	// init heimdall with changed config files
	InitHeimdallConfigWith(homeDir, heimdallConfigFileFromFlag)
}

// InitHeimdallConfigWith initializes passed heimdall/tendermint config files
func InitHeimdallConfigWith(homeDir string, heimdallConfigFileFromFlag string) {
	if strings.Compare(homeDir, "") == 0 {
		Logger.Error("home directory is mentioned")
		return
	}

	// read configuration from the standard configuration file
	configDir := filepath.Join(homeDir, "config")
	heimdallViper := viper.New()
	heimdallViper.SetEnvPrefix("HEIMDALL")
	heimdallViper.AutomaticEnv()

	if heimdallConfigFileFromFlag == "" {
		heimdallViper.SetConfigName("app")     // name of config file (without extension)
		heimdallViper.AddConfigPath(configDir) // call multiple times to add many search paths
	} else {
		heimdallViper.SetConfigFile(heimdallConfigFileFromFlag) // set config file explicitly
	}

	// Handle errors reading the config file
	if err := heimdallViper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	// unmarshal configuration from the standard configuration file
	if err := heimdallViper.UnmarshalExact(&conf); err != nil {
		log.Fatalln("unable to unmarshall config", "Error", err)
	}

	//  if there is a file with overrides submitted via flags => read it and merge it with the already read standard configuration
	if heimdallConfigFileFromFlag != "" {
		heimdallViperFromFlag := viper.New()
		heimdallViperFromFlag.SetConfigFile(heimdallConfigFileFromFlag) // set flag config file explicitly

		err := heimdallViperFromFlag.ReadInConfig()
		if err != nil { // Handle errors reading the config file sybmitted as a flag
			log.Fatalln("unable to read config file submitted via flag", "Error", err)
		}

		var confFromFlag CustomConfig
		// unmarshal configuration from the configuration file submitted as a flag
		if err = heimdallViperFromFlag.UnmarshalExact(&confFromFlag); err != nil {
			log.Fatalln("unable to unmarshall config file submitted via flag", "Error", err)
		}

		conf.Merge(&confFromFlag)
	}

	// update configuration data with submitted flags
	if err := conf.UpdateWithFlags(viper.GetViper(), Logger); err != nil {
		log.Fatalln("unable to read flag values. Check log for details.", "Error", err)
	}

	// perform check for json logging
	if conf.Custom.LogsType == "json" {
		Logger = logger.NewTMJSONLogger(logger.NewSyncWriter(GetLogsWriter(conf.Custom.LogsWriterFile)))
	} else {
		// default fallback
		Logger = logger.NewTMLogger(logger.NewSyncWriter(GetLogsWriter(conf.Custom.LogsWriterFile)))
	}

	// perform checks for timeout
	if conf.Custom.EthRPCTimeout == 0 {
		// fallback to default
		Logger.Debug("Missing ETH RPC timeout or invalid value provided, falling back to default", "timeout", DefaultEthRPCTimeout)
		conf.Custom.EthRPCTimeout = DefaultEthRPCTimeout
	}

	if conf.Custom.BorRPCTimeout == 0 {
		// fallback to default
		Logger.Debug("Missing BOR RPC timeout or invalid value provided, falling back to default", "timeout", DefaultBorRPCTimeout)
		conf.Custom.BorRPCTimeout = DefaultBorRPCTimeout
	}

	if conf.Custom.SHStateSyncedInterval == 0 {
		// fallback to default
		Logger.Debug("Missing self-healing StateSynced interval or invalid value provided, falling back to default", "interval", DefaultSHStateSyncedInterval)
		conf.Custom.SHStateSyncedInterval = DefaultSHStateSyncedInterval
	}

	if conf.Custom.SHStakeUpdateInterval == 0 {
		// fallback to default
		Logger.Debug("Missing self-healing StakeUpdate interval or invalid value provided, falling back to default", "interval", DefaultSHStakeUpdateInterval)
		conf.Custom.SHStakeUpdateInterval = DefaultSHStakeUpdateInterval
	}

	if conf.Custom.SHMaxDepthDuration == 0 {
		// fallback to default
		Logger.Debug("Missing self-healing max depth duration or invalid value provided, falling back to default", "duration", DefaultSHMaxDepthDuration)
		conf.Custom.SHMaxDepthDuration = DefaultSHMaxDepthDuration
	}

	var err error
	if mainRPCClient, err = rpc.Dial(conf.Custom.EthRPCUrl); err != nil {
		log.Fatalln("Unable to dial via ethClient", "URL", conf.Custom.EthRPCUrl, "chain", "eth", "error", err)
	}

	mainChainClient = ethclient.NewClient(mainRPCClient)

	if borRPCClient, err = rpc.Dial(conf.Custom.BorRPCUrl); err != nil {
		log.Fatal(err)
	}

	borClient = ethclient.NewClient(borRPCClient)

	borGRPCClient = borgrpc.NewBorGRPCClient(conf.Custom.BorGRPCUrl)

	// TODO HV2 - Why was this added? We are never using this
	/*
		// Loading genesis doc
		genDoc, err := cmTypes.GenesisDocFromFile(filepath.Join(configDir, "genesis.json"))
		if err != nil {
			log.Fatal(err)
		}

		GenesisDoc = *genDoc
	*/

	// load pv file, unmarshall and set to privKeyObject
	err = file.PermCheck(file.Rootify("priv_validator_key.json", configDir), secretFilePerm)
	if err != nil {
		Logger.Error(err.Error())
	}

	privVal := privval.LoadFilePV(filepath.Join(configDir, "priv_validator_key.json"), filepath.Join(configDir, "priv_validator_key.json"))
	fmt.Println(privVal)
	privKeyObject = privVal.Key.PrivKey.Bytes()
	pubKeyObject = privVal.Key.PubKey.Bytes()

	// TODO HV2 - seems incomplete! Why?
	switch conf.Custom.Chain {
	case MainChain, MumbaiChain, AmoyChain:
	default:

	}
}

// GetDefaultHeimdallConfig returns configuration with default params
func GetDefaultHeimdallConfig() CustomConfig {
	return CustomConfig{
		EthRPCUrl:  DefaultMainRPCUrl,
		BorRPCUrl:  DefaultBorRPCUrl,
		BorGRPCUrl: DefaultBorGRPCUrl,

		CometBFTRPCUrl: DefaultCometBFTNodeURL,

		EthRPCTimeout: DefaultEthRPCTimeout,
		BorRPCTimeout: DefaultBorRPCTimeout,

		AmqpURL:           DefaultAmqpURL,
		HeimdallServerURL: DefaultHeimdallServerURL,
		GRPCServerURL:     DefaultGRPCServerURL,

		MainchainGasLimit: DefaultMainchainGasLimit,

		MainchainMaxGasPrice: DefaultMainchainMaxGasPrice,

		CheckpointPollInterval: DefaultCheckpointPollInterval,
		SyncerPollInterval:     DefaultSyncerPollInterval,
		NoACKPollInterval:      DefaultNoACKPollInterval,
		ClerkPollInterval:      DefaultClerkPollInterval,
		SpanPollInterval:       DefaultSpanPollInterval,
		MilestonePollInterval:  DefaultMilestonePollInterval,
		EnableSH:               DefaultEnableSH,
		SHStateSyncedInterval:  DefaultSHStateSyncedInterval,
		SHStakeUpdateInterval:  DefaultSHStakeUpdateInterval,
		SHMaxDepthDuration:     DefaultSHMaxDepthDuration,

		NoACKWaitTime: NoACKWaitTime,

		LogsType:       DefaultLogsType,
		Chain:          DefaultChain,
		LogsWriterFile: "", // default to stdout
	}
}

// GetConfig returns cached configuration object
func GetConfig() CustomConfig {
	return conf.Custom
}

func GetGenesisDoc() cmTypes.GenesisDoc {
	return GenesisDoc
}

//
// Get main/pos clients
//

// GetMainChainRPCClient returns main chain RPC client
func GetMainChainRPCClient() *rpc.Client {
	return mainRPCClient
}

// GetMainClient returns main chain's eth client
func GetMainClient() *ethclient.Client {
	return mainChainClient
}

// GetBorClient returns bor eth client
func GetBorClient() *ethclient.Client {
	return borClient
}

// GetBorRPCClient returns bor RPC client
func GetBorRPCClient() *rpc.Client {
	return borRPCClient
}

// GetPrivKey returns priv key object
func GetPrivKey() secp256k1.PrivKey {
	return privKeyObject
}

// GetECDSAPrivKey return ecdsa private key
func GetECDSAPrivKey() *ecdsa.PrivateKey {
	// get priv key
	pkObject := GetPrivKey()

	// create ecdsa private key
	ecdsaPrivateKey, _ := ethCrypto.ToECDSA(pkObject[:])

	return ecdsaPrivateKey
}

// GetPubKey returns pub key object
func GetPubKey() secp256k1.PubKey {
	return pubKeyObject
}

// GetAddress returns address object
func GetAddress() []byte {
	return GetPubKey().Address()
}

// GetValidChains returns all the valid chains
func GetValidChains() []string {
	return []string{"mainnet", "mumbai", "amoy", "local"}
}

// GetMilestoneBorBlockHeight returns milestoneBorBlockHeight
func GetMilestoneBorBlockHeight() uint64 {
	return milestoneBorBlockHeight
}

func GetChainManagerAddressMigration(blockNum int64) (ChainManagerAddressMigration, bool) {
	chainMigration := chainManagerAddressMigrations[conf.Custom.Chain]
	if chainMigration == nil {
		chainMigration = chainManagerAddressMigrations["default"]
	}

	result, found := chainMigration[blockNum]

	return result, found
}

// DecorateWithHeimdallFlags adds persistent flags for heimdall-config and bind flags with command
func DecorateWithHeimdallFlags(cmd *cobra.Command, v *viper.Viper, loggerInstance logger.Logger, caller string) {
	// add with-heimdall-config flag
	cmd.PersistentFlags().String(
		WithHeimdallConfigFlag,
		"",
		"Override of Heimdall config file (default <home>/config/config.json)",
	)

	if err := v.BindPFlag(WithHeimdallConfigFlag, cmd.PersistentFlags().Lookup(WithHeimdallConfigFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, WithHeimdallConfigFlag), "Error", err)
	}

	// add MainRPCUrlFlag flag
	cmd.PersistentFlags().String(
		MainRPCUrlFlag,
		"",
		"Set RPC endpoint for ethereum chain",
	)

	if err := v.BindPFlag(MainRPCUrlFlag, cmd.PersistentFlags().Lookup(MainRPCUrlFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, MainRPCUrlFlag), "Error", err)
	}

	// add BorRPCUrlFlag flag
	cmd.PersistentFlags().String(
		BorRPCUrlFlag,
		"",
		"Set RPC endpoint for bor chain",
	)

	if err := v.BindPFlag(BorRPCUrlFlag, cmd.PersistentFlags().Lookup(BorRPCUrlFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, BorRPCUrlFlag), "Error", err)
	}

	// add BorGRPCUrlFlag flag
	cmd.PersistentFlags().String(
		BorGRPCUrlFlag,
		"",
		"Set gRPC endpoint for bor chain",
	)

	if err := v.BindPFlag(BorGRPCUrlFlag, cmd.PersistentFlags().Lookup(BorGRPCUrlFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, BorGRPCUrlFlag), "Error", err)
	}

	// add CometBFTNodeURLFlag flag
	cmd.PersistentFlags().String(
		CometBFTNodeURLFlag,
		"",
		"Set RPC endpoint for CometBFT",
	)

	if err := v.BindPFlag(CometBFTNodeURLFlag, cmd.PersistentFlags().Lookup(CometBFTNodeURLFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, CometBFTNodeURLFlag), "Error", err)
	}

	// add HeimdallServerURLFlag flag
	cmd.PersistentFlags().String(
		HeimdallServerURLFlag,
		"",
		"Set Heimdall REST server endpoint",
	)

	if err := v.BindPFlag(HeimdallServerURLFlag, cmd.PersistentFlags().Lookup(HeimdallServerURLFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, HeimdallServerURLFlag), "Error", err)
	}

	// add GRPCServerURL flag
	cmd.PersistentFlags().String(
		GRPCServerURLFlag,
		"",
		"Set GRPC Server Endpoint",
	)

	if err := v.BindPFlag(GRPCServerURLFlag, cmd.PersistentFlags().Lookup(GRPCServerURLFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, GRPCServerURLFlag), "Error", err)
	}

	// add AmqpURLFlag flag
	cmd.PersistentFlags().String(
		AmqpURLFlag,
		"",
		"Set AMQP endpoint",
	)

	if err := v.BindPFlag(AmqpURLFlag, cmd.PersistentFlags().Lookup(AmqpURLFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, AmqpURLFlag), "Error", err)
	}

	// add CheckpointerPollIntervalFlag flag
	cmd.PersistentFlags().String(
		CheckpointerPollIntervalFlag,
		"",
		"Set check point pull interval",
	)

	if err := v.BindPFlag(CheckpointerPollIntervalFlag, cmd.PersistentFlags().Lookup(CheckpointerPollIntervalFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, CheckpointerPollIntervalFlag), "Error", err)
	}

	// add SyncerPollIntervalFlag flag
	cmd.PersistentFlags().String(
		SyncerPollIntervalFlag,
		"",
		"Set syncer pull interval",
	)

	if err := v.BindPFlag(SyncerPollIntervalFlag, cmd.PersistentFlags().Lookup(SyncerPollIntervalFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, SyncerPollIntervalFlag), "Error", err)
	}

	// add NoACKPollIntervalFlag flag
	cmd.PersistentFlags().String(
		NoACKPollIntervalFlag,
		"",
		"Set no acknowledge pull interval",
	)

	if err := v.BindPFlag(NoACKPollIntervalFlag, cmd.PersistentFlags().Lookup(NoACKPollIntervalFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, NoACKPollIntervalFlag), "Error", err)
	}

	// add ClerkPollIntervalFlag flag
	cmd.PersistentFlags().String(
		ClerkPollIntervalFlag,
		"",
		"Set clerk pull interval",
	)

	if err := v.BindPFlag(ClerkPollIntervalFlag, cmd.PersistentFlags().Lookup(ClerkPollIntervalFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, ClerkPollIntervalFlag), "Error", err)
	}

	// add SpanPollIntervalFlag flag
	cmd.PersistentFlags().String(
		SpanPollIntervalFlag,
		"",
		"Set span pull interval",
	)

	if err := v.BindPFlag(SpanPollIntervalFlag, cmd.PersistentFlags().Lookup(SpanPollIntervalFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, SpanPollIntervalFlag), "Error", err)
	}

	// add MilestonePollIntervalFlag flag
	cmd.PersistentFlags().String(
		MilestonePollIntervalFlag,
		DefaultMilestonePollInterval.String(),
		"Set milestone interval",
	)

	if err := v.BindPFlag(MilestonePollIntervalFlag, cmd.PersistentFlags().Lookup(MilestonePollIntervalFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, MilestonePollIntervalFlag), "Error", err)
	}

	// add MainchainGasLimitFlag flag
	cmd.PersistentFlags().Uint64(
		MainchainGasLimitFlag,
		0,
		"Set main chain gas limit",
	)

	if err := v.BindPFlag(MainchainGasLimitFlag, cmd.PersistentFlags().Lookup(MainchainGasLimitFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, MainchainGasLimitFlag), "Error", err)
	}

	// add MainchainMaxGasPriceFlag flag
	cmd.PersistentFlags().Int64(
		MainchainMaxGasPriceFlag,
		0,
		"Set main chain max gas limit",
	)

	if err := v.BindPFlag(MainchainMaxGasPriceFlag, cmd.PersistentFlags().Lookup(MainchainMaxGasPriceFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, MainchainMaxGasPriceFlag), "Error", err)
	}

	// add NoACKWaitTimeFlag flag
	cmd.PersistentFlags().String(
		NoACKWaitTimeFlag,
		"",
		"Set time ack service waits to clear buffer and elect new proposer",
	)

	if err := v.BindPFlag(NoACKWaitTimeFlag, cmd.PersistentFlags().Lookup(NoACKWaitTimeFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, NoACKWaitTimeFlag), "Error", err)
	}

	// add chain flag
	cmd.PersistentFlags().String(
		ChainFlag,
		"",
		fmt.Sprintf("Set one of the chains: [%s]", strings.Join(GetValidChains(), ",")),
	)

	if err := v.BindPFlag(ChainFlag, cmd.PersistentFlags().Lookup(ChainFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, ChainFlag), "Error", err)
	}

	// add logsWriterFile flag
	cmd.PersistentFlags().String(
		LogsWriterFileFlag,
		"",
		"Set logs writer file, Default is os.Stdout",
	)

	if err := v.BindPFlag(LogsWriterFileFlag, cmd.PersistentFlags().Lookup(LogsWriterFileFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", caller, LogsWriterFileFlag), "Error", err)
	}
}

func (c *CustomAppConfig) UpdateWithFlags(v *viper.Viper, loggerInstance logger.Logger) error {
	const logErrMsg = "Unable to read flag."

	// get endpoint for ethereum chain from viper/cobra
	stringConfgValue := v.GetString(MainRPCUrlFlag)
	if stringConfgValue != "" {
		c.Custom.EthRPCUrl = stringConfgValue
	}

	// get endpoint for bor chain from viper/cobra
	stringConfgValue = v.GetString(BorRPCUrlFlag)
	if stringConfgValue != "" {
		c.Custom.BorRPCUrl = stringConfgValue
	}

	// get endpoint for bor chain from viper/cobra
	stringConfgValue = v.GetString(BorGRPCUrlFlag)
	if stringConfgValue != "" {
		c.Custom.BorGRPCUrl = stringConfgValue
	}

	// get gRPC flag for bor chain from viper/cobra
	boolConfgValue := v.GetBool(BorGRPCFlag)
	if boolConfgValue {
		c.Custom.BorGRPCFlag = boolConfgValue
	}

	// get endpoint for cometBFT from viper/cobra
	stringConfgValue = v.GetString(CometBFTNodeURLFlag)
	if stringConfgValue != "" {
		c.Custom.CometBFTRPCUrl = stringConfgValue
	}

	// get endpoint for CometBFT from viper/cobra
	stringConfgValue = v.GetString(AmqpURLFlag)
	if stringConfgValue != "" {
		c.Custom.AmqpURL = stringConfgValue
	}

	// get Heimdall REST server endpoint from viper/cobra
	stringConfgValue = v.GetString(HeimdallServerURLFlag)
	if stringConfgValue != "" {
		c.Custom.HeimdallServerURL = stringConfgValue
	}

	// get Heimdall GRPC server endpoint from viper/cobra
	stringConfgValue = v.GetString(GRPCServerURLFlag)
	if stringConfgValue != "" {
		c.Custom.GRPCServerURL = stringConfgValue
	}

	// need this error for parsing Duration values
	var err error

	// get check point pull interval from viper/cobra
	stringConfgValue = v.GetString(CheckpointerPollIntervalFlag)
	if stringConfgValue != "" {
		if c.Custom.CheckpointPollInterval, err = time.ParseDuration(stringConfgValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", CheckpointerPollIntervalFlag, "Error", err)
			return err
		}
	}

	// get syncer pull interval from viper/cobra
	stringConfgValue = v.GetString(SyncerPollIntervalFlag)
	if stringConfgValue != "" {
		if c.Custom.SyncerPollInterval, err = time.ParseDuration(stringConfgValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", SyncerPollIntervalFlag, "Error", err)
			return err
		}
	}

	// get poll interval for ack service to send no-ack in case of no checkpoints from viper/cobra
	stringConfgValue = v.GetString(NoACKPollIntervalFlag)
	if stringConfgValue != "" {
		if c.Custom.NoACKPollInterval, err = time.ParseDuration(stringConfgValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", NoACKPollIntervalFlag, "Error", err)
			return err
		}
	}

	// get clerk poll interval from viper/cobra
	stringConfgValue = v.GetString(ClerkPollIntervalFlag)
	if stringConfgValue != "" {
		if c.Custom.ClerkPollInterval, err = time.ParseDuration(stringConfgValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", ClerkPollIntervalFlag, "Error", err)
			return err
		}
	}

	// get span poll interval from viper/cobra
	stringConfgValue = v.GetString(SpanPollIntervalFlag)
	if stringConfgValue != "" {
		if c.Custom.SpanPollInterval, err = time.ParseDuration(stringConfgValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", SpanPollIntervalFlag, "Error", err)
			return err
		}
	}

	// get milestone poll interval from viper/cobra
	stringConfgValue = v.GetString(MilestonePollIntervalFlag)
	if stringConfgValue != "" {
		if c.Custom.MilestonePollInterval, err = time.ParseDuration(stringConfgValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", MilestonePollIntervalFlag, "Error", err)
			return err
		}
	}

	// get time that ack service waits to clear buffer and elect new proposer from viper/cobra
	stringConfgValue = v.GetString(NoACKWaitTimeFlag)
	if stringConfgValue != "" {
		if c.Custom.NoACKWaitTime, err = time.ParseDuration(stringConfgValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", NoACKWaitTimeFlag, "Error", err)
			return err
		}
	}

	// get mainchain gas limit from viper/cobra
	uint64ConfgValue := v.GetUint64(MainchainGasLimitFlag)
	if uint64ConfgValue != 0 {
		c.Custom.MainchainGasLimit = uint64ConfgValue
	}

	// get mainchain max gas price from viper/cobra. if it is greater than  zero => set it as configuration parameter
	int64ConfgValue := v.GetInt64(MainchainMaxGasPriceFlag)
	if int64ConfgValue > 0 {
		c.Custom.MainchainMaxGasPrice = int64ConfgValue
	}

	// get chain from viper/cobra flag
	stringConfgValue = v.GetString(ChainFlag)
	if stringConfgValue != "" {
		c.Custom.Chain = stringConfgValue
	}

	stringConfgValue = v.GetString(LogsWriterFileFlag)
	if stringConfgValue != "" {
		c.Custom.LogsWriterFile = stringConfgValue
	}

	return nil
}

func (c *CustomAppConfig) Merge(cc *CustomConfig) {
	if cc.EthRPCUrl != "" {
		c.Custom.EthRPCUrl = cc.EthRPCUrl
	}

	if cc.BorRPCUrl != "" {
		c.Custom.BorRPCUrl = cc.BorRPCUrl
	}

	if cc.BorGRPCUrl != "" {
		c.Custom.BorGRPCUrl = cc.BorGRPCUrl
	}

	if cc.CometBFTRPCUrl != "" {
		c.Custom.CometBFTRPCUrl = cc.CometBFTRPCUrl
	}

	if cc.AmqpURL != "" {
		c.Custom.AmqpURL = cc.AmqpURL
	}

	if cc.HeimdallServerURL != "" {
		c.Custom.HeimdallServerURL = cc.HeimdallServerURL
	}

	if cc.GRPCServerURL != "" {
		c.Custom.GRPCServerURL = cc.GRPCServerURL
	}

	if cc.MainchainGasLimit != 0 {
		c.Custom.MainchainGasLimit = cc.MainchainGasLimit
	}

	if cc.MainchainMaxGasPrice != 0 {
		c.Custom.MainchainMaxGasPrice = cc.MainchainMaxGasPrice
	}

	if cc.CheckpointPollInterval != 0 {
		c.Custom.CheckpointPollInterval = cc.CheckpointPollInterval
	}

	if cc.SyncerPollInterval != 0 {
		c.Custom.SyncerPollInterval = cc.SyncerPollInterval
	}

	if cc.NoACKPollInterval != 0 {
		c.Custom.NoACKPollInterval = cc.NoACKPollInterval
	}

	if cc.ClerkPollInterval != 0 {
		c.Custom.ClerkPollInterval = cc.ClerkPollInterval
	}

	if cc.SpanPollInterval != 0 {
		c.Custom.SpanPollInterval = cc.SpanPollInterval
	}

	if cc.MilestonePollInterval != 0 {
		c.Custom.MilestonePollInterval = cc.MilestonePollInterval
	}

	if cc.NoACKWaitTime != 0 {
		c.Custom.NoACKWaitTime = cc.NoACKWaitTime
	}

	if cc.Chain != "" {
		c.Custom.Chain = cc.Chain
	}

	if cc.LogsWriterFile != "" {
		c.Custom.LogsWriterFile = cc.LogsWriterFile
	}
}

// DecorateWithCometBFTFlags creates cometBFT flags for desired command and bind them to viper
func DecorateWithCometBFTFlags(cmd *cobra.Command, v *viper.Viper, loggerInstance logger.Logger, message string) {
	// add seeds flag
	cmd.PersistentFlags().String(
		SeedsFlag,
		"",
		"Override seeds",
	)

	if err := v.BindPFlag(SeedsFlag, cmd.PersistentFlags().Lookup(SeedsFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf("%v | BindPFlag | %v", message, SeedsFlag), "Error", err)
	}
}

// UpdateCometBFTConfig updates cometBFT config with flags and default values if needed
func UpdateCometBFTConfig(cometBFTConfig *cfg.Config, v *viper.Viper) {
	// update cometBFTConfig.P2P.Seeds
	seedsFlagValue := v.GetString(SeedsFlag)
	if seedsFlagValue != "" {
		cometBFTConfig.P2P.Seeds = seedsFlagValue
	}

	if cometBFTConfig.P2P.Seeds == "" {
		switch conf.Custom.Chain {
		case MainChain:
			cometBFTConfig.P2P.Seeds = DefaultMainnetSeeds
		case MumbaiChain:
			cometBFTConfig.P2P.Seeds = DefaultMumbaiTestnetSeeds
		case AmoyChain:
			cometBFTConfig.P2P.Seeds = DefaultAmoyTestnetSeeds
		}
	}
}

func GetLogsWriter(logsWriterFile string) io.Writer {
	if logsWriterFile != "" {
		logWriter, err := os.OpenFile(logsWriterFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening log writer file: %v", err)
		}

		return logWriter
	} else {
		return os.Stdout
	}
}

// GetBorGRPCClient returns bor gRPC client
func GetBorGRPCClient() *borgrpc.BorGRPCClient {
	return borGRPCClient
}

// SetTestConfig sets test configuration
func SetTestConfig(_conf CustomConfig) {
	conf.Custom = _conf
}

// TEST PURPOSE ONLY

// SetTestPrivPubKey sets test priv and pub key for testing
func SetTestPrivPubKey(privKey secp256k1.PrivKey) {
	privKeyObject = privKey
	privKeyObject.PubKey()
	pubKey, ok := privKeyObject.PubKey().(secp256k1.PubKey)
	if !ok {
		panic("pub key is not of type secp256k1.PrivKey")
	}
	pubKeyObject = pubKey
}
