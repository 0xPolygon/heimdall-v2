package helper

import (
	"fmt"
	"os"
	"testing"

	cfg "github.com/cometbft/cometbft/config"
	"github.com/spf13/viper"
)

// TestHeimdallConfig checks heimdall configs
func TestHeimdallConfig(t *testing.T) {
	t.Parallel()

	// cli context
	cometBFTNode := "tcp://localhost:26657"
	viper.Set(CometBFTNodeFlag, cometBFTNode)
	viper.Set("log_level", "info")

	InitHeimdallConfig(os.ExpandEnv("$HOME/var/lib/heimdall"))

	fmt.Println("Address", GetAddress())

	pubKey := GetPubKey()

	fmt.Println("PublicKey", pubKey.String())
}

func TestHeimdallConfigUpdateCometBFTConfig(t *testing.T) {
	t.Parallel()

	type teststruct struct {
		chain string
		viper string
		def   string
		value string
	}

	data := []teststruct{
		{chain: "mumbai", viper: "viper", def: "default", value: "viper"},
		{chain: "mumbai", viper: "viper", def: "", value: "viper"},
		{chain: "mumbai", viper: "", def: "default", value: "default"},
		{chain: "mumbai", viper: "", def: "", value: DefaultMumbaiTestnetSeeds},
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
	t.Parallel()

	newPolContractAddress := "0x0000000000000000000000000000000000001234"

	chainManagerAddressMigrations["mumbai"] = map[int64]ChainManagerAddressMigration{
		350: {PolTokenAddress: newPolContractAddress},
	}

	viper.Set("chain", "mumbai")
	InitHeimdallConfig(os.ExpandEnv("$HOME/var/lib/heimdall"))

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

	// test for non-existing chain
	conf.Custom.BorRPCUrl = ""

	viper.Set("chain", "newChain")
	InitHeimdallConfig(os.ExpandEnv("$HOME/var/lib/heimdall"))

	_, found = GetChainManagerAddressMigration(350)
	if found {
		t.Errorf("Expected migration to not be found")
	}
}
