package main

import (
	"fmt"
	"os"

	"github.com/0xPolygon/heimdall-v2/app"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func main() {
	rootCmd := NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "HD", os.ExpandEnv(app.DefaultNodeHome)); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
