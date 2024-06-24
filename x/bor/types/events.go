package types

// bor module event types
const (
	EventTypeProposeSpan = "propose-span"

	AttributeKeySpanID         = "span-id"
	AttributeKeySpanStartBlock = "start-block"
	AttributeKeySpanEndBlock   = "end-block"

	AttributeValueCategory = ModuleName

	// TODO HV2: these should be defined under heimdall-v2/types

	AttributeKeyTxHash       = "txhash"
	AttributeKeySideTxResult = "side-tx-result"
)
