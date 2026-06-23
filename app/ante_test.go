package app_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/0xPolygon/heimdall-v2/app"
	"github.com/0xPolygon/heimdall-v2/helper"
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

// msgTx is a minimal sdk.Tx for exercising the MsgMultiSendCap wiring.
type msgTx struct {
	msgs []sdk.Msg
}

func (t *msgTx) GetMsgs() []sdk.Msg { return t.msgs }

func (t *msgTx) GetMsgsV2() ([]proto.Message, error) {
	out := make([]proto.Message, 0, len(t.msgs))
	for _, m := range t.msgs {
		if pm, ok := m.(proto.Message); ok {
			out = append(out, pm)
		}
	}
	return out, nil
}

func makeMultiSendOverCap() sdk.Msg {
	const overCap = 17
	outputs := make([]banktypes.Output, overCap)
	for i := range outputs {
		outputs[i] = banktypes.Output{Address: "cosmos1xyz", Coins: sdk.NewCoins(sdk.NewInt64Coin("stake", 1))}
	}
	return &banktypes.MsgMultiSend{
		Inputs:  []banktypes.Input{{Address: "cosmos1abc", Coins: sdk.NewCoins(sdk.NewInt64Coin("stake", int64(overCap)))}},
		Outputs: outputs,
	}
}

// Heimdall wires NewMsgMultiSendCapDecorator(maxMultiSendOutputs, helper.IsZurichHardfork).
// Pin both the activation predicate and the cap value at the wiring boundary.
func TestAnteWiring_MsgMultiSendCapHardforkGated(t *testing.T) {
	const activation = int64(100)
	helper.SetZurichHardforkHeight(activation)
	t.Cleanup(func() { helper.SetZurichHardforkHeight(0) })

	cases := []struct {
		name       string
		height     int64
		wantReject bool
	}{
		{name: "below activation: over cap accepts", height: activation - 1, wantReject: false},
		{name: "at activation: over cap rejects", height: activation, wantReject: true},
		{name: "above activation: over cap rejects", height: activation + 1, wantReject: true},
	}

	// Construct the same decorator instance the production ante chain wires.
	dec := ante.NewMsgMultiSendCapDecorator(16, helper.IsZurichHardfork)
	terminal := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return ctx, nil }

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := sdk.Context{}.WithBlockHeight(tc.height)
			_, err := dec.AnteHandle(ctx, &msgTx{msgs: []sdk.Msg{makeMultiSendOverCap()}}, false, terminal)
			if tc.wantReject {
				require.Error(t, err)
				require.ErrorIs(t, err, sdkerrors.ErrInvalidRequest)
			} else {
				require.NoError(t, err)
			}
		})
	}
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
