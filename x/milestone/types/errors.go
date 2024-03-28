package types

import "cosmossdk.io/errors"

// x/milestone module sentinel errors
var (
	ErrNoMilestoneFound         = errors.Register(ModuleName, 2, "milestone not found")
	ErrMilestoneNotInContinuity = errors.Register(ModuleName, 3, "milestone not in continuity")
	ErrMilestoneInvalid         = errors.Register(ModuleName, 4, "milestone msg invalid")
	ErrOldMilestone             = errors.Register(ModuleName, 5, "milestone already exists")
	ErrInvalidMilestoneTimeout  = errors.Register(ModuleName, 6, "invalid milestone timeout msg")
	ErrTooManyMilestoneTimeout  = errors.Register(ModuleName, 7, "too many milestone timeout msg")
	ErrInvalidMilestoneIndex    = errors.Register(ModuleName, 8, "invalid milestone index")
	ErrPrevMilestoneInVoting    = errors.Register(ModuleName, 9, "previous milestone still in voting phase")
	ErrMilestoneParams          = errors.Register(ModuleName, 10, "error in fetching milestone params")
	ErrProposerNotFound         = errors.Register(ModuleName, 11, "milestone proposer not found")
	ErrProposerMismatch         = errors.Register(ModuleName, 12, "milestone and msg proposer mismatch")
)
