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

// InitCheckpointModuleMetrics pre-registers all checkpoint API metrics with zero values.
func InitCheckpointModuleMetrics() {
	metrics := GetModuleMetrics(CheckpointSubsystem)

	for _, method := range AllCheckpointQueryMethods {
		metrics.TotalCalls.WithLabelValues(method, QueryType)
		metrics.SuccessCalls.WithLabelValues(method, QueryType)
		metrics.ResponseTime.WithLabelValues(method, QueryType)
	}

	for _, method := range AllCheckpointTransactionMethods {
		metrics.TotalCalls.WithLabelValues(method, TxType)
		metrics.SuccessCalls.WithLabelValues(method, TxType)
		metrics.ResponseTime.WithLabelValues(method, TxType)
	}
}
