package staking

import (
	"errors"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// ValidateGenesis validates the provided stake genesis state to ensure that listed
// validators and staking sequences are valid
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
