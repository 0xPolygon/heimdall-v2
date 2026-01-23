package helper

import (
	"errors"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
)

func TestFetchAndValidateReceipt(t *testing.T) {
	logger := log.NewNopLogger()

	tests := []struct {
		name           string
		receipt        *ethTypes.Receipt
		err            error
		params         ReceiptValidationParams
		expectedResult bool // true if the receipt should be returned (not nil)
	}{
		{
			name: "valid receipt with matching block number",
			receipt: &ethTypes.Receipt{
				BlockNumber: big.NewInt(100),
			},
			err: nil,
			params: ReceiptValidationParams{
				TxHash:         common.Hex2Bytes("0x1234"),
				MsgBlockNumber: 100,
				Confirmations:  6,
				ModuleName:     "test",
			},
			expectedResult: true,
		},
		{
			name:    "nil receipt",
			receipt: nil,
			err:     nil,
			params: ReceiptValidationParams{
				TxHash:         common.Hex2Bytes("0x1234"),
				MsgBlockNumber: 100,
				Confirmations:  6,
				ModuleName:     "test",
			},
			expectedResult: false,
		},
		{
			name:    "error fetching receipt",
			receipt: nil,
			err:     errors.New("network error"),
			params: ReceiptValidationParams{
				TxHash:         common.Hex2Bytes("0x1234"),
				MsgBlockNumber: 100,
				Confirmations:  6,
				ModuleName:     "test",
			},
			expectedResult: false,
		},
		{
			name: "block number mismatch",
			receipt: &ethTypes.Receipt{
				BlockNumber: big.NewInt(99),
			},
			err: nil,
			params: ReceiptValidationParams{
				TxHash:         common.Hex2Bytes("0x1234"),
				MsgBlockNumber: 100,
				Confirmations:  6,
				ModuleName:     "test",
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCaller := mocks.NewIContractCaller(t)

			// Set up the mock expectation for GetConfirmedTxReceipt
			txHash := common.BytesToHash(tt.params.TxHash)
			mockCaller.On("GetConfirmedTxReceipt", txHash, tt.params.Confirmations).
				Return(tt.receipt, tt.err)

			result := FetchAndValidateReceipt(mockCaller, tt.params, logger)

			if tt.expectedResult {
				require.NotNil(t, result, "expected receipt to be returned")
				require.Equal(t, tt.receipt, result)
			} else {
				require.Nil(t, result, "expected nil receipt")
			}
		})
	}
}
