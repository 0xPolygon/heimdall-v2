package types

import (
	"fmt"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/ethereum/go-ethereum/common"
	// types "github.com/0xPolygon/heimdall-v2/proto/heimdallv2/chainmanager"
)

// Default parameter values
const (
	DefaultMainChainTxConfirmations         uint64 = 6
	DefaultBorChainTxConfirmations          uint64 = 10
	DefaultBorChainMilestoneTxConfirmations uint64 = 16

	// TODO HV2: uncomment when this PR is merged: https://github.com/0xPolygon/cosmos-sdk/pull/3
	// DefaultStateReceiverAddress sdk.AccAddress = sdk.AccAddressFromHex(("0x0000000000000000000000000000000000001001")
	// DefaultValidatorSetAddress  sdk.AccAddress = sdk.AccAddressFromHex(("0x0000000000000000000000000000000000001000")
)

// TODO HV2: probably not be needed since the individual modules store their own params
// Parameter keys
// var (
// 	KeyMainchainTxConfirmations  = []byte("MainchainTxConfirmations")
// 	KeyMaticchainTxConfirmations = []byte("MaticchainTxConfirmations")
// 	KeyChainParams               = []byte("ChainParams")
// )

// var _ subspace.ParamSet = &Params{}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return Params{
		MainChainTxConfirmations: DefaultMainChainTxConfirmations,
		BorChainTxConfirmations:  DefaultBorChainTxConfirmations,
		ChainParams: ChainParams{
			BorChainId: helper.DefaultBorChainID,
			// TODO HV2: uncomment when this PR is merged: https://github.com/0xPolygon/cosmos-sdk/pull/3
			// StateReceiverAddress: DefaultStateReceiverAddress,
			// ValidatorSetAddress:  DefaultValidatorSetAddress,
		},
	}
}

// NewParams creates a new Params object
func NewParams(mainChainTxConfirmations uint64, borChainTxConfirmations uint64, chainParams ChainParams) Params {
	return Params{
		MainChainTxConfirmations: mainChainTxConfirmations,
		BorChainTxConfirmations:  borChainTxConfirmations,
		ChainParams:              chainParams,
	}
}

// TODO HV2: probably not be needed since the individual modules store their own params
// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
// pairs of auth module's parameters.
// nolint
// func (p *Params) ParamSetPairs() subspace.ParamSetPairs {
// 	return subspace.ParamSetPairs{
// 		{KeyMainchainTxConfirmations, &p.MainchainTxConfirmations},
// 		{KeyMaticchainTxConfirmations, &p.MaticchainTxConfirmations},
// 		{KeyChainParams, &p.ChainParams},
// 	}
// }

// Validate checks that the parameters have valid values.
func (p Params) Validate() error {
	if err := validateHeimdallAddress("matic_token_address", p.ChainParams.MaticTokenAddress); err != nil {
		return err
	}

	if err := validateHeimdallAddress("staking_manager_address", p.ChainParams.StakingManagerAddress); err != nil {
		return err
	}

	if err := validateHeimdallAddress("slash_manager_address", p.ChainParams.SlashManagerAddress); err != nil {
		return err
	}

	if err := validateHeimdallAddress("root_chain_address", p.ChainParams.RootChainAddress); err != nil {
		return err
	}

	if err := validateHeimdallAddress("staking_info_address", p.ChainParams.StakingInfoAddress); err != nil {
		return err
	}

	if err := validateHeimdallAddress("state_sender_address", p.ChainParams.StateSenderAddress); err != nil {
		return err
	}

	if err := validateHeimdallAddress("state_receiver_address", p.ChainParams.StateReceiverAddress); err != nil {
		return err
	}

	if err := validateHeimdallAddress("validator_set_address", p.ChainParams.ValidatorSetAddress); err != nil {
		return err
	}

	return nil
}

func validateHeimdallAddress(key string, value string) error {
	if !common.IsHexAddress(value) {
		return fmt.Errorf("Invalid address for value %s for %s in chain_params", value, key)
	}
	if value == "" {
		return fmt.Errorf("Invalid value for key %s in chain_params", key)
	}

	return nil
}
