package api

const (
	// Query API methods.
	GetCheckpointParamsMethod     = "GetCheckpointParams"
	GetCheckpointOverviewMethod   = "GetCheckpointOverview"
	GetAckCountMethod             = "GetAckCount"
	GetCheckpointLatestMethod     = "GetCheckpointLatest"
	GetCheckpointBufferMethod     = "GetCheckpointBuffer"
	GetLastNoAckMethod            = "GetLastNoAck"
	GetNextCheckpointMethod       = "GetNextCheckpoint"
	GetCheckpointListMethod       = "GetCheckpointList"
	GetCheckpointSignaturesMethod = "GetCheckpointSignatures"
	GetCheckpointMethod           = "GetCheckpoint"

	// Transaction API methods.
	CheckpointMethod             = "Checkpoint"
	CheckpointAckMethod          = "CheckpointAck"
	CheckpointNoAckMethod        = "CheckpointNoAck"
	CheckpointUpdateParamsMethod = "UpdateParams"
)

var (
	AllCheckpointQueryMethods = []string{
		GetCheckpointParamsMethod,
		GetCheckpointOverviewMethod,
		GetAckCountMethod,
		GetCheckpointLatestMethod,
		GetCheckpointBufferMethod,
		GetLastNoAckMethod,
		GetNextCheckpointMethod,
		GetCheckpointListMethod,
		GetCheckpointSignaturesMethod,
		GetCheckpointMethod,
	}

	AllCheckpointTransactionMethods = []string{
		CheckpointMethod,
		CheckpointAckMethod,
		CheckpointNoAckMethod,
		CheckpointUpdateParamsMethod,
	}
)
