package rest_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/types/rest"
)

func TestNewBaseReq(t *testing.T) {
	tests := []struct {
		name          string
		from          string
		memo          string
		chainID       string
		gas           string
		gasAdjustment string
		accNumber     uint64
		seq           uint64
		fees          sdk.Coins
		gasPrices     sdk.DecCoins
		simulate      bool
	}{
		{
			name:          "basic request",
			from:          "cosmos1address",
			memo:          "test memo",
			chainID:       "heimdall-1",
			gas:           "200000",
			gasAdjustment: "1.5",
			accNumber:     1,
			seq:           10,
			fees:          sdk.NewCoins(sdk.NewInt64Coin("stake", 100)),
			gasPrices:     sdk.NewDecCoins(sdk.NewInt64DecCoin("stake", 1)),
			simulate:      false,
		},
		{
			name:          "with whitespace",
			from:          "  cosmos1address  ",
			memo:          "  test memo  ",
			chainID:       "  heimdall-1  ",
			gas:           "  200000  ",
			gasAdjustment: "  1.5  ",
			accNumber:     0,
			seq:           0,
			fees:          nil,
			gasPrices:     nil,
			simulate:      true,
		},
		{
			name:          "empty values",
			from:          "",
			memo:          "",
			chainID:       "",
			gas:           "",
			gasAdjustment: "",
			accNumber:     0,
			seq:           0,
			fees:          nil,
			gasPrices:     nil,
			simulate:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := rest.NewBaseReq(
				tt.from, tt.memo, tt.chainID, tt.gas, tt.gasAdjustment,
				tt.accNumber, tt.seq, tt.fees, tt.gasPrices, tt.simulate,
			)

			// Verify basic fields
			require.Equal(t, tt.accNumber, req.AccountNumber)
			require.Equal(t, tt.seq, req.Sequence)
			require.Equal(t, tt.simulate, req.Simulate)

			// Verify string fields don't have leading/trailing whitespace
			require.Equal(t, req.From, strings.TrimSpace(req.From))
			require.Equal(t, req.Memo, strings.TrimSpace(req.Memo))
			require.Equal(t, req.ChainID, strings.TrimSpace(req.ChainID))
			require.Equal(t, req.Gas, strings.TrimSpace(req.Gas))
			require.Equal(t, req.GasAdjustment, strings.TrimSpace(req.GasAdjustment))
		})
	}
}

func TestBaseReqSanitize(t *testing.T) {
	req := rest.BaseReq{
		From:          "  cosmos1address  ",
		Memo:          "  memo  ",
		ChainID:       "  chain  ",
		Gas:           "  100  ",
		GasAdjustment: "  1.5  ",
		AccountNumber: 5,
		Sequence:      10,
		Fees:          sdk.NewCoins(sdk.NewInt64Coin("stake", 100)),
		GasPrices:     sdk.NewDecCoins(sdk.NewInt64DecCoin("stake", 1)),
		Simulate:      true,
	}

	sanitized := req.Sanitize()

	// Verify whitespace is trimmed
	require.Equal(t, "cosmos1address", sanitized.From)
	require.Equal(t, "memo", sanitized.Memo)
	require.Equal(t, "chain", sanitized.ChainID)
	require.Equal(t, "100", sanitized.Gas)
	require.Equal(t, "1.5", sanitized.GasAdjustment)
	require.Equal(t, uint64(5), sanitized.AccountNumber)
	require.Equal(t, uint64(10), sanitized.Sequence)
	require.True(t, sanitized.Simulate)
}

