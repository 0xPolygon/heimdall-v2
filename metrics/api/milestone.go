package api

const (
	// Query API methods.
	GetMilestoneParamsMethod   = "GetMilestoneParams"
	GetMilestoneCountMethod    = "GetMilestoneCount"
	GetLatestMilestoneMethod   = "GetLatestMilestone"
	GetMilestoneByNumberMethod = "GetMilestoneByNumber"

	// Transaction API methods.
	MilestoneUpdateParamsMethod = "UpdateParams"
)

var (
	AllMilestoneQueryMethods = []string{
		GetMilestoneParamsMethod,
		GetMilestoneCountMethod,
		GetLatestMilestoneMethod,
		GetMilestoneByNumberMethod,
	}

	AllMilestoneTransactionMethods = []string{
		MilestoneUpdateParamsMethod,
	}
)
