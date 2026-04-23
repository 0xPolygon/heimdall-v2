package helper

import (
	"fmt"
	"testing"
	"time"

	logger "cosmossdk.io/log"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// TestHeimdallConfig checks heimdall configs
func TestHeimdallConfig(t *testing.T) {
	// Not t.Parallel(): this test mutates package-level configs via InitTestHeimdallConfig

	// cli context
	cometBFTNode := "tcp://localhost:26657"
	viper.Set(CometBFTNodeFlag, cometBFTNode)
	viper.Set(flags.FlagLogLevel, "info")

	InitTestHeimdallConfig("")

	fmt.Println("Address", GetAddress())

	pubKey := GetPubKey()

	fmt.Println("PublicKey", pubKey.String())
}

func TestHeimdallConfigUpdateCometBFTConfig(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		chain string
		viper string
		def   string
		value string
	}

	data := []testStruct{
		{chain: "mumbai", viper: "viper", def: "default", value: "viper"},
		{chain: "mumbai", viper: "viper", def: "", value: "viper"},
		{chain: "mumbai", viper: "", def: "default", value: "default"},
		{chain: "amoy", viper: "viper", def: "default", value: "viper"},
		{chain: "amoy", viper: "viper", def: "", value: "viper"},
		{chain: "amoy", viper: "", def: "default", value: "default"},
		{chain: "amoy", viper: "", def: "", value: DefaultAmoyTestnetSeeds},
		{chain: "mainnet", viper: "viper", def: "default", value: "viper"},
		{chain: "mainnet", viper: "viper", def: "", value: "viper"},
		{chain: "mainnet", viper: "", def: "default", value: "default"},
		{chain: "mainnet", viper: "", def: "", value: DefaultMainnetSeeds},
		{chain: "local", viper: "viper", def: "default", value: "viper"},
		{chain: "local", viper: "viper", def: "", value: "viper"},
		{chain: "local", viper: "", def: "default", value: "default"},
		{chain: "local", viper: "", def: "", value: ""},
	}

	oldConf := conf.Custom.Chain
	viperObj := viper.New()
	cometBFTConfig := cfg.DefaultConfig()

	for _, ts := range data {
		conf.Custom.Chain = ts.chain
		cometBFTConfig.P2P.Seeds = ts.def
		viperObj.Set(SeedsFlag, ts.viper)
		UpdateCometBFTConfig(cometBFTConfig, viperObj)

		if cometBFTConfig.P2P.Seeds != ts.value {
			t.Errorf("invalid UpdateCometBFTConfig, CometBFTConfig.P2P.Seeds not set correctly")
		}
	}

	conf.Custom.Chain = oldConf
}

// TestVerifyBorGRPCHashParityNilClients checks that verifyBorGRPCHashParity
// returns immediately without spawning a goroutine when either client is nil.
// The happy path (hashes match) and the mismatch path (log.Fatal) both require
// live Bor HTTP/gRPC servers and are integration-test only.
func TestVerifyBorGRPCHashParityNilClients(t *testing.T) {
	t.Parallel()

	// Dispatcher must return without panic and without spawning a goroutine
	// when either client is nil.
	verifyBorGRPCHashParity(nil, nil, time.Second)
}

// TestUpdateParityMismatchStreak covers the mismatch-streak state machine that gates log.Fatal on
// confirmed parity mismatches.
func TestUpdateParityMismatchStreak(t *testing.T) {
	t.Parallel()

	const limit = 3

	t.Run("transient failure resets streak", func(t *testing.T) {
		t.Parallel()
		next, fatal := updateParityMismatchStreak(2, false, limit)
		require.Equal(t, 0, next)
		require.False(t, fatal)
	})

	t.Run("single mismatch does not fatal", func(t *testing.T) {
		t.Parallel()
		next, fatal := updateParityMismatchStreak(0, true, limit)
		require.Equal(t, 1, next)
		require.False(t, fatal)
	})

	t.Run("mismatch below threshold does not fatal", func(t *testing.T) {
		t.Parallel()
		next, fatal := updateParityMismatchStreak(1, true, limit)
		require.Equal(t, 2, next)
		require.False(t, fatal)
	})

	t.Run("reaching threshold triggers fatal", func(t *testing.T) {
		t.Parallel()
		next, fatal := updateParityMismatchStreak(2, true, limit)
		require.Equal(t, 3, next)
		require.True(t, fatal)
	})

	t.Run("transient after mismatches resets cleanly", func(t *testing.T) {
		t.Parallel()
		// Simulate: mismatch, mismatch, transient-failure (reset), mismatch
		n, fatal := updateParityMismatchStreak(0, true, limit) // 0 -> 1
		require.Equal(t, 1, n)
		require.False(t, fatal)
		n, fatal = updateParityMismatchStreak(n, true, limit) // 1 -> 2
		require.Equal(t, 2, n)
		require.False(t, fatal)
		n, fatal = updateParityMismatchStreak(n, false, limit) // transient -> reset to 0
		require.Equal(t, 0, n)
		require.False(t, fatal)
		n, fatal = updateParityMismatchStreak(n, true, limit) // 0 -> 1 (not 3)
		require.Equal(t, 1, n)
		require.False(t, fatal)
	})

	t.Run("different streakLimit changes fatal threshold", func(t *testing.T) {
		t.Parallel()
		// Exercises streakLimit as a real parameter rather than a constant: with
		// a higher limit, a single mismatch must not fatal even when it would
		// under the standard limit=3.
		const higher = 5
		next, fatal := updateParityMismatchStreak(2, true, higher)
		require.Equal(t, 3, next)
		require.False(t, fatal, "limit=5 must not fatal at streak=3")

		next, fatal = updateParityMismatchStreak(4, true, higher)
		require.Equal(t, 5, next)
		require.True(t, fatal, "limit=5 must fatal at streak=5")
	})
}

