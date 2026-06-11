package helper

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	logger "cosmossdk.io/log"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/privval"
	"github.com/cosmos/cosmos-sdk/client/flags"
	addressCodec "github.com/cosmos/cosmos-sdk/codec/address"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/0xPolygon/heimdall-v2/file"
	borgrpc "github.com/0xPolygon/heimdall-v2/x/bor/grpc"
)

const (
	CometBFTNodeFlag       = "node"
	WithHeimdallConfigFlag = "app"
	RestServerFlag         = "rest-server"
	BridgeFlag             = "bridge"
	AllProcessesFlag       = "all"
	OnlyProcessesFlag      = "only"
	LogsWriterFileFlag     = "logs_writer_file"
	SeedsFlag              = "seeds"

	MainChain   = "mainnet"
	MumbaiChain = "mumbai"
	AmoyChain   = "amoy"

	// app config flags

	MainRPCUrlFlag   = "eth_rpc_url"
	BorRPCUrlFlag    = "bor_rpc_url"
	BorGRPCUrlFlag   = "bor_grpc_url"
	BorGRPCFlagFlag  = "bor_grpc_flag"
	BorGRPCTokenFlag = "bor_grpc_token" // #nosec G101 -- config key name, not a credential value

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

	MainChainGasFeeCapFlag = "main_chain_gas_fee_cap"
	MainChainGasTipCapFlag = "main_chain_gas_tip_cap"

	NoACKWaitTimeFlag = "no_ack_wait_time"
	ChainFlag         = "chain"
	ProducerVotesFlag = "producer_votes"

	DefaultMainRPCUrl  = "http://localhost:9545"
	DefaultBorRPCUrl   = "http://localhost:8545"
	DefaultBorGRPCUrl  = "localhost:3131"
	DefaultBorGRPCFlag = false

	DefaultEthRPCTimeout = 5 * time.Second
	DefaultBorRPCTimeout = 1 * time.Second

	// DefaultAmqpURL represents default AMQP url
	DefaultAmqpURL = "amqp://guest:guest@localhost:5672/" //nolint:gosec // G101: well-known RabbitMQ default credentials for local development

	DefaultHeimdallServerURL = "tcp://0.0.0.0:1317"

	DefaultCometBFTNodeURL = "http://0.0.0.0:26657"

	NoACKWaitTime = 1800 * time.Second // Time ack service waits to clear the buffer and elect the new proposer (1800 seconds ~ 30 min)

	DefaultCheckpointPollInterval = 5 * time.Minute
	DefaultSyncerPollInterval     = 1 * time.Minute
	DefaultNoACKPollInterval      = 1010 * time.Second
	DefaultClerkPollInterval      = 10 * time.Second
	DefaultSpanPollInterval       = 1 * time.Minute

	DefaultMilestonePollInterval = 30 * time.Second

	// LogTimestampFormat is the millisecond-precision timestamp layout used for
	// all heimdall log output. Matches bor's log format for consistent
	// cross-service log analysis.
	LogTimestampFormat = "2006-01-02T15:04:05.000Z07:00"

	// Self-healing defaults
	DefaultEnableSH                = false
	DefaultSHStateSyncedInterval   = 3 * time.Hour
	DefaultSHStakeUpdateInterval   = 3 * time.Hour
	DefaultSHCheckpointAckInterval = 30 * time.Minute
	DefaultSHMaxDepthDuration      = 24 * time.Hour

	DefaultMainChainGasFeeCap = 500000000000 // 500 Gwei
	DefaultMainChainGasTipCap = 10000000000  // 10 Gwei

	DefaultBorChainID      = "15001"
	DefaultHeimdallChainID = "heimdall-15001"

	DefaultLogsType = "json"
	DefaultChain    = MainChain

	DefaultMainnetSeeds     = "e019e16d4e376723f3adc58eb1761809fea9bee0@35.234.150.253:26656,7f3049e88ac7f820fd86d9120506aaec0dc54b27@34.89.75.187:26656,1f5aff3b4f3193404423c3dd1797ce60cd9fea43@34.142.43.249:26656,2d5484feef4257e56ece025633a6ea132d8cadca@35.246.99.203:26656,17e9efcbd173e81a31579310c502e8cdd8b8ff2e@35.197.233.240:26656,72a83490309f9f63fdca3a0bef16c290e5cbb09c@35.246.95.65:26656,00677b1b2c6282fb060b7bb6e9cc7d2d05cdd599@34.105.180.11:26656,721dd4cebfc4b78760c7ee5d7b1b44d29a0aa854@34.147.169.102:26656,4760b3fc04648522a0bcb2d96a10aadee141ee89@34.89.55.74:26656"
	DefaultAmoyTestnetSeeds = "e4eabef3111155890156221f018b0ea3b8b64820@35.197.249.21:26656,811c3127677a4a34df907b021aad0c9d22f84bf4@34.89.39.114:26656,2ec15d1d33261e8cf42f57236fa93cfdc21c1cfb@35.242.167.175:26656,38120f9d2c003071a7230788da1e3129b6fb9d3f@34.89.15.223:26656,2f16f3857c6c99cc11e493c2082b744b8f36b127@34.105.128.110:26656,2833f06a5e33da2e80541fb1bfde2a7229877fcb@34.89.21.99:26656,2e6f1342416c5d758f5ae32f388bb76f7712a317@34.89.101.16:26656,a596f98b41851993c24de00a28b767c7c5ff8b42@34.89.11.233:26656"

	DefaultMainnetProducers = "91,92,93,94"

	DefaultAmoyTestnetProducers = "4,5"

	DefaultMumbaiTestnetProducers = "1,2,3"

	DefaultLocalTestnetProducers = "1,2,3,4"

	secretFilePerm = 0o600

	// MaxStateSyncSize is the new max state sync size after SpanOverrideHeight hard fork
	MaxStateSyncSize = 30000

	EnforcedMinRetainBlocks = 2000000

	privValJsonFile = "priv_validator_key.json"

	bindPFlagLog = "%v | BindPFlag | %v"

	borGRPCParityRetryInterval = 5 * time.Second
	borGRPCParityMaxAttempts   = 60 // ~5 min total

	// borGRPCParityDepth is how many blocks behind the latest we sample for the
	// parity check. Bor reorgs deeper than this are exceptional and would warrant alerting.
	borGRPCParityDepth = int64(32)

	// borGRPCParityMismatchStreak is the number of consecutive confirmed
	// mismatches required before we log.Fatal. A single mismatch can be a
	// transient race (canonical block at target height changed between the
	// HTTP and gRPC reads). Requiring N-in-a-row at the same height with
	// stable HTTP re-reads virtually rules that out.
	borGRPCParityMismatchStreak = 3

	// borGRPCParityFatalMsg is shared by the boot-time parity check and the
	// runtime per-probe validator so both report the same actionable reason.
	borGRPCParityFatalMsg = "FATAL: bor gRPC hash mismatch with HTTP confirmed across " +
		"multiple consecutive checks. The operator is likely running a new heimdall " +
		"with BorGRPCFlag=true against a bor that doesn't populate the full proto Header. " +
		"Continuing would corrupt milestone propositions on this node. " +
		"Either upgrade bor to a matching version or disable BorGRPCFlag."
)

