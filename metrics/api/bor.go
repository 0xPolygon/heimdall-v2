package api

import (
	"time"
)

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
	ProposeSpanMethod   = "ProposeSpan"
	UpdateParamsMethod  = "UpdateParams"
	VoteProducersMethod = "VoteProducers"
	BackfillSpansMethod = "BackfillSpans"
)

// RecordBorAPI is the single generic function for all Bor module API calls.
func RecordBorAPI(method, apiType string, success bool, start time.Time) {
	RecordAPICallWithStart(BorSubsystem, method, apiType, success, start)
}

// RecordBorQuery records a Bor query API call.
func RecordBorQuery(method string, success bool, start time.Time) {
	RecordBorAPI(method, QueryType, success, start)
}

// RecordBorTransaction records a Bor transaction API call.
func RecordBorTransaction(method string, success bool, start time.Time) {
	RecordBorAPI(method, TxType, success, start)
}
