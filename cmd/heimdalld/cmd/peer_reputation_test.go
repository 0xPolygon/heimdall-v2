package heimdalld

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestPeerReputationStartup(t *testing.T) {
	t.Parallel()
	registry := prometheus.NewRegistry()
	ctx := server.NewDefaultContext()
	cmd := &cobra.Command{Use: "start"}
	cmd.Flags().Bool(peerReputationFlag, false, "")
	cmd.Flags().Bool("with-comet", true, "")
	cmd.Flags().Bool("grpc-only", false, "")
	require.NoError(t, configurePeerReputation(cmd, ctx, registry))
	require.Nil(t, ctx.Config.P2P.PeerObserver)
	require.NoError(t, cmd.Flags().Set(peerReputationFlag, "true"))
	require.NoError(t, cmd.Flags().Set("with-comet", "false"))
	require.ErrorContains(t, configurePeerReputation(cmd, ctx, registry), "in-process CometBFT")
	require.Nil(t, ctx.Config.P2P.PeerObserver)
	require.NoError(t, cmd.Flags().Set("with-comet", "true"))
	require.NoError(t, cmd.Flags().Set("grpc-only", "true"))
	require.ErrorContains(t, configurePeerReputation(cmd, ctx, registry), "in-process CometBFT")
	require.NoError(t, cmd.Flags().Set("grpc-only", "false"))
	require.NoError(t, configurePeerReputation(cmd, ctx, registry))
	require.NotNil(t, ctx.Config.P2P.PeerObserver)
	require.ErrorContains(t, configurePeerReputation(cmd, ctx, registry), "register peer reputation metrics")
	require.NoError(t, installPeerReputation(&cobra.Command{Use: "version"}, ctx))
	require.Error(t, configurePeerReputation(&cobra.Command{Use: "start"}, ctx, registry))
}

func TestPeerReputationMissingStartupFlags(t *testing.T) {
	t.Parallel()
	ctx := server.NewDefaultContext()
	cmd := &cobra.Command{Use: "start"}
	require.Error(t, installPeerReputation(cmd, ctx))
	cmd.Flags().Bool(peerReputationFlag, true, "")
	require.Error(t, configurePeerReputation(cmd, ctx, prometheus.NewRegistry()))
	cmd.Flags().Bool("with-comet", true, "")
	require.Error(t, configurePeerReputation(cmd, ctx, prometheus.NewRegistry()))
}
