package types

// Event Hooks
// These can be utilized to communicate between a staking keeper and another
// keeper which must take particular actions when validators/delegators change
// state. The second keeper must implement this interface, which then the
// staking keeper can call.

// TODO H2 Define the interface so that things won't break when any change in keeper function
// is done
type ValidatorSet interface {
}

// StakingHooks event hooks for staking validator object (noalias)
type StakingHooks interface {
}

// StakingHooksWrapper is a wrapper for modules to inject StakingHooks using depinject.
type StakingHooksWrapper struct{ StakingHooks }

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (StakingHooksWrapper) IsOnePerModuleType() {}
