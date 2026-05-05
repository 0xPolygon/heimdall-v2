package listener

import (
	"testing"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
)

func buildTestEventMap() (map[common.Hash]*abi.Event, []*abi.Event) {
	events := []*abi.Event{
		{Name: "EventA", ID: common.HexToHash("0xaaaa")},
		{Name: "EventB", ID: common.HexToHash("0xbbbb")},
		{Name: "EventC", ID: common.HexToHash("0xcccc")},
	}

	m := make(map[common.Hash]*abi.Event, len(events))
	for _, e := range events {
		m[e.ID] = e
	}

	return m, events
}

func TestEventMap_LookupByTopicHash(t *testing.T) {
	t.Parallel()

	eventMap, events := buildTestEventMap()

	for _, e := range events {
		got, ok := eventMap[e.ID]
		require.True(t, ok, "expected event %s in map", e.Name)
		require.Equal(t, e.Name, got.Name)
	}

	_, ok := eventMap[common.HexToHash("0xdead")]
	require.False(t, ok, "unknown topic should not match")
}

func TestEventMap_NoABIDuplicates(t *testing.T) {
	t.Parallel()

	abiA := abi.ABI{Events: map[string]abi.Event{
		"Transfer": {Name: "Transfer", ID: common.HexToHash("0x1111")},
		"Approval": {Name: "Approval", ID: common.HexToHash("0x2222")},
	}}
	abiB := abi.ABI{Events: map[string]abi.Event{
		"Deposit": {Name: "Deposit", ID: common.HexToHash("0x3333")},
	}}

	eventMap := make(map[common.Hash]*abi.Event)
	for _, abiObj := range []*abi.ABI{&abiA, &abiB} {
		for _, event := range abiObj.Events {
			e := event
			eventMap[e.ID] = &e
		}
	}

	require.Len(t, eventMap, 3)
	require.Equal(t, "Transfer", eventMap[common.HexToHash("0x1111")].Name)
	require.Equal(t, "Approval", eventMap[common.HexToHash("0x2222")].Name)
	require.Equal(t, "Deposit", eventMap[common.HexToHash("0x3333")].Name)
}

func TestNilABI_PanicsOnUnpackLog(t *testing.T) {
	t.Parallel()

	listener := &RootChainListener{
		BaseListener: BaseListener{
			Logger: log.NewNopLogger(),
		},
	}

	// nil stateSenderAbi causes a nil-pointer panic in UnpackLog,
	// indicating a misconfigured listener.
	vLog := types.Log{
		Address:     common.HexToAddress("0x1234"),
		Topics:      []common.Hash{common.HexToHash("0xabc")},
		Data:        []byte("bad abi data"),
		BlockNumber: 100,
	}

	selectedEvent := &abi.Event{Name: helper.StateSyncedEvent}

	// nil stateSenderAbi causes a nil-pointer panic in UnpackLog.
	listener.stateSenderAbi = nil
	require.Panics(t, func() {
		listener.handleStateSyncedLog(vLog, selectedEvent)
	}, "nil ABI must panic, indicates a misconfigured listener")
}

func TestEarlyReturn_UnpackLogError(t *testing.T) {
	t.Parallel()

	listener := &RootChainListener{
		BaseListener: BaseListener{
			Logger: log.NewNopLogger(),
		},
		// empty ABI — UnpackLog will fail because the event signature won't match
		stateSenderAbi: &abi.ABI{},
	}

	vLog := types.Log{
		Address:     common.HexToAddress("0x1234"),
		Topics:      []common.Hash{common.HexToHash("0xabc")},
		Data:        []byte("will not decode"),
		BlockNumber: 100,
	}

	selectedEvent := &abi.Event{Name: helper.StateSyncedEvent}

	// UnpackLog fails → handler should return early without panicking
	// and without calling SendTaskWithDelay (which would panic since queueConnector is nil)
	require.NotPanics(t, func() {
		listener.handleStateSyncedLog(vLog, selectedEvent)
	})
}

func TestEarlyReturn_HandlersWithBadABI(t *testing.T) {
	t.Parallel()

	listener := &RootChainListener{
		BaseListener: BaseListener{
			Logger: log.NewNopLogger(),
		},
		stakingInfoAbi: &abi.ABI{},
		stateSenderAbi: &abi.ABI{},
	}

	vLog := types.Log{
		Address:     common.HexToAddress("0x1234"),
		Topics:      []common.Hash{common.HexToHash("0xabc")},
		Data:        []byte("bad data"),
		BlockNumber: 100,
	}

	// All handlers that decode events should return early on UnpackLog error.
	// If they don't return early, they'd try to call SendTaskWithDelay
	// which would panic because queueConnector is nil.
	handlers := []struct {
		name string
		fn   func()
	}{
		{"StakeUpdate", func() { listener.handleStakeUpdateLog(vLog, &abi.Event{Name: helper.StakeUpdateEvent}) }},
		{"SignerChange", func() { listener.handleSignerChangeLog(vLog, &abi.Event{Name: helper.SignerChangeEvent}) }},
		{"UnstakeInit", func() { listener.handleUnstakeInitLog(vLog, &abi.Event{Name: helper.UnstakeInitEvent}) }},
		{"StateSynced", func() { listener.handleStateSyncedLog(vLog, &abi.Event{Name: helper.StateSyncedEvent}) }},
		{"TopUpFee", func() { listener.handleTopUpFeeLog(vLog, &abi.Event{Name: helper.TopUpFeeEvent}) }},
		{"UnJailed", func() { listener.handleUnJailedLog(vLog, &abi.Event{Name: helper.UnJailedEvent}) }},
	}

	for _, h := range handlers {
		t.Run(h.name, func(t *testing.T) {
			t.Parallel()
			require.NotPanics(t, h.fn,
				"%s should return early on UnpackLog error, not reach SendTaskWithDelay", h.name)
		})
	}
}
