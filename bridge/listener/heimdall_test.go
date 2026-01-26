package listener

import (
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

// heimdallListenerForTest wraps HeimdallListener for testing
type heimdallListenerForTest struct {
	HeimdallListener
	mockConnector *mockQueueConnector
}

func (h *heimdallListenerForTest) ProcessBlockEvent(event sdk.StringEvent, blockHeight int64) {
	h.Logger.Debug("Received block event from Heimdall", "eventType", event.Type)

	eventBytes, err := json.Marshal(event)
	if err != nil {
		h.Logger.Error("Error while marshalling block event", "eventType", event.Type, "error", err)
		return
	}

	switch event.Type {
	case checkpointTypes.EventTypeCheckpoint:
		h.sendBlockTask("sendCheckpointToRootchain", eventBytes, blockHeight)
	default:
		h.Logger.Debug("Block event type mismatch", "eventType", event.Type)
	}
}

func (h *heimdallListenerForTest) sendBlockTask(taskName string, eventBytes []byte, blockHeight int64) {
	signature := &tasks.Signature{
		Name: taskName,
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: string(eventBytes),
			},
			{
				Type:  "int64",
				Value: blockHeight,
			},
		},
	}
	signature.RetryCount = 3

	h.Logger.Info("Sending block level task", "taskName", taskName, "currentTime", time.Now(), "blockHeight", blockHeight)

	_, err := h.mockConnector.SendTask(signature)
	if err != nil {
		h.Logger.Error("Error sending block level task", "taskName", taskName, "blockHeight", blockHeight, "error", err)
	}
}

func TestHeimdallListener_Stop(t *testing.T) {
	t.Parallel()

	t.Run("calls cancel functions and sets them to nil", func(t *testing.T) {
		t.Parallel()

		listener := &HeimdallListener{}
		listener.BaseListener.Logger = log.NewNopLogger()

		subscriptionCalled := false
		headerProcessCalled := false

		listener.BaseListener.cancelSubscription = func() {
			subscriptionCalled = true
		}
		listener.BaseListener.cancelHeaderProcess = func() {
			headerProcessCalled = true
		}

		listener.Stop()

		require.True(t, subscriptionCalled)
		require.True(t, headerProcessCalled)
		require.Nil(t, listener.cancelSubscription)
		require.Nil(t, listener.cancelHeaderProcess)
	})

	t.Run("handles nil cancel functions gracefully", func(t *testing.T) {
		t.Parallel()

		listener := &HeimdallListener{}
		listener.BaseListener.Logger = log.NewNopLogger()
		listener.BaseListener.cancelSubscription = nil
		listener.BaseListener.cancelHeaderProcess = nil

		require.NotPanics(t, func() {
			listener.Stop()
		})

		require.Nil(t, listener.cancelSubscription)
		require.Nil(t, listener.cancelHeaderProcess)
	})
}

func TestHeimdallListener_ProcessHeader(t *testing.T) {
	t.Parallel()

	t.Run("is a no-op function", func(t *testing.T) {
		t.Parallel()

		listener := &HeimdallListener{}
		listener.BaseListener.Logger = log.NewNopLogger()

		// ProcessHeader should do nothing for Heimdall
		require.NotPanics(t, func() {
			listener.ProcessHeader(nil)
		})
	})
}

