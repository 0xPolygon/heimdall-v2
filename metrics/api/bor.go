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
