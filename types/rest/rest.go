// Package rest provides HTTP types and primitives for REST
// requests validation and responses handling.
package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	cmtTypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultPage  = 1
	DefaultLimit = 100
)

// NewResponse creates a new Response instance
func NewResponse(height int64, result json.RawMessage) Response {
	return Response{
		Result: result,
	}
}

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

// Not needed (autocli)
/*
// ReadRESTReq reads and unmarshals a Request's body to the BaseReq struct.
// Writes an error response to ResponseWriter and returns true if errors occurred.
func ReadRESTReq(w http.ResponseWriter, r *http.Request, cdc codec.Codec, req interface{}) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return false
	}

	err = cdc.UnmarshalJSON(body, req)
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("failed to decode JSON payload: %s", err))
		return false
	}

	return true
}
*/

// ErrorResponse defines the attributes of a JSON error response.
type ErrorResponse struct {
	Code  int    `json:"code,omitempty"`
	Error string `json:"error"`
}

// NewErrorResponse creates a new ErrorResponse instance.
func NewErrorResponse(code int, err string) ErrorResponse {
	return ErrorResponse{Code: code, Error: err}
}

// WriteErrorResponse prepares and writes a HTTP error
// given a status code and an error message.
func WriteErrorResponse(w http.ResponseWriter, status int, err string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(codec.NewLegacyAmino().MustMarshalJSON(NewErrorResponse(0, err)))
}

// WriteSimulationResponse prepares and writes an HTTP
// response for transactions simulations.
func WriteSimulationResponse(w http.ResponseWriter, cdc codec.Codec, gas uint64) {
	gasEst := GasEstimateResponse{GasEstimate: gas}

	resp, err := cdc.MarshalJSON(&gasEst)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(resp)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
	}
}

// errorResponse defines the attributes of a JSON error response.
type errorResponse struct {
	Code  int    `json:"code,omitempty"`
	Error string `json:"error"`
}

// newErrorResponse creates a new errorResponse instance.
func newErrorResponse(code int, err string) errorResponse {
	return errorResponse{Code: code, Error: err}
}

// writeErrorResponse prepares and writes a HTTP error
// given a status code and an error message.
func writeErrorResponse(w http.ResponseWriter, status int, err string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err1 := w.Write(legacy.Cdc.MustMarshalJSON(newErrorResponse(0, err)))
	if err1 != nil {
		panic(err1)
	}
}

// ReturnNotFoundIfNoContent returns not found error if no content
func ReturnNotFoundIfNoContent(w http.ResponseWriter, data []byte, message string) bool {
	if len(data) == 0 {
		writeErrorResponse(w, http.StatusNotFound, errors.New(message).Error())
		return false
	}

	return true
}

// Not needed (autocli)
/*
// PostProcessResponse performs post-processing for a REST response. The result
// returned to clients will contain two fields, the height at which the resource
// was queried at and the original result.
func PostProcessResponse(w http.ResponseWriter, cliCtx client.Context, resp interface{}) {
	var result []byte

	if cliCtx.Height < 0 {
		WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("negative height in response").Error())
		return
	}

	switch data := resp.(type) {
	case []byte:
		result = data
	default:
		var err error
		result, err = cliCtx.Codec.MarshalJSON(data)

		if err != nil {
			WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	wrappedResp := NewResponse(cliCtx.Height, result)

	var (
		output []byte
		err    error
	)

	output, err = cliCtx.Codec.MarshalJSON(wrappedResp)

	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(output)
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
	}
}
*/

// ParseHTTPArgsWithLimit parses the request's URL and returns a slice containing
// all arguments pairs. It separates page and limit used for pagination where a
// default limit can be provided.
func ParseHTTPArgsWithLimit(r *http.Request, defaultLimit int) (tags []string, page, limit int, err error) {
	tags = make([]string, 0, len(r.Form))

	for key, values := range r.Form {
		if key == "page" || key == "limit" {
			continue
		}

		var value string

		if value, err = url.QueryUnescape(values[0]); err != nil {
			return tags, page, limit, err
		}

		var tag string
		if key == cmtTypes.TxHeightKey {
			tag = fmt.Sprintf("%s=%s", key, value)
		} else {
			tag = fmt.Sprintf("%s='%s'", key, value)
		}

		tags = append(tags, tag)
	}

	pageStr := r.FormValue("page")
	if pageStr == "" {
		page = DefaultPage
	} else {
		page, err = strconv.Atoi(pageStr)
		if err != nil {
			return tags, page, limit, err
		} else if page <= 0 {
			return tags, page, limit, errors.New("page must greater than 0")
		}
	}

	limitStr := r.FormValue("limit")
	if limitStr == "" {
		limit = defaultLimit
	} else {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return tags, page, limit, err
		} else if limit <= 0 {
			return tags, page, limit, errors.New("limit must greater than 0")
		}
	}

	return tags, page, limit, nil
}

// ParseHTTPArgs parses the request's URL and returns a slice containing all
// arguments pairs. It separates page and limit used for pagination.
func ParseHTTPArgs(r *http.Request) (tags []string, page, limit int, err error) {
	return ParseHTTPArgsWithLimit(r, DefaultLimit)
}
