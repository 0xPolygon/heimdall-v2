package types

import (
	"fmt"
	"sort"
	"strings"

	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// SortValidatorByAddress sorts a slice of validators by address.
// To sort it, we compare the values of the Signer(HeimdallAddress i.e. [20]byte)
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
	return blockNumber <= spanEnd && blockNumber >= (spanEnd-100)
}

func GenerateBorCommittedSpans(latestBorBlock uint64, latestBorUsedSpan *Span) []Span {
	spanLength := latestBorUsedSpan.EndBlock - latestBorUsedSpan.StartBlock
	estimatedSpans := (latestBorBlock - latestBorUsedSpan.EndBlock) / spanLength
	if estimatedSpans > 0 {
		estimatedSpans++ // Add buffer
	}
	spans := make([]Span, 0, estimatedSpans)
	prevSpan := latestBorUsedSpan
	for latestBorBlock > prevSpan.EndBlock {
		startBlock := prevSpan.EndBlock + 1
		newSpan := Span{
			Id:                prevSpan.Id + 1,
			StartBlock:        startBlock,
			EndBlock:          startBlock + spanLength,
			BorChainId:        latestBorUsedSpan.BorChainId,
			SelectedProducers: latestBorUsedSpan.SelectedProducers,
			ValidatorSet:      latestBorUsedSpan.ValidatorSet,
		}
		spans = append(spans, newSpan)
		prevSpan = &newSpan
	}
	return spans
}

// CalcCurrentBorSpanId computes the Bor span ID corresponding to latestBorBlock,
// using latestHeimdallSpan as the reference. It returns an error if inputs are invalid
// (nil span, zero, or negative span length) or if arithmetic overflow is detected.
func CalcCurrentBorSpanId(latestBorBlock uint64, latestHeimdallSpan *Span) (uint64, error) {
	if latestHeimdallSpan == nil {
		return 0, fmt.Errorf("nil Heimdall span provided")
	}
	if latestHeimdallSpan.EndBlock < latestHeimdallSpan.StartBlock {
		return 0, fmt.Errorf(
			"invalid Heimdall span: EndBlock (%d) must be >= StartBlock (%d)",
			latestHeimdallSpan.EndBlock,
			latestHeimdallSpan.StartBlock,
		)
	}

	if latestBorBlock < latestHeimdallSpan.StartBlock {
		return 0, fmt.Errorf(
			"latestBorBlock (%d) must be >= Heimdall span StartBlock (%d)",
			latestBorBlock,
			latestHeimdallSpan.StartBlock,
		)
	}

	if latestBorBlock <= latestHeimdallSpan.EndBlock {
		return latestHeimdallSpan.Id, nil
	}

	spanLength := latestHeimdallSpan.EndBlock - latestHeimdallSpan.StartBlock + 1

	offset := latestBorBlock - latestHeimdallSpan.StartBlock
	quotient := offset / spanLength

	spanId := latestHeimdallSpan.Id + quotient

	if spanId < latestHeimdallSpan.Id {
		return 0, fmt.Errorf(
			"overflow detected computing span ID: reference ID=%d quotient=%d",
			latestHeimdallSpan.Id, quotient,
		)
	}

	return spanId, nil
}

const (
	PlannedDowntimeMinimumTimeInFuture = 150
	PlannedDowntimeMaximumTimeInFuture = 100 * DefaultSpanDuration // ~2 weeks
	PlannedDowntimeMinRange            = 150                       // It will be down minimum for the whole span, this here is just for tx validation
	PlannedDowntimeMaxRange            = 14 * DefaultSpanDuration  // ~48 hours
)

// LogSpan returns a human-readable summary of the span for logging purposes.
// It extracts the key fields without dumping the entire validator set, which causes unreadable logs.
func (s *Span) LogSpan() string {
	if s == nil {
		return "nil"
	}

	selectedProducers := ""
	if len(s.SelectedProducers) > 0 {
		producerIDs := make([]string, 0, len(s.SelectedProducers))
		for _, p := range s.SelectedProducers {
			producerIDs = append(producerIDs, fmt.Sprintf("valID=%d", p.ValId))
		}
		selectedProducers = strings.Join(producerIDs, ", ")
	}

	validatorCount := len(s.ValidatorSet.Validators)

	return fmt.Sprintf(
		"id=%d startBlock=%d endBlock=%d validatorCount=%d selectedProducers=[%s] borChainId=%s",
		s.Id,
		s.StartBlock,
		s.EndBlock,
		validatorCount,
		selectedProducers,
		s.BorChainId,
	)
}
