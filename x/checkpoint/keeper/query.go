package keeper

import (
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

var _ types.QueryServer = Keeper{}
