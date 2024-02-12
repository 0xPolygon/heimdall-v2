package keeper_test

// import (
// 	"math/rand"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"

// 	"cosmossdk.io/math"

// 	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	"github.com/cosmos/cosmos-sdk/x/staking"
// 	"github.com/cosmos/cosmos-sdk/x/staking/testutil"
// 	"github.com/cosmos/cosmos-sdk/x/staking/types"
// )

// func (s *KeeperTestSuite) TestInitExportGenesis() {
// 	// create sub test to check if validator remove
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
// 		validators[i] = hmTypes.NewValidator(
// 			hmTypes.NewValidatorID(uint64(int64(i))),
// 			0,
// 			0,
// 			uint64(i),
// 			int64(simulation.RandIntBetween(r1, 10, 100)), // power
// 			hmTypes.NewPubKey(accounts[i].PubKey.Bytes()),
// 			accounts[i].Address,
// 		)
// 	}

// 	// validator set
// 	validatorSet := hmTypes.NewValidatorSet(validators)

// 	fmt.Print("valSet Proposer", validatorSet.Proposer)

// 	genesisState := types.NewGenesisState(validators, *validatorSet, stakingSequence)
// 	staking.InitGenesis(ctx, app.StakingKeeper, genesisState)

// 	actualParams := staking.ExportGenesis(ctx, app.StakingKeeper)
// 	require.NotNil(t, actualParams)
// 	require.LessOrEqual(t, 5, len(actualParams.Validators))
// }
