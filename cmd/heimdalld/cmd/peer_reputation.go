package heimdalld

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	"github.com/0xPolygon/heimdall-v2/peerpolicy"
)

const (
	peerReputationFlag        = "peer-reputation"
	peerReputationEnforceFlag = "peer-reputation-enforce"
)

func addPeerReputationFlag(cmd *cobra.Command) {
	cmd.Flags().Bool(peerReputationFlag, true, "Score native peers (disable with --peer-reputation=false)")
	cmd.Flags().Bool(peerReputationEnforceFlag, true, "Disconnect and reject peers at risk 100 (observation only with --peer-reputation-enforce=false)")
}

func installPeerReputation(cmd *cobra.Command, ctx *server.Context) error {
	return configurePeerReputation(cmd, ctx, prometheus.DefaultRegisterer)
}

func configurePeerReputation(cmd *cobra.Command, ctx *server.Context, registry prometheus.Registerer) error {
	if cmd.Name() != "start" {
		return nil
	}
	enabled, err := cmd.Flags().GetBool(peerReputationFlag)
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	enforce, err := cmd.Flags().GetBool(peerReputationEnforceFlag)
	if err != nil {
		return err
	}
	withComet, err := cmd.Flags().GetBool("with-comet")
	if err != nil {
		return err
	}
	grpcOnly, err := cmd.Flags().GetBool("grpc-only")
	if err != nil {
		return err
	}
	if !withComet || grpcOnly {
		if cmd.Flags().Changed(peerReputationFlag) || cmd.Flags().Changed(peerReputationEnforceFlag) {
			return fmt.Errorf("--peer-reputation requires in-process CometBFT")
		}
		return nil
	}
	tracker, err := peerpolicy.New(registry, enforce)
	if err != nil {
		return fmt.Errorf("register peer reputation metrics: %w", err)
	}
	ctx.Config.P2P.PeerObserver = tracker
	if enforce {
		ctx.Config.P2P.PeerPolicy = tracker
	}
	ctx.Logger.Info("Peer reputation enabled", "connection_enforcement", enforce)
	return nil
}
