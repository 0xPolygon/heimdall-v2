package keeper

import (
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

var _ types.QueryServer = Keeper{}