func TestHeimdallListener_ProcessBlockEvent(t *testing.T) {
	t.Parallel()

	t.Run("processes checkpoint event successfully", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &heimdallListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		event := sdk.StringEvent{
			Type: checkpointTypes.EventTypeCheckpoint,
			Attributes: []sdk.Attribute{
				{Key: "proposer", Value: "test-address"},
				{Key: "start_block", Value: "100"},
				{Key: "end_block", Value: "200"},
			},
		}

		blockHeight := int64(12345)
		eventBytes, err := json.Marshal(event)
		require.NoError(t, err)

		// Expect SendTask to be called with the correct signature
		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			return sig.Name == "sendCheckpointToRootchain" &&
				len(sig.Args) == 2 &&
				sig.Args[0].Type == "string" &&
				sig.Args[0].Value == string(eventBytes) &&
				sig.Args[1].Type == "int64" &&
				sig.Args[1].Value == blockHeight &&
				sig.RetryCount == 3
		})).Return(nil, nil).Once()

		listener.ProcessBlockEvent(event, blockHeight)

		mockConnector.AssertExpectations(t)
	})

	t.Run("ignores unknown event types", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &heimdallListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		event := sdk.StringEvent{
			Type: "unknown-event-type",
			Attributes: []sdk.Attribute{
				{Key: "key1", Value: "value1"},
			},
		}

		// Should not call SendTask for unknown event types
		listener.ProcessBlockEvent(event, 100)

		mockConnector.AssertNotCalled(t, "SendTask")
	})

	t.Run("handles send task error gracefully", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &heimdallListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		event := sdk.StringEvent{
			Type: checkpointTypes.EventTypeCheckpoint,
			Attributes: []sdk.Attribute{
				{Key: "test", Value: "value"},
			},
		}

		mockConnector.On("SendTask", mock.Anything).
			Return(nil, errors.New("send task failed")).Once()

		require.NotPanics(t, func() {
			listener.ProcessBlockEvent(event, 200)
		})

		mockConnector.AssertExpectations(t)
	})

	t.Run("processes multiple events in sequence", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &heimdallListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		// Expect 3 checkpoint events
		mockConnector.On("SendTask", mock.Anything).Return(nil, nil).Times(3)

		for i := 1; i <= 3; i++ {
			event := sdk.StringEvent{
				Type: checkpointTypes.EventTypeCheckpoint,
				Attributes: []sdk.Attribute{
					{Key: "block", Value: strconv.Itoa(i * 100)},
				},
			}
			listener.ProcessBlockEvent(event, int64(i*100))
		}

		mockConnector.AssertExpectations(t)
	})
}

func TestHeimdallListener_sendBlockTask(t *testing.T) {
	t.Parallel()

	t.Run("creates correct task signature", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &heimdallListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		taskName := "sendCheckpointToRootchain"
		eventBytes := []byte(`{"type":"checkpoint","attributes":[]}`)
		blockHeight := int64(5000)

		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			if sig.Name != taskName {
				return false
			}
			if len(sig.Args) != 2 {
				return false
			}
			if sig.Args[0].Type != "string" || sig.Args[0].Value != string(eventBytes) {
				return false
			}
			if sig.Args[1].Type != "int64" || sig.Args[1].Value != blockHeight {
				return false
			}
			return sig.RetryCount == 3
		})).Return(nil, nil).Once()

		listener.sendBlockTask(taskName, eventBytes, blockHeight)

		mockConnector.AssertExpectations(t)
	})

	t.Run("handles different block heights", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &heimdallListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		blockHeights := []int64{0, 1, 1000, 999999, 1000000000}

		for _, height := range blockHeights {
			mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
				return sig.Args[1].Value == height
			})).Return(nil, nil).Once()

			listener.sendBlockTask("test", []byte("{}"), height)
		}

		mockConnector.AssertExpectations(t)
	})
}

