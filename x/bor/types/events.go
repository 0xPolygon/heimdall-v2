package types

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// bor module event types
const (
	EventTypeSpan          = "span"
	EventTypeProposeSpan   = "propose-span"
	EventTypeBackfillSpans = "backfill-spans"

	AttributeKeySpanID           = "span-id"
	AttributeKeySpanStartBlock   = "start-block"
	AttributeKeySpanEndBlock     = "end-block"
	AttributesKeyLatestSpanId    = "latest-span-id"
	AttributesKeyLatestBorBlock  = "latest-bor-block"
	AttributesKeyLatestBorSpanId = "latest-bor-span-id"

	AttributeValueCategory = ModuleName
)

func NewSpanEvent(span *Span) sdk.Event {
	return sdk.NewEvent(
		EventTypeSpan,
		sdk.NewAttribute("id", strconv.FormatUint(span.Id, 10)),
		sdk.NewAttribute("start_block", strconv.FormatUint(span.StartBlock, 10)),
		sdk.NewAttribute("end_block", strconv.FormatUint(span.EndBlock, 10)),
		// This assumes that we're post veblop where we only have a single producer in the set
		sdk.NewAttribute("block_producer", span.SelectedProducers[0].Signer),
	)
}
