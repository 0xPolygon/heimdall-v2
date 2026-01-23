package types_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/topup/types"
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

	t.Run("can marshal MsgTopupTx after registration", func(t *testing.T) {
		t.Parallel()

		cdc := codec.NewLegacyAmino()
		types.RegisterLegacyAminoCodec(cdc)

		msg := &types.MsgTopupTx{
			Proposer:    "0x1234567890123456789012345678901234567890",
			User:        "0x1234567890123456789012345678901234567891",
			Fee:         sdkmath.NewInt(100),
			TxHash:      make([]byte, 32),
			LogIndex:    1,
			BlockNumber: 100,
		}

		require.NotPanics(t, func() {
			_, err := cdc.MarshalJSON(msg)
			require.NoError(t, err)
		})
	})

	t.Run("can marshal MsgWithdrawFeeTx after registration", func(t *testing.T) {
		t.Parallel()

		cdc := codec.NewLegacyAmino()
		types.RegisterLegacyAminoCodec(cdc)

		msg := &types.MsgWithdrawFeeTx{
			Proposer: "0x1234567890123456789012345678901234567890",
			Amount:   sdkmath.NewInt(200),
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

	t.Run("can resolve MsgTopupTx as sdk.Msg", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)

		msg := &types.MsgTopupTx{
			Proposer:    "0x1234567890123456789012345678901234567890",
			User:        "0x1234567890123456789012345678901234567891",
			Fee:         sdkmath.NewInt(100),
			TxHash:      make([]byte, 32),
			LogIndex:    1,
			BlockNumber: 100,
		}

		anyMsg, err := codectypes.NewAnyWithValue(msg)
		require.NoError(t, err)
		require.NotNil(t, anyMsg)

		var unpackedMsg sdk.Msg
		err = interfaceRegistry.UnpackAny(anyMsg, &unpackedMsg)
		require.NoError(t, err)
		require.NotNil(t, unpackedMsg)
	})

	t.Run("can resolve MsgWithdrawFeeTx as sdk.Msg", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)

		msg := &types.MsgWithdrawFeeTx{
			Proposer: "0x1234567890123456789012345678901234567890",
			Amount:   sdkmath.NewInt(200),
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
		var _ sdk.Msg = &types.MsgTopupTx{}
		var _ sdk.Msg = &types.MsgWithdrawFeeTx{}
	})
}

func TestCodecIntegration(t *testing.T) {
	t.Parallel()

	t.Run("can marshal and unmarshal MsgTopupTx", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		msg := &types.MsgTopupTx{
			Proposer:    "0x1234567890123456789012345678901234567890",
			User:        "0x1234567890123456789012345678901234567891",
			Fee:         sdkmath.NewInt(100),
			TxHash:      make([]byte, 32),
			LogIndex:    1,
			BlockNumber: 100,
		}

		bz, err := cdc.MarshalJSON(msg)
		require.NoError(t, err)
		require.NotEmpty(t, bz)

		var unmarshalledMsg types.MsgTopupTx
		err = cdc.UnmarshalJSON(bz, &unmarshalledMsg)
		require.NoError(t, err)
		require.Equal(t, msg.Proposer, unmarshalledMsg.Proposer)
	})

	t.Run("can marshal and unmarshal MsgWithdrawFeeTx", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		msg := &types.MsgWithdrawFeeTx{
			Proposer: "0x1234567890123456789012345678901234567890",
			Amount:   sdkmath.NewInt(200),
		}

		bz, err := cdc.MarshalJSON(msg)
		require.NoError(t, err)
		require.NotEmpty(t, bz)

		var unmarshalledMsg types.MsgWithdrawFeeTx
		err = cdc.UnmarshalJSON(bz, &unmarshalledMsg)
		require.NoError(t, err)
		require.Equal(t, msg.Proposer, unmarshalledMsg.Proposer)
	})
}
