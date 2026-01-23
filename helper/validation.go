package helper

import (
	"math/big"

	"cosmossdk.io/log"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
)

// ValidateChainID checks if the message chain ID matches the expected Bor chain ID.
// Returns true if the chain ID is valid, false otherwise.
// This is used to ensure messages are from the correct blockchain network.
func ValidateChainID(
	msgChainID string,
	expectedChainID string,
	logger log.Logger,
	moduleName string,
) bool {
	if expectedChainID != msgChainID {
		logger.Error("Invalid chain ID",
			"module", moduleName,
			"expected", expectedChainID,
			"received", msgChainID)
		return false
	}

	return true
}

// ValidateVotingPower validates that the given amount can produce valid voting power.
// This is typically used in stake-related operations to ensure the staked amount
// is enough to generate meaningful voting power.
// Returns true if the amount produces valid voting power, false otherwise.
func ValidateVotingPower(
	amount *big.Int,
	logger log.Logger,
	validatorID uint64,
	moduleName string,
) bool {
	_, err := GetPowerFromAmount(amount)
	if err != nil {
		logger.Error("Invalid voting power from amount",
			"module", moduleName,
			"amount", amount,
			"validatorId", validatorID,
			"error", err)
		return false
	}

	return true
}

// IsSideTxApproved checks if a side transaction received YES votes.
// This is a simple helper to make post-handler code more readable.
// Returns true if the side transaction was approved (VOTE_YES).
func IsSideTxApproved(sideTxResult sidetxs.Vote) bool {
	return sideTxResult == sidetxs.Vote_VOTE_YES
}
