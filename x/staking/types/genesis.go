package types

// import (
// 	"encoding/json"

// 	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
// 	"github.com/cosmos/cosmos-sdk/codec"
// 	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
// )

// // NewGenesisState creates a new GenesisState instanc e
// func NewGenesisState(validators []*hmTypes.Validator,
// 	currentValSet hmTypes.ValidatorSet,
// 	stakingSequences []string) *GenesisState {
// 	return &GenesisState{
// 		Validators:          validators,
// 		CurrentValidatorSet: currentValSet,
// 		StakingSequences:    stakingSequences,
// 	}
// }

// // DefaultGenesisState gets the raw genesis raw message for testing
// func DefaultGenesisState() *GenesisState {
// 	return &GenesisState{}
// }

// // GetGenesisStateFromAppState returns x/staking GenesisState given raw application
// // genesis state.
// func GetGenesisStateFromAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
// 	var genesisState GenesisState

// 	if appState[ModuleName] != nil {
// 		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
// 	}

// 	return &genesisState
// }

// // UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
// func (g GenesisState) UnpackInterfaces(c codectypes.AnyUnpacker) error {
// 	for i := range g.Validators {
// 		if err := g.Validators[i].UnpackInterfaces(c); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }
