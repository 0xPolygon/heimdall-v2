package listener

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockSubscription is a mock implementation of ethereum.Subscription
type mockSubscription struct {
	mock.Mock
	errCh chan error
}

func (m *mockSubscription) Unsubscribe() {
	m.Called()
}

func (m *mockSubscription) Err() <-chan error {
	return m.errCh
}

// mockListener implements the Listener interface for testing
type mockListener struct {
	BaseListener
	startCalled         bool
	processHeaderCalled bool
	lastHeader          *blockHeader
}

func (m *mockListener) Start() error {
	m.startCalled = true
	return nil
}

func (m *mockListener) ProcessHeader(header *blockHeader) {
	m.processHeaderCalled = true
	m.lastHeader = header
}

func (m *mockListener) StartHeaderProcess(ctx context.Context) {
	m.BaseListener.StartHeaderProcess(ctx)
}

func (m *mockListener) StartPolling(ctx context.Context, duration time.Duration, number *big.Int) {
	m.BaseListener.StartPolling(ctx, duration, number)
}

func (m *mockListener) StartSubscription(ctx context.Context, subscription ethereum.Subscription) {
	m.BaseListener.StartSubscription(ctx, subscription)
}

func (m *mockListener) Stop() {
	m.BaseListener.Stop()
}

func (m *mockListener) String() string {
	return m.BaseListener.String()
}

func TestBaseListener_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		listenerName string
	}{
		{
			name:         "rootchain listener",
			listenerName: "rootchain",
		},
		{
			name:         "borchain listener",
			listenerName: "borchain",
		},
		{
			name:         "heimdall listener",
			listenerName: "heimdall",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			listener := &mockListener{}
			listener.BaseListener.name = tt.listenerName
			require.Equal(t, tt.listenerName, listener.String())
		})
	}
}

func TestBaseListener_StartHeaderProcess(t *testing.T) {
	t.Parallel()

	t.Run("processes headers from channel", func(t *testing.T) {
		t.Parallel()

		listener := &mockListener{}
		listener.BaseListener.Logger = log.NewNopLogger()
		listener.BaseListener.HeaderChannel = make(chan *blockHeader, 1)
		listener.BaseListener.impl = listener

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start the header process in a goroutine
		go listener.StartHeaderProcess(ctx)

		// Send a test header
		testHeader := &blockHeader{
			header: &types.Header{
				Number: big.NewInt(100),
			},
			isFinalized: true,
		}
		listener.HeaderChannel <- testHeader

		// Wait a bit for processing
		time.Sleep(50 * time.Millisecond)

		require.True(t, listener.processHeaderCalled)
		require.NotNil(t, listener.lastHeader)
		require.Equal(t, big.NewInt(100), listener.lastHeader.header.Number)
		require.True(t, listener.lastHeader.isFinalized)
	})

	t.Run("stops when context is cancelled", func(t *testing.T) {
		t.Parallel()

		listener := &mockListener{}
		listener.BaseListener.Logger = log.NewNopLogger()
		listener.BaseListener.HeaderChannel = make(chan *blockHeader, 1)
		listener.BaseListener.impl = listener

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan bool)
		go func() {
			listener.StartHeaderProcess(ctx)
			done <- true
		}()

		// Cancel the context
		cancel()

		// Wait for the goroutine to finish
		select {
		case <-done:
			// Success - the goroutine stopped
		case <-time.After(1 * time.Second):
			t.Fatal("StartHeaderProcess did not stop after context cancellation")
		}
	})

	t.Run("handles multiple headers in sequence", func(t *testing.T) {
		t.Parallel()

		listener := &mockListener{}
		listener.BaseListener.Logger = log.NewNopLogger()
		listener.BaseListener.HeaderChannel = make(chan *blockHeader, 10)
		listener.BaseListener.impl = listener

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go listener.StartHeaderProcess(ctx)

		// Send multiple headers
		for i := 1; i <= 3; i++ {
			testHeader := &blockHeader{
				header: &types.Header{
					Number: big.NewInt(int64(i * 100)),
				},
				isFinalized: i%2 == 0,
			}
			listener.HeaderChannel <- testHeader
			time.Sleep(10 * time.Millisecond)
		}

		// the last header should be 300
		require.True(t, listener.processHeaderCalled)
		require.NotNil(t, listener.lastHeader)
		require.Equal(t, big.NewInt(300), listener.lastHeader.header.Number)
	})
}

