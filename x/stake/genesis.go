package staking

import (
	"errors"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// ValidateGenesis validates the provided staking genesis state to ensure the
// expected invariants holds. (i.e. params in correct bounds, no duplicate validators)
func ValidateGenesis(data *types.GenesisState) error {
	for _, validator := range data.Validators {
		if !validator.ValidateBasic() {
			return errors.New("invalid validator")
		}
	}

	for _, sq := range data.StakingSequences {
		if sq == "" {
			return errors.New("invalid sequence")
		}
	}

	return nil
}
