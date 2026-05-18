package helper

import (
	"fmt"
	"testing"

	cfg "github.com/cometbft/cometbft/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestIsV080Hardfork(t *testing.T) {
	tests := []struct {
		name           string
		hardforkHeight int64
		queryHeight    int64
		want           bool
	}{
		{name: "dormant: hardfork height 0 stays false at height 0", hardforkHeight: 0, queryHeight: 0, want: false},
		{name: "dormant: hardfork height 0 stays false at large height", hardforkHeight: 0, queryHeight: 1_000_000, want: false},
		{name: "below hardfork height returns false", hardforkHeight: 100, queryHeight: 99, want: false},
		{name: "exact hardfork height returns true", hardforkHeight: 100, queryHeight: 100, want: true},
		{name: "above hardfork height returns true", hardforkHeight: 100, queryHeight: 101, want: true},
	}

	original := GetV080HardforkHeight()
	t.Cleanup(func() {
		SetV080HardforkHeight(original)
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			SetV080HardforkHeight(tc.hardforkHeight)
			require.Equal(t, tc.want, IsV080Hardfork(tc.queryHeight))
		})
	}
}

// TestHeimdallConfig checks heimdall configs
func TestHeimdallConfig(t *testing.T) {
	t.Parallel()

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

	// Backup and restore viper flags
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
