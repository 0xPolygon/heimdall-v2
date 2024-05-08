package main

import (
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/version"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewRootCmd creates a new root command for simd. It is called once in the main function.
func NewRootCmd() *cobra.Command {
	var (
		// autoCliOpts        autocli.AppOptions
		moduleBasicManager module.BasicManager
		clientCtx          client.Context
	)

	rootCmd := &cobra.Command{
		Use:   "heimdallcli",
		Short: "Heimdall light-client",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Use != version.Cmd.Use {
				// initialise config
				initTendermintViperConfig(cmd)
			}

			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			clientCtx = clientCtx.WithCmdContext(cmd.Context())
			clientCtx, err := client.ReadPersistentCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			clientCtx, err = config.ReadFromClientConfig(clientCtx)
			if err != nil {
				return err
			}

			if err := client.SetCmdClientContextHandler(clientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := initAppConfig()
			customCMTConfig := initCometBFTConfig()

			return server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customCMTConfig)
		},
	}

	// TODO HV2 - once all the commands are added, see if we need to pass the whole clientCtx or just the necessary parts
	// initRootCmd(rootCmd, clientCtx.TxConfig, clientCtx.InterfaceRegistry, clientCtx.Codec, moduleBasicManager)
	initRootCmd(rootCmd, clientCtx, moduleBasicManager)

	// TODO HV2 - I guesss we can remove this
	/*
		if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
			panic(err)
		}
	*/

	return rootCmd
}

func initTendermintViperConfig(cmd *cobra.Command) {
	tendermintNode, _ := cmd.Flags().GetString(helper.TendermintNodeFlag)
	homeValue, _ := cmd.Flags().GetString(helper.HomeFlag)

	// set to viper
	viper.Set(helper.TendermintNodeFlag, tendermintNode)
	viper.Set(helper.HomeFlag, homeValue)

	// start heimdall config
	helper.InitHeimdallConfig("")
}