func TestGetChainManagerAddressMigration(t *testing.T) {
	// Backup and defer restore for chainManagerAddressMigrations
	originalMigrations := make(map[string]map[int64]ChainManagerAddressMigration)
	for k, v := range chainManagerAddressMigrations {
		cp := make(map[int64]ChainManagerAddressMigration)
		for kk, vv := range v {
			cp[kk] = vv
		}
		originalMigrations[k] = cp
	}
	defer func() { chainManagerAddressMigrations = originalMigrations }()

	// Backup and defer restore for conf.Custom
	originalCustom := conf.Custom
	defer func() { conf.Custom = originalCustom }()

	// Back up and restore viper flags
	originalChain := viper.GetString(ChainFlag)
	defer viper.Set(ChainFlag, originalChain)

	// Set up the test
	newPolContractAddress := "0x0000000000000000000000000000000000001234"
	chainManagerAddressMigrations["mumbai"] = map[int64]ChainManagerAddressMigration{
		350: {PolTokenAddress: newPolContractAddress},
	}

	InitTestHeimdallConfig("mumbai")
	migration, found := GetChainManagerAddressMigration(350)
	if !found {
		t.Errorf("Expected migration to be found")
	}
	if migration.PolTokenAddress != newPolContractAddress {
		t.Errorf("Expected pol token address to be %s, got %s", newPolContractAddress, migration.PolTokenAddress)
	}

	// test for non-existing migration
	_, found = GetChainManagerAddressMigration(351)
	if found {
		t.Errorf("Expected migration to not be found")
	}

	// test for the non-existing chain
	conf.Custom.BorRPCUrl = ""
	conf.Custom.Chain = ""

	viper.Set(ChainFlag, "newChain")
	InitTestHeimdallConfig("newChain")

	_, found = GetChainManagerAddressMigration(350)
	if found {
		t.Errorf("Expected migration to not be found")
	}
}

// TestDecorateWithHeimdallFlags_BorGRPCFlags verifies that DecorateWithHeimdallFlags
// registers the BorGRPC flags so that the flag lookup and viper binding work.
func TestDecorateWithHeimdallFlags_BorGRPCFlags(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "test"}
	v := viper.New()
	log := logger.NewNopLogger()

	DecorateWithHeimdallFlags(cmd, v, log, "test")

	// Verify BorGRPCUrlFlag was registered
	flag := cmd.PersistentFlags().Lookup(BorGRPCUrlFlag)
	require.NotNil(t, flag, "BorGRPCUrlFlag must be registered by DecorateWithHeimdallFlags")
	require.Equal(t, BorGRPCUrlFlag, flag.Name)

	// Verify BorGRPCFlagFlag was registered.
	flag = cmd.PersistentFlags().Lookup(BorGRPCFlagFlag)
	require.NotNil(t, flag, "BorGRPCFlagFlag must be registered by DecorateWithHeimdallFlags")

	// Verify BorGRPCTokenFlag was registered.
	flag = cmd.PersistentFlags().Lookup(BorGRPCTokenFlag)
	require.NotNil(t, flag, "BorGRPCTokenFlag must be registered by DecorateWithHeimdallFlags")

	// Verify the viper binding works: set a value and retrieve via viper.
	require.NoError(t, cmd.PersistentFlags().Set(BorGRPCUrlFlag, "grpc://example.com:9090"))
	require.Equal(t, "grpc://example.com:9090", v.GetString(BorGRPCUrlFlag))
}
