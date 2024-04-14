package keeper_test

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/simulation"
)

func (s *KeeperTestSuite) TestInitExportGenesis() {
	ctx, keeper := s.ctx, s.stakeKeeper
	require := s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	n := 5

	stakingSequence := make([]string, n)
	accounts := simulation.RandomAccounts(r1, n)

	for i := range stakingSequence {
		stakingSequence[i] = strconv.Itoa(simulation.RandIntBetween(r1, 1000, 100000))
	}

	validators := make([]*types.Validator, n)
	for i := 0; i < len(validators); i++ {
		pk1 := secp256k1.GenPrivKey().PubKey()
		validators[i] = types.NewValidator(
			uint64(i),
			0,
			0,
			uint64(i),
			int64(simulation.RandIntBetween(r1, 10, 100)), // power
			pk1,
			accounts[i].Address.String(),
		)
	}

	validatorSet := types.NewValidatorSet(validators)

	genesisState := types.NewGenesisState(validators, *validatorSet, stakingSequence)
	keeper.InitGenesis(ctx, genesisState)

	actualParams := keeper.ExportGenesis(ctx)
	require.NotNil(actualParams)
	require.LessOrEqual(5, len(actualParams.Validators))
}
