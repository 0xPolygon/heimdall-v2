package metrics

import (
	"github.com/0xPolygon/heimdall-v2/metrics/api"
	"github.com/0xPolygon/heimdall-v2/version"
)

func InitMetrics() {
	// Update Heimdallv2 Version Info gauge.
	UpdateHeimdallV2Info(version.Version, version.Commit)

	// Init API Module Metrics.
	api.InitAPIModuleMetrics()
}
