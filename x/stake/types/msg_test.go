package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	hmTypes "github.com/0xPolygon/heimdall-v2/x/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var coinPos = sdk.NewInt64Coin(sdk.DefaultBondDenom, 1000)

func TestMsgDecode(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	// firstly we start testing the pubkey serialization

	pk1 := secp256k1.GenPrivKey().PubKey()

	pk1bz, err := cdc.MarshalInterface(pk1)
	require.NoError(t, err)
	var pkUnmarshaled cryptotypes.PubKey
	err = cdc.UnmarshalInterface(pk1bz, &pkUnmarshaled)
	require.NoError(t, err)
	require.True(t, pk1.Equals(pkUnmarshaled.(*secp256k1.PubKey)))

	msgValJoin, err := types.NewMsgValidatorJoin(
		pk1.Address().String(),
		uint64(1),
		uint64(1),
		math.NewInt(int64(1000000000000000000)),
		pk1,
		hmTypes.TxHash{},
		uint64(1),
		uint64(0),
		uint64(1),
	)

	require.NoError(t, err)
	msgSerialized, err := cdc.MarshalInterface(msgValJoin)
	require.NoError(t, err)

	var msgUnmarshaled sdk.Msg
	err = cdc.UnmarshalInterface(msgSerialized, &msgUnmarshaled)
	require.NoError(t, err)
	msgValJoin2, ok := msgUnmarshaled.(*types.MsgValidatorJoin)
	require.True(t, ok)
	require.Equal(t, msgValJoin.From, msgValJoin2.From)
	require.True(t, msgValJoin.SignerPubKey.Equal(msgValJoin2.SignerPubKey))
	require.Equal(t, msgValJoin.ActivationEpoch, msgValJoin2.ActivationEpoch)
	require.Equal(t, msgValJoin.ValId, msgValJoin2.ValId)

	msgSignerUpdate, err := types.NewMsgSignerUpdate(
		pk1.Address().String(),
		uint64(1),
		pk1,
		hmTypes.TxHash{},
		uint64(1),
		uint64(0),
		uint64(1),
	)

	require.NoError(t, err)
	msgSerialized, err = cdc.MarshalInterface(msgSignerUpdate)
	require.NoError(t, err)

	err = cdc.UnmarshalInterface(msgSerialized, &msgUnmarshaled)
	require.NoError(t, err)
	msgSignerUpdate2, ok := msgUnmarshaled.(*types.MsgSignerUpdate)
	require.True(t, ok)
	require.Equal(t, msgSignerUpdate.From, msgSignerUpdate2.From)
	require.True(t, msgSignerUpdate.NewSignerPubKey.Equal(msgSignerUpdate2.NewSignerPubKey))
	require.Equal(t, msgSignerUpdate.ValId, msgSignerUpdate2.ValId)

	msgStakeUpdate, err := types.NewMsgStakeUpdate(
		pk1.Address().String(),
		uint64(1),
		math.NewInt(int64(100000)),
		hmTypes.TxHash{},
		uint64(1),
		uint64(0),
		uint64(1),
	)

	require.NoError(t, err)
	msgSerialized, err = cdc.MarshalInterface(msgStakeUpdate)
	require.NoError(t, err)

	err = cdc.UnmarshalInterface(msgSerialized, &msgUnmarshaled)
	require.NoError(t, err)
	msgStakeUpdate2, ok := msgUnmarshaled.(*types.MsgStakeUpdate)
	require.True(t, ok)
	require.Equal(t, msgStakeUpdate.From, msgStakeUpdate2.From)
	require.Equal(t, msgStakeUpdate.ValId, msgStakeUpdate2.ValId)
	require.Equal(t, msgStakeUpdate.NewAmount, msgStakeUpdate2.NewAmount)

	msgValidatorExit, err := types.NewMsgValidatorExit(
		pk1.Address().String(),
		uint64(1),
		uint64(1),
		pk1,
		hmTypes.TxHash{},
		uint64(1),
		uint64(0),
		uint64(1),
	)

	require.NoError(t, err)
	msgSerialized, err = cdc.MarshalInterface(msgValidatorExit)
	require.NoError(t, err)

	err = cdc.UnmarshalInterface(msgSerialized, &msgUnmarshaled)
	require.NoError(t, err)
	msgValidatorExit2, ok := msgUnmarshaled.(*types.MsgValidatorExit)
	require.True(t, ok)
	require.Equal(t, msgValidatorExit.From, msgValidatorExit2.From)
	require.Equal(t, msgValidatorExit.ValId, msgValidatorExit2.ValId)
	require.Equal(t, msgValidatorExit.DeactivationEpoch, msgValidatorExit2.DeactivationEpoch)

}
