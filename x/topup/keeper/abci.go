package keeper

import (
	"context"
	"github.com/0xPolygon/heimdall-v2/x/topup/types"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
)

// BeginBlocker in x/topup module only initiates the telemetry metrics and returns, returning no errors
func (k *Keeper) BeginBlocker(_ context.Context) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)
	return nil
}

// EndBlocker is called at every block. For x/topup module, it initiates the telemetry metrics and
// returns no updates of the validator set
func (k *Keeper) EndBlocker(_ context.Context) ([]abci.ValidatorUpdate, error) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)
	return []abci.ValidatorUpdate{}, nil
}
