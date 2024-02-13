package keeper

import (
	"context"

	"github.com/gogo/protobuf/codec"
)

type Keeper struct {
	cdc *codec.Codec
}

// GetParams gets the chainmanager module's parameters.
func (k Keeper) GetParams(ctx context.Context) (params Params) {
	params = Params{}
	return
}

type Params struct {
	MainchainTxConfirmations  uint64      `json:"mainchain_tx_confirmations" yaml:"mainchain_tx_confirmations"`
	MaticchainTxConfirmations uint64      `json:"maticchain_tx_confirmations" yaml:"maticchain_tx_confirmations"`
	ChainParams               ChainParams `json:"chain_params" yaml:"chain_params"`
}

type ChainParams struct {
	BorChainID            string `json:"bor_chain_id" yaml:"bor_chain_id"`
	MaticTokenAddress     string `json:"matic_token_address" yaml:"matic_token_address"`
	StakingManagerAddress string `json:"staking_manager_address" yaml:"staking_manager_address"`
	SlashManagerAddress   string `json:"slash_manager_address" yaml:"slash_manager_address"`
	RootChainAddress      string `json:"root_chain_address" yaml:"root_chain_address"`
	StakingInfoAddress    string `json:"staking_info_address" yaml:"staking_info_address"`
	StateSenderAddress    string `json:"state_sender_address" yaml:"state_sender_address"`

	// Bor Chain Contracts
	StateReceiverAddress string `json:"state_receiver_address" yaml:"state_receiver_address"`
	ValidatorSetAddress  string `json:"validator_set_address" yaml:"validator_set_address"`
}
