package heimdalld

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	"github.com/0xPolygon/heimdall-v2/peerpolicy"
)

const peerReputationFlag = "peer-reputation"

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
	withComet, err := cmd.Flags().GetBool("with-comet")
	if err != nil {
		return err
	}
	grpcOnly, err := cmd.Flags().GetBool("grpc-only")
	if err != nil {
		return err
	}
	if !withComet || grpcOnly {
		return fmt.Errorf("--peer-reputation requires in-process CometBFT")
	}
	tracker, err := peerpolicy.New(registry)
	if err != nil {
		return fmt.Errorf("register peer reputation metrics: %w", err)
	}
	ctx.Config.P2P.PeerObserver = tracker
	ctx.Logger.Info("Peer reputation observation enabled; scoring does not enforce limits")
	return nil
}
