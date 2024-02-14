package staking_test

// import (
// 	"fmt"
// 	"math/rand"
// 	"strconv"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"

// 	"cosmossdk.io/math"

// 	"github.com/0xPolygon/heimdall-v2/x/staking"
// 	"github.com/0xPolygon/heimdall-v2/x/staking/testutil"
// 	"github.com/0xPolygon/heimdall-v2/x/staking/types"
// 	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
// 	sdk "github.com/cosmos/cosmos-sdk/types"
// 	"github.com/cosmos/cosmos-sdk/types/simulation"
// )

// func TestValidateGenesis(t *testing.T) {
// 	genValidators1 := make([]types.Validator, 1, 5)
// 	pk := ed25519.GenPrivKey().PubKey()
// 	genValidators1[0] = testutil.NewValidator(t, sdk.ValAddress(pk.Address()), pk)
// 	genValidators1[0].Tokens = math.OneInt()
// 	genValidators1[0].DelegatorShares = math.LegacyOneDec()

// 	tests := []struct {
// 		name    string
// 		mutate  func(*types.GenesisState)
// 		wantErr bool
// 	}{
// 		{"default", func(*types.GenesisState) {}, false},
// 		// validate genesis validators
// 		{"duplicate validator", func(data *types.GenesisState) {
// 			data.Validators = genValidators1
// 			data.Validators = append(data.Validators, genValidators1[0])
// 		}, true},
// 		{"no delegator shares", func(data *types.GenesisState) {
// 			data.Validators = genValidators1
// 			data.Validators[0].DelegatorShares = math.LegacyZeroDec()
// 		}, true},
// 		{"jailed and bonded validator", func(data *types.GenesisState) {
// 			data.Validators = genValidators1
// 			data.Validators[0].Jailed = true
// 			data.Validators[0].Status = types.Bonded
// 		}, true},
// 	}

// 	for _, tt := range tests {
// 		tt := tt

// 		t.Run(tt.name, func(t *testing.T) {
// 			genesisState := types.DefaultGenesisState()
// 			tt.mutate(genesisState)

// 			if tt.wantErr {
// 				assert.Error(t, staking.ValidateGenesis(genesisState))
// 			} else {
// 				assert.NoError(t, staking.ValidateGenesis(genesisState))
// 			}
// 		})
// 	}
// }

// func (suite *GenesisTestSuite) TestInitExportGenesis() {
// 	t, app, ctx := suite.T(), suite.app, suite.ctx
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
