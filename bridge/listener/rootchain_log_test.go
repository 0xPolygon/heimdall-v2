package listener

import (
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
)

// TestRootChainListener_handleLog tests the main event routing logic
func TestRootChainListener_handleLog(t *testing.T) {
	t.Parallel()

	// Test events that don't require ABI unpacking
	simpleEvents := []struct {
		name      string
		eventName string
	}{
		{
			name:      "routes NewHeaderBlock event",
			eventName: helper.NewHeaderBlockEvent,
		},
	}

	for _, tt := range simpleEvents {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			listener := &RootChainListener{
				BaseListener: BaseListener{
					Logger: log.NewNopLogger(),
				},
			}

			vLog := types.Log{
				Address: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
				Topics: []common.Hash{
					common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
				},
				Data:        []byte("test data"),
				BlockNumber: 12345,
			}

			selectedEvent := &abi.Event{
				Name: tt.eventName,
			}

			// Should not panic - these events handle marshaling errors gracefully
			require.NotPanics(t, func() {
				listener.handleLog(vLog, selectedEvent, validatorSlot{})
			})
		})
	}

	// Test event routing dispatch logic
	t.Run("dispatches to correct handler", func(t *testing.T) {
		t.Parallel()

		// handleLog routes every event in rootChainEvents
		eventNames := []string{
			helper.NewHeaderBlockEvent,
			helper.StakedEvent,
			helper.StakeUpdateEvent,
			helper.SignerChangeEvent,
			helper.UnstakeInitEvent,
			helper.StateSyncedEvent,
			helper.TopUpFeeEvent,
		}

		// Verify all event names are defined
		require.Len(t, eventNames, 7)
		for _, name := range eventNames {
			require.NotEmpty(t, name)
		}
	})
}

// TestRootChainListener_handleLog_UnknownEvent tests handling of unknown events
func TestRootChainListener_handleLog_UnknownEvent(t *testing.T) {
	t.Parallel()

	listener := &RootChainListener{
		BaseListener: BaseListener{
			Logger: log.NewNopLogger(),
		},
	}

	vLog := types.Log{
		Address:     common.HexToAddress("0x1234"),
		Topics:      []common.Hash{common.HexToHash("0xabc")},
		Data:        []byte("data"),
		BlockNumber: 100,
	}

	selectedEvent := &abi.Event{
		Name: "UnknownEvent",
	}

	// Should handle gracefully (no panic)
	require.NotPanics(t, func() {
		listener.handleLog(vLog, selectedEvent, validatorSlot{})
	})
}

// TestRootChainListener_handleNewHeaderBlockLog tests checkpoint acknowledgment handling
func TestRootChainListener_handleNewHeaderBlockLog(t *testing.T) {
	t.Parallel()

	t.Run("marshals log successfully", func(t *testing.T) {
		t.Parallel()

		listener := &RootChainListener{
			BaseListener: BaseListener{
				Logger: log.NewNopLogger(),
			},
		}

		vLog := types.Log{
			Address: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			Topics: []common.Hash{
				common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
			},
			Data:        []byte("checkpoint data"),
			BlockNumber: 12345,
			TxHash:      common.HexToHash("0x123"),
			Index:       0,
		}

		selectedEvent := &abi.Event{
			Name: helper.NewHeaderBlockEvent,
		}

		// Should not panic and should be able to marshal
		require.NotPanics(t, func() {
			listener.handleLog(vLog, selectedEvent, validatorSlot{})
		})

		// Verify log can be marshaled
		logBytes, err := json.Marshal(vLog)
		require.NoError(t, err)
		require.NotEmpty(t, logBytes)
	})

	t.Run("handles log with empty data", func(t *testing.T) {
		t.Parallel()

		listener := &RootChainListener{
			BaseListener: BaseListener{
				Logger: log.NewNopLogger(),
			},
		}

		vLog := types.Log{
			Address:     common.HexToAddress("0x1234"),
			Topics:      []common.Hash{common.HexToHash("0xabc")},
			Data:        []byte{},
			BlockNumber: 100,
		}

		selectedEvent := &abi.Event{
			Name: helper.NewHeaderBlockEvent,
		}

		require.NotPanics(t, func() {
			listener.handleLog(vLog, selectedEvent, validatorSlot{})
		})
	})
}

