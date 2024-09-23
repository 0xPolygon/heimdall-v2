package stake

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	_ "cosmossdk.io/api/cosmos/crypto/secp256k1" // register so that it shows up in protoregistry.GlobalTypes
	_ "cosmossdk.io/api/cosmos/crypto/secp256r1" // register so that it shows up in protoregistry.GlobalTypes

	stake "github.com/0xPolygon/heimdall-v2/api/heimdallv2/stake/v1"
)

// AutoCLIOptions returns the auto cli options for the module (query and tx)
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: stake.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod:      "CurrentValidatorSet",
					Use:            "current-validator-set",
					Short:          "Query all validators which are currently active in validator set",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "Signer",
					Use:            "signer [valAddress]",
					Short:          "Query validator info for given validator address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "valAddress"}},
				},
				{
					RpcMethod:      "Validator",
					Use:            "validator [id]",
					Short:          "Query validator info for a given validator id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod: "ValidatorStatus",
					Use:       "validator-status [valAddress]",
					Short:     "Query validator status for given validator address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "valAddress"}},
				},
				{
					RpcMethod:      "TotalPower",
					Use:            "total-power",
					Short:          "Query total power of the validator set",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "StakingIsOldTx",
					Use:            "is-old-tx [txHash] [logIndex]",
					Short:          "Check if a tx is old (already submitted)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "txHash"}, {ProtoField: "logIndex"}},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: stake.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "StakeUpdate",
					Use:       "stake-update [valAddress] [valId] [txHash] [amount] [logIndex] [blockNumber] [nonce]",
					Short:     "Update stake for a validator",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "valAddress"},
						{ProtoField: "valId"},
						{ProtoField: "txHash"},
						{ProtoField: "amount"},
						{ProtoField: "logIndex"},
						{ProtoField: "blockNumber"},
						{ProtoField: "nonce"},
					},
				},
				{
					RpcMethod: "ValidatorExit",
					Use:       "validator-exit [valAddress] [valId] [txHash] [deactivationEpoch] [logIndex] [blockNumber] [nonce]",
					Short:     "Exit validator",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "valAddress"},
						{ProtoField: "valId"},
						{ProtoField: "txHash"},
						{ProtoField: "deactivationEpoch"},
						{ProtoField: "logIndex"},
						{ProtoField: "blockNumber"},
						{ProtoField: "nonce"},
					},
				},
			},
		},
	}
}
