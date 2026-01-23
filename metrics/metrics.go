package metrics

import (
	"github.com/0xPolygon/heimdall-v2/version"
)

const (
	// Namespace is the prefix for all metrics.
	Namespace = "heimdallv2"
)

func InitMetrics() {
	// Update version Info gauge.
	UpdateHeimdallV2Info(version.Version, version.Commit)
}
