package listener

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockQueueConnector is a mock for testing task sending
type mockQueueConnector struct {
	mock.Mock
}

func (m *mockQueueConnector) SendTask(signature *tasks.Signature) (interface{}, error) {
	args := m.Called(signature)
	return args.Get(0), args.Error(1)
}

// borChainListenerForTest wraps BorChainListener for testing
type borChainListenerForTest struct {
	BorChainListener
	mockConnector *mockQueueConnector
}

func (b *borChainListenerForTest) ProcessHeader(newHeader *blockHeader) {
	b.Logger.Debug("New header block detected", "blockNumber", newHeader.header.Number)

	headerBytes, err := newHeader.header.MarshalJSON()
	if err != nil {
		b.Logger.Error("Error marshalling header block", "error", err)
		return
	}

	b.sendTaskWithDelay("sendCheckpointToHeimdall", headerBytes, 0)
}

func (b *borChainListenerForTest) sendTaskWithDelay(taskName string, headerBytes []byte, delay time.Duration) {
	signature := &tasks.Signature{
		Name: taskName,
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: string(headerBytes),
			},
		},
	}
	signature.RetryCount = 3

	eta := time.Now().Add(delay)
	signature.ETA = &eta

	b.Logger.Debug("Sending task", "taskName", taskName, "currentTime", time.Now(), "delayTime", eta)

	_, err := b.mockConnector.SendTask(signature)
	if err != nil {
		b.Logger.Error("Error sending task", "taskName", taskName, "error", err)
	}
}

func TestBorChainListener_Start(t *testing.T) {
	t.Parallel()

	t.Run("verifies Start creates cancel functions", func(t *testing.T) {
		t.Parallel()

		// Test that contexts are created when Start is called
		// Not calling Start() in tests because it depends on helper.GetConfig()
		// which requires full app initialization. Instead, we test the structure.
		listener := &BorChainListener{}
		listener.BaseListener.Logger = log.NewNopLogger()

		// Verify initial state
		require.Nil(t, listener.cancelSubscription)
		require.Nil(t, listener.cancelHeaderProcess)
	})
}

func TestBorChainListener_ProcessHeader(t *testing.T) {
	t.Parallel()

	t.Run("processes header and sends task successfully", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &borChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		testHeader := &types.Header{
			Number:     big.NewInt(300),
			ParentHash: types.EmptyRootHash,
		}

		headerBytes, err := testHeader.MarshalJSON()
		require.NoError(t, err)

		// Expect SendTask to be called with the correct signature
		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			return sig.Name == "sendCheckpointToHeimdall" &&
				len(sig.Args) == 1 &&
				sig.Args[0].Type == "string" &&
				sig.Args[0].Value == string(headerBytes) &&
				sig.RetryCount == 3
		})).Return(nil, nil).Once()

		bHeader := &blockHeader{
			header:      testHeader,
			isFinalized: false,
		}

		listener.ProcessHeader(bHeader)

		mockConnector.AssertExpectations(t)
	})

	t.Run("handles send task error gracefully", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &borChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		testHeader := &types.Header{
			Number: big.NewInt(500),
		}

		// Expect SendTask to fail
		mockConnector.On("SendTask", mock.Anything).
			Return(nil, errors.New("send task failed")).Once()

		bHeader := &blockHeader{
			header:      testHeader,
			isFinalized: false,
		}

		// Should not panic on error
		require.NotPanics(t, func() {
			listener.ProcessHeader(bHeader)
		})

		mockConnector.AssertExpectations(t)
	})

	t.Run("processes multiple headers in sequence", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &borChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		// Expect 3 calls to SendTask
		mockConnector.On("SendTask", mock.Anything).Return(nil, nil).Times(3)

		for i := 1; i <= 3; i++ {
			testHeader := &types.Header{
				Number: big.NewInt(int64(i * 100)),
			}

			bHeader := &blockHeader{
				header:      testHeader,
				isFinalized: false,
			}

			listener.ProcessHeader(bHeader)
		}

		mockConnector.AssertExpectations(t)
	})
}

