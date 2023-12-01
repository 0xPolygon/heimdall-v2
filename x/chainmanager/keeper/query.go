package keeper

import (
	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

var _ types.QueryServer = Keeper{}
