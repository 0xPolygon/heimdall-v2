package app

import (
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtcrypto "github.com/cometbft/cometbft/crypto/secp256k1"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
)

func SetupApp(t *testing.T, numOfVals uint64) *HeimdallApp {
	t.Helper()

	// generate validators, accounts and balances
	validators, accounts, balances := generateValidators(t, numOfVals)

	// setup app with validator set and respective accounts
	app := setupAppWithValidatorSet(t, validators, accounts, balances)
	return app
}

func generateValidators(t *testing.T, numOfVals uint64) ([]*cmttypes.Validator, []authtypes.GenesisAccount, []banktypes.Balance) {
	t.Helper()

	validators := make([]*cmttypes.Validator, 0, numOfVals)
	accounts := make([]authtypes.GenesisAccount, 0, numOfVals)
	balances := make([]banktypes.Balance, 0, numOfVals)

	var i uint64
	for ; i < numOfVals; i++ {
		privKey := cmtcrypto.GenPrivKey()
		pubKey := privKey.PubKey()

		// create validator set
		val := cmttypes.NewValidator(pubKey, 100)
		validators = append(validators, val)

		senderPubKey := secp256k1.GenPrivKey().PubKey()
		acc := authtypes.NewBaseAccount(pubKey.Address().Bytes(), senderPubKey, i, 0)
		balance := banktypes.Balance{
			Address: acc.GetAddress().String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100000000000000))),
		}

		accounts = append(accounts, acc)
		balances = append(balances, balance)
	}

	return validators, accounts, balances
}

func setupAppWithValidatorSet(t *testing.T, validators []*cmttypes.Validator, accounts []authtypes.GenesisAccount, balances []banktypes.Balance) *HeimdallApp {
	t.Helper()

	db := dbm.NewMemDB()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = DefaultNodeHome

	app := NewHeimdallApp(log.NewNopLogger(), db, nil, true, appOptions)
	genesisState := app.DefaultGenesis()

	// initialize validator set
	valSet := cmttypes.NewValidatorSet(validators)

	genesisState, err := simtestutil.GenesisStateWithValSet(app.AppCodec(), genesisState, valSet, accounts, balances...)
	require.NoError(t, err)

	stateBytes, err := jsoniter.ConfigFastest.Marshal(genesisState)
	require.NoError(t, err)

	// initialize chain with the validator set and genesis accounts
	_, err = app.InitChain(&abci.RequestInitChain{
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: simtestutil.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	},
	)
	require.NoError(t, err)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:             app.LastBlockHeight() + 1,
		Hash:               app.LastCommitID().Hash,
		NextValidatorsHash: valSet.Hash(),
	})
	require.NoError(t, err)

	return app
}
