package heimdalld

import (
	"bytes"
	"encoding/json"
	"testing"

	sdklog "cosmossdk.io/log"
	"github.com/0xPolygon/heimdall-v2/peerpolicy"
	"github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/p2p/servebudget"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func servingContext(t *testing.T, args []string) (*cobra.Command, *server.Context, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "start"}
	addPeerServingFlags(cmd)
	cmd.Flags().Bool("with-comet", true, "")
	cmd.Flags().Bool("grpc-only", false, "")
	require.NoError(t, cmd.ParseFlags(args))
	v := viper.New()
	require.NoError(t, v.BindPFlags(cmd.Flags()))
	output := &bytes.Buffer{}
	ctx := &server.Context{Config: config.DefaultConfig(), Viper: v, Logger: sdklog.NewLogger(output, sdklog.OutputJSONOption())}
	return cmd, ctx, output
}

func TestPeerServingStartup(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		mode string
	}{
		{"default", nil, "observe"},
		{"scoring_disabled", []string{"--peer-reputation=false"}, "disabled"},
		{"serving_disabled", []string{"--peer-serving=false"}, ""},
		{"external", []string{"--with-comet=false"}, ""},
		{"query", []string{"--grpc-only"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd, ctx, output := servingContext(t, tc.args)
			require.NoError(t, installPeerServing(cmd, ctx, prometheus.NewRegistry()))
			if tc.mode == "" {
				require.Nil(t, ctx.Config.P2P.ServingPolicy)
				require.Empty(t, output.String())
				return
			}
			g := ctx.Config.P2P.ServingPolicy.(*peerpolicy.Governor)
			const id = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			require.Equal(t, tc.mode, g.Snapshot(id).Mode)
			g.Observe(id, servebudget.InvalidResponse)
			expectedRisk := uint64(60)
			if tc.mode == "disabled" {
				expectedRisk = 0
			}
			require.Equal(t, expectedRisk, g.Snapshot(id).Risk)
			var record map[string]interface{}
			require.NoError(t, json.Unmarshal(output.Bytes(), &record))
			require.Equal(t, tc.mode, record["reputation_mode"])
		})
	}
}

func TestPeerServingStartupErrors(t *testing.T) {
	for _, args := range [][]string{
		{"--with-comet=false", "--peer-serving=true"},
		{"--grpc-only", "--peer-serving=true"},
		{"--peer-serving-inflight=0"},
	} {
		t.Run(args[0], func(t *testing.T) {
			cmd, ctx, output := servingContext(t, args)
			require.Error(t, installPeerServing(cmd, ctx, prometheus.NewRegistry()))
			require.Nil(t, ctx.Config.P2P.ServingPolicy)
			require.Empty(t, output.String())
		})
	}
	t.Run("non_start", func(t *testing.T) {
		cmd, ctx, _ := servingContext(t, nil)
		cmd.Use = "status"
		require.NoError(t, installPeerServing(cmd, ctx, prometheus.NewRegistry()))
		require.Nil(t, ctx.Config.P2P.ServingPolicy)
	})
	t.Run("registration", func(t *testing.T) {
		cmd, ctx, _ := servingContext(t, nil)
		registry := prometheus.NewRegistry()
		_, err := peerpolicy.New(peerpolicy.DefaultConfig(), registry)
		require.NoError(t, err)
		require.ErrorContains(t, installPeerServing(cmd, ctx, registry), "configure peer serving")
		require.Nil(t, ctx.Config.P2P.ServingPolicy)
	})
}
