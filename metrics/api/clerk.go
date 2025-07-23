package api

const (
	// Query API methods.
	GetRecordCountMethod        = "GetRecordCount"
	GetRecordListMethod         = "GetRecordList"
	GetLatestRecordIdMethod     = "GetLatestRecordId"
	GetRecordByIdMethod         = "GetRecordById"
	GetRecordListWithTimeMethod = "GetRecordListWithTime"
	GetRecordSequenceMethod     = "GetRecordSequence"
	IsClerkTxOldMethod          = "IsClerkTxOld"

	// Transaction API methods.
	HandleMsgEventRecordMethod = "HandleMsgEventRecord"
)

var (
	AllClerkQueryMethods = []string{
		GetRecordCountMethod,
		GetRecordListMethod,
		GetLatestRecordIdMethod,
		GetRecordByIdMethod,
		GetRecordListWithTimeMethod,
		GetRecordSequenceMethod,
		IsClerkTxOldMethod,
	}

	AllClerkTransactionMethods = []string{
		HandleMsgEventRecordMethod,
	}
)

// InitClerkModuleMetrics pre-registers all clerk API metrics with zero values.
func InitClerkModuleMetrics() {
	metrics := GetModuleMetrics(ClerkSubsystem)

	for _, method := range AllClerkQueryMethods {
		metrics.TotalCalls.WithLabelValues(method, QueryType)
		metrics.SuccessCalls.WithLabelValues(method, QueryType)
		metrics.ResponseTime.WithLabelValues(method, QueryType)
	}

	for _, method := range AllClerkTransactionMethods {
		metrics.TotalCalls.WithLabelValues(method, TxType)
		metrics.SuccessCalls.WithLabelValues(method, TxType)
		metrics.ResponseTime.WithLabelValues(method, TxType)
	}
}
