package types

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
)

type SideMsgServer interface {
	// SideTxHandler to register specific sideHandler based on methodName
	SideTxHandler(methodName string) sidetxs.SideTxHandler

	// PostTxHandler to register specific postHandler based on methodName
	PostTxHandler(methodName string) sidetxs.PostTxHandler
}

func RegisterSideMsgServer(sideCfg sidetxs.SideTxConfigurator, srv SideMsgServer) {
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
			return
		}

		err = sideCfg.RegisterPostHandler(requestTypeName, postHandler)
		if err != nil {
			return
		}
	}
}

func noopInterceptor(_ context.Context, _ interface{}, _ *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (interface{}, error) {
	return nil, nil
}
