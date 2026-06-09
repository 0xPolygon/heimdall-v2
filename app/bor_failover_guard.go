package app

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"cosmossdk.io/collections"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/0xPolygon/heimdall-v2/helper"
)

// EnforceBorFailoverBPGuard rejects Bor endpoint failover for protected block
// producer validators.
func (app *HeimdallApp) EnforceBorFailoverBPGuard() error {
	if !helper.BorFailoverConfigured(helper.GetConfig()) {
		return nil
	}

	signer, err := helper.GetAddressString()
	if err != nil {
		return fmt.Errorf("failed to resolve local signer for Bor failover BP guard: %w", err)
	}

	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})
	return app.validateBorFailoverBPGuard(ctx, signer)
}

func (app *HeimdallApp) validateBorFailoverBPGuard(ctx sdk.Context, signer string) error {
	localValidatorID, hasValidatorRecord, err := app.validatorIDForSigner(ctx, signer)
	if err != nil {
		return err
	}
	if !hasValidatorRecord {
		app.Logger().Info("Bor endpoint failover allowed; local signer is not a validator", "signer", signer)
		return nil
	}

	protectedProducerIDs, err := app.borFailoverProtectedProducerIDs(ctx)
	if err != nil {
		return err
	}

	if _, ok := protectedProducerIDs[localValidatorID]; ok {
		return fmt.Errorf(
			"bor endpoint failover is not allowed for block producer validators: local signer %s maps to protected producer validator ID %d; configure this node with a single local bor endpoint",
			signer,
			localValidatorID,
		)
	}

	app.Logger().Info(
		"Bor endpoint failover allowed; local validator is not in the protected producer set",
		"signer", signer,
		"validatorID", localValidatorID,
		"protectedProducerIDs", sortedProducerIDs(protectedProducerIDs),
	)

	return nil
}

func (app *HeimdallApp) validatorIDForSigner(ctx sdk.Context, signer string) (uint64, bool, error) {
	validator, err := app.StakeKeeper.GetValidatorInfo(ctx, signer)
	if errors.Is(err, collections.ErrNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("failed to load local signer validator info for Bor failover BP guard: %w", err)
	}

	return validator.ValId, true, nil
}

func (app *HeimdallApp) borFailoverProtectedProducerIDs(ctx sdk.Context) (map[uint64]struct{}, error) {
	protectedProducerIDs := make(map[uint64]struct{})
	addProducerIDs(protectedProducerIDs, helper.GetProducerVotes())
	addProducerIDs(protectedProducerIDs, helper.GetFallbackProducerVotes())

	lastSpan, err := app.BorKeeper.GetLastSpan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load last span for Bor failover BP guard: %w", err)
	}
	for _, producer := range lastSpan.SelectedProducers {
		protectedProducerIDs[producer.ValId] = struct{}{}
	}

	candidates, err := app.BorKeeper.CalculateProducerSet(ctx, helper.GetProducerSetLimit(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to calculate producer set for Bor failover BP guard: %w", err)
	}
	if len(candidates) == 0 {
		candidates = helper.GetFallbackProducerVotes()
	}
	addProducerIDs(protectedProducerIDs, candidates)

	return protectedProducerIDs, nil
}

func addProducerIDs(dst map[uint64]struct{}, ids []uint64) {
	for _, id := range ids {
		dst[id] = struct{}{}
	}
}

func sortedProducerIDs(producers map[uint64]struct{}) string {
	ids := make([]uint64, 0, len(producers))
	for id := range producers {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d", id))
	}

	return strings.Join(parts, ",")
}
