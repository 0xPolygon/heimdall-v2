package main

import (
	"fmt"
	"os"

	"github.com/spf13/viper"

	"github.com/0xPolygon/heimdall-v2/bridge/cmd"
	"github.com/0xPolygon/heimdall-v2/helper"
)

func main() {
	logger := helper.Logger.With("module", "bridge/cmd/")
	rootCmd := cmd.BridgeCommands(viper.GetViper(), logger, "bridge-main")

	// add heimdall flags
	helper.DecorateWithHeimdallFlags(rootCmd, viper.GetViper(), logger, "bridge-main")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
