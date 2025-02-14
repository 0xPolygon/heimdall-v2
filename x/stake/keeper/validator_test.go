package keeper_test

import (
	"testing"

	"github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestSortValidatorByAddress(t *testing.T) {
	randValidators := testutil.GenRandomVals(5, 1, 1, 5, false, 0)

	validators := make([]*types.Validator, len(randValidators))

	for i := range len(randValidators) {
		validators[i] = &randValidators[i]
	}

	validatorSet := types.NewValidatorSet(validators)

	deleteVals := make([]*types.Validator, 2)

	del1 := validators[0].Copy()
	del1.VotingPower = 0

	deleteVals[0] = del1
	deleteVals[1] = del1

	err := validatorSet.UpdateWithChangeSet(deleteVals)
	if err == nil {
		t.Error(err)
	}

	del2 := validators[1].Copy()
	del1.VotingPower = -1

	deleteVals[0] = del1
	deleteVals[1] = del2

	err = validatorSet.UpdateWithChangeSet(deleteVals)
	if err == nil {
		t.Error(err)
	}

	del1.VotingPower = 0
	del2.VotingPower = 0

	err = validatorSet.UpdateWithChangeSet(deleteVals)
	if err != nil {
		t.Error(err)
	}

	i, val := validatorSet.GetByAddress(del1.Signer)

	if i != -1 && val != nil {
		t.Error(err)
	}

	if validatorSet.Len() != 3 {
		t.Error(err)
	}
}
