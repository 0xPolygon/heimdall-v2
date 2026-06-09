package app

import (
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestBorFailoverProtectedProducerIDsDoesNotIncludeEveryValidator(t *testing.T) {
	setup := SetupApp(t, 6)
	app := setup.App
	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})
	validators := app.StakeKeeper.GetCurrentValidators(ctx)
	require.Len(t, validators, 6)
	selectedProducer := requireValidatorByID(t, validators, 0)
	nonProducer := requireValidatorByID(t, validators, 5)

	require.NoError(t, app.BorKeeper.AddNewSpan(ctx, &borTypes.Span{
		Id:                0,
		StartBlock:        1,
		EndBlock:          100,
		ValidatorSet:      stakeTypes.ValidatorSet{Validators: validatorPointers(validators)},
		SelectedProducers: []stakeTypes.Validator{selectedProducer},
		BorChainId:        "bor",
	}))

	protectedProducerIDs, err := app.borFailoverProtectedProducerIDs(ctx)
	require.NoError(t, err)

	require.Contains(t, protectedProducerIDs, selectedProducer.ValId)
	for _, producerID := range helper.GetFallbackProducerVotes() {
		require.Contains(t, protectedProducerIDs, producerID)
	}
	require.NotContains(t, protectedProducerIDs, nonProducer.ValId)
}

func TestValidateBorFailoverBPGuard(t *testing.T) {
	setup := SetupApp(t, 6)
	app := setup.App
	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})
	validators := app.StakeKeeper.GetCurrentValidators(ctx)
	require.Len(t, validators, 6)
	selectedProducer := requireValidatorByID(t, validators, 0)
	fallbackProducer := requireValidatorByID(t, validators, 1)
	nonProducer := requireValidatorByID(t, validators, 5)

	require.NoError(t, app.BorKeeper.AddNewSpan(ctx, &borTypes.Span{
		Id:                0,
		StartBlock:        1,
		EndBlock:          100,
		ValidatorSet:      stakeTypes.ValidatorSet{Validators: validatorPointers(validators)},
		SelectedProducers: []stakeTypes.Validator{selectedProducer},
		BorChainId:        "bor",
	}))

	require.Error(t, app.validateBorFailoverBPGuard(ctx, selectedProducer.Signer))
	require.Error(t, app.validateBorFailoverBPGuard(ctx, fallbackProducer.Signer))
	require.NoError(t, app.validateBorFailoverBPGuard(ctx, nonProducer.Signer))
	require.NoError(t, app.validateBorFailoverBPGuard(ctx, "0x0000000000000000000000000000000000000001"))

	inactiveProducer := selectedProducer
	inactiveProducer.VotingPower = 0
	require.NoError(t, app.StakeKeeper.AddValidator(ctx, inactiveProducer))
	require.Error(t, app.validateBorFailoverBPGuard(ctx, inactiveProducer.Signer))

	jailedProducer := fallbackProducer
	jailedProducer.Jailed = true
	require.NoError(t, app.StakeKeeper.AddValidator(ctx, jailedProducer))
	require.Error(t, app.validateBorFailoverBPGuard(ctx, jailedProducer.Signer))
}

func requireValidatorByID(t *testing.T, validators []stakeTypes.Validator, id uint64) stakeTypes.Validator {
	t.Helper()
	for _, validator := range validators {
		if validator.ValId == id {
			return validator
		}
	}
	t.Fatalf("validator ID %d not found", id)
	return stakeTypes.Validator{}
}

func validatorPointers(validators []stakeTypes.Validator) []*stakeTypes.Validator {
	out := make([]*stakeTypes.Validator, 0, len(validators))
	for i := range validators {
		out = append(out, &validators[i])
	}
	return out
}
