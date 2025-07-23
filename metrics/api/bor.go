package api

const (
	// Query API methods.
	GetSpanListMethod                   = "GetSpanList"
	GetLatestSpanMethod                 = "GetLatestSpan"
	GetNextSpanSeedMethod               = "GetNextSpanSeed"
	GetNextSpanMethod                   = "GetNextSpan"
	GetSpanByIdMethod                   = "GetSpanById"
	GetBorParamsMethod                  = "GetBorParams"
	GetProducerVotesMethod              = "GetProducerVotes"
	GetProducerVotesByValidatorIdMethod = "GetProducerVotesByValidatorId"

	// Transaction API methods.
	ProposeSpanMethod     = "ProposeSpan"
	BorUpdateParamsMethod = "UpdateParams"
	BackfillSpansMethod   = "BackfillSpans"
	VoteProducersMethod   = "VoteProducers"
)

var (
	AllBorQueryMethods = []string{
		GetSpanListMethod,
		GetLatestSpanMethod,
		GetNextSpanSeedMethod,
		GetNextSpanMethod,
		GetSpanByIdMethod,
		GetBorParamsMethod,
		GetProducerVotesMethod,
		GetProducerVotesByValidatorIdMethod,
	}

	AllBorTransactionMethods = []string{
		ProposeSpanMethod,
		BorUpdateParamsMethod,
		BackfillSpansMethod,
		VoteProducersMethod,
	}
)

// InitBorModuleMetrics pre-registers all bor API metrics with zero values.
func InitBorModuleMetrics() {
	metrics := GetModuleMetrics(BorSubsystem)

	for _, method := range AllBorQueryMethods {
		metrics.TotalCalls.WithLabelValues(method, QueryType)
		metrics.SuccessCalls.WithLabelValues(method, QueryType)
		metrics.ResponseTime.WithLabelValues(method, QueryType)
	}

	for _, method := range AllBorTransactionMethods {
		metrics.TotalCalls.WithLabelValues(method, TxType)
		metrics.SuccessCalls.WithLabelValues(method, TxType)
		metrics.ResponseTime.WithLabelValues(method, TxType)
	}
}
