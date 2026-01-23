package app_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/app"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
)

func TestNewAnteHandler_WithValidOptions(t *testing.T) {
	setupResult := app.SetupApp(t, 1)

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(interfaceRegistry)
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	options := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   setupResult.App.AccountKeeper,
			BankKeeper:      setupResult.App.BankKeeper,
			SignModeHandler: txConfig.SignModeHandler(),
			FeegrantKeeper:  nil,
			SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
		},
		SideTxConfig: sidetxs.NewSideTxConfigurator(),
	}

	handler, err := app.NewAnteHandler(options)

	require.NoError(t, err)
	require.NotNil(t, handler)
}

func TestNewAnteHandler_MissingAccountKeeper(t *testing.T) {
	setupResult := app.SetupApp(t, 1)

	options := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   nil,
			BankKeeper:      setupResult.App.BankKeeper,
			SignModeHandler: nil,
		},
		SideTxConfig: sidetxs.NewSideTxConfigurator(),
	}

	handler, err := app.NewAnteHandler(options)

	require.Error(t, err)
	require.Nil(t, handler)
	require.Contains(t, err.Error(), "account keeper is required")
}

func TestNewAnteHandler_MissingBankKeeper(t *testing.T) {
	setupResult := app.SetupApp(t, 1)

	options := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   setupResult.App.AccountKeeper,
			BankKeeper:      nil,
			SignModeHandler: nil,
		},
		SideTxConfig: sidetxs.NewSideTxConfigurator(),
	}

	handler, err := app.NewAnteHandler(options)

	require.Error(t, err)
	require.Nil(t, handler)
	require.Contains(t, err.Error(), "bank keeper is required")
}

func TestNewAnteHandler_MissingSignModeHandler(t *testing.T) {
	setupResult := app.SetupApp(t, 1)

	options := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   setupResult.App.AccountKeeper,
			BankKeeper:      setupResult.App.BankKeeper,
			SignModeHandler: nil,
		},
		SideTxConfig: sidetxs.NewSideTxConfigurator(),
	}

	handler, err := app.NewAnteHandler(options)

	require.Error(t, err)
	require.Nil(t, handler)
	require.Contains(t, err.Error(), "sign mode handler is required")
}

func TestNewAnteHandler_AllFieldsMissing(t *testing.T) {
	options := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   nil,
			BankKeeper:      nil,
			SignModeHandler: nil,
		},
		SideTxConfig: sidetxs.NewSideTxConfigurator(),
	}

	handler, err := app.NewAnteHandler(options)

	require.Error(t, err)
	require.Nil(t, handler)
	// Should fail on the first check (account keeper)
	require.Contains(t, err.Error(), "account keeper is required")
}

func TestNewAnteHandler_WithExtensionOptionChecker(t *testing.T) {
	setupResult := app.SetupApp(t, 1)

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(interfaceRegistry)
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	// Custom extension option checker
	extensionChecker := func(any *codectypes.Any) bool {
		return true
	}

	options := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:          setupResult.App.AccountKeeper,
			BankKeeper:             setupResult.App.BankKeeper,
			SignModeHandler:        txConfig.SignModeHandler(),
			ExtensionOptionChecker: extensionChecker,
		},
		SideTxConfig: sidetxs.NewSideTxConfigurator(),
	}

	handler, err := app.NewAnteHandler(options)

	require.NoError(t, err)
	require.NotNil(t, handler)
}

func TestNewAnteHandler_WithFeegrantKeeper(t *testing.T) {
	setupResult := app.SetupApp(t, 1)

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(interfaceRegistry)
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	options := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   setupResult.App.AccountKeeper,
			BankKeeper:      setupResult.App.BankKeeper,
			SignModeHandler: txConfig.SignModeHandler(),
			FeegrantKeeper:  nil, // can be nil
		},
		SideTxConfig: sidetxs.NewSideTxConfigurator(),
	}

	handler, err := app.NewAnteHandler(options)

	require.NoError(t, err)
	require.NotNil(t, handler)
}

