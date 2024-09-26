package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	k *Keeper
}

// NewQueryServer creates a new querier for chainmanager clients.
func NewQueryServer(k *Keeper) types.QueryServer {
	return queryServer{
		k: k,
	}
}

// GetParams implements the gRPC service handler for querying x/chainmanager parameters.
func (q queryServer) GetParams(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, err := q.k.GetParams(sdkCtx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get params: %s", err)
	}

	return &types.QueryParamsResponse{Params: params}, nil
}
