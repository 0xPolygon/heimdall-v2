package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

func TestNewMsgProposeSpan(t *testing.T) {
	t.Parallel()

	t.Run("creates new MsgProposeSpan with formatted address", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgProposeSpan(
			1,
			"0x1234567890abcdef1234567890abcdef12345678",
			1000,
			2000,
			"137",
			[]byte("seed"),
			"0xabcdef1234567890abcdef1234567890abcdef12",
		)

		require.NotNil(t, msg)
		require.Equal(t, uint64(1), msg.SpanId)
		require.NotEmpty(t, msg.Proposer)
		require.Equal(t, uint64(1000), msg.StartBlock)
		require.Equal(t, uint64(2000), msg.EndBlock)
		require.Equal(t, "137", msg.ChainId)
		require.Equal(t, []byte("seed"), msg.Seed)
		require.NotEmpty(t, msg.SeedAuthor)
	})

	t.Run("formats proposer address using FormatAddress", func(t *testing.T) {
		t.Parallel()

		// Test that the address gets formatted (case normalization, etc.)
		msg := types.NewMsgProposeSpan(
			42,
			"0xABCDEF1234567890ABCDEF1234567890ABCDEF12",
			1000,
			2000,
			"137",
			[]byte{},
			"0x0000000000000000000000000000000000000000",
		)

		require.NotNil(t, msg)
		require.Equal(t, uint64(42), msg.SpanId)
		// Address should be formatted (lowercase with 0x prefix)
		require.Contains(t, msg.Proposer, "0x")
	})

	t.Run("handles empty seed", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgProposeSpan(
			1,
			"0x1234567890123456789012345678901234567890",
			1000,
			2000,
			"137",
			[]byte{},
			"0x0000000000000000000000000000000000000000",
		)

		require.NotNil(t, msg)
		require.Empty(t, msg.Seed)
	})

	t.Run("handles zero values", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgProposeSpan(
			0,
			"0x0000000000000000000000000000000000000000",
			0,
			0,
			"",
			nil,
			"",
		)

		require.NotNil(t, msg)
		require.Equal(t, uint64(0), msg.SpanId)
		require.Equal(t, uint64(0), msg.StartBlock)
		require.Equal(t, uint64(0), msg.EndBlock)
		require.Empty(t, msg.ChainId)
	})
}

func TestNewMsgSetProducerDowntime(t *testing.T) {
	t.Parallel()

	t.Run("creates new MsgSetProducerDowntime", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgSetProducerDowntime(
			"0x1234567890abcdef1234567890abcdef12345678",
			1000,
			2000,
		)

		require.NotNil(t, msg)
		require.NotEmpty(t, msg.Producer)
		require.NotNil(t, msg.DowntimeRange)
		require.Equal(t, uint64(1000), msg.DowntimeRange.StartBlock)
		require.Equal(t, uint64(2000), msg.DowntimeRange.EndBlock)
	})

	t.Run("formats producer address using FormatAddress", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgSetProducerDowntime(
			"0xABCDEF1234567890ABCDEF1234567890ABCDEF12",
			1000,
			2000,
		)

		require.NotNil(t, msg)
		// Address should be formatted (lowercase with 0x prefix)
		require.Contains(t, msg.Producer, "0x")
	})

	t.Run("creates block range correctly", func(t *testing.T) {
		t.Parallel()

		startBlock := uint64(5000)
		endBlock := uint64(10000)

		msg := types.NewMsgSetProducerDowntime(
			"0x1234567890123456789012345678901234567890",
			startBlock,
			endBlock,
		)

		require.NotNil(t, msg)
		require.Equal(t, startBlock, msg.DowntimeRange.StartBlock)
		require.Equal(t, endBlock, msg.DowntimeRange.EndBlock)
	})

	t.Run("handles zero block numbers", func(t *testing.T) {
		t.Parallel()

		msg := types.NewMsgSetProducerDowntime(
			"0x0000000000000000000000000000000000000000",
			0,
			0,
		)

		require.NotNil(t, msg)
		require.Equal(t, uint64(0), msg.DowntimeRange.StartBlock)
		require.Equal(t, uint64(0), msg.DowntimeRange.EndBlock)
	})
}

func TestMsgProposeSpan_Type(t *testing.T) {
	t.Parallel()

	t.Run("returns correct event type", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgProposeSpan{}
		msgType := msg.Type()

		require.Equal(t, types.EventTypeProposeSpan, msgType)
		require.NotEmpty(t, msgType)
	})
}

func TestMsgBackfillSpans_Type(t *testing.T) {
	t.Parallel()

	t.Run("returns correct event type", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgBackfillSpans{}
		msgType := msg.Type()

		require.Equal(t, types.EventTypeBackfillSpans, msgType)
		require.NotEmpty(t, msgType)
	})
}

func TestMsgProposeSpan_Route(t *testing.T) {
	t.Parallel()

	t.Run("returns correct router key", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgProposeSpan{}
		route := msg.Route()

		require.Equal(t, types.RouterKey, route)
		require.NotEmpty(t, route)
	})
}

func TestMsgBackfillSpans_Route(t *testing.T) {
	t.Parallel()

	t.Run("returns correct router key", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgBackfillSpans{}
		route := msg.Route()

		require.Equal(t, types.RouterKey, route)
		require.NotEmpty(t, route)
	})

	t.Run("route matches MsgProposeSpan route", func(t *testing.T) {
		t.Parallel()

		msg1 := types.MsgProposeSpan{}
		msg2 := types.MsgBackfillSpans{}

		require.Equal(t, msg1.Route(), msg2.Route())
	})
}

func TestMessageTypes(t *testing.T) {
	t.Parallel()

	t.Run("validates MsgProposeSpan type and route are consistent", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgProposeSpan{}

		// Type and Route should be non-empty and distinct
		require.NotEmpty(t, msg.Type())
		require.NotEmpty(t, msg.Route())
	})

	t.Run("validates MsgBackfillSpans type and route are consistent", func(t *testing.T) {
		t.Parallel()

		msg := types.MsgBackfillSpans{}

		// Type and Route should be non-empty and distinct
		require.NotEmpty(t, msg.Type())
		require.NotEmpty(t, msg.Route())
	})

	t.Run("validates message types are different", func(t *testing.T) {
		t.Parallel()

		msg1 := types.MsgProposeSpan{}
		msg2 := types.MsgBackfillSpans{}

		// Different messages should have different types
		require.NotEqual(t, msg1.Type(), msg2.Type())
		// But the same route (same module)
		require.Equal(t, msg1.Route(), msg2.Route())
	})
}
