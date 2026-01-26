package types_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
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

	t.Run("can marshal MsgEventRecord after registration", func(t *testing.T) {
		t.Parallel()

		cdc := codec.NewLegacyAmino()
		types.RegisterLegacyAminoCodec(cdc)

		msg := types.NewMsgEventRecord(
			"0x1234567890123456789012345678901234567890",
			"0xabcdef1234567890",
			1,
			1000,
			5,
			make([]byte, 20),
			[]byte("test_data"),
			"137",
		)

		require.NotPanics(t, func() {
			_, err := cdc.MarshalJSON(&msg)
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

	t.Run("can resolve MsgEventRecord as sdk.Msg", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)

		msg := &types.MsgEventRecord{
			From:            "0x1234567890123456789012345678901234567890",
			TxHash:          "0xabcdef",
			LogIndex:        1,
			BlockNumber:     1000,
			Id:              5,
			ContractAddress: "0x0987654321098765432109876543210987654321",
			Data:            []byte("test"),
			ChainId:         "137",
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
		var _ sdk.Msg = &types.MsgEventRecord{}
	})
}

func TestCodecIntegration(t *testing.T) {
	t.Parallel()

	t.Run("can marshal and unmarshal MsgEventRecord", func(t *testing.T) {
		t.Parallel()

		interfaceRegistry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(interfaceRegistry)
		cdc := codec.NewProtoCodec(interfaceRegistry)

		msg := types.NewMsgEventRecord(
			"0x1234567890123456789012345678901234567890",
			"0xabcdef1234567890",
			1,
			1000,
			5,
			make([]byte, 20),
			[]byte("test_data"),
			"137",
		)

		bz, err := cdc.MarshalJSON(&msg)
		require.NoError(t, err)
		require.NotEmpty(t, bz)

		var unmarshalledMsg types.MsgEventRecord
		err = cdc.UnmarshalJSON(bz, &unmarshalledMsg)
		require.NoError(t, err)
		require.Equal(t, msg.TxHash, unmarshalledMsg.TxHash)
		require.Equal(t, msg.LogIndex, unmarshalledMsg.LogIndex)
		require.Equal(t, msg.ChainId, unmarshalledMsg.ChainId)
	})
}
