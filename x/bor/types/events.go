package types

// bor module event types
const (
	EventTypeProposeSpan = "propose-span"

	AttributeKeySuccess        = "success"
	AttributeKeySpanID         = "span-id"
	AttributeKeySpanStartBlock = "start-block"
	AttributeKeySpanEndBlock   = "end-block"

	AttributeValueCategory = ModuleName

	// TODO HV2: these should be defined under heimdall-v2/types

	AttributeKeyTxHash       = "txhash"
	AttributeKeyTxLogIndex   = "tx-log-index"
	AttributeKeySideTxResult = "side-tx-result"
)
