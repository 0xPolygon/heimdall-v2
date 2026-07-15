package keeper

import (
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// eligibleProducerFallback returns a deterministic candidate set for the degenerate case
// where producer election leaves no selectable candidate other than the current producer
// (the elected set has collapsed to the lone incumbent, or every alternative was filtered
// out). It draws from the caller's active set (the milestone supporters) first, then the
// full validator set, each sorted ascending with the current and excluded producers removed.
//
// Only the future-span PreBlocker path reaches this, where an empty candidate set is a fatal
// chain halt. There it is always non-empty: a >2/3 milestone that excludes the current
// producer leaves a non-empty supporting set, so SelectProducer never receives an empty slice
// and the halt cannot occur. The validator-set step is defense-in-depth for that guarantee;
// the non-fatal rotation paths do not opt into this fallback and keep their skip-and-retry
// behavior.
func (k *Keeper) eligibleProducerFallback(ctx sdk.Context, currentProducer uint64, activeValidatorIDs, excludedProducerIDs map[uint64]struct{}) []uint64 {
	if c := sortedEligibleProducers(activeValidatorIDs, currentProducer, excludedProducerIDs); len(c) > 0 {
		return c
	}

	valSet, err := k.sk.GetValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Error("Failed to get validator set for producer fallback", "error", err)
		return nil
	}

	valIDs := make(map[uint64]struct{}, len(valSet.Validators))
	for _, v := range valSet.Validators {
		valIDs[v.ValId] = struct{}{}
	}
	return sortedEligibleProducers(valIDs, currentProducer, excludedProducerIDs)
}

// sortedEligibleProducers returns the ids in set, ascending, omitting the current producer
// and any excluded producer. Sorting makes the result deterministic across validators (Go
// map iteration order is not).
func sortedEligibleProducers(set map[uint64]struct{}, currentProducer uint64, excluded map[uint64]struct{}) []uint64 {
	ids := make([]uint64, 0, len(set))
	for id := range set {
		if id == currentProducer {
			continue
		}
		if _, isExcluded := excluded[id]; isExcluded {
			continue
		}
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}
