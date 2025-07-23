package api

import (
	"time"
)

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
	CpAckMethod                  = "CpAck"
	CpNoAckMethod                = "CpNoAck"
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
		CpAckMethod,
		CpNoAckMethod,
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

// RecordCheckpointAPI is the single generic function for all Checkpoint module API calls.
func RecordCheckpointAPI(method, apiType string, success bool, start time.Time) {
	RecordAPICallWithStart(CheckpointSubsystem, method, apiType, success, start)
}

// RecordCheckpointQuery records a Checkpoint query API call.
func RecordCheckpointQuery(method string, success bool, start time.Time) {
	RecordCheckpointAPI(method, QueryType, success, start)
}

// RecordCheckpointTransaction records a Checkpoint transaction API call.
func RecordCheckpointTransaction(method string, success bool, start time.Time) {
	RecordCheckpointAPI(method, TxType, success, start)
}
