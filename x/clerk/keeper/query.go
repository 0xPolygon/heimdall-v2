package keeper

import (
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

var _ types.QueryServer = Keeper{}
