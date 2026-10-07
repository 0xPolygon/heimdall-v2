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
					RpcMethod: "GetSpanById",
					Use:       "span-by-id [id]",
					Short:     "Query bor span by id",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
					},
				},
				{
					RpcMethod: "GetSpanList",
					Use:       "span-list",
					Short:     "Query list of bor spans",
				},
				{
					RpcMethod: "GetLatestSpan",
					Use:       "latest-span",
					Short:     "Query latest bor span",
				},
				{
					RpcMethod: "GetNextSpanSeed",
					Use:       "next-span-seed [id]",
					Short:     "Query next bor span seed",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "id"},
					},
				},
				{
					RpcMethod: "GetNextSpan",
					Use:       "next-span",
					Short:     "Query next bor span",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "span_id"},
						{ProtoField: "start_block"},
						{ProtoField: "bor_chain_id"},
					},
				},
				{
					RpcMethod: "GetBorParams",
					Use:       "params",
					Short:     "Query bor params",
				},
				{
					RpcMethod: "GetProducerPlannedDowntime",
					Use:       "producer-planned-downtime [producer_id]",
					Short:     "Query planned downtime for a producer",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "producer_id"},
					},
				},
				// Autocli's amino JSON encoder cannot print proto maps, so map responses are REST/gRPC only.
				{RpcMethod: "GetProducerVotes", Skip: true},
				{
					RpcMethod: "GetProducerVotesByValidatorId",
					Use:       "producer-votes-by-validator-id [validator_id]",
					Short:     "Query producer votes cast by a validator",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "validator_id"},
					},
				},
				{RpcMethod: "GetValidatorPerformanceScore", Skip: true},
				{
					RpcMethod: "GetValidatorPerformanceScoreByValidatorId",
					Use:       "validator-performance-score-by-validator-id [validator_id]",
					Short:     "Query performance score of a validator",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{
						{ProtoField: "validator_id"},
					},
				},
			},
		},
	}
}
