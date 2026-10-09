package heimdalld

import (
	sdklog "cosmossdk.io/log"
	"github.com/0xPolygon/heimdall-v2/peerpolicy"
	"github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPeerServingStartup(t *testing.T) {
	for _, tc := range []struct {
		name      string
		args      []string
		installed bool
	}{{"default", nil, true}, {"disabled", []string{"--peer-serving=false"}, false}, {"external", []string{"--with-comet=false"}, false}, {"query", []string{"--grpc-only"}, false}} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "start"}
			addPeerServingFlags(cmd)
			cmd.Flags().Bool("with-comet", true, "")
			cmd.Flags().Bool("grpc-only", false, "")
			require.NoError(t, cmd.ParseFlags(tc.args))
			v := viper.New()
			require.NoError(t, v.BindPFlags(cmd.Flags()))
			ctx := &server.Context{Config: config.DefaultConfig(), Viper: v, Logger: sdklog.NewNopLogger()}
			require.NoError(t, installPeerServing(cmd, ctx, prometheus.NewRegistry()))
			if !tc.installed {
				require.Nil(t, ctx.Config.P2P.ServingPolicy)
				return
			}
			g := ctx.Config.P2P.ServingPolicy.(*peerpolicy.Governor)
			g.Observe("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", servebudget.InvalidResponse)
			require.Equal(t, "observe", g.Snapshot("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").Mode)
		})
	}
}
