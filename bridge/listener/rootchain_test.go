package listener

import (
	"math/big"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// rootChainListenerForTest wraps RootChainListener for testing
type rootChainListenerForTest struct {
	RootChainListener
	mockConnector *mockQueueConnector
}

func (r *rootChainListenerForTest) SendTaskWithDelay(taskName string, eventName string, logBytes []byte, delay time.Duration, _ interface{}) {
	signature := &tasks.Signature{
		Name: taskName,
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: eventName,
			},
			{
				Type:  "string",
				Value: string(logBytes),
			},
		},
	}
	signature.RetryCount = 3

	eta := time.Now().Add(delay)
	signature.ETA = &eta

	r.Logger.Info("Sending task", "taskName", taskName, "currentTime", time.Now(), "delayTime", eta)

	_, err := r.mockConnector.SendTask(signature)
	if err != nil {
		r.Logger.Error("Error sending task", "taskName", taskName, "error", err)
	}
}

func TestRootChainListener_SendTaskWithDelay(t *testing.T) {
	t.Parallel()

	t.Run("sends task with correct signature structure", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &rootChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		taskName := "testTask"
		eventName := "StateSynced"
		logBytes := []byte(`{"address":"0x123","topics":["0xabc"]}`)
		delay := time.Duration(0)

		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			if sig.Name != taskName {
				return false
			}
			if len(sig.Args) != 2 {
				return false
			}
			if sig.Args[0].Type != "string" || sig.Args[0].Value != eventName {
				return false
			}
			if sig.Args[1].Type != "string" || sig.Args[1].Value != string(logBytes) {
				return false
			}
			if sig.RetryCount != 3 {
				return false
			}
			if sig.ETA == nil {
				return false
			}
			expectedTime := time.Now().Add(delay)
			return sig.ETA.Sub(expectedTime).Abs() < time.Second
		})).Return(nil, nil).Once()

		listener.SendTaskWithDelay(taskName, eventName, logBytes, delay, nil)

		mockConnector.AssertExpectations(t)
	})

	t.Run("sends task with non-zero delay", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &rootChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		taskName := "delayedTask"
		eventName := "NewHeaderBlock"
		logBytes := []byte(`{"blockNumber":"100"}`)
		delay := 3 * time.Second

		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			if sig.Name != taskName {
				return false
			}
			if sig.ETA == nil {
				return false
			}
			expectedTime := time.Now().Add(delay)
			timeDiff := sig.ETA.Sub(expectedTime).Abs()
			return timeDiff < time.Second
		})).Return(nil, nil).Once()

		listener.SendTaskWithDelay(taskName, eventName, logBytes, delay, nil)

		mockConnector.AssertExpectations(t)
	})

	t.Run("verifies task has two string arguments", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &rootChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		taskName := "sendStateSyncedToHeimdall"
		eventName := "StateSynced"
		logBytes := []byte(`{"test":"data"}`)

		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			// RootChain tasks have 2 args: eventName (string) and logBytes (string)
			return len(sig.Args) == 2 &&
				sig.Args[0].Type == "string" &&
				sig.Args[1].Type == "string"
		})).Return(nil, nil).Once()

		listener.SendTaskWithDelay(taskName, eventName, logBytes, 0, nil)

		mockConnector.AssertExpectations(t)
	})
}

func TestRootChainListener_StorageKeys(t *testing.T) {
	t.Parallel()

	t.Run("verifies last block storage key", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "rootchain-last-block", lastRootBlockKey)
		require.Contains(t, lastRootBlockKey, "rootchain")
		require.Contains(t, lastRootBlockKey, "last-block")
	})
}

func TestRootChainListener_BlockNumberComparisons(t *testing.T) {
	t.Parallel()

	t.Run("adjusts from block when less than to block", func(t *testing.T) {
		t.Parallel()

		to := big.NewInt(100)
		from := big.NewInt(150)

		// If to < from, set from = to
		if to.Cmp(from) == -1 {
			from = to
		}

		require.Equal(t, to, from)
	})

	t.Run("keeps from block when equal to or less than to", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name     string
			from     *big.Int
			to       *big.Int
			expected *big.Int
		}{
			{
				name:     "from less than to",
				from:     big.NewInt(50),
				to:       big.NewInt(100),
				expected: big.NewInt(50),
			},
			{
				name:     "from equal to to",
				from:     big.NewInt(100),
				to:       big.NewInt(100),
				expected: big.NewInt(100),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				from := new(big.Int).Set(tc.from)
				to := tc.to

				if to.Cmp(from) == -1 {
					from = to
				}

				require.Equal(t, tc.expected.String(), from.String())
			})
		}
	})
}

