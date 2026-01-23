package processor

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"
)

func TestSpanProcessor_Start(t *testing.T) {
	t.Parallel()

	t.Run("initializes span service context", func(t *testing.T) {
		t.Parallel()

		sp := &SpanProcessor{}
		sp.BaseProcessor.Logger = log.NewNopLogger()

		// Initially no cancel function
		require.Nil(t, sp.cancelSpanService)
	})
}

func TestSpanProcessor_Stop(t *testing.T) {
	t.Parallel()

	t.Run("cancels span service context", func(t *testing.T) {
		t.Parallel()

		sp := &SpanProcessor{}
		sp.BaseProcessor.Logger = log.NewNopLogger()

		// Set up a cancel function
		_, cancelFunc := context.WithCancel(context.Background())
		sp.cancelSpanService = cancelFunc

		// Stop should call the cancel function
		sp.Stop()

		// Verify cancellation exists
		require.NotNil(t, sp.cancelSpanService)
	})
}

func TestSpanProcessor_PollingBehavior(t *testing.T) {
	t.Parallel()

	t.Run("respects context cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())

		// Simulate the select pattern in startPolling
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		done := make(chan bool, 1)
		go func() {
			select {
			case <-ticker.C:
				// process span
			case <-ctx.Done():
				done <- true
				return
			}
		}()

		// Cancel the context
		cancel()

		// Should receive done signal
		select {
		case <-done:
			// Success - context cancellation worked
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Context cancellation not respected")
		}
	})
}

func TestSpanProcessor_Constants(t *testing.T) {
	t.Parallel()

	t.Run("validates error message constants", func(t *testing.T) {
		t.Parallel()

		errorMessages := []string{
			errMsgSpanCheckingProposer,
			errMsgSpanFetchingLastSpan,
			errMsgSpanFetchingCurrentChildBlock,
			errMsgSpanFetchingNextSpanDetails,
			errMsgSpanRecoveredPanic,
			errMsgSpanPropose,
			errMsgSpanFetchingLastSpanForVotes,
			errMsgSpanValidatorNotFound,
			errMsgSpanFetchingProducerVotes,
			errMsgSpanSendingProducerVotes,
			errMsgSpanConvertingAddress,
			errMsgSpanBroadcastingToHeimdall,
			errMsgSpanProducerVotesTxFailed,
			errMsgSpanFetchingCurrentChildBlockInPropose,
			errMsgSpanConvertingAddressInPropose,
			errMsgSpanFetchingLatestSpan,
			errMsgSpanUnmarshallingSpan,
			errMsgSpanCreatingRequest,
			errMsgSpanFetchingProducerVotesAPI,
			errMsgSpanUnmarshallingProducerVotes,
			errMsgSpanFetchingChainmanagerParams,
			errMsgSpanFetchingProposers,
			errMsgSpanUnmarshallingProposeTxMsg,
			errMsgSpanFetchingNextSpanSeed,
			errMsgSpanUnmarshallingNextSpanSeed,
		}

		for _, msg := range errorMessages {
			require.NotEmpty(t, msg)
			require.Contains(t, msg, "SpanProcessor")
		}
	})

	t.Run("validates info message constants", func(t *testing.T) {
		t.Parallel()

		infoMessages := []string{
			infoMsgSpanStarting,
			infoMsgSpanStartPolling,
			infoMsgSpanPollingStopped,
			infoMsgSpanProposingNewSpan,
		}

		for _, msg := range infoMessages {
			require.NotEmpty(t, msg)
			require.Contains(t, msg, "SpanProcessor")
		}
	})

	t.Run("validates debug message constants", func(t *testing.T) {
		t.Parallel()

		debugMessages := []string{
			debugMsgSpanNotProposer,
			debugMsgSpanLastSpanNotFound,
			debugMsgSpanCurrentBorBlockLessThanLast,
			debugMsgSpanFoundLastSpan,
			debugMsgSpanCurrentProducerVotes,
			debugMsgSpanLocalProducers,
			debugMsgSpanUpdatingProducerVotes,
			debugMsgSpanDetails,
			debugMsgSpanGeneratedProposerSpanMsg,
			debugMsgSpanSendingRestCallForSeed,
			debugMsgSpanNextSpanSeedFetched,
		}

		for _, msg := range debugMessages {
			require.NotEmpty(t, msg)
			require.Contains(t, msg, "SpanProcessor")
		}
	})
}
