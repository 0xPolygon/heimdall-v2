package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary x/stake interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgValidatorJoin{}, "cosmos-sdk/MsgValidatorJoin")
	legacy.RegisterAminoMsg(cdc, &MsgStakeUpdate{}, "cosmos-sdk/MsgStakeUpdate")
	legacy.RegisterAminoMsg(cdc, &MsgSignerUpdate{}, "cosmos-sdk/MsgSignerUpdate")
	legacy.RegisterAminoMsg(cdc, &MsgValidatorExit{}, "cosmos-sdk/MsgValidatorExit")

}

// RegisterInterfaces registers the x/stake interfaces types with the interface registry
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgValidatorJoin{},
		&MsgStakeUpdate{},
		&MsgSignerUpdate{},
		&MsgValidatorExit{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
