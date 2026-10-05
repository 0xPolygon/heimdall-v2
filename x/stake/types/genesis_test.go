package types_test

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestNewGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("creates genesis state with all parameters", func(t *testing.T) {
		t.Parallel()

		var validators []*types.Validator
		valSet := types.ValidatorSet{}
		stakingSeqs := []string{"seq1", "seq2"}

		gs := types.NewGenesisState(validators, valSet, stakingSeqs)

		require.NotNil(t, gs)
		require.Equal(t, validators, gs.Validators)
		require.Equal(t, valSet, gs.CurrentValidatorSet)
		require.Equal(t, stakingSeqs, gs.StakingSequences)
	})

	t.Run("creates genesis state with nil validators", func(t *testing.T) {
		t.Parallel()

		gs := types.NewGenesisState(nil, types.ValidatorSet{}, nil)

		require.NotNil(t, gs)
		require.Nil(t, gs.Validators)
		require.Nil(t, gs.StakingSequences)
	})
}

func TestDefaultGenesisState(t *testing.T) {
	t.Parallel()

	t.Run("returns default genesis state", func(t *testing.T) {
		t.Parallel()

		gs := types.DefaultGenesisState()

		require.NotNil(t, gs)
	})

	t.Run("default genesis state is valid", func(t *testing.T) {
		t.Parallel()

		gs := types.DefaultGenesisState()

		err := gs.Validate()
		require.NoError(t, err)
	})
}

func TestGenesisState_Validate(t *testing.T) {
	t.Parallel()

	t.Run("validates correct genesis state", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{}

		err := gs.Validate()
		require.NoError(t, err)
	})

	t.Run("rejects empty staking sequence", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			StakingSequences: []string{""},
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid sequence")
	})

	t.Run("rejects sequence with empty string in middle", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			StakingSequences: []string{"seq1", "", "seq3"},
		}

		err := gs.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid sequence")
	})

	t.Run("validates genesis with valid sequences", func(t *testing.T) {
		t.Parallel()

		gs := types.GenesisState{
			StakingSequences: []string{"seq1", "seq2", "seq3"},
		}

		err := gs.Validate()
		require.NoError(t, err)
	})
}

func TestGetGenesisStateFromAppState(t *testing.T) {
	t.Parallel()

	t.Run("retrieves genesis state from app state", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		gs := types.DefaultGenesisState()
		appState := make(map[string]json.RawMessage)
		appState[types.ModuleName] = cdc.MustMarshalJSON(gs)

		result := types.GetGenesisStateFromAppState(cdc, appState)

		require.NotNil(t, result)
	})

	t.Run("returns empty genesis state when module not in app state", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		cdc := codec.NewProtoCodec(interfaceRegistry)

		appState := make(map[string]json.RawMessage)

		result := types.GetGenesisStateFromAppState(cdc, appState)

		require.NotNil(t, result)
	})
}

