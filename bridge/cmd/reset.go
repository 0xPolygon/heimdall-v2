package cmd

import (
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/0xPolygon/heimdall-v2/bridge/util"
)

// resetCmd resets the bridge server data
var resetCmd = &cobra.Command{
	Use:   "unsafe-reset-all",
	Short: "Reset bridge server data",
	RunE: func(cmd *cobra.Command, _ []string) error {
		dbLocation := viper.GetString(util.BridgeDBFlag)
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
