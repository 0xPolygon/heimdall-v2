package api

const (
	// Query API methods.
	IsTopupTxOldMethod                = "IsTopupTxOld"
	GetTopupTxSequenceMethod          = "GetTopupTxSequence"
	GetDividendAccountByAddressMethod = "GetDividendAccountByAddress"
	GetDividendAccountRootHashMethod  = "GetDividendAccountRootHash"
	VerifyAccountProofByAddressMethod = "VerifyAccountProofByAddress"
	GetAccountProofByAddressMethod    = "GetAccountProofByAddress"

	// Transaction API methods.
	HandleTopupTxMethod = "HandleTopupTx"
	WithdrawFeeTxMethod = "WithdrawFeeTx"
)

var (
	AllTopupQueryMethods = []string{
		IsTopupTxOldMethod,
		GetTopupTxSequenceMethod,
		GetDividendAccountByAddressMethod,
		GetDividendAccountRootHashMethod,
		VerifyAccountProofByAddressMethod,
		GetAccountProofByAddressMethod,
	}

	AllTopupTransactionMethods = []string{
		HandleTopupTxMethod,
		WithdrawFeeTxMethod,
	}
)

// InitTopupModuleMetrics pre-registers all topup API metrics with zero values.
func InitTopupModuleMetrics() {
	metrics := GetModuleMetrics(TopupSubsystem)

	for _, method := range AllTopupQueryMethods {
		metrics.TotalCalls.WithLabelValues(method, QueryType)
		metrics.SuccessCalls.WithLabelValues(method, QueryType)
		metrics.ResponseTime.WithLabelValues(method, QueryType)
	}

	for _, method := range AllTopupTransactionMethods {
		metrics.TotalCalls.WithLabelValues(method, TxType)
		metrics.SuccessCalls.WithLabelValues(method, TxType)
		metrics.ResponseTime.WithLabelValues(method, TxType)
	}
}
