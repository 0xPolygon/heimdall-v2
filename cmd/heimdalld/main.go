package main

import (
	"fmt"
	"os"

	"github.com/0xPolygon/heimdall-v2/app"
	heimdalld "github.com/0xPolygon/heimdall-v2/cmd/heimdalld/cmd"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func main() {
	rootCmd := heimdalld.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "HD", app.DefaultNodeHome); err != nil {
		_, _ = fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
