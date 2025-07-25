// Code generated by MockGen. DO NOT EDIT.
// Source: x/bor/types/expected_keepers.go

// Package testutil is a generated GoMock package.
package testutil

import (
	context "context"
	reflect "reflect"

	types "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	types0 "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	types1 "github.com/0xPolygon/heimdall-v2/x/stake/types"
	gomock "github.com/golang/mock/gomock"
)

// MockStakeKeeper is a mock of StakeKeeper interface.
type MockStakeKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockStakeKeeperMockRecorder
}

// MockStakeKeeperMockRecorder is the mock recorder for MockStakeKeeper.
type MockStakeKeeperMockRecorder struct {
	mock *MockStakeKeeper
}

// NewMockStakeKeeper creates a new mock instance.
func NewMockStakeKeeper(ctrl *gomock.Controller) *MockStakeKeeper {
	mock := &MockStakeKeeper{ctrl: ctrl}
	mock.recorder = &MockStakeKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStakeKeeper) EXPECT() *MockStakeKeeperMockRecorder {
	return m.recorder
}

// GetSpanEligibleValidators mocks base method.
func (m *MockStakeKeeper) GetSpanEligibleValidators(ctx context.Context) []types1.Validator {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSpanEligibleValidators", ctx)
	ret0, _ := ret[0].([]types1.Validator)
	return ret0
}

// GetSpanEligibleValidators indicates an expected call of GetSpanEligibleValidators.
func (mr *MockStakeKeeperMockRecorder) GetSpanEligibleValidators(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSpanEligibleValidators", reflect.TypeOf((*MockStakeKeeper)(nil).GetSpanEligibleValidators), ctx)
}

// GetValidatorFromValID mocks base method.
func (m *MockStakeKeeper) GetValidatorFromValID(ctx context.Context, valID uint64) (types1.Validator, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetValidatorFromValID", ctx, valID)
	ret0, _ := ret[0].(types1.Validator)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetValidatorFromValID indicates an expected call of GetValidatorFromValID.
func (mr *MockStakeKeeperMockRecorder) GetValidatorFromValID(ctx, valID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValidatorFromValID", reflect.TypeOf((*MockStakeKeeper)(nil).GetValidatorFromValID), ctx, valID)
}

// GetValidatorSet mocks base method.
func (m *MockStakeKeeper) GetValidatorSet(ctx context.Context) (types1.ValidatorSet, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetValidatorSet", ctx)
	ret0, _ := ret[0].(types1.ValidatorSet)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetValidatorSet indicates an expected call of GetValidatorSet.
func (mr *MockStakeKeeperMockRecorder) GetValidatorSet(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValidatorSet", reflect.TypeOf((*MockStakeKeeper)(nil).GetValidatorSet), ctx)
}

// MockChainManagerKeeper is a mock of ChainManagerKeeper interface.
type MockChainManagerKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockChainManagerKeeperMockRecorder
}

// MockChainManagerKeeperMockRecorder is the mock recorder for MockChainManagerKeeper.
type MockChainManagerKeeperMockRecorder struct {
	mock *MockChainManagerKeeper
}

// NewMockChainManagerKeeper creates a new mock instance.
func NewMockChainManagerKeeper(ctrl *gomock.Controller) *MockChainManagerKeeper {
	mock := &MockChainManagerKeeper{ctrl: ctrl}
	mock.recorder = &MockChainManagerKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChainManagerKeeper) EXPECT() *MockChainManagerKeeperMockRecorder {
	return m.recorder
}

// GetParams mocks base method.
func (m *MockChainManagerKeeper) GetParams(ctx context.Context) (types.Params, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetParams", ctx)
	ret0, _ := ret[0].(types.Params)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetParams indicates an expected call of GetParams.
func (mr *MockChainManagerKeeperMockRecorder) GetParams(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetParams", reflect.TypeOf((*MockChainManagerKeeper)(nil).GetParams), ctx)
}

// MockMilestoneKeeper is a mock of MilestoneKeeper interface.
type MockMilestoneKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockMilestoneKeeperMockRecorder
}

// MockMilestoneKeeperMockRecorder is the mock recorder for MockMilestoneKeeper.
type MockMilestoneKeeperMockRecorder struct {
	mock *MockMilestoneKeeper
}

// NewMockMilestoneKeeper creates a new mock instance.
func NewMockMilestoneKeeper(ctrl *gomock.Controller) *MockMilestoneKeeper {
	mock := &MockMilestoneKeeper{ctrl: ctrl}
	mock.recorder = &MockMilestoneKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMilestoneKeeper) EXPECT() *MockMilestoneKeeperMockRecorder {
	return m.recorder
}

// GetLastMilestone mocks base method.
func (m *MockMilestoneKeeper) GetLastMilestone(ctx context.Context) (*types0.Milestone, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastMilestone", ctx)
	ret0, _ := ret[0].(*types0.Milestone)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLastMilestone indicates an expected call of GetLastMilestone.
func (mr *MockMilestoneKeeperMockRecorder) GetLastMilestone(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastMilestone", reflect.TypeOf((*MockMilestoneKeeper)(nil).GetLastMilestone), ctx)
}
