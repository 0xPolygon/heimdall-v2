package main

import (
	"context"
	"fmt"
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/0xPolygon/heimdall-v2/app"
	heimdalld "github.com/0xPolygon/heimdall-v2/cmd/heimdalld/cmd"
)

func main() {
	rootCmd, heimdallApp := heimdalld.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "HD", app.DefaultNodeHome); err != nil {
		_, _ = fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}

	go func(ctx context.Context) {
		heimdallApp.ProduceELPayload(ctx)
	}(rootCmd.Context())
}
