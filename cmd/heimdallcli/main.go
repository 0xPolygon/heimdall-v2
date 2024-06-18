package main

import (
	"fmt"
	"os"

	cmdhelper "github.com/0xPolygon/heimdall-v2/cmd"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func main() {
	rootCmd := NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "HD", os.ExpandEnv(cmdhelper.GetDefaultHomeDir())); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
