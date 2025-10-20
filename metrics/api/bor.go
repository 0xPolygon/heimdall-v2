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
	GetProducerPlannedDowntimeMethod    = "GetProducerPlannedDowntime"

	// Transaction API methods.
	ProposeSpanMethod     = "ProposeSpan"
	BorUpdateParamsMethod = "UpdateParams"
	BackfillSpansMethod   = "BackfillSpans"
	VoteProducersMethod   = "VoteProducers"

	// Side message handler methods.
	SideHandleMsgSpanMethod          = "SideHandleMsgSpan"
	SideHandleMsgBackfillSpansMethod = "SideHandleMsgBackfillSpans"
	SideHandleMsgSetProducerDowntime = "SideHandleMsgSetProducerDowntime"

	// Post message handler methods.
	PostHandleMsgSpanMethod                = "PostHandleMsgSpan"
	PostHandleMsgBackfillSpansMethod       = "PostHandleMsgBackfillSpans"
	PostHandleMsgSetProducerDowntimeMethod = "PostHandleMsgSetProducerDowntime"
)
