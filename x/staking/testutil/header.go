package testutil

// import (
// 	stakingKeeper "github.com/0xPolygon/heimdall-v2/x/staking/keeper"
// 	sdk "github.com/cosmos/cosmos-sdk/types"

// 	"github.com/stretchr/testify/require"

// 	"github.com/0xPolygon/heimdall-v2/x/types"
// )

// // LoadValidatorSet loads validator set
// func LoadValidatorSet(require *require.Assertions, count int, keeper *stakingKeeper.Keeper, ctx sdk.Context, randomise bool, timeAlive int) types.ValidatorSet {
// 	var valSet types.ValidatorSet

// 	validators := GenRandomVal(count, 0, 10, uint64(timeAlive), randomise, 1)
// 	for _, validator := range validators {
// 		err := keeper.AddValidator(ctx, validator)
// 		require.NoError(err, "Unable to set validator, Error: %v", err)

// 		err = valSet.UpdateWithChangeSet([]*types.Validator{&validator})
// 		require.NoError(err)
// 	}

// 	err := keeper.UpdateValidatorSetInStore(ctx, valSet)
// 	require.NoError(err, "Unable to update validator set")

// 	vals := keeper.GetAllValidators(ctx)
// 	require.NotNil(vals)

// 	return valSet
// }
