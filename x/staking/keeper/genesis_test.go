package keeper_test

// import (
// 	"fmt"
// 	"math/rand"
// 	"strconv"
// 	"time"

// 	"github.com/0xPolygon/heimdall-v2/x/staking/types"
// 	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
// 	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
// 	"github.com/cosmos/cosmos-sdk/types/simulation"
// )

// func (s *KeeperTestSuite) TestInitExportGenesis() {
// 	ctx, keeper := s.ctx, s.stakingKeeper
// 	require := s.Require()

// 	s1 := rand.NewSource(time.Now().UnixNano())
// 	r1 := rand.New(s1)
// 	n := 5

// 	stakingSequence := make([]string, n)
// 	accounts := simulation.RandomAccounts(r1, n)

// 	for i := range stakingSequence {
// 		stakingSequence[i] = strconv.Itoa(simulation.RandIntBetween(r1, 1000, 100000))
// 	}

// 	validators := make([]*hmTypes.Validator, n)
// 	for i := 0; i < len(validators); i++ {
// 		// validator
// 		pk1 := secp256k1.GenPrivKey().PubKey()
// 		validators[i] = hmTypes.NewValidator(
// 			uint64(i),
// 			0,
// 			0,
// 			uint64(i),
// 			int64(simulation.RandIntBetween(r1, 10, 100)), // power
// 			pk1,
// 			accounts[i].Address.String(),
// 		)
// 	}

// 	// validator set
// 	validatorSet := hmTypes.NewValidatorSet(validators)

// 	fmt.Print("valSet Proposer", validatorSet.Proposer)

// 	genesisState := types.NewGenesisState(validators, *validatorSet, stakingSequence)
// 	keeper.InitGenesis(ctx, genesisState)

// 	actualParams := keeper.ExportGenesis(ctx)
// 	require.NotNil(actualParams)
// 	require.LessOrEqual(5, len(actualParams.Validators))
// }
