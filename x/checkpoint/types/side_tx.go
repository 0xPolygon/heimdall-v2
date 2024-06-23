package types

import (
	context "context"
	fmt "fmt"

	hmModule "github.com/0xPolygon/heimdall-v2/module"
	sdk "github.com/cosmos/cosmos-sdk/types"
	grpc "google.golang.org/grpc"
)

type SideMsgServer interface {
	//SideTxHandler to register specific sideHandler based on methodName
	SideTxHandler(methodName string) hmModule.SideTxHandler

	//PostTxHandler to register specific postHandler based on methodName
	PostTxHandler(methodName string) hmModule.PostTxHandler
}

func RegisterSideMsgServer(sideCfg hmModule.SideTxConfigurator, srv SideMsgServer) {
	serviceDesc := _Msg_serviceDesc

	for _, service := range serviceDesc.Methods {

		var requestTypeName string

		// NOTE: This is how we pull the concrete request type for each handler for registering in the InterfaceRegistry.
		// This approach is maybe a bit hacky, but less hacky than reflecting on the handler object itself.
		// We use a no-op interceptor to avoid actually calling into the handler itself.
		_, _ = service.Handler(nil, context.Background(), func(i interface{}) error {
			msg, ok := i.(sdk.Msg)
			if !ok {
				// We panic here because there is no other alternative and the app cannot be initialized correctly
				// this should only happen if there is a problem with code generation in which case the app won't
				// work correctly anyway.
				panic(fmt.Errorf("unable to register service method : %T does not implement sdk.Msg", i))
			}

			requestTypeName = sdk.MsgTypeURL(msg)
			return nil
		}, noopInterceptor)

		sideHandler := srv.SideTxHandler(requestTypeName)

		postHandler := srv.PostTxHandler(requestTypeName)

		if sideHandler == nil || postHandler == nil {
			continue
		}

		err := sideCfg.RegisterSideHandler(requestTypeName, sideHandler)
		if err != nil {
			panic("error in registering the side handler")
		}

		err = sideCfg.RegisterPostHandler(requestTypeName, postHandler)
		if err != nil {
			panic("error in registering the post handler")
		}
	}
}

func noopInterceptor(_ context.Context, _ interface{}, _ *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (interface{}, error) {
	return nil, nil
}
