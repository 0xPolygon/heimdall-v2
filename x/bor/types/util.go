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

const (
	PlannedDowntimeMinimumTimeInFuture = 150
	PlannedDowntimeMaximumTimeInFuture = 100 * DefaultSpanDuration // ~2 weeks
	PlannedDowntimeMinRange            = 150                       // It will be down minimum for the whole span, this here is just for tx validation.
	PlannedDowntimeMaxRange            = 14 * DefaultSpanDuration  // ~48 hours
	RoundRobinDefault                  = 0                         // No preferred replacement, the next producer is chosen via round-robin instead.
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