func TestBaseReqValidateBasic(t *testing.T) {
	tests := []struct {
		name        string
		req         rest.BaseReq
		expectValid bool
		statusCode  int
	}{
		{
			name: "valid request with fees",
			req: rest.BaseReq{
				From:    "cosmos1address",
				ChainID: "heimdall-1",
				Fees:    sdk.NewCoins(sdk.NewInt64Coin("stake", 100)),
			},
			expectValid: true,
		},
		{
			name: "valid request with gas prices",
			req: rest.BaseReq{
				From:      "cosmos1address",
				ChainID:   "heimdall-1",
				GasPrices: sdk.NewDecCoins(sdk.NewInt64DecCoin("stake", 1)),
			},
			expectValid: true,
		},
		{
			name: "valid simulation mode",
			req: rest.BaseReq{
				From:     "cosmos1address",
				Simulate: true,
			},
			expectValid: true,
		},
		{
			name: "missing chain ID (non-simulation)",
			req: rest.BaseReq{
				From:    "cosmos1address",
				ChainID: "",
				Fees:    sdk.NewCoins(sdk.NewInt64Coin("stake", 100)),
			},
			expectValid: false,
			statusCode:  http.StatusBadRequest,
		},
		{
			name: "both fees and gas prices provided",
			req: rest.BaseReq{
				From:      "cosmos1address",
				ChainID:   "heimdall-1",
				Fees:      sdk.NewCoins(sdk.NewInt64Coin("stake", 100)),
				GasPrices: sdk.NewDecCoins(sdk.NewInt64DecCoin("stake", 1)),
			},
			expectValid: false,
			statusCode:  http.StatusBadRequest,
		},
		{
			name: "missing from address",
			req: rest.BaseReq{
				From:    "",
				ChainID: "heimdall-1",
				Fees:    sdk.NewCoins(sdk.NewInt64Coin("stake", 100)),
			},
			expectValid: false,
			statusCode:  http.StatusUnauthorized,
		},
		// Note: Empty fees and gas prices test removed as nil sdk.Coins are considered valid by SDK
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			valid := tt.req.ValidateBasic(w)
			require.Equal(t, tt.expectValid, valid)

			if !tt.expectValid {
				require.Equal(t, tt.statusCode, w.Code)
				require.Contains(t, w.Header().Get("Content-Type"), "application/json")
			}
		})
	}
}

func TestNewErrorResponse(t *testing.T) {
	tests := []struct {
		name string
		code int
		err  string
	}{
		{
			name: "bad request error",
			code: http.StatusBadRequest,
			err:  "invalid request",
		},
		{
			name: "unauthorized error",
			code: http.StatusUnauthorized,
			err:  "unauthorized",
		},
		{
			name: "empty error",
			code: http.StatusInternalServerError,
			err:  "",
		},
		{
			name: "zero code",
			code: 0,
			err:  "some error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := rest.NewErrorResponse(tt.code, tt.err)
			require.Equal(t, tt.code, resp.Code)
			require.Equal(t, tt.err, resp.Error)
		})
	}
}

func TestWriteErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		errMsg     string
		expectCode int
	}{
		{
			name:       "bad request",
			status:     http.StatusBadRequest,
			errMsg:     "bad request error",
			expectCode: http.StatusBadRequest,
		},
		{
			name:       "unauthorized",
			status:     http.StatusUnauthorized,
			errMsg:     "unauthorized error",
			expectCode: http.StatusUnauthorized,
		},
		{
			name:       "internal server error",
			status:     http.StatusInternalServerError,
			errMsg:     "internal error",
			expectCode: http.StatusInternalServerError,
		},
		{
			name:       "not found",
			status:     http.StatusNotFound,
			errMsg:     "not found",
			expectCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rest.WriteErrorResponse(w, tt.status, tt.errMsg)

			require.Equal(t, tt.expectCode, w.Code)
			require.Equal(t, "application/json", w.Header().Get("Content-Type"))
			require.NotEmpty(t, w.Body.String())
			require.Contains(t, w.Body.String(), tt.errMsg)
		})
	}
}

func TestBaseReqFields(t *testing.T) {
	fees := sdk.NewCoins(sdk.NewInt64Coin("stake", 100))
	gasPrices := sdk.NewDecCoins(sdk.NewInt64DecCoin("stake", 1))

	req := rest.BaseReq{
		From:          "cosmos1address",
		Memo:          "test memo",
		ChainID:       "heimdall-1",
		AccountNumber: 42,
		Sequence:      100,
		Fees:          fees,
		GasPrices:     gasPrices,
		Gas:           "200000",
		GasAdjustment: "1.5",
		Simulate:      true,
	}

	// Verify all fields are accessible
	require.Equal(t, "cosmos1address", req.From)
	require.Equal(t, "test memo", req.Memo)
	require.Equal(t, "heimdall-1", req.ChainID)
	require.Equal(t, uint64(42), req.AccountNumber)
	require.Equal(t, uint64(100), req.Sequence)
	require.Equal(t, fees, req.Fees)
	require.Equal(t, gasPrices, req.GasPrices)
	require.Equal(t, "200000", req.Gas)
	require.Equal(t, "1.5", req.GasAdjustment)
	require.True(t, req.Simulate)
}

func TestBaseReqValidateBasicSimulationMode(t *testing.T) {
	// In simulation mode, chain ID and fees are not required
	req := rest.BaseReq{
		From:     "cosmos1address",
		Simulate: true,
	}

	w := httptest.NewRecorder()
	valid := req.ValidateBasic(w)
	require.True(t, valid)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestErrorResponseFields(t *testing.T) {
	resp := rest.ErrorResponse{
		Code:  http.StatusBadRequest,
		Error: "test error",
	}

	require.Equal(t, http.StatusBadRequest, resp.Code)
	require.Equal(t, "test error", resp.Error)
}
