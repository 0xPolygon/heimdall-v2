package helper

import (
	"fmt"
	"os"
	"testing"

	"github.com/spf13/viper"

	cfg "github.com/cometbft/cometbft/config"
)

// TestHeimdallConfig checks heimdall configs
func TestHeimdallConfig(t *testing.T) {
	// TODO HV2: fix this test as it currently depends on the config file
	//  See https://polygon.atlassian.net/browse/POS-2626
	t.Skip("to be enabled")
	t.Parallel()

	// cli context
	cometBFTNode := "tcp://localhost:26657"
	viper.Set(CometBFTNodeFlag, cometBFTNode)
	viper.Set("log_level", "info")

	InitHeimdallConfig(os.ExpandEnv("$HOME/.heimdalld"))

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
	// TODO HV2: fix this test as it currently depends on the config file
	//  See https://polygon.atlassian.net/browse/POS-2626
	t.Skip("to be enabled")
	t.Parallel()

	newMaticContractAddress := "0x0000000000000000000000000000000000001234"

	chainManagerAddressMigrations["mumbai"] = map[int64]ChainManagerAddressMigration{
		350: {PolygonPosTokenAddress: newMaticContractAddress},
	}

	viper.Set("chain", "mumbai")
	InitHeimdallConfig(os.ExpandEnv("$HOME/.heimdalld"))

	migration, found := GetChainManagerAddressMigration(350)

	if !found {
		t.Errorf("Expected migration to be found")
	}

	if migration.PolygonPosTokenAddress != newMaticContractAddress {
		t.Errorf("Expected matic token address to be %s, got %s", newMaticContractAddress, migration.PolygonPosTokenAddress)
	}

	// test for non-existing migration
	_, found = GetChainManagerAddressMigration(351)
	if found {
		t.Errorf("Expected migration to not be found")
	}

	// test for non-existing chain
	conf.Custom.BorRPCUrl = ""

	viper.Set("chain", "newChain")
	InitHeimdallConfig(os.ExpandEnv("$HOME/.heimdalld"))

	_, found = GetChainManagerAddressMigration(350)
	if found {
		t.Errorf("Expected migration to not be found")
	}
}
