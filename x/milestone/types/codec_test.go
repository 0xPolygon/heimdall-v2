package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
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

	t.Run("can marshal MsgUpdateParams after registration", func(t *testing.T) {
		t.Parallel()

		cdc := codec.NewLegacyAmino()
		types.RegisterLegacyAminoCodec(cdc)

		msg := &types.MsgUpdateParams{
			Authority: "cosmos1...",
			Params:    types.DefaultParams(),
		}

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

	t.Run("can resolve MsgUpdateParams as sdk.Msg", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)

		msg := &types.MsgUpdateParams{
			Authority: "cosmos1...",
			Params:    types.DefaultParams(),
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
		var _ sdk.Msg = &types.MsgUpdateParams{}
	})
}

func TestCodecIntegration(t *testing.T) {
	t.Parallel()

	t.Run("can marshal and unmarshal MsgUpdateParams", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		msg := &types.MsgUpdateParams{
			Authority: "cosmos1...",
			Params:    types.DefaultParams(),
		}

		bz, err := cdc.MarshalJSON(msg)
		require.NoError(t, err)
		require.NotEmpty(t, bz)

		var unmarshalledMsg types.MsgUpdateParams
		err = cdc.UnmarshalJSON(bz, &unmarshalledMsg)
		require.NoError(t, err)
		require.Equal(t, msg.Authority, unmarshalledMsg.Authority)
	})
}
