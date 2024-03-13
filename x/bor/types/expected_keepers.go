package types

import (
	"context"
)

type StakeKeeper interface {
	GetSpanEligibleValidators(ctx context.Context) []Validator
	GetValidatorSet(ctx context.Context) ValidatorSet
	GetValidatorFromValID(ctx context.Context, valID uint64) (Validator, bool)
}

type ChainManagerKeeper interface {
	GetParams(ctx context.Context) (Params, error)
}
