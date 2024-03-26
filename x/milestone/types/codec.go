package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary x/milestone interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgMilestone{}, "heimdall-v2/MsgMilestone")
	legacy.RegisterAminoMsg(cdc, &MsgMilestoneTimeout{}, "heimdall-v2/MsgMilestoneTimeout")
}

// RegisterInterfaces registers the x/staking interfaces types with the interface registry
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgMilestone{},
		&MsgMilestoneTimeout{},
	)

	//TODO H2 Please check whether we need this
	// registry.RegisterImplementations(
	// 	(*authz.Authorization)(nil),
	// 	&StakeAuthorization{},
	// )

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
