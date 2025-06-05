package keeper

import (
	"context"
	"fmt"
	"sort"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AddNewVeblopSpan adds a new veblop (Validator-elected block producer) span
func (k *Keeper) AddNewVeblopSpan(ctx sdk.Context, currentProducer uint64, startBlock uint64, endBlock uint64, borChainID string, activeValidatorIDs map[uint64]struct{}) error {
	logger := k.Logger(ctx)

	// select next producers
	newProducerId, err := k.SelectNextSpanProducer(ctx, currentProducer, activeValidatorIDs)
	if err != nil {
		return err
	}

	valSet, err := k.sk.GetValidatorSet(ctx)
	if err != nil {
		return err
	}

	newProducer, err := k.sk.GetValidatorFromValID(ctx, newProducerId)
	if err != nil {
		return err
	}

	lastSpan, err := k.GetLastSpan(ctx)
	if err != nil {
		return err
	}

	// generate new span
	newSpan := &types.Span{
		Id:                lastSpan.Id + 1,
		StartBlock:        startBlock,
		EndBlock:          endBlock,
		ValidatorSet:      valSet,
		SelectedProducers: []staketypes.Validator{newProducer},
		BorChainId:        borChainID,
	}

	logger.Info("Freezing new veblop span", "id", newSpan.Id, "span", newSpan)

	return k.AddNewSpan(ctx, newSpan)
}

func (k *Keeper) FindCurrentProducerID(ctx context.Context, blockNum uint64) (uint64, error) {
	lastSpan, err := k.GetLastSpan(ctx)
	if err != nil {
		return 0, err
	}

	for i := lastSpan.Id; i >= 0; i-- {
		span, err := k.GetSpan(ctx, i)
		if err != nil {
			return 0, err
		}

		if blockNum >= span.StartBlock && blockNum <= span.EndBlock {
			return span.SelectedProducers[0].ValId, nil
		}
	}

	return 0, fmt.Errorf("no active producer found")
}

func (k *Keeper) FindPastBackupProducerIDs(ctx context.Context, blockNum uint64) ([]uint64, error) {
	lastSpan, err := k.GetLastSpan(ctx)
	if err != nil {
		return nil, err
	}

	producerIDs := make([]uint64, 0)
	for i := lastSpan.Id; i > 0; i-- {
		span, err := k.GetSpan(ctx, i)
		if err != nil {
			return nil, err
		}

		if blockNum > span.EndBlock {
			break
		}

		if blockNum == span.StartBlock {
			producerIDs = append(producerIDs, span.SelectedProducers[0].ValId)
		}
	}

	return producerIDs, nil
}

func (k *Keeper) UpdateValidatorPerformanceScore(ctx context.Context, activeValidatorIDs map[uint64]struct{}, blocks uint64) error {
	for validatorID := range activeValidatorIDs {
		hasKey, err := k.PerformanceScore.Has(ctx, validatorID)
		if err != nil {
			return err
		}

		if !hasKey {
			k.PerformanceScore.Set(ctx, validatorID, blocks)
		} else {
			currentScore, err := k.PerformanceScore.Get(ctx, validatorID)
			if err != nil {
				return err
			}
			k.PerformanceScore.Set(ctx, validatorID, currentScore+blocks)
		}
	}

	return nil
}

func (k *Keeper) ResetValidatorPerformanceScore(ctx context.Context) error {
	return k.PerformanceScore.Clear(ctx, nil)
}

func (k *Keeper) GetAllValidatorPerformanceScore(ctx context.Context) (map[uint64]uint64, error) {
	iter, err := k.PerformanceScore.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}

	validatorPerformanceScore := make(map[uint64]uint64)
	for ; iter.Valid(); iter.Next() {
		validatorID, err := iter.Key()
		if err != nil {
			return nil, err
		}

		score, err := iter.Value()
		if err != nil {
			return nil, err
		}

		validatorPerformanceScore[validatorID] = score
	}

	return validatorPerformanceScore, nil
}

