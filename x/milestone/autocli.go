package milestone

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	_ "cosmossdk.io/api/cosmos/crypto/secp256k1" // register so that it shows up in protoregistry.GlobalTypes
	_ "cosmossdk.io/api/cosmos/crypto/secp256r1" // register so that it shows up in protoregistry.GlobalTypes

	milestone "github.com/0xPolygon/heimdall-v2/api/heimdallv2/milestone/v1"
)

// AutoCLIOptions returns the auto cli options for the module (query and tx)
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: milestone.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod:      "Params",
					Use:            "get-params",
					Short:          "Get milestone params",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "Count",
					Use:            "get-count",
					Short:          "Get milestone count",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "LatestMilestone",
					Use:            "get-latest-milestone",
					Short:          "Get latest milestone",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "Milestone",
					Use:            "get-milestone-by-id",
					Short:          "Get milestone by id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod:      "MilestoneProposer",
					Use:            "get-milestone-proposer",
					Short:          "Get milestone proposer",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "times"}},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: milestone.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Milestone",
					Use:       "milestone [proposer] [startBlock] [endBlock] [hash] [borChainId] [milestoneId]",
					Short:     "Send milestone tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "proposer"},
						{ProtoField: "startBlock"},
						{ProtoField: "endBlock"},
						{ProtoField: "hash"},
						{ProtoField: "borChainId"},
						{ProtoField: "milestoneId"},
					},
				},
				{
					RpcMethod:      "MilestoneTimeout",
					Use:            "milestone-timeout [proposer] [startBlock] [endBlock] [hash] [borChainId] [milestoneId]",
					Short:          "Send milestone timeout tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
			},
		},
	}
}
