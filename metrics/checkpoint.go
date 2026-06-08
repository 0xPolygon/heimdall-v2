package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Bridge checkpoint-lag gauges expose how far this node's committed checkpoint
// state trails the latest checkpoint finalized on L1. They are a slow
// authoritative backstop to the peer-height catching_up signal: a node whose
// consensus P2P is partitioned keeps an intact L1 link, so a growing lag here
// flags "behind finalized state" even when peer-height is momentarily fooled by
// not-yet-evicted stale peers.
var (
	// BridgeCheckpointLagBlocks is the raw Bor-block extent between the latest
	// L1-finalized checkpoint end and this node's committed checkpoint end. It is
	// positive even in normal operation (the committed checkpoint trails L1's
	// latest by ~one in-flight checkpoint), so alert on the effective gauge below.
	BridgeCheckpointLagBlocks = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "checkpoint_lag_blocks",
		Help:      "Bor-block extent between the latest L1-finalized checkpoint end and this node's committed checkpoint end (raw; positive in normal operation)",
	})

	// BridgeCheckpointEffectiveLagBlocks equals the raw lag only when the node is
	// genuinely behind (checkpoint id gap >= 2), and 0 otherwise. The normal
	// one-in-flight checkpoint window is suppressed, so this is the alert-safe gauge.
	BridgeCheckpointEffectiveLagBlocks = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: "bridge",
		Name:      "checkpoint_effective_lag_blocks",
		Help:      "Bor-block checkpoint lag, reported only when behind by more than the normal one-in-flight checkpoint (id gap >= 2); 0 otherwise",
	})
)
