// Package rest provides HTTP types and primitives for REST
// requests validation and responses handling.
package rest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BaseReq defines a structure that can be embedded in other request structures
// that all share common "base" fields.
type BaseReq struct {
	From          string       `json:"from"`
	Memo          string       `json:"memo"`
	ChainID       string       `json:"chain_id"`
	AccountNumber uint64       `json:"account_number"`
	Sequence      uint64       `json:"sequence"`
	Fees          sdk.Coins    `json:"fees"`
	GasPrices     sdk.DecCoins `json:"gas_prices"`
	Gas           string       `json:"gas"`
	GasAdjustment string       `json:"gas_adjustment"`
	Simulate      bool         `json:"simulate"`
}

// NewBaseReq creates a new basic request instance and sanitizes its values
func NewBaseReq(
	from, memo, chainID string, gas, gasAdjustment string, accNumber, seq uint64,
	fees sdk.Coins, gasPrices sdk.DecCoins, simulate bool,
) BaseReq {
	return BaseReq{
		From:          strings.TrimSpace(from),
		Memo:          strings.TrimSpace(memo),
		ChainID:       strings.TrimSpace(chainID),
		Fees:          fees,
		GasPrices:     gasPrices,
		Gas:           strings.TrimSpace(gas),
		GasAdjustment: strings.TrimSpace(gasAdjustment),
		AccountNumber: accNumber,
		Sequence:      seq,
		Simulate:      simulate,
	}
}

// Sanitize performs basic sanitization on a BaseReq object.
func (br BaseReq) Sanitize() BaseReq {
	return NewBaseReq(
		br.From, br.Memo, br.ChainID, br.Gas, br.GasAdjustment,
		br.AccountNumber, br.Sequence, br.Fees, br.GasPrices, br.Simulate,
	)
}

// ValidateBasic performs basic validation of a BaseReq. If custom validation
// logic is needed, the implementing request handler should perform those
// checks manually.
func (br BaseReq) ValidateBasic(w http.ResponseWriter) bool {
	if !br.Simulate {
		switch {
		case len(br.ChainID) == 0:
			WriteErrorResponse(w, http.StatusBadRequest, "chain-id required but not specified")
			return false

		case !br.Fees.IsZero() && !br.GasPrices.IsZero():
			// both fees and gas prices were provided
			WriteErrorResponse(w, http.StatusBadRequest, "cannot provide both fees and gas prices")
			return false

		case !br.Fees.IsValid() && !br.GasPrices.IsValid():
			// neither fees nor gas prices were provided
			WriteErrorResponse(w, http.StatusBadRequest, "invalid fees or gas prices provided")
			return false
		}
	}

	if len(br.From) == 0 {
		WriteErrorResponse(w, http.StatusUnauthorized, fmt.Sprintf("invalid from address: %s", br.From))
		return false
	}

	return true
}

// ErrorResponse defines the attributes of a JSON error response.
type ErrorResponse struct {
	Code  int    `json:"code,omitempty"`
	Error string `json:"error"`
}

// NewErrorResponse creates a new ErrorResponse instance.
func NewErrorResponse(code int, err string) ErrorResponse {
	return ErrorResponse{Code: code, Error: err}
}

// WriteErrorResponse prepares and writes an HTTP error
// given a status code and an error message.
func WriteErrorResponse(w http.ResponseWriter, status int, err string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(codec.NewLegacyAmino().MustMarshalJSON(NewErrorResponse(0, err)))
}
