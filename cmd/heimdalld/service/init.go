package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	cmdhelper "github.com/0xPolygon/heimdall-v2/cmd"
	"github.com/0xPolygon/heimdall-v2/helper"
	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/cli"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"

	// TODO HV2 - uncomment once staking module is available
	// stakingcli "github.com/0xPolygon/heimdall-v2/x/staking/client/cli"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type initHeimdallConfig struct {
	clientHome  string
	chainID     string
	validatorID int64
	chain       string
	forceInit   bool
}

// TODO HV2 - this function was heavily modified, review carefully
func heimdallInit(_ *server.Context, cdc *codec.LegacyAmino, initConfig *initHeimdallConfig, config *cmtcfg.Config, mbm module.BasicManager, cliCdc codec.Codec) error {
	conf := helper.GetDefaultHeimdallConfig()
	conf.Chain = initConfig.chain
	WriteDefaultHeimdallConfig(filepath.Join(config.RootDir, "config/heimdall-config.toml"), conf)

	nodeID, _, _, err := InitializeNodeValidatorFiles(config)
	if err != nil {
		return err
	}

	// do not execute init if forceInit is false and genesis.json already exists (or we do not have permission to write to file)
	writeGenesis := initConfig.forceInit

	if !writeGenesis {
		// When not forcing, check if genesis file exists
		_, err := os.Stat(config.GenesisFile())
		if err != nil && errors.Is(err, os.ErrNotExist) {
			logger.Info(fmt.Sprintf("Genesis file %v not found, writing genesis file\n", config.GenesisFile()))

			writeGenesis = true
		} else if err == nil {
			logger.Info(fmt.Sprintf("Found genesis file %v, skipping writing genesis file\n", config.GenesisFile()))
		} else {
			logger.Error(fmt.Sprintf("error checking if genesis file %v exists: %v\n", config.GenesisFile(), err))
			return err
		}
	} else {
		logger.Info(fmt.Sprintf("Force writing genesis file to %v\n", config.GenesisFile()))
	}

	if writeGenesis {
		genesisCreated, err := helper.WriteGenesisFile(initConfig.chain, config.GenesisFile())
		if err != nil {
			return err
		} else if genesisCreated {
			return nil
		}
	} else {
		return nil
	}

	// create chain id
	chainID := initConfig.chainID
	if chainID == "" {
		chainID = fmt.Sprintf("heimdall-%v", cmdhelper.RandStringRunes(6))
	}

	appGenState := mbm.DefaultGenesis(cliCdc)

	appState, err := json.MarshalIndent(appGenState, "", " ")
	if err != nil {
		return err
	}

	// app state json
	appStateJSON, err := json.Marshal(appState)
	if err != nil {
		return err
	}

	toPrint := struct {
		ChainID string `json:"chain_id"`
		NodeID  string `json:"node_id"`
	}{
		chainID,
		nodeID,
	}

	out, err := codec.MarshalJSONIndent(cdc, toPrint)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "%s\n", string(out))

	return genutil.ExportGenesisFileWithTime(config.GenesisFile(), chainID, nil, appStateJSON, cmttime.Now())
}

/*
TODO HV2 - check `func InitCmd(mbm module.BasicManager, defaultNodeHome string) *cobra.Command {}` in cosmos-sdk
maybe we can use it
*/

// InitCmd initialises files required to start heimdall
func initCmd(ctx *server.Context, cdc *codec.LegacyAmino, mbm module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize genesis config, priv-validator file, and p2p-node file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			cliCdc := clientCtx.Codec
			initConfig := &initHeimdallConfig{
				chainID: viper.GetString(flags.FlagChainID),
				chain:   viper.GetString(helper.ChainFlag),
				// TODO HV2 - uncomment once staking module is available
				// validatorID: viper.GetInt64(stakingcli.FlagValidatorID),
				clientHome: viper.GetString(helper.FlagClientHome),
				forceInit:  viper.GetBool(helper.OverwriteGenesisFlag),
			}
			config := ctx.Config
			config.SetRoot(viper.GetString(cli.HomeFlag))
			return heimdallInit(ctx, cdc, initConfig, config, mbm, cliCdc)
		},
	}

	cmd.Flags().String(cli.HomeFlag, helper.DefaultNodeHome, "node's home directory")
	cmd.Flags().String(helper.FlagClientHome, helper.DefaultCLIHome, "client's home directory")
	cmd.Flags().String(flags.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")
	// TODO HV2 - uncomment once staking module is available
	// cmd.Flags().Int(stakingcli.FlagValidatorID, 1, "--id=<validator ID here>, if left blank will be assigned 1")
	cmd.Flags().Bool(helper.OverwriteGenesisFlag, false, "overwrite the genesis.json file if it exists")

	return cmd
}
