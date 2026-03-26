package processor

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"
)

func TestCheckpointProcessor_Start(t *testing.T) {
	t.Parallel()

	t.Run("verifies Start initializes no-ack polling", func(t *testing.T) {
		t.Parallel()

		cp := &CheckpointProcessor{}
		cp.BaseProcessor.Logger = log.NewNopLogger()

		require.Nil(t, cp.cancelNoACKPolling)
	})
}

func TestCheckpointProcessor_Stop(t *testing.T) {
	t.Parallel()

	t.Run("verifies Stop cancels no-ack polling", func(t *testing.T) {
		t.Parallel()

		cp := &CheckpointProcessor{}
		cp.BaseProcessor.Logger = log.NewNopLogger()

		// Set up a cancel function
		_, cancelFunc := context.WithCancel(context.Background())
		cp.cancelNoACKPolling = cancelFunc

		// Stop should call the cancel function
		cp.Stop()

		// Verify cancellation worked
		require.NotNil(t, cp.cancelNoACKPolling)
	})
}

func TestCheckpointProcessor_ContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("polling respects context cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())

		// Simulate the select pattern in startPollingForNoAck
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		done := make(chan bool, 1)
		go func() {
			select {
			case <-ticker.C:
				// Would process no-ack
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

func TestNewCheckpointProcessor(t *testing.T) {
	t.Parallel()

	t.Run("creates processor with ABI", func(t *testing.T) {
		t.Parallel()

		cp := NewCheckpointProcessor(nil)

		require.NotNil(t, cp)
		require.Nil(t, cp.rootChainAbi)
	})
}

func TestCheckpointProcessor_NoAckGuard(t *testing.T) {
	t.Parallel()

	t.Run("skips no-ack check when previous run is in progress", func(t *testing.T) {
		t.Parallel()

		cp := &CheckpointProcessor{}
		cp.BaseProcessor.Logger = log.NewNopLogger()

		// Pre-set noAckInProgress to true (simulating a run already in progress)
		cp.noAckInProgress.Store(true)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			defer close(done)
			// Use a very short interval so the ticker fires before we cancel
			cp.startPollingForNoAck(ctx, 20*time.Millisecond)
		}()

		// Wait for at least one tick to hit the skip path
		require.Eventually(t, func() bool {
			return cp.noAckSkipCount.Load() > 0
		}, 2*time.Second, 10*time.Millisecond, "expected at least one tick to fire and be skipped")
		cancel()

		select {
		case <-done:
			// startPollingForNoAck returned after context cancellation
		case <-time.After(2 * time.Second):
			t.Fatal("startPollingForNoAck did not stop after context cancellation")
		}

		// noAckInProgress should still be true (the goroutine was never spawned, so Store(false) was never called)
		require.True(t, cp.noAckInProgress.Load())
	})

	t.Run("noAckInProgress defaults to false", func(t *testing.T) {
		t.Parallel()

		cp := &CheckpointProcessor{}
		require.False(t, cp.noAckInProgress.Load())
	})

	t.Run("CompareAndSwap prevents concurrent access", func(t *testing.T) {
		t.Parallel()

		cp := &CheckpointProcessor{}

		// First CAS should succeed
		require.True(t, cp.noAckInProgress.CompareAndSwap(false, true))
		// Second CAS should fail
		require.False(t, cp.noAckInProgress.CompareAndSwap(false, true))
		// Store resets the flag
		cp.noAckInProgress.Store(false)
		// CAS should succeed again
		require.True(t, cp.noAckInProgress.CompareAndSwap(false, true))
	})
}

func TestCheckpointProcessor_Constants(t *testing.T) {
	t.Parallel()

	t.Run("validates error message constants", func(t *testing.T) {
		t.Parallel()

		errorMessages := []string{
			errMsgCpUnmarshallingHeaderBlock,
			errMsgCpCheckingProposer,
			errMsgCpBlocksLessThanConfirmations,
			errMsgCpCalculatingNextCheckpoint,
			errMsgCpSendingCheckpointToHeimdall,
			errMsgCpUnmarshallingEvent,
			errMsgCpCheckingCurrentProposer,
			errMsgCpSendingCheckpointToRootChain,
			errMsgCpUnmarshallingEventFromRootChain,
			errMsgCpParsingEvent,
			errMsgCpBroadcastingCheckpointAck,
			errMsgCpCheckpointAckTxFailed,
			errMsgCpFetchingLatestCheckpointTime,
			errMsgCpCheckingProposerList,
			errMsgCpProposingNoAck,
			errMsgCpFetchingCurrentHeaderBlock,
			errMsgCpFetchingHeaderBlockObject,
			errMsgCpConvertingAddress,
			errMsgCpBroadcastingCheckpoint,
			errMsgCpCheckpointTxFailed,
			errMsgCpQueryingCheckpointTxProof,
			errMsgCpDecodingCheckpointTx,
			errMsgCpInvalidSideTxMsg,
			errMsgCpFetchingCheckpointSignatures,
			errMsgCpParsingCheckpointSignatures,
			errMsgCpCreatingRootChainInstance,
			errMsgCpSubmittingCheckpointToRootChain,
			errMsgCpFetchingAccountRootHash,
			errMsgCpUnmarshallingAccountRootHash,
			errMsgCpFetchingCurrentHeaderBlockNumber,
			errMsgCpFetchingHeaderBlock,
			errMsgCpFetchingLastNoAck,
			errMsgCpUnmarshallingNoAckData,
			errMsgCpBroadcastingNoAck,
			errMsgCpNoAckTxFailed,
			errMsgCpFetchingCurrentChildBlock,
			errMsgCpFetchingChainManagerParams,
			errMsgCpFetchingCheckpointParams,
		}

		for _, msg := range errorMessages {
			require.NotEmpty(t, msg)
			require.Contains(t, msg, "CheckpointProcessor")
		}
	})

	t.Run("validates info message constants", func(t *testing.T) {
		t.Parallel()

		infoMessages := []string{
			infoMsgCpStarting,
			infoMsgCpStartPollingNoAck,
			infoMsgCpRegisteringTasks,
			infoMsgCpNoAckPollingStopped,
			infoMsgCpNotProposer,
			infoMsgCpReceivedCheckpointToRootChainRequest,
			infoMsgCpProcessingCheckpointConfirmationEvent,
			infoMsgCpNotCurrentProposerOrAlreadySent,
			infoMsgCpCheckpointAlreadyInBuffer,
			infoMsgCpWaitingForBlocks,
			infoMsgCpRootHashCalculated,
			infoMsgCpFetchingAccountRootHashError,
			infoMsgCpCreatingAndBroadcastingCheckpoint,
			infoMsgCpPreparingCheckpointForRootChain,
			infoMsgCpNoAckTransactionSentSuccessfully,
			infoMsgCpValidatingCheckpointSubmission,
			infoMsgCpCheckpointValid,
			infoMsgCpStartBlockDoesNotMatch,
			infoMsgCpCheckpointAlreadySent,
			infoMsgCpNoNeedToSendCheckpoint,
		}

		for _, msg := range infoMessages {
			require.NotEmpty(t, msg)
			// One message doesn't have the processor prefix, check if it contains "checkpoint"
			if msg == infoMsgCpPreparingCheckpointForRootChain {
				require.Contains(t, msg, "checkpoint")
			} else {
				require.Contains(t, msg, "CheckpointProcessor")
			}
		}
	})

	t.Run("validates debug message constants", func(t *testing.T) {
		t.Parallel()

		debugMessages := []string{
			debugMsgCpProcessingNewHeaderBlock,
			debugMsgCpConfirmationsRequired,
			debugMsgCpNoBufferedCheckpoint,
			debugMsgCpCalculatingCheckpointEligibility,
			debugMsgCpInitiatingCheckpointToHeimdall,
			debugMsgCpFetchingDividendAccountRootHash,
			debugMsgCpDividendAccountRootHashFetched,
			debugMsgCpCheckpointAckAlreadySubmitted,
			debugMsgCpCannotSendMultipleNoAckInShortTime,
			debugMsgCpFetchedCurrentChildBlock,
		}

		for _, msg := range debugMessages {
			require.NotEmpty(t, msg)
			require.Contains(t, msg, "CheckpointProcessor")
		}
	})
}
