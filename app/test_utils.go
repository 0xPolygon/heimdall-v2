package app

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtcrypto "github.com/cometbft/cometbft/crypto/secp256k1"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

const (
	VoteExtBlockHeight = 100
	CurrentHeight      = 101
	TxHash1            = "000000000000000000000000000000000000000000000000000000000001dead"
	TxHash2            = "000000000000000000000000000000000000000000000000000000000002dead"
	TxHash3            = "000000000000000000000000000000000000000000000000000000000003dead"
	ValAddr1           = "0x000000000000000000000000000000000001dEaD"
	ValAddr2           = "0x000000000000000000000000000000000002dEaD"
	ValAddr3           = "0x000000000000000000000000000000000003dEaD"
)

func SetupApp(t *testing.T, numOfVals uint64) (*HeimdallApp, *dbm.MemDB, log.Logger) {
	t.Helper()

	// generate validators, accounts and balances
	validators, accounts, balances := generateValidators(t, numOfVals)

	// setup app with validator set and respective accounts
	return setupAppWithValidatorSet(t, validators, accounts, balances)
}

func generateValidators(t *testing.T, numOfVals uint64) ([]*stakeTypes.Validator, []authtypes.GenesisAccount, []banktypes.Balance) {
	t.Helper()

	validators := make([]*stakeTypes.Validator, 0, numOfVals)
	accounts := make([]authtypes.GenesisAccount, 0, numOfVals)
	balances := make([]banktypes.Balance, 0, numOfVals)

	var i uint64
	for ; i < numOfVals; i++ {
		privKey := cmtcrypto.GenPrivKey()
		pubKey := privKey.PubKey()
		pk, err := cryptocodec.FromCmtPubKeyInterface(pubKey)
		if err != nil {
			_ = fmt.Errorf("failed to convert pubkey: %w", err)
		}

		// create validator set
		val, _ := stakeTypes.NewValidator(i, 0, 0, i, 100, pk, pubKey.Address().String())

		validators = append(validators, val)

		senderPubKey := secp256k1.GenPrivKey().PubKey()
		acc := authtypes.NewBaseAccount(senderPubKey.Address().Bytes(), senderPubKey, i+1, 0) // fee_collector is the first initialized (module) account (AccountNumber = 0)
		balance := banktypes.Balance{
			Address: acc.GetAddress().String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100000000000000))),
		}

		accounts = append(accounts, acc)
		balances = append(balances, balance)
	}

	return validators, accounts, balances
}

func setupAppWithValidatorSet(t *testing.T, validators []*stakeTypes.Validator, accounts []authtypes.GenesisAccount, balances []banktypes.Balance) (*HeimdallApp, *dbm.MemDB, log.Logger) {
	t.Helper()

	db := dbm.NewMemDB()

	appOptions := make(simtestutil.AppOptionsMap)
	appOptions[flags.FlagHome] = DefaultNodeHome

	logger := log.NewTestLogger(t)
	app := NewHeimdallApp(logger, db, nil, true, appOptions)
	genesisState := app.DefaultGenesis()

	// initialize validator set
	valSet := stakeTypes.NewValidatorSet(validators)

	genesisState, err := GenesisStateWithValSet(app.AppCodec(), genesisState, valSet, accounts, balances...)
	require.NoError(t, err)

	stateBytes, err := json.Marshal(genesisState)
	require.NoError(t, err)

	// initialize chain with the validator set and genesis accounts
	_, err = app.InitChain(&abci.RequestInitChain{
		Validators:      []abci.ValidatorUpdate{},
		ConsensusParams: simtestutil.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
		InitialHeight:   100,
	},
	)
	require.NoError(t, err)

	return app, db, logger
}

func mustMarshalSideTxResponses(t *testing.T, respVotes ...[]sidetxs.SideTxResponse) []byte {
	t.Helper()
	responses := make([]sidetxs.SideTxResponse, 0)
	for _, r := range respVotes {
		responses = append(responses, r...)
	}

	sideTxResponses := sidetxs.ConsolidatedSideTxResponse{
		SideTxResponses: responses,
		Height:          VoteExtBlockHeight,
	}

	voteExtension, err := sideTxResponses.Marshal()
	require.NoError(t, err)
	return voteExtension
}

func createSideTxResponses(vote sidetxs.Vote, txHashes ...string) []sidetxs.SideTxResponse {
	responses := make([]sidetxs.SideTxResponse, len(txHashes))
	for i, txHash := range txHashes {
		responses[i] = sidetxs.SideTxResponse{
			TxHash: common.Hex2Bytes(txHash),
			Result: vote,
		}
	}
	return responses
}

// GenesisStateWithValSet returns a new genesis state with the validator set
func GenesisStateWithValSet(codec codec.Codec, genesisState map[string]json.RawMessage, valSet *stakeTypes.ValidatorSet, genAccs []authtypes.GenesisAccount, balances ...banktypes.Balance) (map[string]json.RawMessage, error) {
	// set genesis accounts
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = codec.MustMarshalJSON(authGenesis)

	validators := make([]*stakeTypes.Validator, 0, len(valSet.Validators))
	seqs := make([]string, 0, len(valSet.Validators))
	r := rand.New(rand.NewSource(time.Now().UnixMilli()))

	for i, val := range valSet.Validators {

		validator := stakeTypes.Validator{
			// #nosec G115
			ValId:      uint64(i),
			StartEpoch: 0,
			EndEpoch:   0,
			// #nosec G115
			Nonce:       uint64(i),
			VotingPower: 100,
			PubKey:      val.PubKey,
			Signer:      val.Signer,
			LastUpdated: time.Now().String(),
		}

		validators = append(validators, &validator)
		seqs = append(seqs, strconv.Itoa(simulation.RandIntBetween(r, 1, 1000000)))
	}

	// set validators and delegations
	stakingGenesis := stakeTypes.NewGenesisState(validators, *valSet, seqs)
	genesisState[stakeTypes.ModuleName] = codec.MustMarshalJSON(stakingGenesis)

	totalSupply := sdk.NewCoins()
	for _, b := range balances {
		// add genesis acc tokens to total supply
		totalSupply = totalSupply.Add(b.Coins...)
	}

	// update total supply
	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, totalSupply, []banktypes.Metadata{}, []banktypes.SendEnabled{})
	genesisState[banktypes.ModuleName] = codec.MustMarshalJSON(bankGenesis)

	return genesisState, nil
}
