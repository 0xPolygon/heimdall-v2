package simulation_test

import (
	"encoding/json"
	"math/rand"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/0xPolygon/heimdall-v2/x/chainmanager/simulation"
	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/stretchr/testify/require"
)

// TestRandomizedGenState tests the normal scenario of applying RandomizedGenState.
// Abnormal scenarios are not tested here.
func TestRandomizedGenState(t *testing.T) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	s := rand.NewSource(1)
	r := rand.New(s)

	simState := module.SimulationState{
		AppParams:    make(simtypes.AppParams),
		Cdc:          cdc,
		Rand:         r,
		NumBonded:    3,
		BondDenom:    "matic",
		Accounts:     simtypes.RandomAccounts(r, 3),
		InitialStake: sdkmath.NewInt(1000),
		GenState:     make(map[string]json.RawMessage),
	}

	simulation.RandomizedGenState(&simState)

	var chainmanagerGenesis types.GenesisState
	simState.Cdc.MustUnmarshalJSON(simState.GenState[types.ModuleName], &chainmanagerGenesis)

	require.Equal(t, "646203300", chainmanagerGenesis.Params.ChainParams.BorChainId)
	require.Equal(t, "0x00000000000000000000000041f27Cc6F3875D04", chainmanagerGenesis.Params.ChainParams.MaticTokenAddress)
	require.Equal(t, "0x00000000000000000000000068255aaf95e94627", chainmanagerGenesis.Params.ChainParams.StakingManagerAddress)
	require.Equal(t, "0x0000000000000000000000001B6cffa2BA517936", chainmanagerGenesis.Params.ChainParams.SlashManagerAddress)
	require.Equal(t, "0x00000000000000000000000030b95fF183c471d4", chainmanagerGenesis.Params.ChainParams.RootChainAddress)
	require.Equal(t, "0x00000000000000000000000028B621587CB3AD0B", chainmanagerGenesis.Params.ChainParams.StakingInfoAddress)
	require.Equal(t, "0x0000000000000000000000003c04951aa42655d9", chainmanagerGenesis.Params.ChainParams.StateSenderAddress)
	require.Equal(t, "0x000000000000000000000000243A768b7C4E0B68", chainmanagerGenesis.Params.ChainParams.StateReceiverAddress)
	require.Equal(t, "0x00000000000000000000000025845c95d4491d1b", chainmanagerGenesis.Params.ChainParams.ValidatorSetAddress)

	require.Equal(t, uint64(41), chainmanagerGenesis.Params.MainChainTxConfirmations)
	require.Equal(t, uint64(57), chainmanagerGenesis.Params.BorChainTxConfirmations)
}

// TestRandomizedGenState1 tests abnormal scenarios of applying RandomizedGenState.
func TestRandomizedGenState1(t *testing.T) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	s := rand.NewSource(1)
	r := rand.New(s)

	// all these tests will panic
	tests := []struct {
		simState module.SimulationState
		panicMsg string
	}{
		{ // panic => reason: incomplete initialization of the simState
			module.SimulationState{}, "invalid memory address or nil pointer dereference"},
		{ // panic => reason: incomplete initialization of the simState
			module.SimulationState{
				AppParams: make(simtypes.AppParams),
				Cdc:       cdc,
				Rand:      r,
			}, "assignment to entry in nil map"},
	}

	for _, tt := range tests {
		require.Panicsf(t, func() { simulation.RandomizedGenState(&tt.simState) }, tt.panicMsg)
	}
}
