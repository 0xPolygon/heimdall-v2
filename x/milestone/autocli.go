package milestone

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	_ "cosmossdk.io/api/cosmos/crypto/secp256k1" // register so that it shows up in protoregistry.GlobalTypes
	_ "cosmossdk.io/api/cosmos/crypto/secp256r1" // register so that it shows up in protoregistry.GlobalTypes

	"github.com/0xPolygon/heimdall-v2/api/heimdallv2/milestone"
)

// AutoCLIOptions returns the auto cli options for the module (query and tx)
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: milestone.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod:      "GetParams",
					Use:            "get-params",
					Short:          "Get milestone params",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "GetMilestoneCount",
					Use:            "get-count",
					Short:          "Get milestone count",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "GetLatestMilestone",
					Use:            "get-latest-milestone",
					Short:          "Get latest milestone",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "GetMilestoneByNumber",
					Use:            "get-milestone-by-number",
					Short:          "Get milestone by number",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "number"}},
				},
				{
					RpcMethod:      "GetMilestoneProposerByTimes",
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
						{ProtoField: "start_block"},
						{ProtoField: "end_block"},
						{ProtoField: "hash"},
						{ProtoField: "bor_chain_id"},
						{ProtoField: "milestone_id"},
					},
				},
				{
					RpcMethod:      "MilestoneTimeout",
					Use:            "milestone-timeout",
					Short:          "Send milestone timeout tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
			},
		},
	}
}
