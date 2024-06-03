package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary x/topup interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	// TODO HV2: we are using heimdall-v2 namespace. Other modules need to be double checked to be sure they are consistent with this choice (and not using cosmos-sdk).
	legacy.RegisterAminoMsg(cdc, &MsgTopupTx{}, "heimdall-v2/MsgTopupTx")
	legacy.RegisterAminoMsg(cdc, &MsgWithdrawFeeTx{}, "heimdall-v2/MsgWithdrawFeeTx")
}

// RegisterInterfaces registers the topup msg implementations in the registry
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgTopupTx{},
		&MsgWithdrawFeeTx{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
