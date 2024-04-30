package topup

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	_ "cosmossdk.io/api/cosmos/crypto/secp256k1" // register so that it shows up in protoregistry.GlobalTypes
	_ "cosmossdk.io/api/cosmos/crypto/secp256r1" // register so that it shows up in protoregistry.GlobalTypes

	topupv1 "github.com/0xPolygon/heimdall-v2/api/heimdallv2/topup/v1"
)

// AutoCLIOptions returns the auto cli options for the module (query and tx)
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: topupv1.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod:      "TopupSequence",
					Use:            "topup-sequence [txHash] [logIndex]",
					Short:          "Query the sequence of a topup tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "txHash"}, {ProtoField: "logIndex"}},
				},
				{
					RpcMethod:      "IsOldTx",
					Use:            "is-old-tx [txHash] [logIndex]",
					Short:          "Check if a tx is old (already submitted)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "txHash"}, {ProtoField: "logIndex"}},
				},
				{
					RpcMethod:      "GetDividendAccount",
					Use:            "dividend-account [address]",
					Short:          "Query a dividend account by its address",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
				},
				{
					RpcMethod:      "GetDividendAccountRootHash",
					Use:            "dividend-account-root",
					Short:          "Query dividend account root hash",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "GetAccountProof",
					Use:            "account-proof [address]",
					Short:          "Query account proof",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
				},
				{
					RpcMethod: "GetAccountProofVerify",
					Use:       "verify-account-proof [address] [accountProof]",
					Short:     "Verify account proof",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "address"}, {ProtoField: "accountProof"}},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: topupv1.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "CreateTopupTx",
					Use:       "create-topup-tx [proposer] [user] [fee] [txHash] [logIndex] [blockNumber]",
					Short:     "Create a topup tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "proposer"}, {ProtoField: "user"}, {ProtoField: "fee"},
						{ProtoField: "txHash"}, {ProtoField: "logIndex"}, {ProtoField: "blockNumber"}},
				},
				{
					RpcMethod: "WithdrawFee",
					Use:       "withdraw-fee [proposer] [fee]",
					Short:     "Withdraw fee",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "proposer"}, {ProtoField: "fee"}},
				},
			},
		},
	}
}