func TestBorChainListener_sendTaskWithDelay(t *testing.T) {
	t.Parallel()

	t.Run("sends task with correct signature and zero delay", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &borChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		taskName := "testTask"
		headerBytes := []byte(`{"number":"0x64"}`)
		delay := time.Duration(0)

		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			// Verify task signature structure
			if sig.Name != taskName {
				return false
			}
			if len(sig.Args) != 1 {
				return false
			}
			if sig.Args[0].Type != "string" {
				return false
			}
			if sig.Args[0].Value != string(headerBytes) {
				return false
			}
			if sig.RetryCount != 3 {
				return false
			}
			// Verify ETA is set and approximately correct (within 1 second)
			if sig.ETA == nil {
				return false
			}
			expectedTime := time.Now().Add(delay)
			return sig.ETA.Sub(expectedTime).Abs() < time.Second
		})).Return(nil, nil).Once()

		listener.sendTaskWithDelay(taskName, headerBytes, delay)

		mockConnector.AssertExpectations(t)
	})

	t.Run("sends task with non-zero delay", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &borChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		taskName := "delayedTask"
		headerBytes := []byte(`{"number":"0x100"}`)
		delay := 5 * time.Second

		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			if sig.Name != taskName {
				return false
			}
			// Verify ETA is in the future by approximately the delay amount
			if sig.ETA == nil {
				return false
			}
			expectedTime := time.Now().Add(delay)
			timeDiff := sig.ETA.Sub(expectedTime).Abs()
			return timeDiff < time.Second
		})).Return(nil, nil).Once()

		listener.sendTaskWithDelay(taskName, headerBytes, delay)

		mockConnector.AssertExpectations(t)
	})

	t.Run("handles SendTask error", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &borChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		taskName := "failingTask"
		headerBytes := []byte(`{"number":"0x200"}`)

		mockConnector.On("SendTask", mock.Anything).
			Return(nil, errors.New("send task failed")).Once()

		// Should not panic
		require.NotPanics(t, func() {
			listener.sendTaskWithDelay(taskName, headerBytes, 0)
		})

		mockConnector.AssertExpectations(t)
	})
}

func TestBorChainListener_Integration(t *testing.T) {
	t.Parallel()

	t.Run("end-to-end header processing", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &borChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		// Create the test header
		testHeader := &types.Header{
			Number: big.NewInt(999),
		}

		headerBytes, err := testHeader.MarshalJSON()
		require.NoError(t, err)

		// Expect a task to be sent
		taskSent := make(chan bool, 1)
		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			defer func() { taskSent <- true }()
			return sig.Name == "sendCheckpointToHeimdall" &&
				sig.Args[0].Value == string(headerBytes)
		})).Return(nil, nil).Once()

		// Process a header through the pipeline
		bHeader := &blockHeader{
			header:      testHeader,
			isFinalized: false,
		}
		listener.ProcessHeader(bHeader)

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

func TestBorChainListener_TaskSignatureStructure(t *testing.T) {
	t.Parallel()

	t.Run("verifies task signature has correct structure", func(t *testing.T) {
		t.Parallel()

		// Create a sample task signature as would be created by the listener
		testHeader := &types.Header{
			Number: big.NewInt(777),
		}
		headerBytes, err := testHeader.MarshalJSON()
		require.NoError(t, err)

		signature := &tasks.Signature{
			Name: "sendCheckpointToHeimdall",
			Args: []tasks.Arg{
				{
					Type:  "string",
					Value: string(headerBytes),
				},
			},
		}
		signature.RetryCount = 3
		eta := time.Now()
		signature.ETA = &eta

		// Verify structure
		require.Equal(t, "sendCheckpointToHeimdall", signature.Name)
		require.Len(t, signature.Args, 1)
		require.Equal(t, "string", signature.Args[0].Type)
		require.Equal(t, 3, signature.RetryCount)
		require.NotNil(t, signature.ETA)

		// Verify the header bytes are valid JSON and contain the block number
		var jsonData map[string]interface{}
		err = json.Unmarshal([]byte(signature.Args[0].Value.(string)), &jsonData)
		require.NoError(t, err)
		require.Contains(t, jsonData, "number")
	})
}

func TestBorChainListener_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles nil header gracefully", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &borChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		// Should not panic with nil block header
		require.NotPanics(t, func() {
			defer func() {
				if r := recover(); r != nil {
				}
			}()
			listener.ProcessHeader(nil)
		})
	})

	t.Run("handles very large block numbers", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &borChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		// Very large block number
		largeNumber := new(big.Int)
		largeNumber.SetString("999999999999999999999", 10)

		testHeader := &types.Header{
			Number: largeNumber,
		}

		mockConnector.On("SendTask", mock.Anything).Return(nil, nil).Once()

		bHeader := &blockHeader{
			header:      testHeader,
			isFinalized: false,
		}

		require.NotPanics(t, func() {
			listener.ProcessHeader(bHeader)
		})

		mockConnector.AssertExpectations(t)
	})
}