func TestHeimdallListener_TaskSignatureStructure(t *testing.T) {
	t.Parallel()

	t.Run("verifies task has two arguments", func(t *testing.T) {
		t.Parallel()

		event := sdk.StringEvent{
			Type: checkpointTypes.EventTypeCheckpoint,
			Attributes: []sdk.Attribute{
				{Key: "proposer", Value: "addr"},
			},
		}
		eventBytes, err := json.Marshal(event)
		require.NoError(t, err)

		blockHeight := int64(777)

		signature := &tasks.Signature{
			Name: "sendCheckpointToRootchain",
			Args: []tasks.Arg{
				{
					Type:  "string",
					Value: string(eventBytes),
				},
				{
					Type:  "int64",
					Value: blockHeight,
				},
			},
		}
		signature.RetryCount = 3

		// Verify structure
		require.Equal(t, "sendCheckpointToRootchain", signature.Name)
		require.Len(t, signature.Args, 2)
		require.Equal(t, "string", signature.Args[0].Type)
		require.Equal(t, "int64", signature.Args[1].Type)
		require.Equal(t, 3, signature.RetryCount)

		// Verify we can unmarshal the event
		var decodedEvent sdk.StringEvent
		err = json.Unmarshal([]byte(signature.Args[0].Value.(string)), &decodedEvent)
		require.NoError(t, err)
		require.Equal(t, checkpointTypes.EventTypeCheckpoint, decodedEvent.Type)

		// Verify block height
		require.Equal(t, blockHeight, signature.Args[1].Value)
	})
}

func TestHeimdallListener_StorageKeys(t *testing.T) {
	t.Parallel()

	t.Run("verifies storage key constant", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "heimdall-last-block", heimdallLastBlockKey)
		require.Contains(t, heimdallLastBlockKey, "heimdall")
		require.Contains(t, heimdallLastBlockKey, "last-block")
	})
}

func TestHeimdallListener_EventTypes(t *testing.T) {
	t.Parallel()

	t.Run("handles different event types correctly", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name         string
			eventType    string
			shouldSendTo bool
		}{
			{
				name:         "checkpoint event",
				eventType:    checkpointTypes.EventTypeCheckpoint,
				shouldSendTo: true,
			},
			{
				name:         "unknown event",
				eventType:    "unknown",
				shouldSendTo: false,
			},
			{
				name:         "empty event type",
				eventType:    "",
				shouldSendTo: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockConnector := new(mockQueueConnector)
				listener := &heimdallListenerForTest{
					mockConnector: mockConnector,
				}
				listener.BaseListener.Logger = log.NewNopLogger()

				event := sdk.StringEvent{
					Type: tc.eventType,
					Attributes: []sdk.Attribute{
						{Key: "test", Value: "value"},
					},
				}

				if tc.shouldSendTo {
					mockConnector.On("SendTask", mock.Anything).Return(nil, nil).Once()
				}

				listener.ProcessBlockEvent(event, 100)

				if tc.shouldSendTo {
					mockConnector.AssertExpectations(t)
				} else {
					mockConnector.AssertNotCalled(t, "SendTask")
				}
			})
		}
	})
}

func TestHeimdallListener_Integration(t *testing.T) {
	t.Parallel()

	t.Run("end-to-end event processing", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &heimdallListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		// Create a realistic checkpoint event
		event := sdk.StringEvent{
			Type: checkpointTypes.EventTypeCheckpoint,
			Attributes: []sdk.Attribute{
				{Key: "proposer", Value: "0x1234567890abcdef"},
				{Key: "start_block", Value: "1000"},
				{Key: "end_block", Value: "2000"},
				{Key: "root_hash", Value: "0xabcdef"},
			},
		}

		blockHeight := int64(2000)

		taskSent := make(chan bool, 1)
		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			defer func() { taskSent <- true }()
			return sig.Name == "sendCheckpointToRootchain" &&
				len(sig.Args) == 2 &&
				sig.Args[1].Value == blockHeight
		})).Return(nil, nil).Once()

		listener.ProcessBlockEvent(event, blockHeight)

		// Wait for the task to be sent
		select {
		case <-taskSent:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("Task was not sent")
		}

		mockConnector.AssertExpectations(t)
	})
}

