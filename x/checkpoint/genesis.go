package checkpoint

import (
	"errors"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

// ValidateGenesis validates the provided staking genesis state to ensure the
// expected invariants holds. (i.e. params in correct bounds, no duplicate validators)
func ValidateGenesis(data *types.GenesisState) error {
	if err := data.Params.Validate(); err != nil {
		return err
	}

	if len(data.Checkpoints) != 0 {
		if int(data.AckCount) != len(data.Checkpoints) {
			return errors.New("Incorrect state in state-dump , Please Check")
		}
	}

	return nil
}
