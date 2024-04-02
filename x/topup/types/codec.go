package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary topup interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	// TODO HV2: are we using cosmos-sdk namespace here or we change it to heimdall-v2? Check common and custom modules
	legacy.RegisterAminoMsg(cdc, &MsgTopupTx{}, "cosmos-sdk/MsgTopupTx")
	legacy.RegisterAminoMsg(cdc, &MsgWithdrawFeeTx{}, "cosmos-sdk/MsgWithdrawFeeTx")
}

// RegisterInterfaces registers the topup msg implementations in the registry
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgTopupTx{},
		&MsgWithdrawFeeTx{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
