package types

import (
	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
)

type SideMsgServer interface {
	// SideTxHandler to register specific sideHandler based on methodName
	SideTxHandler(methodName string) hmTypes.SideTxHandler

	// PostTxHandler to register specific postHandler based on methodName
	PostTxHandler(methodName string) hmTypes.PostTxHandler
}