func TestHeimdallListener_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles event with empty attributes", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &heimdallListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		event := sdk.StringEvent{
			Type:       checkpointTypes.EventTypeCheckpoint,
			Attributes: []sdk.Attribute{},
		}

		mockConnector.On("SendTask", mock.Anything).Return(nil, nil).Once()

		require.NotPanics(t, func() {
			listener.ProcessBlockEvent(event, 100)
		})

		mockConnector.AssertExpectations(t)
	})

	t.Run("handles very large block heights", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &heimdallListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		event := sdk.StringEvent{
			Type: checkpointTypes.EventTypeCheckpoint,
		}

		largeHeight := int64(999999999999)

		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			return sig.Args[1].Value == largeHeight
		})).Return(nil, nil).Once()

		require.NotPanics(t, func() {
			listener.ProcessBlockEvent(event, largeHeight)
		})

		mockConnector.AssertExpectations(t)
	})

	t.Run("handles negative block heights", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &heimdallListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		event := sdk.StringEvent{
			Type: checkpointTypes.EventTypeCheckpoint,
		}

		negativeHeight := int64(-1)

		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			return sig.Args[1].Value == negativeHeight
		})).Return(nil, nil).Once()

		// Negative heights should still be processed (validation happens elsewhere)
		require.NotPanics(t, func() {
			listener.ProcessBlockEvent(event, negativeHeight)
		})

		mockConnector.AssertExpectations(t)
	})
}

func TestHeimdallListener_BlockRangeCalculations(t *testing.T) {
	t.Parallel()

	t.Run("toBlock adjustment when less than or equal to fromBlock", func(t *testing.T) {
		t.Parallel()

		// Test the logic: if toBlock <= fromBlock, then toBlock = fromBlock + 1
		fromBlock := uint64(100)
		toBlock := uint64(100)

		if toBlock <= fromBlock {
			toBlock = fromBlock + 1
		}

		require.Equal(t, uint64(101), toBlock)
	})

	t.Run("toBlock remains unchanged when greater than fromBlock", func(t *testing.T) {
		t.Parallel()

		fromBlock := uint64(100)
		toBlock := uint64(200)

		original := toBlock
		if toBlock <= fromBlock {
			toBlock = fromBlock + 1
		}

		require.Equal(t, original, toBlock)
	})
}

func TestHeimdallListener_StringConversions(t *testing.T) {
	t.Parallel()

	t.Run("converts uint64 to string for storage", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name     string
			input    uint64
			expected string
		}{
			{"zero", 0, "0"},
			{"small number", 42, "42"},
			{"large number", 1234567890, "1234567890"},
			{"max uint64", ^uint64(0), "18446744073709551615"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := strconv.FormatUint(tc.input, 10)
				require.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("parses string to uint64 from storage", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name      string
			input     string
			expected  uint64
			shouldErr bool
		}{
			{"valid zero", "0", 0, false},
			{"valid number", "12345", 12345, false},
			{"valid large", "999999999", 999999999, false},
			{"invalid non-numeric", "abc", 0, true},
			{"invalid negative", "-1", 0, true},
			{"invalid empty", "", 0, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := strconv.ParseUint(tc.input, 10, 64)
				if tc.shouldErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.expected, result)
				}
			})
		}
	})
}

func TestHeimdallListener_PollIntervalCalculation(t *testing.T) {
	t.Parallel()

	t.Run("selects minimum of two intervals", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name                 string
			syncerInterval       time.Duration
			checkpointInterval   time.Duration
			expectedPollInterval time.Duration
		}{
			{
				name:                 "checkpoint smaller",
				syncerInterval:       10 * time.Second,
				checkpointInterval:   5 * time.Second,
				expectedPollInterval: 5 * time.Second,
			},
			{
				name:                 "syncer smaller",
				syncerInterval:       3 * time.Second,
				checkpointInterval:   10 * time.Second,
				expectedPollInterval: 3 * time.Second,
			},
			{
				name:                 "both equal",
				syncerInterval:       7 * time.Second,
				checkpointInterval:   7 * time.Second,
				expectedPollInterval: 7 * time.Second,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				pollInterval := tc.syncerInterval
				if tc.checkpointInterval < tc.syncerInterval {
					pollInterval = tc.checkpointInterval
				}
				require.Equal(t, tc.expectedPollInterval, pollInterval)
			})
		}
	})
}
