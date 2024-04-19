package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary x/checkpoint interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgCheckpointAdjust{}, "cosmos-sdk/MsgCheckpointAdjust")
	legacy.RegisterAminoMsg(cdc, &MsgCheckpoint{}, "cosmos-sdk/MsgCheckpoint")
	legacy.RegisterAminoMsg(cdc, &MsgCheckpointAck{}, "cosmos-sdk/MsgCheckpointAck")
	legacy.RegisterAminoMsg(cdc, &MsgCheckpointNoAck{}, "cosmos-sdk/MsgCheckpointNoAck")

}

// RegisterInterfaces registers the x/checkpoint interfaces types with the interface registry
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCheckpointAdjust{},
		&MsgCheckpoint{},
		&MsgCheckpointAck{},
		&MsgCheckpointNoAck{},
	)
	//TODO HV2 Please check whether we need this
	// registry.RegisterImplementations(
	// 	(*authz.Authorization)(nil),
	// 	&StakeAuthorization{},
	// )

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
