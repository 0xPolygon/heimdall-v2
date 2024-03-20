package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterLegacyAminoCodec registers the necessary x/chainmanager interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&Params{}, "heimdall-v2/x/chainmanmager/Params", nil)
}

// NOTE(Heimdall-v2): RegisterInterfaces is a no-op as the chainmanager module doesn't have any Msg types
func RegisterInterfaces(registry types.InterfaceRegistry) {}