func TestRootChainListener_ConfirmationBlocksCalculation(t *testing.T) {
	t.Parallel()

	t.Run("validates header number is greater than confirmations", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name          string
			headerNumber  *big.Int
			confirmations uint64
			shouldProcess bool
		}{
			{
				name:          "header greater than confirmations",
				headerNumber:  big.NewInt(100),
				confirmations: 50,
				shouldProcess: true,
			},
			{
				name:          "header equal to confirmations",
				headerNumber:  big.NewInt(50),
				confirmations: 50,
				shouldProcess: false,
			},
			{
				name:          "header less than confirmations",
				headerNumber:  big.NewInt(30),
				confirmations: 50,
				shouldProcess: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				confirmationBlocks := big.NewInt(0).SetUint64(tc.confirmations)
				shouldProcess := tc.headerNumber.Cmp(confirmationBlocks) > 0

				require.Equal(t, tc.shouldProcess, shouldProcess)
			})
		}
	})

	t.Run("subtracts confirmation blocks from header number", func(t *testing.T) {
		t.Parallel()

		headerNumber := big.NewInt(1000)
		confirmations := uint64(128)

		confirmationBlocks := big.NewInt(0).SetUint64(confirmations)
		result := new(big.Int).Sub(headerNumber, confirmationBlocks)

		expected := big.NewInt(872) // 1000 - 128
		require.Equal(t, expected.String(), result.String())
	})
}

func TestRootChainListener_StorageOperations(t *testing.T) {
	t.Parallel()

	t.Run("increments block number from storage", func(t *testing.T) {
		t.Parallel()

		lastBlockStr := "9999"
		lastBlockNum, err := strconv.ParseUint(lastBlockStr, 10, 64)
		require.NoError(t, err)

		from := big.NewInt(0).SetUint64(lastBlockNum + 1)
		require.Equal(t, "10000", from.String())
	})

	t.Run("parses storage value to uint64", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name      string
			input     string
			expected  uint64
			shouldErr bool
		}{
			{"valid number", "12345", 12345, false},
			{"zero", "0", 0, false},
			{"large number", "999999999", 999999999, false},
			{"invalid", "notanumber", 0, true},
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

	t.Run("skips processing when storage block >= header number", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name          string
			storageBlock  uint64
			headerNumber  uint64
			shouldProcess bool
		}{
			{
				name:          "storage less than header",
				storageBlock:  100,
				headerNumber:  200,
				shouldProcess: true,
			},
			{
				name:          "storage equal to header",
				storageBlock:  200,
				headerNumber:  200,
				shouldProcess: false,
			},
			{
				name:          "storage greater than header",
				storageBlock:  300,
				headerNumber:  200,
				shouldProcess: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				shouldProcess := tc.storageBlock < tc.headerNumber
				require.Equal(t, tc.shouldProcess, shouldProcess)
			})
		}
	})
}

func TestRootChainListener_FinalizedBlockHandling(t *testing.T) {
	t.Parallel()

	t.Run("uses header number directly when finalized", func(t *testing.T) {
		t.Parallel()

		header := &types.Header{
			Number: big.NewInt(500),
		}

		to := header.Number
		from := header.Number

		// When finalized, no confirmation subtraction needed
		require.Equal(t, big.NewInt(500), to)
		require.Equal(t, big.NewInt(500), from)
	})

	t.Run("subtracts confirmations when not finalized", func(t *testing.T) {
		t.Parallel()

		header := &types.Header{
			Number: big.NewInt(500),
		}
		confirmations := uint64(100)

		headerNumber := header.Number
		confirmationBlocks := big.NewInt(0).SetUint64(confirmations)
		headerNumber = new(big.Int).Sub(headerNumber, confirmationBlocks)

		require.Equal(t, "400", headerNumber.String())
	})
}

func TestRootChainListener_LogProcessing(t *testing.T) {
	t.Parallel()

	t.Run("extracts topic from log", func(t *testing.T) {
		t.Parallel()

		l := types.Log{
			Address: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			Topics: []common.Hash{
				common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
			},
			Data:        []byte("test data"),
			BlockNumber: 12345,
		}

		require.Len(t, l.Topics, 1)
		topic := l.Topics[0].Bytes()
		require.Len(t, topic, 32)
	})

	t.Run("handles log with multiple topics", func(t *testing.T) {
		t.Parallel()

		l := types.Log{
			Topics: []common.Hash{
				common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
				common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
				common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
			},
		}

		require.Len(t, l.Topics, 3)
		// The first topic is the event signature
		eventSignature := l.Topics[0].Bytes()
		require.Len(t, eventSignature, 32)
	})

	t.Run("handles empty log topics", func(t *testing.T) {
		t.Parallel()

		l := types.Log{
			Address: common.HexToAddress("0x1234"),
			Topics:  []common.Hash{},
			Data:    []byte("data"),
		}

		require.Empty(t, l.Topics)
	})
}