// UpdateLatestActiveValidator updates the latest active validator
func (k *Keeper) UpdateLatestActiveValidator(ctx context.Context, activeValidatorIDs map[uint64]struct{}) error {
	err := k.LatestActiveValidator.Clear(ctx, nil)
	if err != nil {
		return err
	}

	for validatorID := range activeValidatorIDs {
		err := k.LatestActiveValidator.Set(ctx, validatorID)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetLatestActiveValidator returns the latest active validator
func (k *Keeper) GetLatestActiveValidator(ctx context.Context) (map[uint64]struct{}, error) {
	iter, err := k.LatestActiveValidator.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}

	latestActiveValidator := make(map[uint64]struct{})
	for ; iter.Valid(); iter.Next() {
		validatorID, err := iter.Key()
		if err != nil {
			return nil, err
		}
		latestActiveValidator[validatorID] = struct{}{}
	}

	return latestActiveValidator, nil
}

func (k *Keeper) AddLatestFailedValidator(ctx context.Context, validatorID uint64) error {
	err := k.LatestFailedValidator.Set(ctx, validatorID)
	if err != nil {
		return err
	}

	return nil
}

func (k *Keeper) GetLatestFailedValidator(ctx context.Context) (map[uint64]struct{}, error) {
	iter, err := k.LatestFailedValidator.Iterate(ctx, nil)
	if err != nil {
		return nil, err
	}

	latestFailedValidator := make(map[uint64]struct{})
	for ; iter.Valid(); iter.Next() {
		validatorID, err := iter.Key()
		if err != nil {
			return nil, err
		}
		latestFailedValidator[validatorID] = struct{}{}
	}

	return latestFailedValidator, nil
}

func (k *Keeper) ClearLatestFailedValidator(ctx context.Context) error {
	err := k.LatestFailedValidator.Clear(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}

// SelectNextSpanProducer selects the next producer for a new span.
// It calculates candidate set, filters by active producers and selects one.
func (k *Keeper) SelectNextSpanProducer(ctx context.Context, currentProducer uint64, activeValidatorIDs map[uint64]struct{}) (uint64, error) {
	candidates, err := k.CalculateProducerSet(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate producer set: %w", err)
	}

	activeCandidates := k.FilterByActiveProducerSet(ctx, candidates, activeValidatorIDs)

	// If no candidate is available after threshold filtering,
	// the candidate list will be rotated to the next producer EVEN IF the the producer is not active.
	if len(activeCandidates) == 0 {
		newCandidates := make([]uint64, 0)
		for _, validatorID := range candidates {
			if validatorID != currentProducer {
				newCandidates = append(newCandidates, validatorID)
			}
		}
		activeCandidates = newCandidates
	}

	nextProducer, err := k.SelectProducer(ctx, currentProducer, activeCandidates)
	if err != nil {
		return 0, fmt.Errorf("failed to select producer: %w", err)
	}

	return nextProducer, nil
}

// CalculateProducerSet ranks producer candidates by the sum of the stake from validators who voted for them,
// weighted by their relative position in the candidate list.
func (k *Keeper) CalculateProducerSet(ctx context.Context) ([]uint64, error) {
	params, err := k.FetchParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bor params: %w", err)
	}

	allValidators, err := k.sk.GetValidatorSet(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator set: %w", err)
	}

	totalPotentialProducers := uint64(len(allValidators.Validators))
	if totalPotentialProducers == 0 {
		k.Logger(ctx).Info("No validators found, cannot calculate producer set.")
		return []uint64{}, nil
	}

	producerWeightedScores := make(map[uint64]int64) // Will now be sum of stakes

	votesIterator, err := k.ProducerVotes.Iterate(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to iterate producer votes: %w", err)
	}
	defer votesIterator.Close()

	for ; votesIterator.Valid(); votesIterator.Next() {
		validatorID, err := votesIterator.Key()
		if err != nil {
			return nil, fmt.Errorf("failed to get key from producer votes iterator: %w", err)
		}

		producerVoteData, err := votesIterator.Value()
		if err != nil {
			k.Logger(ctx).Error("Failed to get value from producer votes iterator, skipping", "validatorID", validatorID, "error", err)
			continue
		}

		validator, err := k.sk.GetValidatorFromValID(ctx, validatorID)
		if err != nil {
			k.Logger(ctx).Debug("Failed to get validator for producer vote, skipping", "validatorID", validatorID, "error", err)
			continue
		}

		if validator.VotingPower <= 0 {
			k.Logger(ctx).Debug("Validator has no voting power, skipping votes", "validatorID", validatorID)
			continue
		}

		validatorStake := validator.VotingPower

		// Consider only the first 'totalPotentialProducers' candidates from the vote list.
		// Apply positional weighting: higher positions get higher weights.
		for i, candidateID := range producerVoteData.Votes {
			if uint64(i) >= totalPotentialProducers {
				break // Only consider top N votes where N is totalPotentialProducers
			}
			// Weight decreases by position: (totalPotentialProducers - position) * validatorStake
			positionWeight := int64(totalPotentialProducers - uint64(i))
			producerWeightedScores[candidateID] += positionWeight * validatorStake
		}
	}

	if len(producerWeightedScores) == 0 {
		k.Logger(ctx).Warn("No producer votes recorded or no valid votes found.")
		return []uint64{}, nil
	}

	type scoredProducer struct {
		ID    uint64
		Score int64
	}

	var rankedProducers []scoredProducer
	for id, score := range producerWeightedScores {
		rankedProducers = append(rankedProducers, scoredProducer{ID: id, Score: score})
	}

	// Sort producers by score in descending order.
	// If scores are equal, sort by ID ascending for determinism.
	sort.SliceStable(rankedProducers, func(i, j int) bool {
		if rankedProducers[i].Score == rankedProducers[j].Score {
			return rankedProducers[i].ID < rankedProducers[j].ID
		}
		return rankedProducers[i].Score > rankedProducers[j].Score
	})

	// Calculate total stake of all validators for threshold calculation
	var totalStakeOfAllValidators int64
	for _, val := range allValidators.Validators {
		totalStakeOfAllValidators += val.VotingPower
	}

	if totalStakeOfAllValidators == 0 {
		k.Logger(ctx).Warn("Total stake of all validators is 0. No producers can qualify under threshold logic.")
		return []uint64{}, nil
	}

	producerSetLimit := int(params.ProducerCount)
	if producerSetLimit == 0 {
		k.Logger(ctx).Warn("ProducerCount is 0, returning empty producer set.")
		return []uint64{}, nil
	}

	finalCandidates := make([]uint64, 0, producerSetLimit)
	for i, sp := range rankedProducers {
		if i >= producerSetLimit {
			break // Reached producer set limit
		}

		// Calculate positional threshold: candidate needs >= 2/3 of max possible weighted vote at their position
		position := uint64(i) + 1 // 1-indexed position
		maxPossibleWeightedVoteAtPosition := int64(totalPotentialProducers-position+1) * totalStakeOfAllValidators
		positionalRequiredScore := (maxPossibleWeightedVoteAtPosition * 2 / 3) + 1

		k.Logger(ctx).Debug("Threshold check for candidate",
			"candidateID", sp.ID,
			"candidateScore", sp.Score,
			"position", position,
			"maxPossibleWeightedVoteAtPosition", maxPossibleWeightedVoteAtPosition,
			"positionalRequiredScore", positionalRequiredScore)

		if sp.Score >= positionalRequiredScore {
			finalCandidates = append(finalCandidates, sp.ID)
		} else {
			k.Logger(ctx).Debug("Candidate failed to meet positional threshold, stopping further selection.",
				"candidateID", sp.ID,
				"candidateScore", sp.Score,
				"positionalRequiredScore", positionalRequiredScore)
			break // Stop adding candidates if one fails to qualify
		}
	}

	k.Logger(ctx).Debug("Calculated producer set", "count", len(finalCandidates), "candidates", finalCandidates)
	return finalCandidates, nil
}

// FilterByActiveProducerSet filters candidates based on whether each candidate has voted for the last X milestones.
func (k *Keeper) FilterByActiveProducerSet(ctx context.Context, candidates []uint64, activeValidatorIDs map[uint64]struct{}) []uint64 {
	activeCandidates := make([]uint64, 0, len(candidates))

	for _, candidate := range candidates {
		if _, ok := activeValidatorIDs[candidate]; ok {
			activeCandidates = append(activeCandidates, candidate)
		}
	}
	return activeCandidates
}

// SelectProducer selects a producer from the candidates list.
// The selected candidate will be the next candidate to the current producer.
// If the current producer is not in the candidate list, the first candidate in the list will be chosen.
func (k *Keeper) SelectProducer(ctx context.Context, currentProducer uint64, candidates []uint64) (uint64, error) {
	if len(candidates) == 0 {
		k.Logger(ctx).Error("SelectProducer called with no candidates")
		return 0, fmt.Errorf("no candidates found")
	}

	currentIndex := -1
	for i, candidate := range candidates {
		if candidate == currentProducer {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		// Current producer not in the list, select the first candidate
		k.Logger(ctx).Debug("Current producer not in candidate list, selecting first candidate", "currentProducer", currentProducer, "selected", candidates[0])
		return candidates[0], nil
	}

	// Select the next candidate in the list, wrapping around
	nextIndex := (currentIndex + 1) % len(candidates)
	k.Logger(ctx).Info("Selecting next producer in list", "currentProducer", currentProducer, "currentIndex", currentIndex, "nextIndex", nextIndex, "selected", candidates[nextIndex])
	return candidates[nextIndex], nil
}
