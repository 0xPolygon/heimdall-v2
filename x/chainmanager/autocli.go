package chainmanager

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
)

func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			// TODO HV2: add Service once grpc.pb.go is generated
			// Service: types._Query_serviceDesc.ServiceName,
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
