package types

import (
	"sort"
	"strings"

	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// SortValidatorByAddress sorts a slice of validators by address
// to sort it we compare the values of the Signer(HeimdallAddress i.e. [20]byte)
func SortValidatorByAddress(a []staketypes.Validator) []staketypes.Validator {
	sort.Slice(a, func(i, j int) bool {
		return strings.Compare(a[i].Signer, a[j].Signer) < 0
	})

	return a
}

// SortSpansById sorts spans by SpanID
func SortSpansById(a []Span) {
	sort.Slice(a, func(i, j int) bool {
		return a[i].Id < a[j].Id
	})
}

func IsBlockCloseToSpanEnd(blockNumber, spanEnd uint64) bool {
	// Check if the block number is within 100 blocks of the span end
	// TODO: Move the 100 to params?
	return blockNumber <= spanEnd && blockNumber >= (spanEnd-100)
}

func GenerateBorCommittedSpans(latestBorBlock uint64, latestHeimdallSpan *Span) []Span {
	spans := []Span{}
	spanLength := latestHeimdallSpan.EndBlock - latestHeimdallSpan.StartBlock
	prevSpan := latestHeimdallSpan
	for latestBorBlock > latestHeimdallSpan.EndBlock {
		startBlock := prevSpan.EndBlock + 1
		newSpan := Span{
			Id:                prevSpan.Id + 1,
			StartBlock:        startBlock,
			EndBlock:          startBlock + spanLength,
			BorChainId:        latestHeimdallSpan.BorChainId,
			SelectedProducers: latestHeimdallSpan.SelectedProducers,
			ValidatorSet:      latestHeimdallSpan.ValidatorSet,
		}
		spans = append(spans, newSpan)
		prevSpan = &newSpan
	}
	return spans
}

func CalcCurrentBorSpanId(latestBorBlock uint64, latestHeimdallSpan *Span) uint64 {
	// Calculate the current span based on the latest bor block and latest heimdall span
	spanLength := latestHeimdallSpan.EndBlock - latestHeimdallSpan.StartBlock
	spanId := latestHeimdallSpan.Id + ((latestBorBlock - latestHeimdallSpan.StartBlock) / spanLength)
	if (latestBorBlock-latestHeimdallSpan.StartBlock)%spanLength != 0 {
		spanId++
	}
	return spanId
}
