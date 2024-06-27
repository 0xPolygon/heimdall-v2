package types

import (
	"context"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

type StakeKeeper interface {
	MilestoneIncrementAccum(ctx context.Context, times int)
	GetMilestoneValidatorSet(ctx context.Context) (validatorSet types.ValidatorSet, err error)
}
