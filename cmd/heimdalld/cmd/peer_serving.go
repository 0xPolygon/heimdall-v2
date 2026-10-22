package heimdalld

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	"github.com/0xPolygon/heimdall-v2/peerpolicy"
)

func addPeerServingFlags(cmd *cobra.Command) {
	d := peerpolicy.DefaultConfig()
	f := cmd.Flags()
	f.Bool("peer-serving", true, "Apply shared native bulk serving budgets")
	f.Bool("peer-reputation", true, "Observe native peer scores without disconnecting or jailing peers")
	f.Uint64("peer-serving-byte-rate", d.BytesPerSecond, "Node-wide native bulk bytes per second")
	f.Uint64("peer-serving-byte-burst", d.BurstBytes, "Node-wide native bulk byte burst")
	f.Uint64("peer-serving-memory", d.MemoryBytes, "Native serving payload reservation capacity in bytes")
	f.Uint64("peer-serving-inflight", d.MaxInflight, "Maximum native serving reservations")
	f.Uint64("peer-serving-request-rate", d.RequestsPerSecond, "Node-wide native bulk work admissions per second")
	f.Uint64("peer-serving-peer-byte-rate", d.PeerBytesPerSecond, "Per-peer bulk bytes per second")
	f.Uint64("peer-serving-peer-byte-burst", d.PeerBurstBytes, "Per-peer bulk byte burst")
	f.Uint64("peer-serving-peer-request-rate", d.PeerRequestsPerSecond, "Per-peer native bulk work admissions per second")
	f.Uint64("peer-serving-repeat-bytes", d.RepeatBytes, "Repeated-object bytes per peer per ten-second interval after one retry")
	f.StringSlice("peer-serving-protected-ids", nil, "Authenticated CometBFT node IDs sharing reserved operator headroom")
}

func installPeerServing(cmd *cobra.Command, ctx *server.Context, reg prometheus.Registerer) error {
	if cmd.Name() != "start" {
		return nil
	}
	if !ctx.Viper.GetBool("peer-serving") {
		return nil
	}
	if !ctx.Viper.GetBool("with-comet") || ctx.Viper.GetBool("grpc-only") {
		if cmd.Flags().Changed("peer-serving") {
			return fmt.Errorf("peer serving requires in-process CometBFT")
		}
		return nil
	}
	cfg := peerServingConfig(ctx)
	governor, err := peerpolicy.New(cfg, reg)
	if err != nil {
		return fmt.Errorf("configure peer serving: %w", err)
	}
	ctx.Config.P2P.ServingPolicy = governor
	ctx.Logger.Info("Native peer serving enabled", "reputation_mode", governor.Mode())
	return nil
}

func peerServingConfig(ctx *server.Context) peerpolicy.Config {
	v := ctx.Viper
	return peerpolicy.Config{
		Observe:        v.GetBool("peer-reputation"),
		BytesPerSecond: v.GetUint64("peer-serving-byte-rate"), BurstBytes: v.GetUint64("peer-serving-byte-burst"),
		MemoryBytes: v.GetUint64("peer-serving-memory"), MaxInflight: v.GetUint64("peer-serving-inflight"),
		RequestsPerSecond: v.GetUint64("peer-serving-request-rate"), PeerBytesPerSecond: v.GetUint64("peer-serving-peer-byte-rate"),
		PeerBurstBytes: v.GetUint64("peer-serving-peer-byte-burst"), PeerRequestsPerSecond: v.GetUint64("peer-serving-peer-request-rate"),
		RepeatBytes: v.GetUint64("peer-serving-repeat-bytes"), ProtectedIDs: v.GetStringSlice("peer-serving-protected-ids"),
	}
}