func init() {
	Logger = logger.NewLogger(os.Stdout, logger.LevelOption(zerolog.InfoLevel))
}

// CustomConfig represents heimdall config
type CustomConfig struct {
	EthRPCUrl      string `mapstructure:"eth_rpc_url"`       // RPC endpoint for the main chain
	BorRPCUrl      string `mapstructure:"bor_rpc_url"`       // RPC endpoint for bor chain
	BorGRPCFlag    bool   `mapstructure:"bor_grpc_flag"`     // gRPC flag for bor chain
	BorGRPCUrl     string `mapstructure:"bor_grpc_url"`      // gRPC endpoint for bor chain
	BorGRPCToken   string `mapstructure:"bor_grpc_token"`    // bearer token for bor gRPC; empty = no auth
	CometBFTRPCUrl string `mapstructure:"comet_bft_rpc_url"` // cometBft node url
	SubGraphUrl    string `mapstructure:"sub_graph_url"`     // sub graph url

	EthRPCTimeout time.Duration `mapstructure:"eth_rpc_timeout"` // timeout for eth rpc
	BorRPCTimeout time.Duration `mapstructure:"bor_rpc_timeout"` // timeout for bor rpc

	AmqpURL string `mapstructure:"amqp_url"` // amqp url

	MainChainGasFeeCap int64 `mapstructure:"main_chain_gas_fee_cap"` // max fee per gas for EIP-1559 txs (in wei)
	MainChainGasTipCap int64 `mapstructure:"main_chain_gas_tip_cap"` // max priority fee per gas for EIP-1559 txs (in wei)

	// config related to bridge
	CheckpointPollInterval  time.Duration `mapstructure:"checkpoint_poll_interval"` // Poll interval for checkpointer service to send new checkpoints or missing ACK
	SyncerPollInterval      time.Duration `mapstructure:"syncer_poll_interval"`     // Poll interval for syncer service to sync for changes on the main chain
	NoACKPollInterval       time.Duration `mapstructure:"noack_poll_interval"`      // Poll interval for ack service to send no-ack in case of no checkpoints
	ClerkPollInterval       time.Duration `mapstructure:"clerk_poll_interval"`
	SpanPollInterval        time.Duration `mapstructure:"span_poll_interval"`
	MilestonePollInterval   time.Duration `mapstructure:"milestone_poll_interval"`
	EnableSH                bool          `mapstructure:"enable_self_heal"`           // Enable self-healing
	SHStateSyncedInterval   time.Duration `mapstructure:"sh_state_synced_interval"`   // Interval to self-heal StateSynced events if missing
	SHStakeUpdateInterval   time.Duration `mapstructure:"sh_stake_update_interval"`   // Interval to self-heal StakeUpdate events if missing
	SHCheckpointAckInterval time.Duration `mapstructure:"sh_checkpoint_ack_interval"` // Interval to self-heal Checkpoint ACKs (New Header Blocks) events if missing
	SHMaxDepthDuration      time.Duration `mapstructure:"sh_max_depth_duration"`      // Max duration that allows to suggest self-healing is not needed

	// wait-time-related options
	NoACKWaitTime time.Duration `mapstructure:"no_ack_wait_time"` // Time ack service waits to clear the buffer and elect the new proposer

	// Log related options
	LogsType       string `mapstructure:"logs_type"`        // if true, enable logging in json format
	LogsWriterFile string `mapstructure:"logs_writer_file"` // if given, Logs will be written to this file else os.Stdout

	Chain string `mapstructure:"chain"`

	ProducerVotes string `mapstructure:"producer_votes"`

	// #### Health check configs ####
	// MaxGoRoutineThreshold is the maximum number of goroutines before heimdall health check fails.
	MaxGoRoutineThreshold int `mapstructure:"max_goroutine_threshold"`

	// WarnGoRoutineThreshold is the maximum number of goroutines before heimdall health check warns.
	WarnGoRoutineThreshold int `mapstructure:"warn_goroutine_threshold"`

	// MinPeerThreshold is the minimum number of peers before heimdall health check fails.
	MinPeerThreshold int `mapstructure:"min_peer_threshold"`

	// WarnPeerThreshold is the minimum number of peers before heimdall health check warns.
	WarnPeerThreshold int `mapstructure:"warn_peer_threshold"`
}

type CustomAppConfig struct {
	serverconfig.Config `mapstructure:",squash"`
	Custom              CustomConfig `mapstructure:"custom"`
}

var conf CustomAppConfig

// MainChainClient stores the eth client for mainChain
var (
	mainChainClient *ethclient.Client
	mainRPCClient   *rpc.Client
)

// borClient stores eth/rpc client for bor
var (
	borClient     *ethclient.Client
	borRPCClient  *rpc.Client
	borGRPCClient borgrpc.Client
)

// private key object
var privKeyObject secp256k1.PrivKey

var pubKeyObject secp256k1.PubKey

var producerVotes []uint64

// Logger stores global logger object
var Logger logger.Logger

var rioHeight int64 = 0

var tallyFixHeight int64 = 0

var disableVPCheckHeight int64 = 0

var disableValSetCheckHeight int64 = 0

var initialHeight int64 = 0

var milestoneDeletionHeight int64 = 0

var faultyMilestoneNumber int64 = 0

var producerDowntimeHeight int64 = 0

var phuketHardforkHeight int64 = 0

var feeWithdrawValidatorGateHeight int64 = 0

var zurichHardforkHeight int64 = 0

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

// parseProducerVotes parses a comma-separated string of producer IDs into a slice of uint64
func parseProducerVotes(producerVotesStr string) []uint64 {
	if producerVotesStr == "" {
		return []uint64{}
	}

	producerStrings := strings.Split(producerVotesStr, ",")
	if len(producerStrings) > 0 && producerStrings[0] != "" {
		votes := make([]uint64, len(producerStrings))
		for i, p := range producerStrings {
			pTrimmed := strings.TrimSpace(p)
			if pTrimmed == "" {
				log.Fatalf("Empty producer ID found in producer votes list: '%s'", producerVotesStr)
			}
			var parseErr error
			votes[i], parseErr = strconv.ParseUint(pTrimmed, 10, 64)
			if parseErr != nil {
				log.Fatalf("Failed to parse producer ID '%s': %v", pTrimmed, parseErr)
			}
		}
		return votes
	}

	return []uint64{}
}

// InitHeimdallConfig initializes with viper config (from heimdall configuration)
func InitHeimdallConfig(homeDir string) {
	if strings.Compare(homeDir, "") == 0 {
		// get home dir from viper
		homeDir = viper.GetString(flags.FlagHome)
	}

	// get heimdall config filepath from the viper/cobra flag
	heimdallConfigFileFromFlag := viper.GetString(WithHeimdallConfigFlag)

	// init heimdall with changed config files
	InitHeimdallConfigWith(homeDir, heimdallConfigFileFromFlag)
}

