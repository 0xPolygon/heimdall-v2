package chainmanager

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	chainmanager "github.com/0xPolygon/heimdall-v2/api/heimdallv2/chainmanager"
)

func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: chainmanager.Query_ServiceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Query values set as chainmanager parameters.",
					Long:      "Query the current chainmanager parameters information",
				},
			},
		},
	}
}
