package types

import (
	"context"

	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

type StakeKeeper interface {
	GetSpanEligibleValidators(ctx context.Context) []staketypes.Validator
	GetValidatorSet(ctx context.Context) (staketypes.ValidatorSet, error)
	GetValidatorFromValID(ctx context.Context, valID uint64) (staketypes.Validator, error)
}

type ChainManagerKeeper interface {
	GetParams(ctx context.Context) (chainmanagertypes.Params, error)
}