func TestBaseListener_StartSubscription(t *testing.T) {
	t.Parallel()

	t.Run("handles subscription errors", func(t *testing.T) {
		t.Parallel()

		listener := &mockListener{}
		listener.BaseListener.Logger = log.NewNopLogger()

		mockSub := &mockSubscription{
			errCh: make(chan error, 1),
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan bool)
		go func() {
			listener.StartSubscription(ctx, mockSub)
			done <- true
		}()

		// Send an error
		mockSub.errCh <- errors.New("subscription error")

		select {
		case <-done:
			// Success - subscription handler stopped
		case <-time.After(1 * time.Second):
			t.Fatal("StartSubscription did not stop after error")
		}
	})

	t.Run("stops when context is cancelled", func(t *testing.T) {
		t.Parallel()

		listener := &mockListener{}
		listener.BaseListener.Logger = log.NewNopLogger()

		mockSub := &mockSubscription{
			errCh: make(chan error, 1),
		}

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan bool)
		go func() {
			listener.StartSubscription(ctx, mockSub)
			done <- true
		}()

		// Cancel the context
		cancel()

		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("StartSubscription did not stop after context cancellation")
		}
	})

	t.Run("cancels subscription on error if cancelFunc is set", func(t *testing.T) {
		t.Parallel()

		listener := &mockListener{}
		listener.BaseListener.Logger = log.NewNopLogger()

		cancelled := false
		_, cancelFunc := context.WithCancel(context.Background())
		listener.BaseListener.cancelSubscription = func() {
			cancelled = true
			cancelFunc()
		}

		mockSub := &mockSubscription{
			errCh: make(chan error, 1),
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan bool)
		go func() {
			listener.StartSubscription(ctx, mockSub)
			done <- true
		}()

		// Send an error
		mockSub.errCh <- errors.New("test error")

		select {
		case <-done:
			require.True(t, cancelled, "cancelSubscription was not called")
		case <-time.After(1 * time.Second):
			t.Fatal("StartSubscription did not handle error properly")
		}
	})
}

func TestBaseListener_Stop(t *testing.T) {
	t.Parallel()

	t.Run("calls cancel functions when set", func(t *testing.T) {
		t.Parallel()

		listener := &mockListener{}
		listener.BaseListener.Logger = log.NewNopLogger()

		subscriptionCancelled := false
		headerProcessCancelled := false

		ctx1, cancel1 := context.WithCancel(context.Background())
		ctx2, cancel2 := context.WithCancel(context.Background())

		// Wrap cancel functions to track calls
		listener.BaseListener.cancelSubscription = func() {
			subscriptionCancelled = true
			cancel1()
		}
		listener.BaseListener.cancelHeaderProcess = func() {
			headerProcessCancelled = true
			cancel2()
		}

		listener.Stop()

		require.True(t, subscriptionCancelled)
		require.True(t, headerProcessCancelled)

		// Verify contexts were canceled
		select {
		case <-ctx1.Done():
			// Expected
		default:
			t.Fatal("ctx1 was not cancelled")
		}

		select {
		case <-ctx2.Done():
			// Expected
		default:
			t.Fatal("ctx2 was not cancelled")
		}
	})

	t.Run("handles nil cancel functions gracefully", func(t *testing.T) {
		t.Parallel()

		listener := &mockListener{}
		listener.BaseListener.Logger = log.NewNopLogger()
		listener.BaseListener.cancelSubscription = nil
		listener.BaseListener.cancelHeaderProcess = nil

		// Should not panic
		require.NotPanics(t, func() {
			listener.Stop()
		})
	})

	t.Run("can be called multiple times safely", func(t *testing.T) {
		t.Parallel()

		listener := &mockListener{}
		listener.BaseListener.Logger = log.NewNopLogger()

		callCount := 0
		_, cancel := context.WithCancel(context.Background())
		listener.BaseListener.cancelSubscription = func() {
			callCount++
			cancel()
		}

		// Call Stop multiple times
		listener.Stop()
		listener.Stop()

		// Should be called twice (once per Stop call)
		require.Equal(t, 2, callCount)
	})
}

func TestBlockHeader_Fields(t *testing.T) {
	t.Parallel()

	t.Run("finalized block header", func(t *testing.T) {
		t.Parallel()

		header := &types.Header{
			Number:     big.NewInt(500),
			ParentHash: types.EmptyRootHash,
		}

		bHeader := &blockHeader{
			header:      header,
			isFinalized: true,
		}

		require.Equal(t, big.NewInt(500), bHeader.header.Number)
		require.True(t, bHeader.isFinalized)
	})

	t.Run("non-finalized block header", func(t *testing.T) {
		t.Parallel()

		header := &types.Header{
			Number: big.NewInt(600),
		}

		bHeader := &blockHeader{
			header:      header,
			isFinalized: false,
		}

		require.Equal(t, big.NewInt(600), bHeader.header.Number)
		require.False(t, bHeader.isFinalized)
	})

	t.Run("block header with various fields", func(t *testing.T) {
		t.Parallel()

		parentHash := types.EmptyRootHash
		header := &types.Header{
			Number:     big.NewInt(12345),
			ParentHash: parentHash,
			GasLimit:   8000000,
			GasUsed:    5000000,
		}

		bHeader := &blockHeader{
			header:      header,
			isFinalized: true,
		}

		require.Equal(t, big.NewInt(12345), bHeader.header.Number)
		require.Equal(t, parentHash, bHeader.header.ParentHash)
		require.Equal(t, uint64(8000000), bHeader.header.GasLimit)
		require.Equal(t, uint64(5000000), bHeader.header.GasUsed)
		require.True(t, bHeader.isFinalized)
	})
}