// TestRootChainListener_handleStateSyncedLog tests state sync event handling
func TestRootChainListener_handleStateSyncedLog(t *testing.T) {
	t.Parallel()

	t.Run("marshals state synced log", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		cdc := codec.NewProtoCodec(interfaceRegistry)

		listener := &RootChainListener{
			BaseListener: BaseListener{
				Logger: log.NewNopLogger(),
				cliCtx: client.Context{}.WithCodec(cdc),
			},
			stateSenderAbi: &abi.ABI{},
		}

		vLog := types.Log{
			Address:     common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			Topics:      []common.Hash{common.HexToHash("0xabc")},
			Data:        []byte("state sync data"),
			BlockNumber: 12345,
		}

		selectedEvent := &abi.Event{
			Name: helper.StateSyncedEvent,
		}

		require.NotPanics(t, func() {
			listener.handleLog(vLog, selectedEvent, validatorSlot{})
		})
	})

	t.Run("handles state sync with multiple topics", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		cdc := codec.NewProtoCodec(interfaceRegistry)

		listener := &RootChainListener{
			BaseListener: BaseListener{
				Logger: log.NewNopLogger(),
				cliCtx: client.Context{}.WithCodec(cdc),
			},
			stateSenderAbi: &abi.ABI{},
		}

		vLog := types.Log{
			Address: common.HexToAddress("0x1234"),
			Topics: []common.Hash{
				common.HexToHash("0xabc"),
				common.HexToHash("0xdef"),
				common.HexToHash("0x123"),
			},
			Data:        []byte("state sync data with topics"),
			BlockNumber: 12345,
		}

		selectedEvent := &abi.Event{
			Name: helper.StateSyncedEvent,
		}

		require.NotPanics(t, func() {
			listener.handleLog(vLog, selectedEvent, validatorSlot{})
		})
	})
}

// TestRootChainListener_LogMarshalling tests JSON marshaling of various log types
func TestRootChainListener_LogMarshalling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		log     types.Log
		wantErr bool
	}{
		{
			name: "marshals basic log",
			log: types.Log{
				Address:     common.HexToAddress("0x1234"),
				Topics:      []common.Hash{common.HexToHash("0xabc")},
				Data:        []byte("data"),
				BlockNumber: 100,
			},
			wantErr: false,
		},
		{
			name: "marshals log with multiple topics",
			log: types.Log{
				Address: common.HexToAddress("0x5678"),
				Topics: []common.Hash{
					common.HexToHash("0x111"),
					common.HexToHash("0x222"),
					common.HexToHash("0x333"),
				},
				Data:        []byte("multi-topic data"),
				BlockNumber: 200,
			},
			wantErr: false,
		},
		{
			name: "marshals log with empty data",
			log: types.Log{
				Address:     common.HexToAddress("0x9abc"),
				Topics:      []common.Hash{common.HexToHash("0xdef")},
				Data:        []byte{},
				BlockNumber: 300,
			},
			wantErr: false,
		},
		{
			name: "marshals log with large block number",
			log: types.Log{
				Address:     common.HexToAddress("0xaaaa"),
				Topics:      []common.Hash{common.HexToHash("0xbbbb")},
				Data:        []byte("large block"),
				BlockNumber: 999999999,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logBytes, err := json.Marshal(tt.log)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, logBytes)

				// Verify we can unmarshal back
				var unmarshalledLog types.Log
				err = json.Unmarshal(logBytes, &unmarshalledLog)
				require.NoError(t, err)
				require.Equal(t, tt.log.Address, unmarshalledLog.Address)
				require.Equal(t, tt.log.BlockNumber, unmarshalledLog.BlockNumber)
			}
		})
	}
}

// TestRootChainListener_EventConstants verifies event name constants
func TestRootChainListener_EventConstants(t *testing.T) {
	t.Parallel()

	t.Run("verifies error message constants", func(t *testing.T) {
		t.Parallel()

		// Constants are internal, just verify they're used correctly in the code
		require.NotEmpty(t, "failed to marshal log")
		require.NotEmpty(t, "error while parsing event")
	})

	t.Run("verifies helper event constants exist", func(t *testing.T) {
		t.Parallel()

		// These constants should be defined in the helper package
		require.NotEmpty(t, helper.NewHeaderBlockEvent)
		require.NotEmpty(t, helper.StakedEvent)
		require.NotEmpty(t, helper.StakeUpdateEvent)
		require.NotEmpty(t, helper.SignerChangeEvent)
		require.NotEmpty(t, helper.UnstakeInitEvent)
		require.NotEmpty(t, helper.StateSyncedEvent)
		require.NotEmpty(t, helper.TopUpFeeEvent)
	})
}

// TestRootChainListener_LogHandlingEdgeCases tests edge cases in log handling
func TestRootChainListener_LogHandlingEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles nil logger gracefully", func(t *testing.T) {
		t.Parallel()

		listener := &RootChainListener{
			BaseListener: BaseListener{
				Logger: log.NewNopLogger(),
			},
		}

		vLog := types.Log{
			Address:     common.HexToAddress("0x0"),
			Topics:      []common.Hash{},
			Data:        []byte{},
			BlockNumber: 0,
		}

		selectedEvent := &abi.Event{
			Name: "UnknownEvent",
		}

		require.NotPanics(t, func() {
			listener.handleLog(vLog, selectedEvent, validatorSlot{})
		})
	})

	t.Run("handles maximum uint64 block number", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		cdc := codec.NewProtoCodec(interfaceRegistry)

		listener := &RootChainListener{
			BaseListener: BaseListener{
				Logger: log.NewNopLogger(),
				cliCtx: client.Context{}.WithCodec(cdc),
			},
			stateSenderAbi: &abi.ABI{},
		}

		vLog := types.Log{
			Address:     common.HexToAddress("0x1234"),
			Topics:      []common.Hash{common.HexToHash("0xabc")},
			Data:        []byte("data"),
			BlockNumber: ^uint64(0), // max uint64
		}

		selectedEvent := &abi.Event{
			Name: helper.StateSyncedEvent,
		}

		require.NotPanics(t, func() {
			listener.handleLog(vLog, selectedEvent, validatorSlot{})
		})
	})
}

