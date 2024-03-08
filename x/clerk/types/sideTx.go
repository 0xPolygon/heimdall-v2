package types

import (
	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
)

type SideMsgServer interface {
	//SideHandler to register specific sideHandler based on methodName
	SideTxHandler(methodName string) hmTypes.SideTxHandler

	//PostHandler to register specific postHandler based on methodName
	PostTxHandler(methodName string) hmTypes.PostTxHandler
}
