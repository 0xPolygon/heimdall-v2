package api

const (
	// Query API methods.
	GetCurrentValidatorSetMethod      = "GetCurrentValidatorSet"
	GetSignerByAddressMethod          = "GetSignerByAddress"
	GetValidatorByIdMethod            = "GetValidatorById"
	GetValidatorStatusByAddressMethod = "GetValidatorStatusByAddress"
	GetTotalPowerMethod               = "GetTotalPower"
	IsStakeTxOldMethod                = "IsStakeTxOld"
	GetCurrentProposerMethod          = "GetCurrentProposer"
	GetProposersByTimesMethod         = "GetProposersByTimes"

	// Transaction API methods.
	ValidatorJoinMethod = "ValidatorJoin"
	StakeUpdateMethod   = "StakeUpdate"
	SignerUpdateMethod  = "SignerUpdate"
	ValidatorExitMethod = "ValidatorExit"
)

var (
	AllStakeQueryMethods = []string{
		GetCurrentValidatorSetMethod,
		GetSignerByAddressMethod,
		GetValidatorByIdMethod,
		GetValidatorStatusByAddressMethod,
		GetTotalPowerMethod,
		IsStakeTxOldMethod,
		GetCurrentProposerMethod,
		GetProposersByTimesMethod,
	}

	AllStakeTransactionMethods = []string{
		ValidatorJoinMethod,
		StakeUpdateMethod,
		SignerUpdateMethod,
		ValidatorExitMethod,
	}
)

// InitStakeModuleMetrics pre-registers all stake API metrics with zero values.
func InitStakeModuleMetrics() {
	metrics := GetModuleMetrics(StakeSubsystem)

	for _, method := range AllStakeQueryMethods {
		metrics.TotalCalls.WithLabelValues(method, QueryType)
		metrics.SuccessCalls.WithLabelValues(method, QueryType)
		metrics.ResponseTime.WithLabelValues(method, QueryType)
	}

	for _, method := range AllStakeTransactionMethods {
		metrics.TotalCalls.WithLabelValues(method, TxType)
		metrics.SuccessCalls.WithLabelValues(method, TxType)
		metrics.ResponseTime.WithLabelValues(method, TxType)
	}
}
