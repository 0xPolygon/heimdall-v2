// Code generated by MockGen. DO NOT EDIT.
// Source: x/stake/types/expected_keepers.go

// Package testutil is a generated GoMock package.
package testutil

import (
	context "context"
	reflect "reflect"

	types "github.com/cosmos/cosmos-sdk/types"
	gomock "github.com/golang/mock/gomock"
)

// MockCheckpointKeeper is a mock of CheckpointKeeper interface.
type MockCheckpointKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockCheckpointKeeperMockRecorder
}

// MockCheckpointKeeperMockRecorder is the mock recorder for MockCheckpointKeeper.
type MockCheckpointKeeperMockRecorder struct {
	mock *MockCheckpointKeeper
}

// NewMockCheckpointKeeper creates a new mock instance.
func NewMockCheckpointKeeper(ctrl *gomock.Controller) *MockCheckpointKeeper {
	mock := &MockCheckpointKeeper{ctrl: ctrl}
	mock.recorder = &MockCheckpointKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCheckpointKeeper) EXPECT() *MockCheckpointKeeperMockRecorder {
	return m.recorder
}

// GetACKCount mocks base method.
func (m *MockCheckpointKeeper) GetACKCount(ctx context.Context) uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetACKCount", ctx)
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetACKCount indicates an expected call of GetACKCount.
func (mr *MockCheckpointKeeperMockRecorder) GetACKCount(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetACKCount", reflect.TypeOf((*MockCheckpointKeeper)(nil).GetACKCount), ctx)
}

// MockBankKeeper is a mock of BankKeeper interface.
type MockBankKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockBankKeeperMockRecorder
}

// MockBankKeeperMockRecorder is the mock recorder for MockBankKeeper.
type MockBankKeeperMockRecorder struct {
	mock *MockBankKeeper
}

// NewMockBankKeeper creates a new mock instance.
func NewMockBankKeeper(ctrl *gomock.Controller) *MockBankKeeper {
	mock := &MockBankKeeper{ctrl: ctrl}
	mock.recorder = &MockBankKeeperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBankKeeper) EXPECT() *MockBankKeeperMockRecorder {
	return m.recorder
}

// GetBalance mocks base method.
func (m *MockBankKeeper) GetBalance(ctx context.Context, addr types.AccAddress, denom string) types.Coin {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBalance", ctx, addr, denom)
	ret0, _ := ret[0].(types.Coin)
	return ret0
}

// GetBalance indicates an expected call of GetBalance.
func (mr *MockBankKeeperMockRecorder) GetBalance(ctx, addr, denom interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBalance", reflect.TypeOf((*MockBankKeeper)(nil).GetBalance), ctx, addr, denom)
}

// SendCoins mocks base method.
func (m *MockBankKeeper) SendCoins(ctx context.Context, fromAddr, toAddr types.AccAddress, amt types.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendCoins", ctx, fromAddr, toAddr, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendCoins indicates an expected call of SendCoins.
func (mr *MockBankKeeperMockRecorder) SendCoins(ctx, fromAddr, toAddr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoins", reflect.TypeOf((*MockBankKeeper)(nil).SendCoins), ctx, fromAddr, toAddr, amt)
}
