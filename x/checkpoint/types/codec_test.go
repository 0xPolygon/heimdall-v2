package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

func TestRegisterLegacyAminoCodec(t *testing.T) {
	t.Parallel()

	t.Run("registers without panic", func(t *testing.T) {
		t.Parallel()

		cdc := codec.NewLegacyAmino()

		require.NotPanics(t, func() {
			types.RegisterLegacyAminoCodec(cdc)
		})
	})

	t.Run("can marshal MsgCheckpoint after registration", func(t *testing.T) {
		t.Parallel()

		cdc := codec.NewLegacyAmino()
		types.RegisterLegacyAminoCodec(cdc)

		msg := types.NewMsgCheckpointBlock(
			"0x1234567890123456789012345678901234567890",
			1000,
			2000,
			make([]byte, 32),
			make([]byte, 32),
			"137",
		)

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

	t.Run("can resolve MsgCheckpoint as sdk.Msg", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)

		msg := &types.MsgCheckpoint{
			Proposer:   "0x1234567890123456789012345678901234567890",
			StartBlock: 1000,
			EndBlock:   2000,
			BorChainId: "137",
		}

		anyMsg, err := codectypes.NewAnyWithValue(msg)
		require.NoError(t, err)
		require.NotNil(t, anyMsg)

		var unpackedMsg sdk.Msg
		err = interfaceRegistry.UnpackAny(anyMsg, &unpackedMsg)
		require.NoError(t, err)
		require.NotNil(t, unpackedMsg)
	})

	t.Run("can resolve MsgCpAck as sdk.Msg", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)

		msg := &types.MsgCpAck{
			From:       "0x1234567890123456789012345678901234567890",
			Proposer:   "0x0987654321098765432109876543210987654321",
			StartBlock: 1000,
			EndBlock:   2000,
		}

		anyMsg, err := codectypes.NewAnyWithValue(msg)
		require.NoError(t, err)
		require.NotNil(t, anyMsg)

		var unpackedMsg sdk.Msg
		err = interfaceRegistry.UnpackAny(anyMsg, &unpackedMsg)
		require.NoError(t, err)
		require.NotNil(t, unpackedMsg)
	})

	t.Run("can resolve MsgCpNoAck as sdk.Msg", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)

		msg := &types.MsgCpNoAck{
			From: "0x1234567890123456789012345678901234567890",
		}

		anyMsg, err := codectypes.NewAnyWithValue(msg)
		require.NoError(t, err)
		require.NotNil(t, anyMsg)

		var unpackedMsg sdk.Msg
		err = interfaceRegistry.UnpackAny(anyMsg, &unpackedMsg)
		require.NoError(t, err)
		require.NotNil(t, unpackedMsg)
	})

	t.Run("registered messages implement sdk.Msg interface", func(t *testing.T) {
		t.Parallel()

		// Verify at compile time
		var _ sdk.Msg = &types.MsgCheckpoint{}
		var _ sdk.Msg = &types.MsgCpAck{}
		var _ sdk.Msg = &types.MsgCpNoAck{}
	})
}

func TestCodecIntegration(t *testing.T) {
	t.Parallel()

	t.Run("can marshal and unmarshal MsgCheckpoint", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		msg := types.NewMsgCheckpointBlock(
			"0x1234567890123456789012345678901234567890",
			1000,
			2000,
			make([]byte, 32),
			make([]byte, 32),
			"137",
		)

		bz, err := cdc.MarshalJSON(msg)
		require.NoError(t, err)
		require.NotEmpty(t, bz)

		var unmarshalledMsg types.MsgCheckpoint
		err = cdc.UnmarshalJSON(bz, &unmarshalledMsg)
		require.NoError(t, err)
		require.Equal(t, msg.StartBlock, unmarshalledMsg.StartBlock)
		require.Equal(t, msg.EndBlock, unmarshalledMsg.EndBlock)
		require.Equal(t, msg.BorChainId, unmarshalledMsg.BorChainId)
	})
}
