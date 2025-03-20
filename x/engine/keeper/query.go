package keeper

import (
	"github.com/0xPolygon/heimdall-v2/x/engine/types"
)

var _ types.QueryServer = Keeper{}