// InitHeimdallConfigWith initializes passed heimdall/tendermint config files
func InitHeimdallConfigWith(homeDir string, heimdallConfigFileFromFlag string) {
	var err error

	if strings.Compare(homeDir, "") == 0 {
		panic("home directory is not specified")
	}

	if strings.Compare(conf.Custom.BorRPCUrl, "") != 0 || strings.Compare(conf.Custom.BorGRPCUrl, "") != 0 {
		return
	}

	// read configuration from the standard configuration file
	configDir := filepath.Join(homeDir, "config")
	heimdallViper := viper.New()
	heimdallViper.SetEnvPrefix("HEIMDALL")
	heimdallViper.AutomaticEnv()

	if heimdallConfigFileFromFlag == "" {
		heimdallViper.SetConfigName("app")     // name of the config file (without extension)
		heimdallViper.AddConfigPath(configDir) // call multiple times to add many search paths
	} else {
		heimdallViper.SetConfigFile(heimdallConfigFileFromFlag) // set the config file explicitly
	}

	// Handle errors reading the config file
	if err = heimdallViper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	// unmarshal configuration from the standard configuration file
	if err = heimdallViper.UnmarshalExact(&conf); err != nil {
		log.Fatalln("unable to unmarshall config", "Error", err)
	}

	//  if there is a file with overrides submitted via flags => read it and merge it with the already read standard configuration
	if heimdallConfigFileFromFlag != "" {
		heimdallViperFromFlag := viper.New()
		heimdallViperFromFlag.SetConfigFile(heimdallConfigFileFromFlag) // set the flag config file explicitly

		err = heimdallViperFromFlag.ReadInConfig()
		if err != nil { // Handle errors reading the config file submitted as a flag
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
	if err = conf.UpdateWithFlags(viper.GetViper(), Logger); err != nil {
		log.Fatalln("unable to read flag values. Check log for details.", "Error", err)
	}

	logLevelStr := viper.GetString(flags.FlagLogLevel)
	logLevel, err := zerolog.ParseLevel(logLevelStr)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	logNoColor := viper.GetBool(flags.FlagLogNoColor)
	var logOpts []logger.Option
	if conf.Custom.LogsType == "json" {
		logOpts = append(logOpts, logger.OutputJSONOption())
	} else {
		logOpts = append(logOpts, logger.ColorOption(!logNoColor))
	}
	logOpts = append(logOpts,
		logger.LevelOption(logLevel),
		logger.TimeFormatOption(LogTimestampFormat),
	)

	Logger = logger.NewLogger(GetLogsWriter(conf.Custom.LogsWriterFile), logOpts...)

	// perform checks for timeout
	if conf.Custom.EthRPCTimeout == 0 {
		// fallback to default
		Logger.Debug("Missing ETH RPC timeout or invalid value provided, falling back to default", "timeout", DefaultEthRPCTimeout)
		conf.Custom.EthRPCTimeout = DefaultEthRPCTimeout
	}

	if clamped := clampBorRPCTimeout(conf.Custom.BorRPCTimeout); clamped != conf.Custom.BorRPCTimeout {
		if conf.Custom.BorRPCTimeout <= 0 {
			Logger.Debug("Missing BOR RPC timeout or invalid value provided, falling back to default", "timeout", DefaultBorRPCTimeout)
		} else {
			// GetBorChainCallTimeout multiplies this by up to maxBudgetedEndpoints for
			// the failover cascade; cap it so one Bor call in a milestone/checkpoint
			// vote extension can't run past CometBFT's ~10s ABCI budget and miss votes.
			Logger.Warn("bor_rpc_timeout exceeds maximum; clamping", "configured", conf.Custom.BorRPCTimeout, "max", MaxBorRPCTimeout)
		}
		conf.Custom.BorRPCTimeout = clamped
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

	if conf.Custom.SHCheckpointAckInterval == 0 {
		// fallback to default
		Logger.Debug("Missing self-healing Checkpoint ACK interval or invalid value provided, falling back to default", "interval", DefaultSHCheckpointAckInterval)
		conf.Custom.SHCheckpointAckInterval = DefaultSHCheckpointAckInterval
	}

	if conf.Custom.SHMaxDepthDuration == 0 {
		// fallback to default
		Logger.Debug("Missing self-healing max depth duration or invalid value provided, falling back to default", "duration", DefaultSHMaxDepthDuration)
		conf.Custom.SHMaxDepthDuration = DefaultSHMaxDepthDuration
	}

	// validate EIP-1559 gas config: tip cap must not exceed fee cap
	if conf.Custom.MainChainGasTipCap > conf.Custom.MainChainGasFeeCap {
		log.Fatal("invalid gas config: main_chain_gas_tip_cap must not exceed main_chain_gas_fee_cap",
			"tip_cap", conf.Custom.MainChainGasTipCap,
			"fee_cap", conf.Custom.MainChainGasFeeCap,
		)
	}

	if mainRPCClient, err = rpc.Dial(conf.Custom.EthRPCUrl); err != nil {
		log.Fatal("unable to dial main chain RPC client", "URL", conf.Custom.EthRPCUrl, "error", err)
	}

	mainChainClient = ethclient.NewClient(mainRPCClient)

	initBorRPCClient()
	initBorGRPCClient()

	// Set default producers based on the chain if not already set by config or flags
	if conf.Custom.ProducerVotes == "" {
		switch conf.Custom.Chain {
		case MainChain:
			conf.Custom.ProducerVotes = DefaultMainnetProducers
			Logger.Debug("Using default mainnet producers", "producers", DefaultMainnetProducers)
		case AmoyChain:
			conf.Custom.ProducerVotes = DefaultAmoyTestnetProducers
			Logger.Debug("Using default amoy producers", "producers", DefaultAmoyTestnetProducers)
		case MumbaiChain:
			conf.Custom.ProducerVotes = DefaultMumbaiTestnetProducers
			Logger.Debug("Using default mumbai producers", "producers", DefaultMumbaiTestnetProducers)
		default:
			conf.Custom.ProducerVotes = DefaultLocalTestnetProducers
			Logger.Debug("Using default local producers", "producers", DefaultLocalTestnetProducers)
		}
	}

	producerVotes = parseProducerVotes(conf.Custom.ProducerVotes)
	if len(producerVotes) == 0 {
		Logger.Info("No producer votes configured or parsed.")
	}

	// load pv file, unmarshall and set to privKeyObject
	err = file.PermCheck(file.Rootify(privValJsonFile, configDir), secretFilePerm)
	if err != nil {
		Logger.Error(err.Error())
	}

	privVal := privval.LoadFilePV(filepath.Join(configDir, privValJsonFile), filepath.Join(configDir, privValJsonFile))
	privKeyObject = privVal.Key.PrivKey.Bytes()
	pubKeyObject = privVal.Key.PubKey.Bytes()

	switch conf.Custom.Chain {
	case MainChain:
		milestoneDeletionHeight = 28525000
		faultyMilestoneNumber = 1941439
		rioHeight = 77414656
		tallyFixHeight = 28913694
		disableVPCheckHeight = 25723000
		disableValSetCheckHeight = 25723063
		initialHeight = 24404501
		producerDowntimeHeight = 34966593
		phuketHardforkHeight = 44070000
		feeWithdrawValidatorGateHeight = 46361000
		zurichHardforkHeight = 0 // TODO marcello set HF height
	case MumbaiChain:
		milestoneDeletionHeight = 0
		faultyMilestoneNumber = -1
		rioHeight = 48473856
		tallyFixHeight = 0
		disableVPCheckHeight = 0
		disableValSetCheckHeight = 0
		initialHeight = 0
		producerDowntimeHeight = 0
		phuketHardforkHeight = 0
		feeWithdrawValidatorGateHeight = 0
		zurichHardforkHeight = 0
	case AmoyChain:
		milestoneDeletionHeight = 0
		faultyMilestoneNumber = -1
		rioHeight = 26272256
		tallyFixHeight = 13143851
		disableVPCheckHeight = 10618199
		disableValSetCheckHeight = 10618299
		initialHeight = 8788501
		producerDowntimeHeight = 20457139
		phuketHardforkHeight = 32276400
		feeWithdrawValidatorGateHeight = 35914000
		zurichHardforkHeight = 0 // TODO marcello set HF height
	default:
		milestoneDeletionHeight = 0
		faultyMilestoneNumber = -1
		rioHeight = 128
		tallyFixHeight = 0
		disableVPCheckHeight = 0
		disableValSetCheckHeight = 0
		initialHeight = 0
		producerDowntimeHeight = 0
		phuketHardforkHeight = 0
		feeWithdrawValidatorGateHeight = 0
		zurichHardforkHeight = 0
	}
}

// warnIfBorRPCInaccessible checks if the Bor RPC endpoint is accessible by making a simple call to get the latest block number.
// If the call fails, it logs a warning message. This is useful to detect issues with the Bor RPC endpoint at startup.
func warnIfBorRPCInaccessible(client *ethclient.Client, timeout time.Duration, url string) {
	if client == nil {
		Logger.Warn("Bor RPC client is nil", "URL", url)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if _, err := client.BlockNumber(ctx); err != nil {
		Logger.Warn("Bor RPC endpoint appears inaccessible at startup", "URL", url, "error", err)
	}
}

// warnIfBorGRPCInaccessible checks if the Bor gRPC endpoint is accessible by making a simple call to get the latest block header.
// If the call fails, it logs a warning message. This is useful to detect issues with the Bor gRPC endpoint at startup.
func warnIfBorGRPCInaccessible(client *borgrpc.BorGRPCClient, timeout time.Duration, url string) {
	if client == nil {
		Logger.Warn("Bor gRPC client is nil", "URL", url)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if _, err := client.HeaderByNumber(ctx, -2); err != nil {
		Logger.Warn("Bor gRPC endpoint appears inaccessible at startup", "URL", url, "error", err)
	}
}

// verifyBorGRPCHashParity launches a background goroutine that periodically
// asserts both transports return the same ethTypes.Header.Hash() for the same
// bor block. A mismatch typically means the operator is running a new heimdall
// with BorGRPCFlag=true against an old bor that doesn't populate the full proto
// Header — which would silently corrupt milestone propositions on this node.
// The goroutine retries until one of:
//   - Both transports return a header, and the hashes match (log.Info, exit goroutine)
//   - Both return a header, and hashes differ (log.Fatal, halt the node)
//   - Retry budget is exhausted (log Error, exit goroutine)
func verifyBorGRPCHashParity(httpClient *ethclient.Client, grpcClient *borgrpc.BorGRPCClient, timeout time.Duration) {
	// Negate_conditional requires injecting typed-nil stubs mixed with non-nil, which the production path never does.
	// mutator-disable-next-line defensive nil-client guard
	if httpClient == nil || grpcClient == nil {
		return
	}
	go runBorGRPCHashParityCheck(httpClient, grpcClient, timeout)
}

// runBorGRPCHashParityCheck launches the hash parity check
// mutator-disable-func thin production-init wiring around runBorGRPCHashParityCheckWith
// the full retry+fatal logic is tested via runBorGRPCHashParityCheckWith directly
func runBorGRPCHashParityCheck(httpClient *ethclient.Client, grpcClient *borgrpc.BorGRPCClient, timeout time.Duration) {
	runBorGRPCHashParityCheckWith(
		httpClient, grpcClient, timeout,
		borGRPCParityMaxAttempts, borGRPCParityRetryInterval,
		borGRPCParityFatalFunc,
	)
}

// borGRPCParityFatalFunc is the production fatal action for a confirmed parity
// mismatch streak: log the reason and exit so the node stops voting rather than
// feeding wrong-version Header hashes into side-handler reads.
func borGRPCParityFatalFunc(msg string, keysAndValues ...interface{}) {
	Logger.Error(msg, keysAndValues...)
	os.Exit(1)
}

// runBorGRPCHashParityCheckWith is the core of runBorGRPCHashParityCheck.
// It accepts the parity interfaces plus injectable max-attempts, retry-interval, and
// fatalFunc so unit tests can exercise the full loop without network access or process exit.
func runBorGRPCHashParityCheckWith(
	httpClient parityHTTPFetcher,
	grpcClient parityGRPCFetcher,
	timeout time.Duration,
	maxAttempts int,
	retryInterval time.Duration,
	fatalFunc func(msg string, keysAndValues ...interface{}),
) {
	mismatches := 0
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ok, mismatch := checkBorGRPCHashParityOnceWith(httpClient, grpcClient, timeout)
		if ok {
			return
		}
		next, fatal := updateParityMismatchStreak(mismatches, mismatch, borGRPCParityMismatchStreak)
		if fatal {
			fatalFunc(borGRPCParityFatalMsg, "consecutiveMismatches", next)
			// Return so control flow does not depend on fatalFunc exiting the
			// process.
			return
		}
		mismatches = next
		// mutator-disable-next-line retry pacing
		if attempt < maxAttempts {
			time.Sleep(retryInterval)
		}
	}
	// Statement_deletion only drops an advisory message; no logic change.
	// mutator-disable-next-line operator-log line
	Logger.Error("Bor gRPC hash parity check gave up after retries — could not confirm transport equivalence. "+
		"If BorGRPCFlag=true, verify that bor is running a version that populates the full proto Header. "+
		"Continuing without parity confirmation.",
		"attempts", maxAttempts,
		"interval", retryInterval,
	)
}

// updateParityMismatchStreak evolves the consecutive-mismatch counter given
// the outcome of one parity check. Returns the new value and whether
// the caller should log.Fatal (reached the configured threshold).
func updateParityMismatchStreak(current int, mismatch bool, streakLimit int) (next int, fatal bool) {
	if !mismatch {
		// Transient / unavailable — reset the streak so a flaky network
		// window can't ladder into false fatal later.
		return 0, false
	}
	next = current + 1
	return next, next >= streakLimit
}

// checkBorGRPCHashParityOnceWith runs a single parity comparison at a block a
// few confirmations behind the current head (to avoid head-churn / reorg
// races).
// Returns (ok, mismatch):
//   - Ok=true: both transports returned the same hash for the same block → check passed, caller exits
//   - Ok=false, mismatch=false: one transport was unavailable, a reorg was detected during the check,
//     or the chain is too young for the target depth → retry later, do not count toward mismatch streak
//   - Ok=false, mismatch=true: both transports returned headers but with different hashes → count toward
//     mismatch streak; the caller logs fatal only after borGRPCParityMismatchStreak consecutive mismatches
func checkBorGRPCHashParityOnceWith(httpClient parityHTTPFetcher, grpcClient parityGRPCFetcher, timeout time.Duration) (ok, mismatch bool) {
	return checkBorGRPCHashParityOnce(context.Background(), httpClient, grpcClient, timeout, true)
}

func checkBorGRPCHashParityOnceQuiet(ctx context.Context, httpClient parityHTTPFetcher, grpcClient parityGRPCFetcher, timeout time.Duration) (ok, mismatch bool) {
	return checkBorGRPCHashParityOnce(ctx, httpClient, grpcClient, timeout, false)
}

func checkBorGRPCHashParityOnce(parent context.Context, httpClient parityHTTPFetcher, grpcClient parityGRPCFetcher, timeout time.Duration, emitLogs bool) (ok, mismatch bool) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	targetNum, okDepth := resolveParityTargetHeight(ctx, httpClient)
	if !okDepth {
		return false, false
	}
	httpHeader, grpcHeader, stable := fetchStableHeadersAtHeight(ctx, httpClient, grpcClient, targetNum)
	if !stable {
		return false, false
	}

	if grpcHeader.Hash() != httpHeader.Hash() {
		if emitLogs {
			// Statement_deletion drops an advisory log; the (false, true) return below still signals a mismatch.
			// mutator-disable-next-line operator-log line
			Logger.Warn("Bor gRPC hash mismatch with HTTP for the same block — counting toward mismatch streak before fatal",
				"block", httpHeader.Number.String(),
				"httpHash", httpHeader.Hash().Hex(),
				"grpcHash", grpcHeader.Hash().Hex(),
			)
		}
		// Streak bookkeeping is covered directly via updateParityMismatchStreak tests.
		// mutator-disable-next-line boolean_substitution on mismatch signal
		return false, true
	}

	if emitLogs {
		// Statement_deletion drops a success message; no branch logic affected.
		// mutator-disable-next-line operator-log line
		Logger.Info("Bor gRPC hash parity check passed",
			"block", httpHeader.Number.String(),
			"hash", httpHeader.Hash().Hex(),
		)
	}
	// Ok-signal flip is observable only via runBorGRPCHashParityCheckWith's caller-side early-return, which is tested directly.
	// mutator-disable-next-line boolean_substitution on ok signal
	return true, false
}

// parityHTTPFetcher is the subset of *ethclient.Client used by the parity check.
// Defined as an interface so unit tests can inject stubs without dialing a real
// Bor HTTP endpoint. *ethclient.Client satisfies this interface.
type parityHTTPFetcher interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*ethTypes.Header, error)
}

// parityGRPCFetcher is the subset of *borgrpc.BorGRPCClient used by the parity
// check. Defined as an interface so unit tests can inject stubs without dialing
// a real gRPC endpoint. *borgrpc.BorGRPCClient satisfies this interface.
type parityGRPCFetcher interface {
	HeaderByNumber(ctx context.Context, blockID int64) (*ethTypes.Header, error)
}

// resolveParityTargetHeight returns (targetNum, ok). ok=false when the chain
// is too young, the HTTP head lookup failed, or the result is otherwise
// unusable — in any of those cases the parity check should retry later.
func resolveParityTargetHeight(ctx context.Context, httpClient parityHTTPFetcher) (int64, bool) {
	latest, err := httpClient.HeaderByNumber(ctx, nil)
	if err != nil || latest == nil {
		return 0, false
	}
	latestNum := latest.Number.Int64()
	if latestNum < borGRPCParityDepth {
		return 0, false
	}
	return latestNum - borGRPCParityDepth, true
}

// fetchStableHeadersAtHeight pulls the block at targetNum via HTTP, then via
// gRPC, then via HTTP again, and only returns (httpHeader, grpcHeader, true)
// when both HTTP reads agree (ruling out a reorg mid-check). Any transport
// error or reorg returns stable=false, so the caller defers the decision.
func fetchStableHeadersAtHeight(ctx context.Context, httpClient parityHTTPFetcher, grpcClient parityGRPCFetcher, targetNum int64) (httpHeader, grpcHeader *ethTypes.Header, stable bool) {
	target := big.NewInt(targetNum)
	httpHeader, err := httpClient.HeaderByNumber(ctx, target)
	if err != nil || httpHeader == nil {
		return nil, nil, false
	}
	grpcHeader, err = grpcClient.HeaderByNumber(ctx, targetNum)
	if err != nil || grpcHeader == nil {
		return nil, nil, false
	}
	httpHeader2, err := httpClient.HeaderByNumber(ctx, target)
	if err != nil || httpHeader2 == nil || httpHeader2.Hash() != httpHeader.Hash() {
		return nil, nil, false
	}
	return httpHeader, grpcHeader, true
}

// GetDefaultHeimdallConfig returns configuration with default params
func GetDefaultHeimdallConfig() CustomConfig {
	return CustomConfig{
		EthRPCUrl:   DefaultMainRPCUrl,
		BorRPCUrl:   DefaultBorRPCUrl,
		BorGRPCFlag: DefaultBorGRPCFlag,
		BorGRPCUrl:  DefaultBorGRPCUrl,

		ProducerVotes: DefaultMainnetProducers,

		CometBFTRPCUrl: DefaultCometBFTNodeURL,

		EthRPCTimeout: DefaultEthRPCTimeout,
		BorRPCTimeout: DefaultBorRPCTimeout,

		AmqpURL: DefaultAmqpURL,

		MainChainGasFeeCap: DefaultMainChainGasFeeCap,
		MainChainGasTipCap: DefaultMainChainGasTipCap,

		CheckpointPollInterval:  DefaultCheckpointPollInterval,
		SyncerPollInterval:      DefaultSyncerPollInterval,
		NoACKPollInterval:       DefaultNoACKPollInterval,
		ClerkPollInterval:       DefaultClerkPollInterval,
		SpanPollInterval:        DefaultSpanPollInterval,
		MilestonePollInterval:   DefaultMilestonePollInterval,
		EnableSH:                DefaultEnableSH,
		SHStateSyncedInterval:   DefaultSHStateSyncedInterval,
		SHStakeUpdateInterval:   DefaultSHStakeUpdateInterval,
		SHCheckpointAckInterval: DefaultSHCheckpointAckInterval,
		SHMaxDepthDuration:      DefaultSHMaxDepthDuration,

		NoACKWaitTime: NoACKWaitTime,

		LogsType:       DefaultLogsType,
		Chain:          DefaultChain,
		LogsWriterFile: "", // default to stdout

		MaxGoRoutineThreshold:  0,
		WarnGoRoutineThreshold: 0,
		MinPeerThreshold:       0,
		WarnPeerThreshold:      0,
	}
}

// GetConfig returns the cached configuration object
func GetConfig() CustomConfig {
	return conf.Custom
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

// GetPrivKey returns the priv key object
func GetPrivKey() secp256k1.PrivKey {
	return privKeyObject
}

// GetPubKey returns the pub key object
func GetPubKey() secp256k1.PubKey {
	return pubKeyObject
}

// GetAddress returns address object
func GetAddress() []byte {
	return GetPubKey().Address()
}

// GetAddressString returns the address object as string
func GetAddressString() (string, error) {
	address := GetAddress()
	ac := addressCodec.NewHexCodec()
	addressString, err := ac.BytesToString(address)
	if err != nil {
		return "", err
	}
	return addressString, nil
}

// GetValidChains returns all the valid chains
func GetValidChains() []string {
	return []string{"mainnet", "mumbai", "amoy", "local"}
}

func GetRioHeight() int64 {
	return rioHeight
}

func IsRio(blockNum uint64) bool {
	return blockNum >= uint64(rioHeight)
}

func SetRioHeight(height int64) {
	rioHeight = height
}

func GetTallyFixHeight() int64 {
	return tallyFixHeight
}

func GetDisableVPCheckHeight() int64 {
	return disableVPCheckHeight
}

func GetDisableValSetCheckHeight() int64 {
	return disableValSetCheckHeight
}

func GetInitialHeight() int64 {
	return initialHeight
}

func GetMilestoneDeletionHeight() int64 {
	return milestoneDeletionHeight
}

func GetFaultyMilestoneNumber() uint64 {
	return uint64(faultyMilestoneNumber)
}

func GetSetProducerDowntimeHeight() int64 {
	return producerDowntimeHeight
}

func IsPhuketHardfork(height int64) bool {
	return phuketHardforkHeight > 0 && height >= phuketHardforkHeight
}

func SetPhuketHardforkHeight(height int64) {
	phuketHardforkHeight = height
}

func GetPhuketHardforkHeight() int64 {
	return phuketHardforkHeight
}

func IsFeeWithdrawValidatorGate(height int64) bool {
	return feeWithdrawValidatorGateHeight > 0 && height >= feeWithdrawValidatorGateHeight
}

func SetFeeWithdrawValidatorGateHeight(height int64) {
	feeWithdrawValidatorGateHeight = height
}

func GetFeeWithdrawValidatorGateHeight() int64 {
	return feeWithdrawValidatorGateHeight
}

func IsZurichHardfork(height int64) bool {
	return zurichHardforkHeight > 0 && height >= zurichHardforkHeight
}

func SetZurichHardforkHeight(height int64) {
	zurichHardforkHeight = height
}

func GetZurichHardforkHeight() int64 {
	return zurichHardforkHeight
}

func GetChainManagerAddressMigration(blockNum int64) (ChainManagerAddressMigration, bool) {
	chainMigration := chainManagerAddressMigrations[conf.Custom.Chain]
	if chainMigration == nil {
		chainMigration = chainManagerAddressMigrations["default"]
	}

	result, found := chainMigration[blockNum]

	return result, found
}

func GetProducerVotes() []uint64 {
	return producerVotes
}

func GetFallbackProducerVotes() []uint64 {
	switch conf.Custom.Chain {
	case MainChain:
		return parseProducerVotes(DefaultMainnetProducers)
	case AmoyChain:
		return parseProducerVotes(DefaultAmoyTestnetProducers)
	case MumbaiChain:
		return parseProducerVotes(DefaultMumbaiTestnetProducers)
	default:
		return parseProducerVotes(DefaultLocalTestnetProducers)
	}
}

const (
	producerSetLimit    = uint64(3)
	newProducerSetLimit = uint64(4)
)

func GetProducerSetLimit(ctx sdk.Context) uint64 {
	if ctx.BlockHeight() >= GetSetProducerDowntimeHeight() {
		return newProducerSetLimit
	}
	return producerSetLimit
}

const (
	changeProducerThreshold    = 5
	spanRotationBuffer         = 10
	newChangeProducerThreshold = 10
	newSpanRotationBuffer      = 20
)

func GetChangeProducerThreshold(ctx sdk.Context) int64 {
	if ctx.BlockHeight() >= GetSetProducerDowntimeHeight() {
		return newChangeProducerThreshold
	}
	return changeProducerThreshold
}

func GetSpanRotationBuffer(ctx sdk.Context) uint64 {
	if ctx.BlockHeight() >= GetSetProducerDowntimeHeight() {
		return newSpanRotationBuffer
	}
	return spanRotationBuffer
}

// DecorateWithHeimdallFlags adds persistent flags for app configs and bind flags with command
func DecorateWithHeimdallFlags(cmd *cobra.Command, v *viper.Viper, loggerInstance logger.Logger, caller string) {
	// add the with-app-config flag
	cmd.PersistentFlags().String(
		WithHeimdallConfigFlag,
		"",
		"Override of Heimdall app config file (default <home>/config/config.json)",
	)

	if err := v.BindPFlag(WithHeimdallConfigFlag, cmd.PersistentFlags().Lookup(WithHeimdallConfigFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, WithHeimdallConfigFlag), "Error", err)
	}

	// add MainRPCUrlFlag flag
	cmd.PersistentFlags().String(
		MainRPCUrlFlag,
		"",
		"Set RPC endpoint for ethereum chain",
	)

	if err := v.BindPFlag(MainRPCUrlFlag, cmd.PersistentFlags().Lookup(MainRPCUrlFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, MainRPCUrlFlag), "Error", err)
	}

	// add BorRPCUrlFlag flag
	cmd.PersistentFlags().String(
		BorRPCUrlFlag,
		"",
		"Set RPC endpoint for bor chain",
	)

	if err := v.BindPFlag(BorRPCUrlFlag, cmd.PersistentFlags().Lookup(BorRPCUrlFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, BorRPCUrlFlag), "Error", err)
	}

	// add BorGRPCUrlFlag flag
	cmd.PersistentFlags().String(
		BorGRPCUrlFlag,
		"",
		"Set gRPC endpoint for bor chain",
	)

	if err := v.BindPFlag(BorGRPCUrlFlag, cmd.PersistentFlags().Lookup(BorGRPCUrlFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, BorGRPCUrlFlag), "Error", err)
	}

	// add BorGRPCFlagFlag flag
	cmd.PersistentFlags().String(
		BorGRPCFlagFlag,
		"",
		"gRPC flag for bor chain",
	)

	if err := v.BindPFlag(BorGRPCFlagFlag, cmd.PersistentFlags().Lookup(BorGRPCFlagFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, BorGRPCFlagFlag), "Error", err)
	}

	// add BorGRPCTokenFlag flag
	cmd.PersistentFlags().String(
		BorGRPCTokenFlag,
		"",
		"Bearer token for bor gRPC authentication (must match bor [grpc] token; empty disables auth)",
	)

	// viper.BindPFlag only errors if the flag doesn't exist, which the PersistentFlags().String call above guarantees.
	// mutator-disable-next-line CLI flag-binding error guard
	if err := v.BindPFlag(BorGRPCTokenFlag, cmd.PersistentFlags().Lookup(BorGRPCTokenFlag)); err != nil {
		// mutator-disable-next-line operator-log line in unreachable error branch
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, BorGRPCTokenFlag), "Error", err)
	}

	// add CometBFTNodeURLFlag flag
	cmd.PersistentFlags().String(
		CometBFTNodeURLFlag,
		"",
		"Set RPC endpoint for CometBFT",
	)

	if err := v.BindPFlag(CometBFTNodeURLFlag, cmd.PersistentFlags().Lookup(CometBFTNodeURLFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, CometBFTNodeURLFlag), "Error", err)
	}

	// add HeimdallServerURLFlag flag
	cmd.PersistentFlags().String(
		HeimdallServerURLFlag,
		"",
		"Set Heimdall REST server endpoint",
	)

	if err := v.BindPFlag(HeimdallServerURLFlag, cmd.PersistentFlags().Lookup(HeimdallServerURLFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, HeimdallServerURLFlag), "Error", err)
	}

	// add GRPCServerURL flag
	cmd.PersistentFlags().String(
		GRPCServerURLFlag,
		"",
		"Set GRPC Server Endpoint",
	)

	if err := v.BindPFlag(GRPCServerURLFlag, cmd.PersistentFlags().Lookup(GRPCServerURLFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, GRPCServerURLFlag), "Error", err)
	}

	// add AmqpURLFlag flag
	cmd.PersistentFlags().String(
		AmqpURLFlag,
		"",
		"Set AMQP endpoint",
	)

	if err := v.BindPFlag(AmqpURLFlag, cmd.PersistentFlags().Lookup(AmqpURLFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, AmqpURLFlag), "Error", err)
	}

	// add CheckpointerPollIntervalFlag flag
	cmd.PersistentFlags().String(
		CheckpointerPollIntervalFlag,
		"",
		"Set check point pull interval",
	)

	if err := v.BindPFlag(CheckpointerPollIntervalFlag, cmd.PersistentFlags().Lookup(CheckpointerPollIntervalFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, CheckpointerPollIntervalFlag), "Error", err)
	}

	// add SyncerPollIntervalFlag flag
	cmd.PersistentFlags().String(
		SyncerPollIntervalFlag,
		"",
		"Set syncer pull interval",
	)

	if err := v.BindPFlag(SyncerPollIntervalFlag, cmd.PersistentFlags().Lookup(SyncerPollIntervalFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, SyncerPollIntervalFlag), "Error", err)
	}

	// add NoACKPollIntervalFlag flag
	cmd.PersistentFlags().String(
		NoACKPollIntervalFlag,
		"",
		"Set no acknowledge pull interval",
	)

	if err := v.BindPFlag(NoACKPollIntervalFlag, cmd.PersistentFlags().Lookup(NoACKPollIntervalFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, NoACKPollIntervalFlag), "Error", err)
	}

	// add ClerkPollIntervalFlag flag
	cmd.PersistentFlags().String(
		ClerkPollIntervalFlag,
		"",
		"Set clerk pull interval",
	)

	if err := v.BindPFlag(ClerkPollIntervalFlag, cmd.PersistentFlags().Lookup(ClerkPollIntervalFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, ClerkPollIntervalFlag), "Error", err)
	}

	// add SpanPollIntervalFlag flag
	cmd.PersistentFlags().String(
		SpanPollIntervalFlag,
		"",
		"Set span pull interval",
	)

	if err := v.BindPFlag(SpanPollIntervalFlag, cmd.PersistentFlags().Lookup(SpanPollIntervalFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, SpanPollIntervalFlag), "Error", err)
	}

	// add MilestonePollIntervalFlag flag
	cmd.PersistentFlags().String(
		MilestonePollIntervalFlag,
		DefaultMilestonePollInterval.String(),
		"Set milestone interval",
	)

	if err := v.BindPFlag(MilestonePollIntervalFlag, cmd.PersistentFlags().Lookup(MilestonePollIntervalFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, MilestonePollIntervalFlag), "Error", err)
	}

	// add MainChainGasFeeCapFlag flag
	cmd.PersistentFlags().Int64(
		MainChainGasFeeCapFlag,
		0,
		"Set main chain max gas fee cap for EIP-1559 transactions (in wei)",
	)

	if err := v.BindPFlag(MainChainGasFeeCapFlag, cmd.PersistentFlags().Lookup(MainChainGasFeeCapFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, MainChainGasFeeCapFlag), "Error", err)
	}

	// add MainChainGasTipCapFlag flag
	cmd.PersistentFlags().Int64(
		MainChainGasTipCapFlag,
		0,
		"Set main chain max priority fee (tip) for EIP-1559 transactions (in wei)",
	)

	if err := v.BindPFlag(MainChainGasTipCapFlag, cmd.PersistentFlags().Lookup(MainChainGasTipCapFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, MainChainGasTipCapFlag), "Error", err)
	}

	// add NoACKWaitTimeFlag flag
	cmd.PersistentFlags().String(
		NoACKWaitTimeFlag,
		"",
		"Set time ack service waits to clear buffer and elect new proposer",
	)

	if err := v.BindPFlag(NoACKWaitTimeFlag, cmd.PersistentFlags().Lookup(NoACKWaitTimeFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, NoACKWaitTimeFlag), "Error", err)
	}

	// add chain flag
	cmd.PersistentFlags().String(
		ChainFlag,
		"",
		fmt.Sprintf("Set one of the chains: [%s]", strings.Join(GetValidChains(), ",")),
	)

	if err := v.BindPFlag(ChainFlag, cmd.PersistentFlags().Lookup(ChainFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, ChainFlag), "Error", err)
	}

	// add logsWriterFile flag
	cmd.PersistentFlags().String(
		LogsWriterFileFlag,
		"",
		"Set logs writer file, Default is os.Stdout",
	)

	if err := v.BindPFlag(LogsWriterFileFlag, cmd.PersistentFlags().Lookup(LogsWriterFileFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, LogsWriterFileFlag), "Error", err)
	}

	// add producerVotes flag
	cmd.PersistentFlags().String(
		ProducerVotesFlag,
		"",
		"Set comma-separated list of producer IDs",
	)

	if err := v.BindPFlag(ProducerVotesFlag, cmd.PersistentFlags().Lookup(ProducerVotesFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, caller, ProducerVotesFlag), "Error", err)
	}
}

