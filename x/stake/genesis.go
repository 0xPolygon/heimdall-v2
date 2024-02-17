package staking

import (
	"errors"

	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/0xPolygon/heimdall-v2/x/stake/keeper"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO H2 Please look into this function
// WriteValidators returns a slice of bonded genesis validators.
func WriteValidators(ctx sdk.Context, keeper *keeper.Keeper) (vals []cmttypes.GenesisValidator, returnErr error) {

	return
}

// ValidateGenesis validates the provided staking genesis state to ensure the
// expected invariants holds. (i.e. params in correct bounds, no duplicate validators)
func ValidateGenesis(data *types.GenesisState) error {
	for _, validator := range data.Validators {
		if !validator.ValidateBasic() {
			return errors.New("Invalid validator")
		}
	}

	for _, sq := range data.StakingSequences {
		if sq == "" {
			return errors.New("Invalid Sequence")
		}
	}

	return nil
}
