package app

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/helper"
)

// normalizeCancelCharge zeroes the proposal cancel ratio from Purley while no ProposalCancelDest is set,
// since the charge would otherwise stay locked in the gov module account.
// It runs in every PreBlocker: gov applies MsgUpdateParams in EndBlocker, so a later update that restores
// a ratio without a destination is undone before the next block's txs (queries show it for that one block).
func (app *HeimdallApp) normalizeCancelCharge(ctx sdk.Context, height int64) error {
	if !helper.IsPurley(height) {
		return nil
	}
	params, err := app.GovKeeper.Params.Get(ctx)
	if err != nil {
		return err
	}
	if params.ProposalCancelDest != "" {
		return nil
	}
	if ratio, err := math.LegacyNewDecFromStr(params.ProposalCancelRatio); err == nil && ratio.IsZero() {
		return nil
	}
	params.ProposalCancelRatio = math.LegacyZeroDec().String()
	return app.GovKeeper.Params.Set(ctx, params)
}
