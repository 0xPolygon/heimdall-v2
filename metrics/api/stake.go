package api

const (
	// Query API methods.
	GetCurrentValidatorSetMethod      = "GetCurrentValidatorSet"
	GetSignerByAddressMethod          = "GetSignerByAddress"
	GetValidatorByIdMethod            = "GetValidatorById"
	GetValidatorStatusByAddressMethod = "GetValidatorStatusByAddress"
	GetTotalPowerMethod               = "GetTotalPower"
	IsStakeTxOldMethod                = "IsStakeTxOld"
	GetCurrentProposerMethod          = "GetCurrentProposer"
	GetProposersByTimesMethod         = "GetProposersByTimes"

	// Transaction API methods.
	ValidatorJoinMethod = "ValidatorJoin"
	StakeUpdateMethod   = "StakeUpdate"
	SignerUpdateMethod  = "SignerUpdate"
	ValidatorExitMethod = "ValidatorExit"
)

var (
	AllStakeQueryMethods = []string{
		GetCurrentValidatorSetMethod,
		GetSignerByAddressMethod,
		GetValidatorByIdMethod,
		GetValidatorStatusByAddressMethod,
		GetTotalPowerMethod,
		IsStakeTxOldMethod,
		GetCurrentProposerMethod,
		GetProposersByTimesMethod,
	}

	AllStakeTransactionMethods = []string{
		ValidatorJoinMethod,
		StakeUpdateMethod,
		SignerUpdateMethod,
		ValidatorExitMethod,
	}
)
