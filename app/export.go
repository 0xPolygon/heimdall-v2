package app

import (
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	jsoniter "github.com/json-iterator/go"
)

// ExportAppStateAndValidators exports the state of the application for a genesis
// file.
func (app *HeimdallApp) ExportAppStateAndValidators(
	forZeroHeight bool,
	jailAllowedAddrs []string,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	// as if they could withdraw from the start of the next block
	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})

	// We export at last height + 1, because that's the height at which
	// Tendermint will start InitChain.
	height := app.LastBlockHeight() + 1
	if forZeroHeight {
		height = 0
		app.prepForZeroHeightGenesis(ctx, jailAllowedAddrs)
	}

	genState, err := app.mm.ExportGenesisForModules(ctx, app.appCodec, modulesToExport)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}
	appState, err := jsoniter.ConfigFastest.MarshalIndent(genState, "", "  ")
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	// TODO: uncomment when implemented
	// validators, err := staking.WriteValidators(ctx, app.StakingKeeper)
	return servertypes.ExportedApp{
		AppState: appState,
		// TODO: uncomment when implemented
		// Validators:      validators,
		Height:          height,
		ConsensusParams: app.BaseApp.GetConsensusParams(ctx),
	}, err
}

// prepare for fresh start at zero height
// NOTE zero height genesis is a temporary feature which will be deprecated
// in favour of export at a block height
// TODO: What would a "fresh start at zero height" mean for Heimdall ?
// What would we need to preserve (checkpoints, state sync, validator state) and what data would be reset ?
// Decide and implement accordingly
func (app *HeimdallApp) prepForZeroHeightGenesis(ctx sdk.Context, jailAllowedAddrs []string) {
	// TODO: uncomment when implemented
	// applyAllowedAddrs := false

	// // check if there is a allowed address list
	// if len(jailAllowedAddrs) > 0 {
	// 	applyAllowedAddrs = true
	// }

	allowedAddrsMap := make(map[string]bool)

	for _, addr := range jailAllowedAddrs {
		_, err := sdk.ValAddressFromBech32(addr)
		if err != nil {
			panic(err)
		}
		allowedAddrsMap[addr] = true
	}

	/* Handle fee distribution state. */

	// withdraw all validator commission
	//nolint:ineffassign
	// TODO: uncomment when implemented
	// err := app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
	// 	valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	_, _ = app.DistrKeeper.WithdrawValidatorCommission(ctx, valBz)
	// 	return false
	// })
	// if err != nil {
	// 	panic(err)
	// }

	// TODO: uncomment when implemented
	// withdraw all delegator rewards
	// dels, err := app.StakingKeeper.GetAllDelegations(ctx)
	// for _, delegation := range dels {
	// 	valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	delAddr, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	_, err = app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, valAddr)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }

	// clear validator slash events
	app.DistrKeeper.DeleteAllValidatorSlashEvents(ctx)

	// clear validator historical rewards
	app.DistrKeeper.DeleteAllValidatorHistoricalRewards(ctx)

	// set context height to zero
	height := ctx.BlockHeight()
	ctx = ctx.WithBlockHeight(0)

	// TODO: uncomment when implemented
	// reinitialize all validators
	// err = app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
	// 	valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	// donate any unwithdrawn outstanding reward fraction tokens to the community pool
	// 	scraps, err := app.DistrKeeper.GetValidatorOutstandingRewardsCoins(ctx, valBz)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	feePool, err := app.DistrKeeper.FeePool.Get(ctx)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	feePool.CommunityPool = feePool.CommunityPool.Add(scraps...)
	// 	if err := app.DistrKeeper.FeePool.Set(ctx, feePool); err != nil {
	// 		panic(err)
	// 	}

	// 	if err := app.DistrKeeper.Hooks().AfterValidatorCreated(ctx, valBz); err != nil {
	// 		panic(err)
	// 	}
	// 	return false
	// })

	// TODO: uncomment when implemented
	// reinitialize all delegations
	// for _, del := range dels {
	// 	valAddr, err := sdk.ValAddressFromBech32(del.ValidatorAddress)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	delAddr, err := sdk.AccAddressFromBech32(del.DelegatorAddress)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	if err := app.DistrKeeper.Hooks().BeforeDelegationCreated(ctx, delAddr, valAddr); err != nil {
	// 		panic(err)
	// 	}
	// 	if err := app.DistrKeeper.Hooks().AfterDelegationModified(ctx, delAddr, valAddr); err != nil {
	// 		panic(err)
	// 	}
	// }

	// reset context height
	ctx = ctx.WithBlockHeight(height)

	/* Handle staking state. */

	// TODO: uncomment when implemented
	// iterate through redelegations, reset creation height
	//nolint:errcheck
	// app.StakingKeeper.IterateRedelegations(ctx, func(_ int64, red stakingtypes.Redelegation) (stop bool) {
	// 	for i := range red.Entries {
	// 		red.Entries[i].CreationHeight = 0
	// 	}
	// 	app.StakingKeeper.SetRedelegation(ctx, red) //nolint:errcheck
	// 	return false
	// })

	// // iterate through unbonding delegations, reset creation height
	// //nolint:errcheck
	// app.StakingKeeper.IterateUnbondingDelegations(ctx, func(_ int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
	// 	for i := range ubd.Entries {
	// 		ubd.Entries[i].CreationHeight = 0
	// 	}
	// 	err = app.StakingKeeper.SetUnbondingDelegation(ctx, ubd)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	return false
	// })

	// Iterate through validators by power descending, reset bond heights, and
	// update bond intra-tx counters.
	store := ctx.KVStore(app.GetKey(stakingtypes.StoreKey))
	iter := storetypes.KVStoreReversePrefixIterator(store, stakingtypes.ValidatorsKey)
	// TODO: uncomment when implemented
	// counter := int16(0)

	// // Closure to ensure iterator doesn't leak.
	// for ; iter.Valid(); iter.Next() {
	// 	addr := sdk.ValAddress(stakingtypes.AddressFromValidatorsKey(iter.Key()))
	// 	validator, err := app.StakingKeeper.GetValidator(ctx, addr)
	// 	if err != nil {
	// 		panic("expected validator, not found")
	// 	}

	// 	validator.UnbondingHeight = 0
	// 	if applyAllowedAddrs && !allowedAddrsMap[addr.String()] {
	// 		validator.Jailed = true
	// 	}

	// 	app.StakingKeeper.SetValidator(ctx, validator) //nolint:errcheck
	// 	counter++
	// }

	if err := iter.Close(); err != nil {
		app.Logger().Error("error while closing the key-value store reverse prefix iterator: ", err)
		return
	}
	/* Handle slashing state. */
}
