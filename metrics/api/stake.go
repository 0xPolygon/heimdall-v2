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
