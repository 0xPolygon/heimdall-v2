package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
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

	t.Run("can marshal messages after registration", func(t *testing.T) {
		t.Parallel()

		cdc := codec.NewLegacyAmino()
		types.RegisterLegacyAminoCodec(cdc)

		// Create a simple message without requiring full validator join setup
		msg := &types.MsgValidatorExit{
			From:  "0x1234567890123456789012345678901234567890",
			ValId: 1,
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

	t.Run("can resolve MsgValidatorJoin as sdk.Msg", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)

		msg := &types.MsgValidatorJoin{
			From:         "0x1234567890123456789012345678901234567890",
			ValId:        1,
			SignerPubKey: make([]byte, 65),
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
		var _ sdk.Msg = &types.MsgValidatorJoin{}
		var _ sdk.Msg = &types.MsgStakeUpdate{}
		var _ sdk.Msg = &types.MsgSignerUpdate{}
		var _ sdk.Msg = &types.MsgValidatorExit{}
	})
}
