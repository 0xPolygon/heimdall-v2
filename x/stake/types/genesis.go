package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"

	util "github.com/0xPolygon/heimdall-v2/common/hex"
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

// DefaultGenesisState gets the raw genesis message for testing
func DefaultGenesisState() *GenesisState {
	return &GenesisState{}
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

// SetGenesisStateToAppState sets x/stake GenesisState into the raw application
// genesis state.
func SetGenesisStateToAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage, validators []*Validator, currentValSet ValidatorSet) (map[string]json.RawMessage, error) {
	stakeState := GetGenesisStateFromAppState(cdc, appState)
	stakeState.Validators = validators
	stakeState.CurrentValidatorSet = currentValSet
	// the replaced validators no longer match any exported signer mapping
	stakeState.ValidatorSigners = nil
	appState[ModuleName] = cdc.MustMarshalJSON(stakeState)

	return appState, nil
}

// Validate validates the provided stake genesis state to ensure that listed
// validators and staking sequences are valid
func (data GenesisState) Validate() error {
	for _, validator := range data.Validators {
		if err := validator.ValidateBasic(); err != nil {
			return errors.New("invalid validator")
		}
	}

	for _, sq := range data.StakingSequences {
		if sq == "" {
			return errors.New("invalid sequence")
		}
	}

	return data.ValidateValidatorSigners()
}

// ValidateValidatorSigners checks that a genesis carrying ValidatorSigners describes a
// store InitGenesis can import verbatim: a non-empty current set, one record per signer,
// every validator ID mapped exactly once to one of its own records, and every current
// set member backed by a record with the same ID and public key.
func (data GenesisState) ValidateValidatorSigners() error {
	if len(data.ValidatorSigners) == 0 {
		return nil
	}
	if len(data.CurrentValidatorSet.Validators) == 0 {
		return errors.New("validator signers require a current validator set")
	}

	records, err := data.validatorRecords()
	if err != nil {
		return err
	}

	mapped, err := data.validateSignerMappings(records)
	if err != nil {
		return err
	}

	return data.validateCurrentSetRecords(records, mapped)
}

func (data GenesisState) validateSignerMappings(records map[string]*Validator) (map[uint64]string, error) {
	mapped := make(map[uint64]string, len(data.ValidatorSigners))
	for _, vs := range data.ValidatorSigners {
		if _, dup := mapped[vs.ValId]; dup {
			return nil, fmt.Errorf("duplicate signer mapping for validator %d", vs.ValId)
		}
		signer := util.FormatAddress(vs.Signer)
		if record, ok := records[signer]; !ok || record.ValId != vs.ValId {
			return nil, fmt.Errorf("validator %d is mapped to signer %s with no matching record", vs.ValId, vs.Signer)
		}
		mapped[vs.ValId] = signer
	}

	for _, v := range data.Validators {
		if _, ok := mapped[v.ValId]; !ok {
			return nil, fmt.Errorf("validator %d has no signer mapping", v.ValId)
		}
	}

	return mapped, nil
}

// validateCurrentSetRecords requires every member to be the record its ID maps to, with the
// same voting power: a signer change writes the new mapping and the same block's stake
// EndBlocker swaps the member, and that EndBlocker also applies every power change, so a
// committed export always satisfies both. CometBFT's genesis validators take their power from
// the records while vote extensions are tallied against this set. Nonce and last_updated
// aren't compared, as the set copy keeps them stale. The set is stored as exported, so its
// total power must match its members.
func (data GenesisState) validateCurrentSetRecords(records map[string]*Validator, mapped map[uint64]string) error {
	set := data.CurrentValidatorSet
	if err := validateSetTotalPower(set); err != nil {
		return err
	}

	inSet := make(map[uint64]struct{}, len(set.Validators))
	for _, v := range set.Validators {
		if _, dup := inSet[v.ValId]; dup {
			return fmt.Errorf("current set lists validator %d twice", v.ValId)
		}
		inSet[v.ValId] = struct{}{}

		if err := validateSetMember(v, records, mapped); err != nil {
			return err
		}
	}

	return nil
}

func validateSetMember(v *Validator, records map[string]*Validator, mapped map[uint64]string) error {
	signer := util.FormatAddress(v.Signer)
	record, ok := records[signer]
	if !ok {
		return fmt.Errorf("current set validator %d has no record", v.ValId)
	}
	if record.ValId != v.ValId || !bytes.Equal(record.PubKey, v.PubKey) {
		return fmt.Errorf("current set validator %d does not match its record", v.ValId)
	}
	if mapped[v.ValId] != signer {
		return fmt.Errorf("current set validator %d is not the signer its id maps to", v.ValId)
	}
	if record.VotingPower != v.VotingPower {
		return fmt.Errorf("current set validator %d has power %d but its record has %d", v.ValId, v.VotingPower, record.VotingPower)
	}

	return nil
}

// validateSetTotalPower doesn't check the proposer: the stake EndBlocker stores the set
// before re-electing one, so a proposer removed by that block's updates stays recorded.
func validateSetTotalPower(set ValidatorSet) error {
	var total int64
	for _, v := range set.Validators {
		total += v.VotingPower
	}

	if set.TotalVotingPower != 0 && set.TotalVotingPower != total {
		return fmt.Errorf("current set total voting power %d does not match its members' %d", set.TotalVotingPower, total)
	}

	return nil
}

func (data GenesisState) validatorRecords() (map[string]*Validator, error) {
	records := make(map[string]*Validator, len(data.Validators))
	for _, v := range data.Validators {
		signer := util.FormatAddress(v.Signer)
		if _, dup := records[signer]; dup {
			return nil, fmt.Errorf("duplicate validator record for signer %s", signer)
		}
		records[signer] = v
	}

	return records, nil
}
