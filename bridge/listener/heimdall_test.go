package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/metrics"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func TestHeimdallListener_Stop(t *testing.T) {
	t.Parallel()

	t.Run("calls the cancel function and sets it to nil", func(t *testing.T) {
		t.Parallel()

		listener := &HeimdallListener{}
		listener.BaseListener.Logger = log.NewNopLogger()

		headerProcessCalled := false
		listener.BaseListener.cancelHeaderProcess = func() {
			headerProcessCalled = true
		}

		listener.Stop()

		require.True(t, headerProcessCalled)
		require.Nil(t, listener.cancelHeaderProcess)
	})

	t.Run("handles nil cancel functions gracefully", func(t *testing.T) {
		t.Parallel()

		listener := &HeimdallListener{}
		listener.BaseListener.Logger = log.NewNopLogger()
		listener.BaseListener.cancelHeaderProcess = nil

		require.NotPanics(t, func() {
			listener.Stop()
		})

		require.Nil(t, listener.cancelHeaderProcess)
	})
}

func TestHeimdallListener_ProcessBlockEvent(t *testing.T) {
	t.Parallel()

	t.Run("processes checkpoint event successfully", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &HeimdallListener{BaseListener: newTestBaseListener(mockConnector)}
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
		listener := &HeimdallListener{BaseListener: newTestBaseListener(mockConnector)}
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
		listener := &HeimdallListener{BaseListener: newTestBaseListener(mockConnector)}
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
		listener := &HeimdallListener{BaseListener: newTestBaseListener(mockConnector)}
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

func TestHeimdallListener_sendTask(t *testing.T) {
	t.Parallel()

	t.Run("creates correct task signature", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &HeimdallListener{BaseListener: newTestBaseListener(mockConnector)}
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

		_ = listener.sendTask(taskName, blockTaskRetryCount, 0, stringArg(string(eventBytes)), tasks.Arg{Type: "int64", Value: blockHeight})

		mockConnector.AssertExpectations(t)
	})

	t.Run("handles different block heights", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &HeimdallListener{BaseListener: newTestBaseListener(mockConnector)}
		listener.BaseListener.Logger = log.NewNopLogger()

		blockHeights := []int64{0, 1, 1000, 999999, 1000000000}

		for _, height := range blockHeights {
			mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
				return sig.Args[1].Value == height
			})).Return(nil, nil).Once()

			_ = listener.sendTask("test", blockTaskRetryCount, 0, stringArg(string([]byte("{}"))), tasks.Arg{Type: "int64", Value: height})
		}

		mockConnector.AssertExpectations(t)
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
				listener := &HeimdallListener{BaseListener: newTestBaseListener(mockConnector)}
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
		listener := &HeimdallListener{BaseListener: newTestBaseListener(mockConnector)}
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
		listener := &HeimdallListener{BaseListener: newTestBaseListener(mockConnector)}
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
		listener := &HeimdallListener{BaseListener: newTestBaseListener(mockConnector)}
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
		listener := &HeimdallListener{BaseListener: newTestBaseListener(mockConnector)}
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

func TestHeimdallListener_EarliestBlockClamping(t *testing.T) {
	t.Parallel()

	t.Run("clamps fromBlock to earliestBlock when node has pruned", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name             string
			fromBlock        uint64
			earliestBlock    uint64
			expectedFromBlock uint64
		}{
			{
				name:              "fromBlock before earliest, should clamp",
				fromBlock:         100,
				earliestBlock:     5000,
				expectedFromBlock: 5000,
			},
			{
				name:              "fromBlock equal to earliest, no clamp",
				fromBlock:         5000,
				earliestBlock:     5000,
				expectedFromBlock: 5000,
			},
			{
				name:              "fromBlock after earliest, no clamp",
				fromBlock:         10000,
				earliestBlock:     5000,
				expectedFromBlock: 10000,
			},
			{
				name:              "earliestBlock is zero (non-pruned node), no clamp",
				fromBlock:         100,
				earliestBlock:     0,
				expectedFromBlock: 100,
			},
			{
				name:              "earliestBlock is 1 (genesis), fromBlock before",
				fromBlock:         0,
				earliestBlock:     1,
				expectedFromBlock: 1,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fromBlock := tc.fromBlock
				earliestBlock := tc.earliestBlock

				// This mirrors the logic in fetchFromAndToBlock
				if earliestBlock > 0 && fromBlock < earliestBlock {
					fromBlock = earliestBlock
				}

				require.Equal(t, tc.expectedFromBlock, fromBlock)
			})
		}
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

// fakeBlockResultsServer serves CometBFT block_results with one checkpoint
// event per block, and fails every request for failingHeight.
func fakeBlockResultsServer(t *testing.T, failingHeight int64) *rpchttp.HTTP {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params struct {
				Height string `json:"height"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if req.Method != "block_results" || req.Params.Height == strconv.FormatInt(failingHeight, 10) {
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32603,"message":"unavailable"}}`, req.ID)
			return
		}
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"height":%q,"finalize_block_events":[{"type":%q,"attributes":[]}]}}`,
			req.ID, req.Params.Height, checkpointTypes.EventTypeCheckpoint)
	}))
	t.Cleanup(srv.Close)

	client, err := rpchttp.New(srv.URL, "/websocket")
	require.NoError(t, err)
	return client
}

// TestHeimdallListener_processBlockRange guards against skipping a block whose
// checkpoint event couldn't be fetched or queued: processing stops before it,
// so the next poll retries it.
func TestHeimdallListener_processBlockRange(t *testing.T) {
	t.Parallel()

	t.Run("stops before a block whose events can't be fetched", func(t *testing.T) {
		t.Parallel()

		queue := new(mockQueueConnector)
		queue.On("SendTask", mock.Anything).Return(nil, nil).Twice()
		listener := &HeimdallListener{BaseListener: newTestBaseListener(queue)}
		listener.httpClient = fakeBlockResultsServer(t, 12)
		queryFailures := metrics.BridgeListenerRangeRetries.WithLabelValues(heimdallListenerStr, "query_failed")
		before := testutil.ToFloat64(queryFailures)

		require.Equal(t, uint64(12), listener.processBlockRange(context.Background(), 10, 14))
		queue.AssertExpectations(t)
		require.GreaterOrEqual(t, testutil.ToFloat64(queryFailures)-before, 1.0)
	})

	t.Run("stops before a block whose checkpoint can't be queued", func(t *testing.T) {
		t.Parallel()

		queue := new(mockQueueConnector)
		queue.On("SendTask", mock.Anything).Return(nil, errors.New("broker down")).Once()
		listener := &HeimdallListener{BaseListener: newTestBaseListener(queue)}
		listener.httpClient = fakeBlockResultsServer(t, -1)
		notQueued := metrics.BridgeListenerRangeRetries.WithLabelValues(heimdallListenerStr, "not_queued")
		before := testutil.ToFloat64(notQueued)

		require.Equal(t, uint64(10), listener.processBlockRange(context.Background(), 10, 14))
		queue.AssertExpectations(t)
		require.GreaterOrEqual(t, testutil.ToFloat64(notQueued)-before, 1.0)
	})

	t.Run("finishes the whole range", func(t *testing.T) {
		t.Parallel()

		queue := new(mockQueueConnector)
		queue.On("SendTask", mock.Anything).Return(nil, nil).Times(3)
		listener := &HeimdallListener{BaseListener: newTestBaseListener(queue)}
		listener.httpClient = fakeBlockResultsServer(t, -1)

		require.Equal(t, uint64(13), listener.processBlockRange(context.Background(), 10, 12))
		queue.AssertExpectations(t)
	})
}
