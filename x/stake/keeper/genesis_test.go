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

	prevN := 3
	prevAccounts := simulation.RandomAccounts(r, prevN)
	prevValidators := make([]*types.Validator, prevN)
	for i := 0; i < prevN; i++ {
		pkPrev := secp256k1.GenPrivKey().PubKey()
		prevValidators[i], err = types.NewValidator(uint64(100+i), 0, 0, uint64(i), int64(simulation.RandIntBetween(r, 10, 100)), pkPrev, prevAccounts[i].Address.String())
		require.NoError(err)
	}
	prevSet := types.NewValidatorSet(prevValidators)
	err = keeper.UpdatePreviousBlockValidatorSetInStore(ctx, *prevSet)
	require.NoError(err)

	penN := 2
	penAccounts := simulation.RandomAccounts(r, penN)
	penValidators := make([]*types.Validator, penN)
	for i := 0; i < penN; i++ {
		pkPen := secp256k1.GenPrivKey().PubKey()
		penValidators[i], err = types.NewValidator(uint64(200+i), 0, 0, uint64(i), int64(simulation.RandIntBetween(r, 10, 100)), pkPen, penAccounts[i].Address.String())
		require.NoError(err)
	}
	penultimate := types.NewValidatorSet(penValidators)
	err = keeper.UpdatePenultimateBlockValidatorSetInStore(ctx, *penultimate)
	require.NoError(err)

	customTxs := [][]byte{[]byte("tx-a"), []byte("tx-b")}
	require.NoError(keeper.SetLastBlockTxs(ctx, customTxs))

	actualParams := keeper.ExportGenesis(ctx)
	require.NotNil(actualParams)
	require.LessOrEqual(n, len(actualParams.Validators))
	require.True(genesisState.CurrentValidatorSet.Equal(actualParams.CurrentValidatorSet))

	require.True(prevSet.Equal(actualParams.PreviousBlockValidatorSet))
	require.LessOrEqual(prevN, len(actualParams.Validators))

	require.True(penultimate.Equal(actualParams.PenultimateBlockValidatorSet))
	require.LessOrEqual(penN, len(actualParams.Validators))

	require.Equal(customTxs, actualParams.LastBlockTxs.Txs)
}
