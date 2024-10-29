package cmd

import (
	"os"
	"path"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
)

// resetCmd resets the bridge server data
var resetCmd = &cobra.Command{
	Use:   "unsafe-reset-all",
	Short: "Reset bridge server data",
	RunE: func(cmd *cobra.Command, _ []string) error {
		serverCtx := server.GetServerContextFromCmd(cmd)

		dbLocation := serverCtx.Viper.GetString(bridgeDBFlag)
		dir, err := os.ReadDir(dbLocation)
		if err != nil {
			return err
		}

		for _, d := range dir {
			err = os.RemoveAll(path.Join([]string{dbLocation, d.Name()}...))
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}
