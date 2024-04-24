package types

import (
	"context"

	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

type StakeKeeper interface {
	GetSpanEligibleValidators(ctx context.Context) []Validator
	GetValidatorSet(ctx context.Context) ValidatorSet
	GetValidatorFromValID(ctx context.Context, valID uint64) (Validator, bool)
}

type ChainManagerKeeper interface {
	GetParams(ctx context.Context) (chainmanagertypes.Params, error)
}
