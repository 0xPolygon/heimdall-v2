package keeper

import (
	"context"
	"time"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
)

// EndBlocker called at the end of every block, and returns validator updates
func (k *Keeper) EndBlocker(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	k.PanicIfSetupIsIncomplete()
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)
	return k.ApplyAndReturnValidatorSetUpdates(ctx)
}