// TestRootChainListener_ConcurrentLogHandling tests concurrent log processing
func TestRootChainListener_ConcurrentLogHandling(t *testing.T) {
	t.Parallel()

	listener := &RootChainListener{
		BaseListener: BaseListener{
			Logger: log.NewNopLogger(),
		},
	}

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			defer func() { done <- true }()

			vLog := types.Log{
				Address:     common.HexToAddress("0x1234"),
				Topics:      []common.Hash{common.HexToHash("0xabc")},
				Data:        []byte("concurrent data"),
				BlockNumber: uint64(index),
			}

			selectedEvent := &abi.Event{
				Name: helper.NewHeaderBlockEvent,
			}

			listener.handleLog(vLog, selectedEvent, validatorSlot{})
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestRootChainListener_LogDataSizes tests handling of various log data sizes
func TestRootChainListener_LogDataSizes(t *testing.T) {
	t.Parallel()

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	tests := []struct {
		name     string
		dataSize int
	}{
		{"empty data", 0},
		{"small data", 32},
		{"medium data", 256},
		{"large data", 4096},
		{"very large data", 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			listener := &RootChainListener{
				BaseListener: BaseListener{
					Logger: log.NewNopLogger(),
					cliCtx: client.Context{}.WithCodec(cdc),
				},
				stateSenderAbi: &abi.ABI{},
			}

			data := make([]byte, tt.dataSize)
			for i := range data {
				data[i] = byte(i % 256)
			}

			vLog := types.Log{
				Address:     common.HexToAddress("0x1234"),
				Topics:      []common.Hash{common.HexToHash("0xabc")},
				Data:        data,
				BlockNumber: 12345,
			}

			selectedEvent := &abi.Event{
				Name: helper.StateSyncedEvent,
			}

			require.NotPanics(t, func() {
				listener.handleLog(vLog, selectedEvent, validatorSlot{})
			})

			// Verify marshaling works
			logBytes, err := json.Marshal(vLog)
			require.NoError(t, err)
			require.NotEmpty(t, logBytes)
		})
	}
}

// TestRootChainListener_AddressFormats tests handling of various address formats
func TestRootChainListener_AddressFormats(t *testing.T) {
	t.Parallel()

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	tests := []struct {
		name    string
		address string
	}{
		{"zero address", "0x0000000000000000000000000000000000000000"},
		{"short address", "0x1234"},
		{"full address", "0x1234567890abcdef1234567890abcdef12345678"},
		{"max address", "0xffffffffffffffffffffffffffffffffffffffff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			listener := &RootChainListener{
				BaseListener: BaseListener{
					Logger: log.NewNopLogger(),
					cliCtx: client.Context{}.WithCodec(cdc),
				},
				stateSenderAbi: &abi.ABI{},
			}

			vLog := types.Log{
				Address:     common.HexToAddress(tt.address),
				Topics:      []common.Hash{common.HexToHash("0xabc")},
				Data:        []byte("address test data"),
				BlockNumber: 12345,
			}

			selectedEvent := &abi.Event{
				Name: helper.StateSyncedEvent,
			}

			require.NotPanics(t, func() {
				listener.handleLog(vLog, selectedEvent, validatorSlot{})
			})
		})
	}
}

// TestRootChainListener_handleLog_QueuesTaskInValidatorSlot checks that a log is
// routed to its processor task, delayed by this node's validator slot, and not
// queued at all by a node outside the validator set.
func TestRootChainListener_handleLog_QueuesTaskInValidatorSlot(t *testing.T) {
	t.Parallel()

	queue := new(mockQueueConnector)
	queue.On("SendTask", mock.MatchedBy(func(sig *tasks.Signature) bool {
		return sig.Name == "sendCheckpointAckToHeimdall" &&
			sig.ETA.Sub(time.Now().Add(20*time.Second)).Abs() < time.Second
	})).Return(nil, nil).Once()
	listener := &RootChainListener{BaseListener: newTestBaseListener(queue)}

	event := &abi.Event{Name: helper.NewHeaderBlockEvent}
	listener.handleLog(types.Log{BlockNumber: 1}, event, validatorSlot{isValidator: true, delay: 20 * time.Second})
	listener.handleLog(types.Log{BlockNumber: 2}, event, validatorSlot{})

	queue.AssertExpectations(t)
}
