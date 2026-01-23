package processor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewStakingProcessor(t *testing.T) {
	t.Parallel()

	t.Run("creates processor with ABI", func(t *testing.T) {
		t.Parallel()

		sp := NewStakingProcessor(nil)

		require.NotNil(t, sp)
		require.Nil(t, sp.stakingInfoAbi)
	})
}

func TestStakingProcessor_Constants(t *testing.T) {
	t.Parallel()

	t.Run("validates delay duration", func(t *testing.T) {
		t.Parallel()

		delay := 15 * time.Second
		require.Equal(t, defaultDelayDuration, delay)
		require.Greater(t, delay, time.Duration(0))
	})

	t.Run("validates error message constants", func(t *testing.T) {
		t.Parallel()

		errorMessages := []string{
			errMsgUnmarshallingEvent,
			errMsgParsingEvent,
			errMsgValidatingNonce,
			errMsgCreatingMsg,
			errMsgBroadcasting,
			errMsgTxFailed,
			errMsgConvertingAddress,
			errMsgCheckingOldTx,
			errMsgInvalidSignerPubkey,
			errMsgFetchValidatorNonce,
			errMsgNonceNotInOrder,
			errMsgQueryStakeTxs,
			errMsgSearchTxs,
		}

		for _, msg := range errorMessages {
			require.NotEmpty(t, msg)
			require.Contains(t, msg, "StakingProcessor")
		}
	})

	t.Run("validates info message constants", func(t *testing.T) {
		t.Parallel()

		infoMessages := []string{
			infoMsgIgnoringAlreadyProcessed,
			infoMsgIgnoringNonceOutOfOrder,
			infoMsgAccountDoesNotExist,
			infoMsgRecentStakingTxnNotZero,
		}

		for _, msg := range infoMessages {
			require.NotEmpty(t, msg)
			require.Contains(t, msg, "StakingProcessor")
		}
	})

	t.Run("validates other message constants", func(t *testing.T) {
		t.Parallel()

		infoMessages := []string{
			msgNonceOutOfOrder,
			msgAccountDoesNotExist,
		}

		for _, msg := range infoMessages {
			require.NotEmpty(t, msg)
			require.Contains(t, msg, "StakingProcessor")
		}
	})
}
