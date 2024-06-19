package checkpoint

import (
	"errors"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

// ValidateGenesis validates the provided checkpoint data
func ValidateGenesis(data *types.GenesisState) error {
	if err := data.Params.Validate(); err != nil {
		return err
	}

	if len(data.Checkpoints) != 0 {
		if int(data.AckCount) != len(data.Checkpoints) {
			return errors.New("incorrect state in state-dump , please Check")
		}
	}

	return nil
}
