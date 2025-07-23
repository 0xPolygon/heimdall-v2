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