func TestSetGenesisStateToAppState(t *testing.T) {
	t.Parallel()

	t.Run("sets genesis state to app state", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		appState := make(map[string]json.RawMessage)
		var validators []*types.Validator
		valSet := types.ValidatorSet{}

		result, err := types.SetGenesisStateToAppState(cdc, appState, validators, valSet)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Contains(t, result, types.ModuleName)
	})

	t.Run("updates existing app state", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		// Initialize with the default state
		gs := types.DefaultGenesisState()
		appState := make(map[string]json.RawMessage)
		appState[types.ModuleName] = cdc.MustMarshalJSON(gs)

		// Update with new validators
		var validators []*types.Validator
		valSet := types.ValidatorSet{}

		result, err := types.SetGenesisStateToAppState(cdc, appState, validators, valSet)

		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestGenesisState_ValidateValidatorSigners(t *testing.T) {
	t.Parallel()

	newVal := func(id uint64) *types.Validator {
		pub := secp256k1.GenPrivKey().PubKey()
		v, err := types.NewValidator(id, 0, 0, 1, 10, pub, pub.Address().String())
		require.NoError(t, err)
		return v
	}
	v1, v2, outside := newVal(1), newVal(2), newVal(3)
	rotated := *v2
	rotatedNew := newVal(2)

	base := func() types.GenesisState {
		return types.GenesisState{
			Validators:          []*types.Validator{v1, v2, outside, rotatedNew},
			CurrentValidatorSet: types.ValidatorSet{Validators: []*types.Validator{v1, rotatedNew}},
			ValidatorSigners: []types.ValidatorSigner{
				{ValId: 1, Signer: v1.Signer},
				{ValId: 2, Signer: rotatedNew.Signer},
				{ValId: 3, Signer: outside.Signer},
			},
		}
	}

	tests := []struct {
		name      string
		mutate    func(gs *types.GenesisState)
		wantErr   string
		keepOrder bool
	}{
		{name: "consistent export", mutate: func(*types.GenesisState) {}},
		{name: "current set not sorted by signer", wantErr: "current set is not sorted by signer", keepOrder: true, mutate: func(gs *types.GenesisState) {
			sort.Sort(sort.Reverse(types.ValidatorsByAddress(gs.CurrentValidatorSet.Validators)))
		}},
		{name: "legacy genesis without signers", mutate: func(gs *types.GenesisState) {
			gs.ValidatorSigners = nil
			gs.CurrentValidatorSet = types.ValidatorSet{}
		}},
		{name: "empty current set", wantErr: "require a current validator set", mutate: func(gs *types.GenesisState) {
			gs.CurrentValidatorSet = types.ValidatorSet{}
		}},
		{name: "duplicate signer record", wantErr: "duplicate validator record", mutate: func(gs *types.GenesisState) {
			gs.Validators = append(gs.Validators, &rotated)
		}},
		{name: "duplicate id mapping", wantErr: "duplicate signer mapping", mutate: func(gs *types.GenesisState) {
			gs.ValidatorSigners = append(gs.ValidatorSigners, types.ValidatorSigner{ValId: 2, Signer: v2.Signer})
		}},
		{name: "mapping to unknown signer", wantErr: "no matching record", mutate: func(gs *types.GenesisState) {
			gs.ValidatorSigners[2].Signer = newVal(3).Signer
		}},
		{name: "mapping to another validator's record", wantErr: "no matching record", mutate: func(gs *types.GenesisState) {
			gs.ValidatorSigners[0].Signer = outside.Signer
		}},
		{name: "validator without mapping", wantErr: "validator 3 has no signer mapping", mutate: func(gs *types.GenesisState) {
			gs.ValidatorSigners = gs.ValidatorSigners[:2]
		}},
		{name: "current set member without record", wantErr: "current set validator 4 has no record", mutate: func(gs *types.GenesisState) {
			gs.CurrentValidatorSet.Validators = append(gs.CurrentValidatorSet.Validators, newVal(4))
		}},
		{name: "current set member with another record's id", wantErr: "current set validator 7 does not match its record", mutate: func(gs *types.GenesisState) {
			member := *v1
			member.ValId = 7
			gs.CurrentValidatorSet.Validators[0] = &member
		}},
		{name: "current set member with another public key", wantErr: "current set validator 1 does not match its record", mutate: func(gs *types.GenesisState) {
			member := *v1
			member.PubKey = outside.PubKey
			gs.CurrentValidatorSet.Validators[0] = &member
		}},
		{name: "current set member with another power", wantErr: "current set validator 1 has power 70 but its record has 10", mutate: func(gs *types.GenesisState) {
			member := *v1
			member.VotingPower = 70
			gs.CurrentValidatorSet.Validators[0] = &member
		}},
		{name: "current set member with stale nonce", mutate: func(gs *types.GenesisState) {
			member := *v1
			member.Nonce = v1.Nonce - 1
			gs.CurrentValidatorSet.Validators[0] = &member
		}},
		{name: "current set lists an id twice", wantErr: "current set lists validator 1 twice", mutate: func(gs *types.GenesisState) {
			gs.CurrentValidatorSet.Validators = append(gs.CurrentValidatorSet.Validators, v1)
		}},
		{name: "current set total power off", wantErr: "total voting power 7 does not match", mutate: func(gs *types.GenesisState) {
			gs.CurrentValidatorSet.TotalVotingPower = 7
		}},
		{name: "current set total power consistent", mutate: func(gs *types.GenesisState) {
			gs.CurrentValidatorSet.TotalVotingPower = v1.VotingPower + rotatedNew.VotingPower
		}},
		{name: "current set proposer removed by the last update", mutate: func(gs *types.GenesisState) {
			gs.CurrentValidatorSet.Proposer = outside
		}},
		{name: "current set member on its pre-rotation signer", wantErr: "current set validator 2 is not the signer its id maps to", mutate: func(gs *types.GenesisState) {
			gs.CurrentValidatorSet.Validators[1] = v2
		}},
		{name: "id mapped back to its pre-rotation record", wantErr: "current set validator 2 is not the signer its id maps to", mutate: func(gs *types.GenesisState) {
			gs.ValidatorSigners[1].Signer = v2.Signer
		}},
		{name: "signers compared case-insensitively", mutate: func(gs *types.GenesisState) {
			gs.ValidatorSigners[0].Signer = strings.ToUpper(strings.TrimPrefix(v1.Signer, "0x"))
			member := *v1
			member.Signer = strings.ToUpper(v1.Signer)
			gs.CurrentValidatorSet.Validators[0] = &member
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gs := base()
			tc.mutate(&gs)
			if !tc.keepOrder {
				sort.Sort(types.ValidatorsByAddress(gs.CurrentValidatorSet.Validators))
			}

			err := gs.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestGenesisState_ValidatorSignersJSONRoundTrip(t *testing.T) {
	t.Parallel()

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	gs := types.GenesisState{ValidatorSigners: []types.ValidatorSigner{{ValId: 2, Signer: "0xabc"}, {ValId: 5, Signer: "0xdef"}}}

	bz, err := cdc.MarshalJSON(&gs)
	require.NoError(t, err)

	var got types.GenesisState
	require.NoError(t, cdc.UnmarshalJSON(bz, &got))
	require.Equal(t, gs.ValidatorSigners, got.ValidatorSigners)
}

func TestSetGenesisStateToAppState_ClearsValidatorSigners(t *testing.T) {
	t.Parallel()

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	exported := types.GenesisState{ValidatorSigners: []types.ValidatorSigner{{ValId: 1, Signer: "0xabc"}}}
	appState := map[string]json.RawMessage{types.ModuleName: cdc.MustMarshalJSON(&exported)}

	appState, err := types.SetGenesisStateToAppState(cdc, appState, nil, types.ValidatorSet{})
	require.NoError(t, err)
	require.Empty(t, types.GetGenesisStateFromAppState(cdc, appState).ValidatorSigners)
}

func TestGenesisState_ValidateCurrentSetMembership(t *testing.T) {
	t.Parallel()

	const ackCount = 10
	newVal := func(id, startEpoch uint64, power int64) *types.Validator {
		pub := secp256k1.GenPrivKey().PubKey()
		v, err := types.NewValidator(id, startEpoch, 0, 1, power, pub, pub.Address().String())
		require.NoError(t, err)
		return v
	}
	a, b := newVal(1, 0, 10), newVal(2, 0, 10)
	pending, retired := newVal(3, ackCount+5, 10), newVal(4, 0, 0)

	tests := []struct {
		name    string
		set     []*types.Validator
		signers []types.ValidatorSigner
		wantErr string
	}{
		{name: "set holds exactly the current records", set: []*types.Validator{a, b}, signers: []types.ValidatorSigner{{ValId: 1}}},
		{name: "legacy genesis is not checked", set: []*types.Validator{a}},
		{name: "current record missing from the set", set: []*types.Validator{a}, signers: []types.ValidatorSigner{{ValId: 1}}, wantErr: "validator 2 is current but not in the current set"},
		{name: "pending record in the set", set: []*types.Validator{a, b, pending}, signers: []types.ValidatorSigner{{ValId: 1}}, wantErr: "current set has 3 members but 2 validators are current"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gs := types.GenesisState{
				Validators:          []*types.Validator{a, b, pending, retired},
				CurrentValidatorSet: types.ValidatorSet{Validators: tc.set},
				ValidatorSigners:    tc.signers,
			}

			err := gs.ValidateCurrentSetMembership(ackCount)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tc.wantErr)
		})
	}
}
