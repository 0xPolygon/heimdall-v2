package helper

import (
	"math/big"
	"testing"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
)

func TestValidateChainID(t *testing.T) {
	logger := log.NewNopLogger()

	tests := []struct {
		name            string
		msgChainID      string
		expectedChainID string
		moduleName      string
		expected        bool
	}{
		{
			name:            "matching chain IDs",
			msgChainID:      "80001",
			expectedChainID: "80001",
			moduleName:      "test",
			expected:        true,
		},
		{
			name:            "non-matching chain IDs",
			msgChainID:      "80001",
			expectedChainID: "137",
			moduleName:      "test",
			expected:        false,
		},
		{
			name:            "empty chain ID mismatch",
			msgChainID:      "",
			expectedChainID: "137",
			moduleName:      "test",
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateChainID(
				tt.msgChainID,
				tt.expectedChainID,
				logger,
				tt.moduleName,
			)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateVotingPower(t *testing.T) {
	logger := log.NewNopLogger()

	// 1 token = 10^18 wei
	oneToken := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	lessThanOneToken := new(big.Int).Sub(oneToken, big.NewInt(1))
	tenTokens := new(big.Int).Mul(oneToken, big.NewInt(10))

	tests := []struct {
		name        string
		amount      *big.Int
		validatorID uint64
		moduleName  string
		expected    bool
	}{
		{
			name:        "valid amount - exactly one token",
			amount:      oneToken,
			validatorID: 1,
			moduleName:  "stake",
			expected:    true,
		},
		{
			name:        "valid amount - ten tokens",
			amount:      tenTokens,
			validatorID: 2,
			moduleName:  "stake",
			expected:    true,
		},
		{
			name:        "invalid amount - less than one token",
			amount:      lessThanOneToken,
			validatorID: 3,
			moduleName:  "stake",
			expected:    false,
		},
		{
			name:        "invalid amount - zero",
			amount:      big.NewInt(0),
			validatorID: 4,
			moduleName:  "stake",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateVotingPower(
				tt.amount,
				logger,
				tt.validatorID,
				tt.moduleName,
			)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSideTxApproved(t *testing.T) {
	tests := []struct {
		name         string
		sideTxResult sidetxs.Vote
		expected     bool
	}{
		{
			name:         "approved - VOTE_YES",
			sideTxResult: sidetxs.Vote_VOTE_YES,
			expected:     true,
		},
		{
			name:         "not approved - VOTE_NO",
			sideTxResult: sidetxs.Vote_VOTE_NO,
			expected:     false,
		},
		{
			name:         "not approved - UNSPECIFIED",
			sideTxResult: sidetxs.Vote_UNSPECIFIED,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSideTxApproved(tt.sideTxResult)
			require.Equal(t, tt.expected, result)
		})
	}
}
