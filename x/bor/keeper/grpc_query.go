package keeper

import "github.com/0xPolygon/heimdall-v2/x/bor/types"

type Querier struct {
	keeper Keeper
}

var _ types.QueryServer = Querier{}

func NewQuerier(keeper Keeper) Querier {
	return Querier{keeper: keeper}
}