func (c *CustomAppConfig) UpdateWithFlags(v *viper.Viper, loggerInstance logger.Logger) error {
	const logErrMsg = "Unable to read flag."

	// get the endpoint for the ethereum chain from viper/cobra
	stringConfigValue := v.GetString(MainRPCUrlFlag)
	if stringConfigValue != "" {
		c.Custom.EthRPCUrl = stringConfigValue
	}

	// get endpoint for bor chain from viper/cobra
	stringConfigValue = v.GetString(BorRPCUrlFlag)
	if stringConfigValue != "" {
		c.Custom.BorRPCUrl = stringConfigValue
	}

	// get gRPC flag for bor chain from viper/cobra. Use IsSet so an explicit
	// --bor_grpc_flag=false from CLI/env can override a config-file true,
	// rather than silently being indistinguishable from "unset" via the bool
	// zero value.
	if v.IsSet(BorGRPCFlagFlag) {
		c.Custom.BorGRPCFlag = v.GetBool(BorGRPCFlagFlag)
	}

	// get endpoint for bor chain from viper/cobra
	stringConfigValue = v.GetString(BorGRPCUrlFlag)
	if stringConfigValue != "" {
		c.Custom.BorGRPCUrl = stringConfigValue
	}

	// get bearer token for bor gRPC from viper/cobra
	stringConfigValue = v.GetString(BorGRPCTokenFlag)
	if stringConfigValue != "" {
		c.Custom.BorGRPCToken = stringConfigValue
	}

	// get endpoint for cometBFT from viper/cobra
	stringConfigValue = v.GetString(CometBFTNodeURLFlag)
	if stringConfigValue != "" {
		c.Custom.CometBFTRPCUrl = stringConfigValue
	}

	// get endpoint for CometBFT from viper/cobra
	stringConfigValue = v.GetString(AmqpURLFlag)
	if stringConfigValue != "" {
		c.Custom.AmqpURL = stringConfigValue
	}

	// get Heimdall REST server endpoint from viper/cobra
	stringConfigValue = v.GetString(HeimdallServerURLFlag)
	if stringConfigValue != "" {
		c.API.Enable = true
		c.API.Address = stringConfigValue
	}

	// get Heimdall GRPC server endpoint from viper/cobra
	stringConfigValue = v.GetString(GRPCServerURLFlag)
	if stringConfigValue != "" {
		c.GRPC.Enable = true
		c.GRPC.Address = stringConfigValue
	}

	// need this error for parsing Duration values
	var err error

	// get the checkpoint poll interval from viper/cobra
	stringConfigValue = v.GetString(CheckpointerPollIntervalFlag)
	if stringConfigValue != "" {
		if c.Custom.CheckpointPollInterval, err = time.ParseDuration(stringConfigValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", CheckpointerPollIntervalFlag, "Error", err)
			return err
		}
	}

	// get syncer pull interval from viper/cobra
	stringConfigValue = v.GetString(SyncerPollIntervalFlag)
	if stringConfigValue != "" {
		if c.Custom.SyncerPollInterval, err = time.ParseDuration(stringConfigValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", SyncerPollIntervalFlag, "Error", err)
			return err
		}
	}

	// get the poll interval for ack service to send no-ack in case of no checkpoints from viper/cobra
	stringConfigValue = v.GetString(NoACKPollIntervalFlag)
	if stringConfigValue != "" {
		if c.Custom.NoACKPollInterval, err = time.ParseDuration(stringConfigValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", NoACKPollIntervalFlag, "Error", err)
			return err
		}
	}

	// get clerk poll interval from viper/cobra
	stringConfigValue = v.GetString(ClerkPollIntervalFlag)
	if stringConfigValue != "" {
		if c.Custom.ClerkPollInterval, err = time.ParseDuration(stringConfigValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", ClerkPollIntervalFlag, "Error", err)
			return err
		}
	}

	// get span poll interval from viper/cobra
	stringConfigValue = v.GetString(SpanPollIntervalFlag)
	if stringConfigValue != "" {
		if c.Custom.SpanPollInterval, err = time.ParseDuration(stringConfigValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", SpanPollIntervalFlag, "Error", err)
			return err
		}
	}

	// get milestone poll interval from viper/cobra
	stringConfigValue = v.GetString(MilestonePollIntervalFlag)
	if stringConfigValue != "" {
		if c.Custom.MilestonePollInterval, err = time.ParseDuration(stringConfigValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", MilestonePollIntervalFlag, "Error", err)
			return err
		}
	}

	// get time that ack service waits to clear buffer and elect the new proposer from viper/cobra
	stringConfigValue = v.GetString(NoACKWaitTimeFlag)
	if stringConfigValue != "" {
		if c.Custom.NoACKWaitTime, err = time.ParseDuration(stringConfigValue); err != nil {
			loggerInstance.Error(logErrMsg, "Flag", NoACKWaitTimeFlag, "Error", err)
			return err
		}
	}

	// get mainChain gas fee cap from viper/cobra. if it is greater than zero, set it as a configuration parameter
	int64ConfigValue := v.GetInt64(MainChainGasFeeCapFlag)
	if int64ConfigValue > 0 {
		c.Custom.MainChainGasFeeCap = int64ConfigValue
	}

	// get mainChain gas tip cap from viper/cobra
	int64ConfigValue = v.GetInt64(MainChainGasTipCapFlag)
	if int64ConfigValue > 0 {
		c.Custom.MainChainGasTipCap = int64ConfigValue
	}

	// get chain from viper/cobra flag
	stringConfigValue = v.GetString(ChainFlag)
	if stringConfigValue != "" {
		c.Custom.Chain = stringConfigValue
	}

	stringConfigValue = v.GetString(LogsWriterFileFlag)
	if stringConfigValue != "" {
		c.Custom.LogsWriterFile = stringConfigValue
	}

	// get producer votes from viper/cobra flag
	stringConfigValue = v.GetString(ProducerVotesFlag)
	if stringConfigValue != "" {
		c.Custom.ProducerVotes = stringConfigValue
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

	// Only adopt cc.BorGRPCFlag when cc explicitly configures the gRPC block,
	// signaled by a non-empty BorGRPCUrl. Without this guard, a layered config
	// that omits the gRPC block would silently flip BorGRPCFlag to its bool
	// zero value (false) and disable gRPC for an operator who set it elsewhere.
	if cc.BorGRPCUrl != "" {
		c.Custom.BorGRPCFlag = cc.BorGRPCFlag
		c.Custom.BorGRPCUrl = cc.BorGRPCUrl
	}

	if cc.BorGRPCToken != "" {
		c.Custom.BorGRPCToken = cc.BorGRPCToken
	}

	if cc.CometBFTRPCUrl != "" {
		c.Custom.CometBFTRPCUrl = cc.CometBFTRPCUrl
	}

	if cc.AmqpURL != "" {
		c.Custom.AmqpURL = cc.AmqpURL
	}

	if cc.MainChainGasFeeCap != 0 {
		c.Custom.MainChainGasFeeCap = cc.MainChainGasFeeCap
	}

	if cc.MainChainGasTipCap != 0 {
		c.Custom.MainChainGasTipCap = cc.MainChainGasTipCap
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

	if cc.SHCheckpointAckInterval != 0 {
		c.Custom.SHCheckpointAckInterval = cc.SHCheckpointAckInterval
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

	// Add merge logic for Producers if necessary, though flags and direct config usually take precedence.
	// If the direct config file sets it, it's already in c.Custom.Producers before merge.
	// If the override file (cc) sets it, we might want to let it override.
	if cc.ProducerVotes != "" {
		c.Custom.ProducerVotes = cc.ProducerVotes
	}
}

// DecorateWithCometBFTFlags creates cometBFT flags for the desired command and binds them to viper
func DecorateWithCometBFTFlags(cmd *cobra.Command, v *viper.Viper, loggerInstance logger.Logger, message string) {
	// add seedsFlag
	cmd.PersistentFlags().String(
		SeedsFlag,
		"",
		"Override seeds",
	)

	if err := v.BindPFlag(SeedsFlag, cmd.PersistentFlags().Lookup(SeedsFlag)); err != nil {
		loggerInstance.Error(fmt.Sprintf(bindPFlagLog, message, SeedsFlag), "Error", err)
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
		case AmoyChain:
			cometBFTConfig.P2P.Seeds = DefaultAmoyTestnetSeeds
		}
	}
}

func GetLogsWriter(logsWriterFile string) io.Writer {
	if logsWriterFile != "" {
		logWriter, err := os.OpenFile(logsWriterFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
		if err != nil {
			log.Fatalf("error opening log writer file: %v", err)
		}

		return logWriter
	}
	return os.Stdout
}

// GetBorGRPCClient returns bor gRPC client
func GetBorGRPCClient() borgrpc.Client {
	return borGRPCClient
}

// Sanitize enforces minimums and returns notes and corrected key/values
func (c *CustomAppConfig) Sanitize() (notes []string, kv map[string]any) {
	kv = make(map[string]any)

	if c.MinRetainBlocks != 0 && c.MinRetainBlocks < EnforcedMinRetainBlocks {
		c.MinRetainBlocks = EnforcedMinRetainBlocks
		notes = append(notes, fmt.Sprintf("min-retain-blocks=%d (minimum enforced)", EnforcedMinRetainBlocks))
		kv["min-retain-blocks"] = EnforcedMinRetainBlocks
	}

	return notes, kv
}
