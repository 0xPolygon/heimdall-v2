package metrics

import (
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	// PreBlockerTimer tracks the time taken by PreBlocker function.
	PreBlockerTimer = metrics.NewRegisteredTimer("heimdallv2/abci/preblocker", nil)

	// BeginBlockerTimer tracks the time taken by BeginBlocker function.
	BeginBlockerTimer = metrics.NewRegisteredTimer("heimdallv2/abci/beginblocker", nil)

	// EndBlockerTimer tracks the time taken by EndBlocker function.
	EndBlockerTimer = metrics.NewRegisteredTimer("heimdallv2/abci/endblocker", nil)
)
