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
