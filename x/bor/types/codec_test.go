package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

func TestRegisterLegacyAminoCodec(t *testing.T) {
	t.Parallel()

	t.Run("registers legacy amino codec without panic", func(t *testing.T) {
		t.Parallel()

		cdc := codec.NewLegacyAmino()

		require.NotPanics(t, func() {
			types.RegisterLegacyAminoCodec(cdc)
		})
	})

	t.Run("can marshal MsgProposeSpan after registration", func(t *testing.T) {
		t.Parallel()

		cdc := codec.NewLegacyAmino()
		types.RegisterLegacyAminoCodec(cdc)

		msg := types.NewMsgProposeSpan(
			1,
			"0x1234567890123456789012345678901234567890",
			1000,
			2000,
			"137",
			[]byte("seed"),
			"0xabcd",
		)

		// Should be able to marshal without panic
		require.NotPanics(t, func() {
			_, err := cdc.MarshalJSON(msg)
			require.NoError(t, err)
		})
	})
}

func TestRegisterInterfaces(t *testing.T) {
	t.Parallel()

	t.Run("registers interfaces without panic", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()

		require.NotPanics(t, func() {
			types.RegisterInterfaces(interfaceRegistry)
		})
	})

	t.Run("can resolve MsgProposeSpan as sdk.Msg", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)

		msg := &types.MsgProposeSpan{
			SpanId:     1,
			Proposer:   "0x1234567890123456789012345678901234567890",
			StartBlock: 1000,
			EndBlock:   2000,
			ChainId:    "137",
		}

		// Should be able to pack as sdk.Msg interface
		anyMsg, err := codectypes.NewAnyWithValue(msg)
		require.NoError(t, err)
		require.NotNil(t, anyMsg)

		// Should be able to unpack back
		var unpackedMsg sdk.Msg
		err = interfaceRegistry.UnpackAny(anyMsg, &unpackedMsg)
		require.NoError(t, err)
		require.NotNil(t, unpackedMsg)
	})

	t.Run("can resolve MsgUpdateParams as sdk.Msg", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)

		msg := &types.MsgUpdateParams{
			Authority: "0x1234567890123456789012345678901234567890",
			Params: types.Params{
				SprintDuration: 16,
				SpanDuration:   6400,
				ProducerCount:  4,
			},
		}

		// Should be able to pack as sdk.Msg interface
		anyMsg, err := codectypes.NewAnyWithValue(msg)
		require.NoError(t, err)
		require.NotNil(t, anyMsg)

		// Should be able to unpack back
		var unpackedMsg sdk.Msg
		err = interfaceRegistry.UnpackAny(anyMsg, &unpackedMsg)
		require.NoError(t, err)
		require.NotNil(t, unpackedMsg)
	})

	t.Run("registered messages implement sdk.Msg interface", func(t *testing.T) {
		t.Parallel()

		// Verify at compile time that messages implement sdk.Msg
		var _ sdk.Msg = &types.MsgProposeSpan{}
		var _ sdk.Msg = &types.MsgUpdateParams{}
	})
}

func TestCodecIntegration(t *testing.T) {
	t.Parallel()

	t.Run("can marshal and unmarshal with proto codec", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		msg := types.NewMsgProposeSpan(
			42,
			"0x1234567890123456789012345678901234567890",
			1000,
			2000,
			"137",
			[]byte("test seed"),
			"0xabcdef",
		)

		// Marshal
		bz, err := cdc.MarshalJSON(msg)
		require.NoError(t, err)
		require.NotEmpty(t, bz)

		// Unmarshal
		var unmarshalledMsg types.MsgProposeSpan
		err = cdc.UnmarshalJSON(bz, &unmarshalledMsg)
		require.NoError(t, err)
		require.Equal(t, msg.SpanId, unmarshalledMsg.SpanId)
		require.Equal(t, msg.StartBlock, unmarshalledMsg.StartBlock)
		require.Equal(t, msg.EndBlock, unmarshalledMsg.EndBlock)
		require.Equal(t, msg.ChainId, unmarshalledMsg.ChainId)
	})
}
