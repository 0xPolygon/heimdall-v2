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
// Every candidate is restricted to a positive-power member of the current validator set.
// The active set carries milestone supporter IDs resolved against the penultimate validator
// set, which can name a validator that has since exited: its record survives with zero power
// and it is absent from the current set. Selecting such a validator freezes a span whose sole
// producer has zero power, which Bor reads as a validator deletion and cannot build a producer
// snapshot from. Intersecting with the current positive-power set keeps that producer out.
//
// Only the future-span PreBlocker path reaches this, where an empty candidate set is a fatal
// chain halt. The current-validator-set step makes the result deterministically non-empty
// whenever the set holds a positive-power validator other than the current and excluded
// producers, so selection does not depend on how many supporters remain eligible. The non-fatal
// rotation paths do not opt into this fallback and keep their skip-and-retry behavior.
func (k *Keeper) eligibleProducerFallback(ctx sdk.Context, currentProducer uint64, activeValidatorIDs, excludedProducerIDs map[uint64]struct{}) []uint64 {
	valSet, err := k.sk.GetValidatorSet(ctx)
	if err != nil {
		k.Logger(ctx).Error("Failed to get validator set for producer fallback", "error", err)
		return nil
	}

	eligible := make(map[uint64]struct{}, len(valSet.Validators))
	for _, v := range valSet.Validators {
		if v.VotingPower > 0 {
			eligible[v.ValId] = struct{}{}
		}
	}

	// Prefer the milestone supporters, restricted to currently-eligible validators.
	supporters := make(map[uint64]struct{}, len(activeValidatorIDs))
	for id := range activeValidatorIDs {
		if _, ok := eligible[id]; ok {
			supporters[id] = struct{}{}
		}
	}
	if c := sortedEligibleProducers(supporters, currentProducer, excludedProducerIDs); len(c) > 0 {
		return c
	}

	return sortedEligibleProducers(eligible, currentProducer, excludedProducerIDs)
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