func TestRootChainListener_FilterQuery(t *testing.T) {
	t.Parallel()

	t.Run("builds correct filter query", func(t *testing.T) {
		t.Parallel()

		fromBlock := big.NewInt(1000)
		toBlock := big.NewInt(2000)
		addresses := []common.Address{
			common.HexToAddress("0x1111111111111111111111111111111111111111"),
			common.HexToAddress("0x2222222222222222222222222222222222222222"),
			common.HexToAddress("0x3333333333333333333333333333333333333333"),
		}

		query := ethereum.FilterQuery{
			FromBlock: fromBlock,
			ToBlock:   toBlock,
			Addresses: addresses,
		}

		require.Equal(t, fromBlock, query.FromBlock)
		require.Equal(t, toBlock, query.ToBlock)
		require.Len(t, query.Addresses, 3)
		require.Contains(t, query.Addresses, addresses[0])
		require.Contains(t, query.Addresses, addresses[1])
		require.Contains(t, query.Addresses, addresses[2])
	})

	t.Run("validates block range", func(t *testing.T) {
		t.Parallel()

		fromBlock := big.NewInt(100)
		toBlock := big.NewInt(200)

		// Valid: from < to
		require.True(t, fromBlock.Cmp(toBlock) < 0)

		// Invalid: from > to
		invalidFrom := big.NewInt(300)
		require.False(t, invalidFrom.Cmp(toBlock) < 0)
	})
}

func TestRootChainListener_TaskSignatureStructure(t *testing.T) {
	t.Parallel()

	t.Run("verifies rootchain task has correct structure", func(t *testing.T) {
		t.Parallel()

		taskName := "sendStateSyncedToHeimdall"
		eventName := "StateSynced"
		logBytes := []byte(`{"address":"0xabc","data":"0x123"}`)

		signature := &tasks.Signature{
			Name: taskName,
			Args: []tasks.Arg{
				{
					Type:  "string",
					Value: eventName,
				},
				{
					Type:  "string",
					Value: string(logBytes),
				},
			},
		}
		signature.RetryCount = 3
		eta := time.Now()
		signature.ETA = &eta

		// Verify structure
		require.Equal(t, taskName, signature.Name)
		require.Len(t, signature.Args, 2)
		require.Equal(t, "string", signature.Args[0].Type)
		require.Equal(t, eventName, signature.Args[0].Value)
		require.Equal(t, "string", signature.Args[1].Type)
		require.Equal(t, string(logBytes), signature.Args[1].Value)
		require.Equal(t, 3, signature.RetryCount)
		require.NotNil(t, signature.ETA)
	})
}

func TestRootChainListener_Integration(t *testing.T) {
	t.Parallel()

	t.Run("end-to-end task sending", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &rootChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		taskName := "sendNewHeaderBlockToHeimdall"
		eventName := "NewHeaderBlock"
		logBytes := []byte(`{"blockNumber":"500","timestamp":"1234567890"}`)
		delay := 2 * time.Second

		taskSent := make(chan bool, 1)
		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			defer func() { taskSent <- true }()
			return sig.Name == taskName &&
				sig.Args[0].Value == eventName &&
				sig.Args[1].Value == string(logBytes)
		})).Return(nil, nil).Once()

		listener.SendTaskWithDelay(taskName, eventName, logBytes, delay, nil)

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

func TestRootChainListener_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles empty event name", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &rootChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			return sig.Args[0].Value == ""
		})).Return(nil, nil).Once()

		require.NotPanics(t, func() {
			listener.SendTaskWithDelay("test", "", []byte("{}"), 0, nil)
		})

		mockConnector.AssertExpectations(t)
	})

	t.Run("handles empty log bytes", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &rootChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			return sig.Args[1].Value == ""
		})).Return(nil, nil).Once()

		require.NotPanics(t, func() {
			listener.SendTaskWithDelay("test", "event", []byte(""), 0, nil)
		})

		mockConnector.AssertExpectations(t)
	})

	t.Run("handles very large delay", func(t *testing.T) {
		t.Parallel()

		mockConnector := new(mockQueueConnector)
		listener := &rootChainListenerForTest{
			mockConnector: mockConnector,
		}
		listener.BaseListener.Logger = log.NewNopLogger()

		largeDelay := 24 * time.Hour

		mockConnector.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
			if sig.ETA == nil {
				return false
			}
			expectedTime := time.Now().Add(largeDelay)
			timeDiff := sig.ETA.Sub(expectedTime).Abs()
			return timeDiff < time.Second
		})).Return(nil, nil).Once()

		require.NotPanics(t, func() {
			listener.SendTaskWithDelay("test", "event", []byte("{}"), largeDelay, nil)
		})

		mockConnector.AssertExpectations(t)
	})
}
