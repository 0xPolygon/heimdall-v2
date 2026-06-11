package app

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	borTypes "github.com/0xPolygon/heimdall-v2/x/bor/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestBorFailoverProtectedProducerIDsDoesNotIncludeEveryValidator(t *testing.T) {
	setup := SetupApp(t, 6)
	app := setup.App
	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})
	validators := app.StakeKeeper.GetCurrentValidators(ctx)
	require.Len(t, validators, 6)
	selectedProducer := requireValidatorByID(t, validators, 0)
	nonProducer := requireValidatorByID(t, validators, 5)

	require.NoError(t, app.BorKeeper.AddNewSpan(ctx, &borTypes.Span{
		Id:                0,
		StartBlock:        1,
		EndBlock:          100,
		ValidatorSet:      stakeTypes.ValidatorSet{Validators: validatorPointers(validators)},
		SelectedProducers: []stakeTypes.Validator{selectedProducer},
		BorChainId:        "bor",
	}))

	protectedProducerIDs, err := app.borFailoverProtectedProducerIDs(ctx)
	require.NoError(t, err)

	require.Contains(t, protectedProducerIDs, selectedProducer.ValId)
	for _, producerID := range helper.GetFallbackProducerVotes() {
		require.Contains(t, protectedProducerIDs, producerID)
	}
	require.NotContains(t, protectedProducerIDs, nonProducer.ValId)
}

func TestValidateBorFailoverBPGuard(t *testing.T) {
	setup := SetupApp(t, 6)
	app := setup.App
	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})
	validators := app.StakeKeeper.GetCurrentValidators(ctx)
	require.Len(t, validators, 6)
	selectedProducer := requireValidatorByID(t, validators, 0)
	fallbackProducer := requireValidatorByID(t, validators, 1)
	nonProducer := requireValidatorByID(t, validators, 5)

	require.NoError(t, app.BorKeeper.AddNewSpan(ctx, &borTypes.Span{
		Id:                0,
		StartBlock:        1,
		EndBlock:          100,
		ValidatorSet:      stakeTypes.ValidatorSet{Validators: validatorPointers(validators)},
		SelectedProducers: []stakeTypes.Validator{selectedProducer},
		BorChainId:        "bor",
	}))

	require.Error(t, app.validateBorFailoverBPGuard(ctx, selectedProducer.Signer))
	require.Error(t, app.validateBorFailoverBPGuard(ctx, fallbackProducer.Signer))
	require.NoError(t, app.validateBorFailoverBPGuard(ctx, nonProducer.Signer))
	require.NoError(t, app.validateBorFailoverBPGuard(ctx, "0x0000000000000000000000000000000000000001"))

	inactiveProducer := selectedProducer
	inactiveProducer.VotingPower = 0
	require.NoError(t, app.StakeKeeper.AddValidator(ctx, inactiveProducer))
	require.Error(t, app.validateBorFailoverBPGuard(ctx, inactiveProducer.Signer))

	jailedProducer := fallbackProducer
	jailedProducer.Jailed = true
	require.NoError(t, app.StakeKeeper.AddValidator(ctx, jailedProducer))
	require.Error(t, app.validateBorFailoverBPGuard(ctx, jailedProducer.Signer))
}

// TestInitChainFailsClosedForGenesisBorFailoverProducer covers the fresh-DB /
// pre-InitChain path: the startup guard in newStartApp runs before genesis state
// exists, so a genesis producer's signer has no validator record yet. The
// InitChainer re-check must catch it once InitGenesis has written validator and
// span state. The sole genesis validator is necessarily the span-0 producer.
func TestInitChainFailsClosedForGenesisBorFailoverProducer(t *testing.T) {
	validatorPrivKeys, validators, accounts, balances := generateValidators(t, 1)

	appCfg := helper.CustomAppConfig{
		Config: *serverconfig.DefaultConfig(),
		Custom: helper.GetDefaultHeimdallConfig(),
	}
	appCfg.Custom.BorRPCUrl = "http://primary:8545,http://fallback:8545"
	helper.SetTestConfig(appCfg)
	helper.SetTestPrivPubKey(validatorPrivKeys[0])
	t.Cleanup(func() {
		helper.SetTestConfig(helper.CustomAppConfig{
			Config: *serverconfig.DefaultConfig(),
			Custom: helper.GetDefaultHeimdallConfig(),
		})
	})

	db := dbm.NewMemDB()
	appOptions := make(simtestutil.AppOptionsMap)
	appOptions[flags.FlagHome] = DefaultNodeHome
	hApp := NewHeimdallApp(log.NewTestLogger(t), db, nil, true, appOptions)

	genesisState := hApp.DefaultGenesis()
	valSet := stakeTypes.NewValidatorSet(validators)
	genesisState, err := GenesisStateWithValSet(hApp.AppCodec(), genesisState, valSet, accounts, balances...)
	require.NoError(t, err)
	// Build span 0 from the validator set (genFirstSpan), as a real genesis does;
	// the sole validator becomes the span producer and thus a protected producer.
	genesisState, err = borTypes.SetGenesisStateToAppState(hApp.AppCodec(), genesisState, *valSet)
	require.NoError(t, err)
	stateBytes, err := json.Marshal(genesisState)
	require.NoError(t, err)

	helper.SetTestInitialHeight(VoteExtBlockHeight)
	_, err = hApp.InitChain(&abci.RequestInitChain{
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: simtestutil.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
		InitialHeight:   VoteExtBlockHeight,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "block producer")
}

func requireValidatorByID(t *testing.T, validators []stakeTypes.Validator, id uint64) stakeTypes.Validator {
	t.Helper()
	for _, validator := range validators {
		if validator.ValId == id {
			return validator
		}
	}
	t.Fatalf("validator ID %d not found", id)
	return stakeTypes.Validator{}
}

func validatorPointers(validators []stakeTypes.Validator) []*stakeTypes.Validator {
	out := make([]*stakeTypes.Validator, 0, len(validators))
	for i := range validators {
		out = append(out, &validators[i])
	}
	return out
}
