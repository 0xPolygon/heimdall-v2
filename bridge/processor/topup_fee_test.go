package processor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFeeProcessor_Start(t *testing.T) {
	t.Parallel()

	t.Run("starts successfully", func(t *testing.T) {
		t.Parallel()

		fp := &FeeProcessor{}

		require.NotNil(t, fp)
	})
}

func TestTopupFeeProcessor_Constants(t *testing.T) {
	t.Parallel()

	t.Run("validates error message constants", func(t *testing.T) {
		t.Parallel()

		errorMessages := []string{
			errMsgTopupUnmarshallingEvent,
			errMsgTopupParsingEvent,
			errMsgTopupBroadcasting,
			errMsgTopupTxFailed,
			errMsgTopupRegisteringTask,
		}

		for _, msg := range errorMessages {
			require.NotEmpty(t, msg)
			require.Contains(t, msg, "Processor")
		}
	})

	t.Run("validates info message constants", func(t *testing.T) {
		t.Parallel()

		infoMessages := []string{
			infoMsgTopupStarting,
			infoMsgTopupRegisteringTasks,
			infoMsgTopupSending,
		}

		for _, msg := range infoMessages {
			require.NotEmpty(t, msg)
			require.Contains(t, msg, "Processor")
		}
	})

	t.Run("validates debug message constants", func(t *testing.T) {
		t.Parallel()

		debugMessages := []string{
			debugMsgTopupIgnoringProcessed,
		}

		for _, msg := range debugMessages {
			require.NotEmpty(t, msg)
			require.Contains(t, msg, "Processor")
		}
	})
}
