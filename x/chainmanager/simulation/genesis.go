package simulation

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strconv"

	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/ethereum/go-ethereum/common"
)

const (
	MainChainTxConfirmations = "main_chain_tx_confirmations"
	BorChainTxConfirmations  = "bor_chain_tx_confirmations"

	BorChainID = "bor_chain_id"
	// #nosec G101 -- suppress gosec warning
	PolTokenAddress       = "pol_token_address"
	StakingManagerAddress = "staking_manager_address"
	SlashManagerAddress   = "slash_manager_address"
	RootChainAddress      = "root_chain_address"
	StakingInfoAddress    = "staking_info_address"
	StateSenderAddress    = "state_sender_address"

	// Bor Chain Contracts

	StateReceiverAddress = "state_receiver_address"
	ValidatorSetAddress  = "validator_set_address"
)

// genMainChainTxConfirmations returns a randomized mainchain tx confirmations
func genMainChainTxConfirmations(r *rand.Rand) uint64 {
	return uint64(r.Intn(100) + 1)
}

// genBorChainTxConfirmations returns a randomized mainchain tx confirmations
func genBorChainTxConfirmations(r *rand.Rand) uint64 {
	return uint64(r.Intn(100) + 1)
}

// genBorChainID returns a randomized bor chain id
func genBorChainID(r *rand.Rand) string {
	return strconv.Itoa(r.Intn(math.MaxInt32))
}

func genAddress(r *rand.Rand) string {
	return common.BigToAddress(big.NewInt(int64(r.Intn(math.MaxInt64)))).String()
}

// RandomizedGenState generates a random GenesisState for chainmanager
func RandomizedGenState(simState *module.SimulationState) {
	var (
		mainChainTxConfirmations uint64
		borChainTxConfirmations  uint64
		borChainID               string
		polTokenAddress          string
		stakingManagerAddress    string
		slashManagerAddress      string
		rootChainAddress         string
		stakingInfoAddress       string
		stateSenderAddress       string
		stateReceiverAddress     string
		validatorSetAddress      string
	)

	simState.AppParams.GetOrGenerate(MainChainTxConfirmations, &mainChainTxConfirmations, simState.Rand, func(r *rand.Rand) {
		mainChainTxConfirmations = genMainChainTxConfirmations(r)
	})

	simState.AppParams.GetOrGenerate(BorChainTxConfirmations, &borChainTxConfirmations, simState.Rand, func(r *rand.Rand) {
		borChainTxConfirmations = genBorChainTxConfirmations(r)
	})

	simState.AppParams.GetOrGenerate(BorChainID, &borChainID, simState.Rand, func(r *rand.Rand) {
		borChainID = genBorChainID(r)
	})

	simState.AppParams.GetOrGenerate(PolTokenAddress, &polTokenAddress, simState.Rand, func(r *rand.Rand) {
		polTokenAddress = genAddress(r)
	})

	simState.AppParams.GetOrGenerate(StakingManagerAddress, &stakingManagerAddress, simState.Rand, func(r *rand.Rand) {
		stakingManagerAddress = genAddress(r)
	})

	simState.AppParams.GetOrGenerate(SlashManagerAddress, &slashManagerAddress, simState.Rand, func(r *rand.Rand) {
		slashManagerAddress = genAddress(r)
	})

	simState.AppParams.GetOrGenerate(RootChainAddress, &rootChainAddress, simState.Rand, func(r *rand.Rand) {
		rootChainAddress = genAddress(r)
	})

	simState.AppParams.GetOrGenerate(StakingInfoAddress, &stakingInfoAddress, simState.Rand, func(r *rand.Rand) {
		stakingInfoAddress = genAddress(r)
	})

	simState.AppParams.GetOrGenerate(StateSenderAddress, &stateSenderAddress, simState.Rand, func(r *rand.Rand) {
		stateSenderAddress = genAddress(r)
	})

	simState.AppParams.GetOrGenerate(StateReceiverAddress, &stateReceiverAddress, simState.Rand, func(r *rand.Rand) {
		stateReceiverAddress = genAddress(r)
	})

	simState.AppParams.GetOrGenerate(ValidatorSetAddress, &validatorSetAddress, simState.Rand, func(r *rand.Rand) {
		validatorSetAddress = genAddress(r)
	})

	chainParams := types.ChainParams{
		BorChainId:            borChainID,
		PolTokenAddress:       polTokenAddress,
		StakingManagerAddress: stakingManagerAddress,
		SlashManagerAddress:   slashManagerAddress,
		RootChainAddress:      rootChainAddress,
		StakingInfoAddress:    stakingInfoAddress,
		StateSenderAddress:    stateSenderAddress,
		StateReceiverAddress:  stateReceiverAddress,
		ValidatorSetAddress:   validatorSetAddress,
	}

	params := types.NewParams(mainChainTxConfirmations, borChainTxConfirmations, chainParams)
	chainManagerGenesis := types.NewGenesisState(params)

	bz, err := json.MarshalIndent(&chainManagerGenesis.Params, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated chainmanager parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(chainManagerGenesis)
}