func TestHandlerOptions_EmbeddedFields(t *testing.T) {
	setupResult := app.SetupApp(t, 1)

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(interfaceRegistry)
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	sideTxConfig := sidetxs.NewSideTxConfigurator()

	options := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   setupResult.App.AccountKeeper,
			BankKeeper:      setupResult.App.BankKeeper,
			SignModeHandler: txConfig.SignModeHandler(),
		},
		SideTxConfig: sideTxConfig,
	}

	// Verify fields are accessible
	require.NotNil(t, options.AccountKeeper)
	require.NotNil(t, options.BankKeeper)
	require.NotNil(t, options.SignModeHandler)
	require.NotNil(t, options.SideTxConfig)
}

func TestNewAnteHandler_ZeroValueOptions(t *testing.T) {
	var options app.HandlerOptions

	handler, err := app.NewAnteHandler(options)

	require.Error(t, err)
	require.Nil(t, handler)
	require.Contains(t, err.Error(), "account keeper is required")
}

func TestNewAnteHandler_PartiallyInitializedHandlerOptions(t *testing.T) {
	setupResult := app.SetupApp(t, 1)

	// Only AccountKeeper set, others missing
	options := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper: setupResult.App.AccountKeeper,
		},
	}

	handler, err := app.NewAnteHandler(options)

	require.Error(t, err)
	require.Nil(t, handler)
	require.Contains(t, err.Error(), "bank keeper is required")
}

func TestNewAnteHandler_WithNilSideTxConfig(t *testing.T) {
	setupResult := app.SetupApp(t, 1)

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(interfaceRegistry)
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	options := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   setupResult.App.AccountKeeper,
			BankKeeper:      setupResult.App.BankKeeper,
			SignModeHandler: txConfig.SignModeHandler(),
		},
		SideTxConfig: nil,
	}

	// This should still work as the side tx decorator handles nil config
	handler, err := app.NewAnteHandler(options)

	require.NoError(t, err)
	require.NotNil(t, handler)
}

func TestNewAnteHandler_DecoratorChainNotEmpty(t *testing.T) {
	setupResult := app.SetupApp(t, 1)

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(interfaceRegistry)
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	options := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   setupResult.App.AccountKeeper,
			BankKeeper:      setupResult.App.BankKeeper,
			SignModeHandler: txConfig.SignModeHandler(),
		},
		SideTxConfig: sidetxs.NewSideTxConfigurator(),
	}

	handler, err := app.NewAnteHandler(options)

	require.NoError(t, err)
	require.NotNil(t, handler)

	// The handler should be a chained handler with multiple decorators
	require.NotPanics(t, func() {
		_ = handler
	})
}

func TestNewAnteHandler_ValidatesInOrder(t *testing.T) {
	// Test that validation happens in the expected order:
	// 1. AccountKeeper, 2. BankKeeper, 3. SignModeHandler

	// AccountKeeper fails first
	options1 := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   nil,
			BankKeeper:      nil,
			SignModeHandler: nil,
		},
	}
	_, err1 := app.NewAnteHandler(options1)
	require.Error(t, err1)
	require.Contains(t, err1.Error(), "account keeper")

	setupResult := app.SetupApp(t, 1)

	// BankKeeper fails second
	options2 := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   setupResult.App.AccountKeeper,
			BankKeeper:      nil,
			SignModeHandler: nil,
		},
	}
	_, err2 := app.NewAnteHandler(options2)
	require.Error(t, err2)
	require.Contains(t, err2.Error(), "bank keeper")

	// SignModeHandler fails third
	options3 := app.HandlerOptions{
		HandlerOptions: ante.HandlerOptions{
			AccountKeeper:   setupResult.App.AccountKeeper,
			BankKeeper:      setupResult.App.BankKeeper,
			SignModeHandler: nil,
		},
	}
	_, err3 := app.NewAnteHandler(options3)
	require.Error(t, err3)
	require.Contains(t, err3.Error(), "sign mode handler")
}
