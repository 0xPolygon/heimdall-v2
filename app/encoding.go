package app

import (
	"cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/cosmos/gogoproto/proto"
)

// EncodingConfig specifies the concrete encoding types to use for a given app.
// This is provided for compatibility between protobuf and amino implementations.
type EncodingConfig struct {
	InterfaceRegistry types.InterfaceRegistry
	Marshaler         codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

// MakeEncodingConfig creates an EncodingConfig for an amino based test configuration.
func MakeEncodingConfig() EncodingConfig {
	interfaceRegistry, _ := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec: address.Bech32Codec{
				Bech32Prefix: sdk.Bech32MainPrefix,
			},
			ValidatorAddressCodec: address.Bech32Codec{
				Bech32Prefix: sdk.Bech32MainPrefix,
			},
		},
	})

	appCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()
	txConfig := tx.NewTxConfig(appCodec, tx.DefaultSignModes)

	// amino := codec.NewLegacyAmino()
	// interfaceRegistry := codectypes.NewInterfaceRegistry()
	// cdc := codec.NewProtoCodec(interfaceRegistry)
	// txCfg := tx.NewTxConfig(cdc, tx.DefaultSignModes)
	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Marshaler:         appCodec,
		TxConfig:          txConfig,
		Amino:             legacyAmino,
	}
}

func RegisterEncodingConfig() EncodingConfig {
	encConfig := MakeEncodingConfig()

	// std.RegisterLegacyAminoCodec(encConfig.Amino)
	// std.RegisterInterfaces(encConfig.InterfaceRegistry)
	// ModuleBasics.RegisterLegacyAminoCodec(encConfig.Amino)
	// ModuleBasics.RegisterInterfaces(encConfig.InterfaceRegistry)

	return encConfig
}
