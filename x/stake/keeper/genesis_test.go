package keeper_test

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func (s *KeeperTestSuite) TestInitExportGenesis() {
	ctx, keeper, require := s.ctx, s.stakeKeeper, s.Require()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := 5

	stakingSequence := make([]string, n)
	accounts := simulation.RandomAccounts(r, n)

	for i := range stakingSequence {
		stakingSequence[i] = strconv.Itoa(simulation.RandIntBetween(r, 1000, 100000))
	}

	validators := make([]*types.Validator, n)
	var err error
	for i := 0; i < len(validators); i++ {
		pk1 := secp256k1.GenPrivKey().PubKey()
		validators[i], err = types.NewValidator(
			uint64(i),
			0,
			0,
			uint64(i),
			int64(simulation.RandIntBetween(r, 10, 100)), // power
			pk1,
			accounts[i].Address.String(),
		)

		require.NoError(err)
	}

	validatorSet := types.NewValidatorSet(validators)

	genesisState := types.NewGenesisState(validators, *validatorSet, stakingSequence)
	keeper.InitGenesis(ctx, genesisState)
	valSet, err := keeper.GetPreviousBlockValidatorSet(ctx)
	require.NoError(err)
	require.Equal(validatorSet.Len(), valSet.Len())
	require.Equal(validatorSet.Proposer.Signer, valSet.Proposer.Signer)
	require.Equal(validatorSet.TotalVotingPower, valSet.TotalVotingPower)

	actualParams := keeper.ExportGenesis(ctx)
	require.NotNil(actualParams)
	require.LessOrEqual(n, len(actualParams.Validators))
	require.True(genesisState.CurrentValidatorSet.Equal(actualParams.CurrentValidatorSet))
}
