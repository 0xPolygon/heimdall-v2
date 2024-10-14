package app

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
)

// HandlerOptions are the options required for constructing a default SDK AnteHandler.
type HandlerOptions struct {
	ante.HandlerOptions
}

// NewAnteHandler returns an AnteHandler that checks and increments sequence
// numbers, checks signatures & account numbers, and deducts fees from the first
// signer.
func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, errors.New("account keeper is required for ante builder")
	}

	if options.BankKeeper == nil {
		return nil, errors.New("bank keeper is required for ante builder")
	}

	if options.SignModeHandler == nil {
		return nil, errors.New("sign mode handler is required for ante builder")
	}

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
		ante.NewSetPubKeyDecorator(options.AccountKeeper, options.SignModeHandler), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		// TODO HV2: Do we need a sideTxDecorator? See https://polygon.atlassian.net/browse/POS-2705
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
