package heimdalld

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func peerReputationCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "start"}
	addPeerReputationFlag(cmd)
	cmd.Flags().Bool("with-comet", true, "")
	cmd.Flags().Bool("grpc-only", false, "")
	return cmd
}

func TestPeerReputationStartup(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		args          []string
		enabled, fail bool
	}{
		{"default", nil, true, false},
		{"disabled", []string{"--peer-reputation=false"}, false, false},
		{"explicit", []string{"--peer-reputation=true"}, true, false},
		{"external-comet", []string{"--with-comet=false"}, false, false},
		{"query-only", []string{"--grpc-only"}, false, false},
		{"explicit-external", []string{"--with-comet=false", "--peer-reputation=true"}, false, true},
		{"explicit-query", []string{"--grpc-only", "--peer-reputation=true"}, false, true},
		{"disabled-query", []string{"--grpc-only", "--peer-reputation=false"}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := server.NewDefaultContext()
			cmd := peerReputationCommand()
			require.NoError(t, cmd.ParseFlags(tc.args))
			registry := prometheus.NewRegistry()
			err := configurePeerReputation(cmd, ctx, registry)
			if tc.fail {
				require.ErrorContains(t, err, "in-process CometBFT")
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.enabled, ctx.Config.P2P.PeerObserver != nil)
			if tc.enabled {
				require.ErrorContains(t, configurePeerReputation(cmd, ctx, registry), "register peer reputation metrics")
			}
		})
	}
}

func TestPeerReputationMissingStartupFlags(t *testing.T) {
	t.Parallel()
	ctx := server.NewDefaultContext()
	cmd := &cobra.Command{Use: "start"}
	require.Error(t, installPeerReputation(cmd, ctx))
	addPeerReputationFlag(cmd)
	require.Error(t, configurePeerReputation(cmd, ctx, prometheus.NewRegistry()))
	cmd.Flags().Bool("with-comet", true, "")
	require.Error(t, configurePeerReputation(cmd, ctx, prometheus.NewRegistry()))
	require.NoError(t, installPeerReputation(&cobra.Command{Use: "version"}, ctx))
}
