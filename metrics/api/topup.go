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
