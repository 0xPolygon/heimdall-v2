package types

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
)

// NewGenesisState creates a new GenesisState instance
func NewGenesisState(validators []*Validator,
	currentValSet ValidatorSet,
	stakingSequences []string,
) *GenesisState {
	return &GenesisState{
		Validators:          validators,
		CurrentValidatorSet: currentValSet,
		StakingSequences:    stakingSequences,
	}
}

type PublicKey struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type PrivKey struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ValidatorKey struct {
	Address string    `json:"address"`
	PubKey  PublicKey `json:"pub_key"`
	PrivKey PrivKey   `json:"priv_key"`
}

// DefaultGenesisState gets the raw genesis raw message for testing
func DefaultGenesisState() *GenesisState {
	content, err := os.ReadFile("/Users/alanjewar/var/lib/heimdall/config/priv_validator_key.json")
	if err != nil {
		fmt.Print(err)
	}

	var validatorKey ValidatorKey
	err = json.Unmarshal(content, &validatorKey)
	if err != nil {
		fmt.Print(err)
	}

	signer := "0x" + validatorKey.Address
	pubKeyValue := validatorKey.PubKey.Value

	fmt.Println("Signer Address: ", signer)
	fmt.Println("Public Key: ", pubKeyValue)

	return &GenesisState{
		Validators: []*Validator{
			{
				ValId:            1,
				StartEpoch:       0,
				EndEpoch:         1000000,
				Nonce:            0,
				VotingPower:      1000,
				PubKey:           []byte(pubKeyValue),
				Signer:           signer,
				LastUpdated:      "hello",
				Jailed:           false,
				ProposerPriority: 0,
			},
		},
		CurrentValidatorSet: ValidatorSet{
			Validators: []*Validator{
				{
					ValId:            1,
					StartEpoch:       0,
					EndEpoch:         1000000,
					Nonce:            0,
					VotingPower:      1000,
					PubKey:           []byte(pubKeyValue),
					Signer:           signer,
					LastUpdated:      "hello",
					Jailed:           false,
					ProposerPriority: 0,
				},
			},
		},
		StakingSequences: []string{"initial_stake"},
	}
}

// GetGenesisStateFromAppState returns x/stake GenesisState given raw application
// genesis state.
func GetGenesisStateFromAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return &genesisState
}

// SetGenesisStateToAppState sets x/stake GenesisState into raw application
// genesis state.
func SetGenesisStateToAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage, validators []*Validator, currentValSet ValidatorSet) (map[string]json.RawMessage, error) {
	stakeState := GetGenesisStateFromAppState(cdc, appState)
	stakeState.Validators = validators
	stakeState.CurrentValidatorSet = currentValSet
	appState[ModuleName] = cdc.MustMarshalJSON(stakeState)

	return appState, nil
}
