package api

const (
	// Query API methods.
	GetMilestoneParamsMethod   = "GetMilestoneParams"
	GetMilestoneCountMethod    = "GetMilestoneCount"
	GetLatestMilestoneMethod   = "GetLatestMilestone"
	GetMilestoneByNumberMethod = "GetMilestoneByNumber"

	// Transaction API methods.
	MilestoneUpdateParamsMethod = "UpdateParams"
)

var (
	AllMilestoneQueryMethods = []string{
		GetMilestoneParamsMethod,
		GetMilestoneCountMethod,
		GetLatestMilestoneMethod,
		GetMilestoneByNumberMethod,
	}

	AllMilestoneTransactionMethods = []string{
		MilestoneUpdateParamsMethod,
	}
)

// InitMilestoneModuleMetrics pre-registers all milestone API metrics with zero values.
func InitMilestoneModuleMetrics() {
	metrics := GetModuleMetrics(MilestoneSubsystem)

	for _, method := range AllMilestoneQueryMethods {
		metrics.TotalCalls.WithLabelValues(method, QueryType)
		metrics.SuccessCalls.WithLabelValues(method, QueryType)
		metrics.ResponseTime.WithLabelValues(method, QueryType)
	}

	for _, method := range AllMilestoneTransactionMethods {
		metrics.TotalCalls.WithLabelValues(method, TxType)
		metrics.SuccessCalls.WithLabelValues(method, TxType)
		metrics.ResponseTime.WithLabelValues(method, TxType)
	}
}
