package api

const (
	// Query API methods.

	GetSpanListMethod                               = "GetSpanList"
	GetLatestSpanMethod                             = "GetLatestSpan"
	GetNextSpanSeedMethod                           = "GetNextSpanSeed"
	GetNextSpanMethod                               = "GetNextSpan"
	GetSpanByIdMethod                               = "GetSpanById"
	GetBorParamsMethod                              = "GetBorParams"
	GetProducerVotesMethod                          = "GetProducerVotes"
	GetProducerVotesByValidatorIdMethod             = "GetProducerVotesByValidatorId"
	GetProducerPlannedDowntimeMethod                = "GetProducerPlannedDowntime"
	GetValidatorPerformanceScoreMethod              = "GetValidatorPerformanceScore"
	GetValidatorPerformanceScoreByValidatorIdMethod = "GetValidatorPerformanceScoreByValidatorId"

	// Transaction API methods.

	ProposeSpanMethod      = "ProposeSpan"
	BorUpdateParamsMethod  = "UpdateParams"
	BackfillSpansMethod    = "BackfillSpans"
	VoteProducersMethod    = "VoteProducers"
	ProducerDowntimeMethod = "ProducerDowntime"

	// Side message handler methods.

	SideHandleMsgSpanMethod                = "SideHandleMsgSpan"
	SideHandleMsgSetProducerDowntimeMethod = "SideHandleMsgSetProducerDowntime"

	// Post message handler methods.

	PostHandleMsgSpanMethod                = "PostHandleMsgSpan"
	PostHandleMsgBackfillSpansMethod       = "PostHandleMsgBackfillSpans"
	PostHandleMsgSetProducerDowntimeMethod = "PostHandleMsgSetProducerDowntime"
)
