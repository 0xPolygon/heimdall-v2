package bor

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	"github.com/0xPolygon/heimdall-v2/api/heimdallv2/bor"
)

func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: bor.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "SpanById",
					Use:       "span-by-id [id]",
					Short:     "Query bor span by id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "span_id"},
					},
				},
				{
					RpcMethod: "SpanList",
					Use:       "span-list",
					Short:     "Query list of bor spans",
				},
				{
					RpcMethod: "LatestSpan",
					Use:       "latest-span",
					Short:     "Query latest bor span",
				},
				{
					RpcMethod: "NextSpanSeed",
					Use:       "next-span-seed",
					Short:     "Query next bor span seed",
				},
				{
					RpcMethod: "NextSpan",
					Use:       "next-span",
					Short:     "Query next bor span",
				},
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Query bor params",
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: bor.Msg_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "ProposeSpan",
					Use:       "propose-span [proposer] [span-id] [start-block] [chain-id]",
					Short:     "Propose a new bor span",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "proposer"},
						{ProtoField: "span_id"},
						{ProtoField: "start_block"},
						{ProtoField: "chain_id"},
					},
				},
			},
		},
	}
}
