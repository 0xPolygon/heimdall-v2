// Code generated by MockGen. DO NOT EDIT.
// Source: x/topup/types/expected_keepers.go

// Package testutil is a generated GoMock package.
package testutil

import (
	context "context"
	reflect "reflect"

	types "github.com/cosmos/cosmos-sdk/types"
	query "github.com/cosmos/cosmos-sdk/types/query"
	keeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	types0 "github.com/cosmos/cosmos-sdk/x/bank/types"
	gomock "github.com/golang/mock/gomock"
)

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

// AllBalances mocks base method.
func (m *MockBankKeeper) AllBalances(arg0 context.Context, arg1 *types0.QueryAllBalancesRequest) (*types0.QueryAllBalancesResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AllBalances", arg0, arg1)
	ret0, _ := ret[0].(*types0.QueryAllBalancesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AllBalances indicates an expected call of AllBalances.
func (mr *MockBankKeeperMockRecorder) AllBalances(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AllBalances", reflect.TypeOf((*MockBankKeeper)(nil).AllBalances), arg0, arg1)
}

// AppendSendRestriction mocks base method.
func (m *MockBankKeeper) AppendSendRestriction(restriction types0.SendRestrictionFn) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AppendSendRestriction", restriction)
}

// AppendSendRestriction indicates an expected call of AppendSendRestriction.
func (mr *MockBankKeeperMockRecorder) AppendSendRestriction(restriction interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AppendSendRestriction", reflect.TypeOf((*MockBankKeeper)(nil).AppendSendRestriction), restriction)
}

// Balance mocks base method.
func (m *MockBankKeeper) Balance(arg0 context.Context, arg1 *types0.QueryBalanceRequest) (*types0.QueryBalanceResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Balance", arg0, arg1)
	ret0, _ := ret[0].(*types0.QueryBalanceResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Balance indicates an expected call of Balance.
func (mr *MockBankKeeperMockRecorder) Balance(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Balance", reflect.TypeOf((*MockBankKeeper)(nil).Balance), arg0, arg1)
}

// BlockedAddr mocks base method.
func (m *MockBankKeeper) BlockedAddr(addr types.AccAddress) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BlockedAddr", addr)
	ret0, _ := ret[0].(bool)
	return ret0
}

// BlockedAddr indicates an expected call of BlockedAddr.
func (mr *MockBankKeeperMockRecorder) BlockedAddr(addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BlockedAddr", reflect.TypeOf((*MockBankKeeper)(nil).BlockedAddr), addr)
}

// BurnCoins mocks base method.
func (m *MockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt types.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BurnCoins", ctx, moduleName, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// BurnCoins indicates an expected call of BurnCoins.
func (mr *MockBankKeeperMockRecorder) BurnCoins(ctx, moduleName, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BurnCoins", reflect.TypeOf((*MockBankKeeper)(nil).BurnCoins), ctx, moduleName, amt)
}

// ClearSendRestriction mocks base method.
func (m *MockBankKeeper) ClearSendRestriction() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ClearSendRestriction")
}

// ClearSendRestriction indicates an expected call of ClearSendRestriction.
func (mr *MockBankKeeperMockRecorder) ClearSendRestriction() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ClearSendRestriction", reflect.TypeOf((*MockBankKeeper)(nil).ClearSendRestriction))
}

// DelegateCoins mocks base method.
func (m *MockBankKeeper) DelegateCoins(ctx context.Context, delegatorAddr, moduleAccAddr types.AccAddress, amt types.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DelegateCoins", ctx, delegatorAddr, moduleAccAddr, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// DelegateCoins indicates an expected call of DelegateCoins.
func (mr *MockBankKeeperMockRecorder) DelegateCoins(ctx, delegatorAddr, moduleAccAddr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DelegateCoins", reflect.TypeOf((*MockBankKeeper)(nil).DelegateCoins), ctx, delegatorAddr, moduleAccAddr, amt)
}

// DelegateCoinsFromAccountToModule mocks base method.
func (m *MockBankKeeper) DelegateCoinsFromAccountToModule(ctx context.Context, senderAddr types.AccAddress, recipientModule string, amt types.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DelegateCoinsFromAccountToModule", ctx, senderAddr, recipientModule, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// DelegateCoinsFromAccountToModule indicates an expected call of DelegateCoinsFromAccountToModule.
func (mr *MockBankKeeperMockRecorder) DelegateCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DelegateCoinsFromAccountToModule", reflect.TypeOf((*MockBankKeeper)(nil).DelegateCoinsFromAccountToModule), ctx, senderAddr, recipientModule, amt)
}

// DeleteSendEnabled mocks base method.
func (m *MockBankKeeper) DeleteSendEnabled(ctx context.Context, denoms ...string) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx}
	for _, a := range denoms {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "DeleteSendEnabled", varargs...)
}

// DeleteSendEnabled indicates an expected call of DeleteSendEnabled.
func (mr *MockBankKeeperMockRecorder) DeleteSendEnabled(ctx interface{}, denoms ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx}, denoms...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSendEnabled", reflect.TypeOf((*MockBankKeeper)(nil).DeleteSendEnabled), varargs...)
}

// DenomMetadata mocks base method.
func (m *MockBankKeeper) DenomMetadata(arg0 context.Context, arg1 *types0.QueryDenomMetadataRequest) (*types0.QueryDenomMetadataResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DenomMetadata", arg0, arg1)
	ret0, _ := ret[0].(*types0.QueryDenomMetadataResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DenomMetadata indicates an expected call of DenomMetadata.
func (mr *MockBankKeeperMockRecorder) DenomMetadata(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DenomMetadata", reflect.TypeOf((*MockBankKeeper)(nil).DenomMetadata), arg0, arg1)
}

// DenomMetadataByQueryString mocks base method.
func (m *MockBankKeeper) DenomMetadataByQueryString(arg0 context.Context, arg1 *types0.QueryDenomMetadataByQueryStringRequest) (*types0.QueryDenomMetadataByQueryStringResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DenomMetadataByQueryString", arg0, arg1)
	ret0, _ := ret[0].(*types0.QueryDenomMetadataByQueryStringResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DenomMetadataByQueryString indicates an expected call of DenomMetadataByQueryString.
func (mr *MockBankKeeperMockRecorder) DenomMetadataByQueryString(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DenomMetadataByQueryString", reflect.TypeOf((*MockBankKeeper)(nil).DenomMetadataByQueryString), arg0, arg1)
}

// DenomOwners mocks base method.
func (m *MockBankKeeper) DenomOwners(arg0 context.Context, arg1 *types0.QueryDenomOwnersRequest) (*types0.QueryDenomOwnersResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DenomOwners", arg0, arg1)
	ret0, _ := ret[0].(*types0.QueryDenomOwnersResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DenomOwners indicates an expected call of DenomOwners.
func (mr *MockBankKeeperMockRecorder) DenomOwners(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DenomOwners", reflect.TypeOf((*MockBankKeeper)(nil).DenomOwners), arg0, arg1)
}

// DenomsMetadata mocks base method.
func (m *MockBankKeeper) DenomsMetadata(arg0 context.Context, arg1 *types0.QueryDenomsMetadataRequest) (*types0.QueryDenomsMetadataResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DenomsMetadata", arg0, arg1)
	ret0, _ := ret[0].(*types0.QueryDenomsMetadataResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DenomsMetadata indicates an expected call of DenomsMetadata.
func (mr *MockBankKeeperMockRecorder) DenomsMetadata(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DenomsMetadata", reflect.TypeOf((*MockBankKeeper)(nil).DenomsMetadata), arg0, arg1)
}

// ExportGenesis mocks base method.
func (m *MockBankKeeper) ExportGenesis(arg0 context.Context) *types0.GenesisState {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExportGenesis", arg0)
	ret0, _ := ret[0].(*types0.GenesisState)
	return ret0
}

// ExportGenesis indicates an expected call of ExportGenesis.
func (mr *MockBankKeeperMockRecorder) ExportGenesis(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExportGenesis", reflect.TypeOf((*MockBankKeeper)(nil).ExportGenesis), arg0)
}

// GetAccountsBalances mocks base method.
func (m *MockBankKeeper) GetAccountsBalances(ctx context.Context) []types0.Balance {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAccountsBalances", ctx)
	ret0, _ := ret[0].([]types0.Balance)
	return ret0
}

// GetAccountsBalances indicates an expected call of GetAccountsBalances.
func (mr *MockBankKeeperMockRecorder) GetAccountsBalances(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccountsBalances", reflect.TypeOf((*MockBankKeeper)(nil).GetAccountsBalances), ctx)
}

// GetAllBalances mocks base method.
func (m *MockBankKeeper) GetAllBalances(ctx context.Context, addr types.AccAddress) types.Coins {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllBalances", ctx, addr)
	ret0, _ := ret[0].(types.Coins)
	return ret0
}

// GetAllBalances indicates an expected call of GetAllBalances.
func (mr *MockBankKeeperMockRecorder) GetAllBalances(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllBalances", reflect.TypeOf((*MockBankKeeper)(nil).GetAllBalances), ctx, addr)
}

// GetAllDenomMetaData mocks base method.
func (m *MockBankKeeper) GetAllDenomMetaData(ctx context.Context) []types0.Metadata {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllDenomMetaData", ctx)
	ret0, _ := ret[0].([]types0.Metadata)
	return ret0
}

// GetAllDenomMetaData indicates an expected call of GetAllDenomMetaData.
func (mr *MockBankKeeperMockRecorder) GetAllDenomMetaData(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllDenomMetaData", reflect.TypeOf((*MockBankKeeper)(nil).GetAllDenomMetaData), ctx)
}

// GetAllSendEnabledEntries mocks base method.
func (m *MockBankKeeper) GetAllSendEnabledEntries(ctx context.Context) []types0.SendEnabled {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllSendEnabledEntries", ctx)
	ret0, _ := ret[0].([]types0.SendEnabled)
	return ret0
}

// GetAllSendEnabledEntries indicates an expected call of GetAllSendEnabledEntries.
func (mr *MockBankKeeperMockRecorder) GetAllSendEnabledEntries(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllSendEnabledEntries", reflect.TypeOf((*MockBankKeeper)(nil).GetAllSendEnabledEntries), ctx)
}

// GetAuthority mocks base method.
func (m *MockBankKeeper) GetAuthority() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAuthority")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAuthority indicates an expected call of GetAuthority.
func (mr *MockBankKeeperMockRecorder) GetAuthority() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAuthority", reflect.TypeOf((*MockBankKeeper)(nil).GetAuthority))
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

// GetBlockedAddresses mocks base method.
func (m *MockBankKeeper) GetBlockedAddresses() map[string]bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockedAddresses")
	ret0, _ := ret[0].(map[string]bool)
	return ret0
}

// GetBlockedAddresses indicates an expected call of GetBlockedAddresses.
func (mr *MockBankKeeperMockRecorder) GetBlockedAddresses() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockedAddresses", reflect.TypeOf((*MockBankKeeper)(nil).GetBlockedAddresses))
}

// GetDenomMetaData mocks base method.
func (m *MockBankKeeper) GetDenomMetaData(ctx context.Context, denom string) (types0.Metadata, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDenomMetaData", ctx, denom)
	ret0, _ := ret[0].(types0.Metadata)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// GetDenomMetaData indicates an expected call of GetDenomMetaData.
func (mr *MockBankKeeperMockRecorder) GetDenomMetaData(ctx, denom interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDenomMetaData", reflect.TypeOf((*MockBankKeeper)(nil).GetDenomMetaData), ctx, denom)
}

// GetPaginatedTotalSupply mocks base method.
func (m *MockBankKeeper) GetPaginatedTotalSupply(ctx context.Context, pagination *query.PageRequest) (types.Coins, *query.PageResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPaginatedTotalSupply", ctx, pagination)
	ret0, _ := ret[0].(types.Coins)
	ret1, _ := ret[1].(*query.PageResponse)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetPaginatedTotalSupply indicates an expected call of GetPaginatedTotalSupply.
func (mr *MockBankKeeperMockRecorder) GetPaginatedTotalSupply(ctx, pagination interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPaginatedTotalSupply", reflect.TypeOf((*MockBankKeeper)(nil).GetPaginatedTotalSupply), ctx, pagination)
}

// GetParams mocks base method.
func (m *MockBankKeeper) GetParams(ctx context.Context) types0.Params {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetParams", ctx)
	ret0, _ := ret[0].(types0.Params)
	return ret0
}

// GetParams indicates an expected call of GetParams.
func (mr *MockBankKeeperMockRecorder) GetParams(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetParams", reflect.TypeOf((*MockBankKeeper)(nil).GetParams), ctx)
}

// GetSendEnabledEntry mocks base method.
func (m *MockBankKeeper) GetSendEnabledEntry(ctx context.Context, denom string) (types0.SendEnabled, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSendEnabledEntry", ctx, denom)
	ret0, _ := ret[0].(types0.SendEnabled)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// GetSendEnabledEntry indicates an expected call of GetSendEnabledEntry.
func (mr *MockBankKeeperMockRecorder) GetSendEnabledEntry(ctx, denom interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSendEnabledEntry", reflect.TypeOf((*MockBankKeeper)(nil).GetSendEnabledEntry), ctx, denom)
}

// GetSupply mocks base method.
func (m *MockBankKeeper) GetSupply(ctx context.Context, denom string) types.Coin {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSupply", ctx, denom)
	ret0, _ := ret[0].(types.Coin)
	return ret0
}

// GetSupply indicates an expected call of GetSupply.
func (mr *MockBankKeeperMockRecorder) GetSupply(ctx, denom interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSupply", reflect.TypeOf((*MockBankKeeper)(nil).GetSupply), ctx, denom)
}

// HasBalance mocks base method.
func (m *MockBankKeeper) HasBalance(ctx context.Context, addr types.AccAddress, amt types.Coin) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasBalance", ctx, addr, amt)
	ret0, _ := ret[0].(bool)
	return ret0
}

// HasBalance indicates an expected call of HasBalance.
func (mr *MockBankKeeperMockRecorder) HasBalance(ctx, addr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasBalance", reflect.TypeOf((*MockBankKeeper)(nil).HasBalance), ctx, addr, amt)
}

// HasDenomMetaData mocks base method.
func (m *MockBankKeeper) HasDenomMetaData(ctx context.Context, denom string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasDenomMetaData", ctx, denom)
	ret0, _ := ret[0].(bool)
	return ret0
}

// HasDenomMetaData indicates an expected call of HasDenomMetaData.
func (mr *MockBankKeeperMockRecorder) HasDenomMetaData(ctx, denom interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasDenomMetaData", reflect.TypeOf((*MockBankKeeper)(nil).HasDenomMetaData), ctx, denom)
}

// HasSupply mocks base method.
func (m *MockBankKeeper) HasSupply(ctx context.Context, denom string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HasSupply", ctx, denom)
	ret0, _ := ret[0].(bool)
	return ret0
}

// HasSupply indicates an expected call of HasSupply.
func (mr *MockBankKeeperMockRecorder) HasSupply(ctx, denom interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HasSupply", reflect.TypeOf((*MockBankKeeper)(nil).HasSupply), ctx, denom)
}

// InitGenesis mocks base method.
func (m *MockBankKeeper) InitGenesis(arg0 context.Context, arg1 *types0.GenesisState) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "InitGenesis", arg0, arg1)
}

// InitGenesis indicates an expected call of InitGenesis.
func (mr *MockBankKeeperMockRecorder) InitGenesis(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InitGenesis", reflect.TypeOf((*MockBankKeeper)(nil).InitGenesis), arg0, arg1)
}

// InputOutputCoins mocks base method.
func (m *MockBankKeeper) InputOutputCoins(ctx context.Context, input types0.Input, outputs []types0.Output) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InputOutputCoins", ctx, input, outputs)
	ret0, _ := ret[0].(error)
	return ret0
}

// InputOutputCoins indicates an expected call of InputOutputCoins.
func (mr *MockBankKeeperMockRecorder) InputOutputCoins(ctx, input, outputs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InputOutputCoins", reflect.TypeOf((*MockBankKeeper)(nil).InputOutputCoins), ctx, input, outputs)
}

// IsSendEnabledCoin mocks base method.
func (m *MockBankKeeper) IsSendEnabledCoin(ctx context.Context, coin types.Coin) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsSendEnabledCoin", ctx, coin)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsSendEnabledCoin indicates an expected call of IsSendEnabledCoin.
func (mr *MockBankKeeperMockRecorder) IsSendEnabledCoin(ctx, coin interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsSendEnabledCoin", reflect.TypeOf((*MockBankKeeper)(nil).IsSendEnabledCoin), ctx, coin)
}

// IsSendEnabledCoins mocks base method.
func (m *MockBankKeeper) IsSendEnabledCoins(ctx context.Context, coins ...types.Coin) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx}
	for _, a := range coins {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "IsSendEnabledCoins", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// IsSendEnabledCoins indicates an expected call of IsSendEnabledCoins.
func (mr *MockBankKeeperMockRecorder) IsSendEnabledCoins(ctx interface{}, coins ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx}, coins...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsSendEnabledCoins", reflect.TypeOf((*MockBankKeeper)(nil).IsSendEnabledCoins), varargs...)
}

// IsSendEnabledDenom mocks base method.
func (m *MockBankKeeper) IsSendEnabledDenom(ctx context.Context, denom string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsSendEnabledDenom", ctx, denom)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsSendEnabledDenom indicates an expected call of IsSendEnabledDenom.
func (mr *MockBankKeeperMockRecorder) IsSendEnabledDenom(ctx, denom interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsSendEnabledDenom", reflect.TypeOf((*MockBankKeeper)(nil).IsSendEnabledDenom), ctx, denom)
}

// IterateAccountBalances mocks base method.
func (m *MockBankKeeper) IterateAccountBalances(ctx context.Context, addr types.AccAddress, cb func(types.Coin) bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "IterateAccountBalances", ctx, addr, cb)
}

// IterateAccountBalances indicates an expected call of IterateAccountBalances.
func (mr *MockBankKeeperMockRecorder) IterateAccountBalances(ctx, addr, cb interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IterateAccountBalances", reflect.TypeOf((*MockBankKeeper)(nil).IterateAccountBalances), ctx, addr, cb)
}

// IterateAllBalances mocks base method.
func (m *MockBankKeeper) IterateAllBalances(ctx context.Context, cb func(types.AccAddress, types.Coin) bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "IterateAllBalances", ctx, cb)
}

// IterateAllBalances indicates an expected call of IterateAllBalances.
func (mr *MockBankKeeperMockRecorder) IterateAllBalances(ctx, cb interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IterateAllBalances", reflect.TypeOf((*MockBankKeeper)(nil).IterateAllBalances), ctx, cb)
}

// IterateAllDenomMetaData mocks base method.
func (m *MockBankKeeper) IterateAllDenomMetaData(ctx context.Context, cb func(types0.Metadata) bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "IterateAllDenomMetaData", ctx, cb)
}

// IterateAllDenomMetaData indicates an expected call of IterateAllDenomMetaData.
func (mr *MockBankKeeperMockRecorder) IterateAllDenomMetaData(ctx, cb interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IterateAllDenomMetaData", reflect.TypeOf((*MockBankKeeper)(nil).IterateAllDenomMetaData), ctx, cb)
}

// IterateSendEnabledEntries mocks base method.
func (m *MockBankKeeper) IterateSendEnabledEntries(ctx context.Context, cb func(string, bool) bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "IterateSendEnabledEntries", ctx, cb)
}

// IterateSendEnabledEntries indicates an expected call of IterateSendEnabledEntries.
func (mr *MockBankKeeperMockRecorder) IterateSendEnabledEntries(ctx, cb interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IterateSendEnabledEntries", reflect.TypeOf((*MockBankKeeper)(nil).IterateSendEnabledEntries), ctx, cb)
}

// IterateTotalSupply mocks base method.
func (m *MockBankKeeper) IterateTotalSupply(ctx context.Context, cb func(types.Coin) bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "IterateTotalSupply", ctx, cb)
}

// IterateTotalSupply indicates an expected call of IterateTotalSupply.
func (mr *MockBankKeeperMockRecorder) IterateTotalSupply(ctx, cb interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IterateTotalSupply", reflect.TypeOf((*MockBankKeeper)(nil).IterateTotalSupply), ctx, cb)
}

// LockedCoins mocks base method.
func (m *MockBankKeeper) LockedCoins(ctx context.Context, addr types.AccAddress) types.Coins {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LockedCoins", ctx, addr)
	ret0, _ := ret[0].(types.Coins)
	return ret0
}

// LockedCoins indicates an expected call of LockedCoins.
func (mr *MockBankKeeperMockRecorder) LockedCoins(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LockedCoins", reflect.TypeOf((*MockBankKeeper)(nil).LockedCoins), ctx, addr)
}

// MintCoins mocks base method.
func (m *MockBankKeeper) MintCoins(ctx context.Context, moduleName string, amt types.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MintCoins", ctx, moduleName, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// MintCoins indicates an expected call of MintCoins.
func (mr *MockBankKeeperMockRecorder) MintCoins(ctx, moduleName, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MintCoins", reflect.TypeOf((*MockBankKeeper)(nil).MintCoins), ctx, moduleName, amt)
}

// Params mocks base method.
func (m *MockBankKeeper) Params(arg0 context.Context, arg1 *types0.QueryParamsRequest) (*types0.QueryParamsResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Params", arg0, arg1)
	ret0, _ := ret[0].(*types0.QueryParamsResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Params indicates an expected call of Params.
func (mr *MockBankKeeperMockRecorder) Params(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Params", reflect.TypeOf((*MockBankKeeper)(nil).Params), arg0, arg1)
}

// PrependSendRestriction mocks base method.
func (m *MockBankKeeper) PrependSendRestriction(restriction types0.SendRestrictionFn) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "PrependSendRestriction", restriction)
}

// PrependSendRestriction indicates an expected call of PrependSendRestriction.
func (mr *MockBankKeeperMockRecorder) PrependSendRestriction(restriction interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PrependSendRestriction", reflect.TypeOf((*MockBankKeeper)(nil).PrependSendRestriction), restriction)
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

// SendCoinsFromAccountToModule mocks base method.
func (m *MockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr types.AccAddress, recipientModule string, amt types.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendCoinsFromAccountToModule", ctx, senderAddr, recipientModule, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendCoinsFromAccountToModule indicates an expected call of SendCoinsFromAccountToModule.
func (mr *MockBankKeeperMockRecorder) SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoinsFromAccountToModule", reflect.TypeOf((*MockBankKeeper)(nil).SendCoinsFromAccountToModule), ctx, senderAddr, recipientModule, amt)
}

// SendCoinsFromModuleToAccount mocks base method.
func (m *MockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr types.AccAddress, amt types.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendCoinsFromModuleToAccount", ctx, senderModule, recipientAddr, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendCoinsFromModuleToAccount indicates an expected call of SendCoinsFromModuleToAccount.
func (mr *MockBankKeeperMockRecorder) SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoinsFromModuleToAccount", reflect.TypeOf((*MockBankKeeper)(nil).SendCoinsFromModuleToAccount), ctx, senderModule, recipientAddr, amt)
}

// SendCoinsFromModuleToModule mocks base method.
func (m *MockBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt types.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendCoinsFromModuleToModule", ctx, senderModule, recipientModule, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendCoinsFromModuleToModule indicates an expected call of SendCoinsFromModuleToModule.
func (mr *MockBankKeeperMockRecorder) SendCoinsFromModuleToModule(ctx, senderModule, recipientModule, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoinsFromModuleToModule", reflect.TypeOf((*MockBankKeeper)(nil).SendCoinsFromModuleToModule), ctx, senderModule, recipientModule, amt)
}

// SendEnabled mocks base method.
func (m *MockBankKeeper) SendEnabled(arg0 context.Context, arg1 *types0.QuerySendEnabledRequest) (*types0.QuerySendEnabledResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendEnabled", arg0, arg1)
	ret0, _ := ret[0].(*types0.QuerySendEnabledResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendEnabled indicates an expected call of SendEnabled.
func (mr *MockBankKeeperMockRecorder) SendEnabled(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendEnabled", reflect.TypeOf((*MockBankKeeper)(nil).SendEnabled), arg0, arg1)
}

// SetAllSendEnabled mocks base method.
func (m *MockBankKeeper) SetAllSendEnabled(ctx context.Context, sendEnableds []*types0.SendEnabled) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetAllSendEnabled", ctx, sendEnableds)
}

// SetAllSendEnabled indicates an expected call of SetAllSendEnabled.
func (mr *MockBankKeeperMockRecorder) SetAllSendEnabled(ctx, sendEnableds interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetAllSendEnabled", reflect.TypeOf((*MockBankKeeper)(nil).SetAllSendEnabled), ctx, sendEnableds)
}

// SetDenomMetaData mocks base method.
func (m *MockBankKeeper) SetDenomMetaData(ctx context.Context, denomMetaData types0.Metadata) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetDenomMetaData", ctx, denomMetaData)
}

// SetDenomMetaData indicates an expected call of SetDenomMetaData.
func (mr *MockBankKeeperMockRecorder) SetDenomMetaData(ctx, denomMetaData interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetDenomMetaData", reflect.TypeOf((*MockBankKeeper)(nil).SetDenomMetaData), ctx, denomMetaData)
}

// SetParams mocks base method.
func (m *MockBankKeeper) SetParams(ctx context.Context, params types0.Params) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetParams", ctx, params)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetParams indicates an expected call of SetParams.
func (mr *MockBankKeeperMockRecorder) SetParams(ctx, params interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetParams", reflect.TypeOf((*MockBankKeeper)(nil).SetParams), ctx, params)
}

// SetSendEnabled mocks base method.
func (m *MockBankKeeper) SetSendEnabled(ctx context.Context, denom string, value bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetSendEnabled", ctx, denom, value)
}

// SetSendEnabled indicates an expected call of SetSendEnabled.
func (mr *MockBankKeeperMockRecorder) SetSendEnabled(ctx, denom, value interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetSendEnabled", reflect.TypeOf((*MockBankKeeper)(nil).SetSendEnabled), ctx, denom, value)
}

// SpendableBalanceByDenom mocks base method.
func (m *MockBankKeeper) SpendableBalanceByDenom(arg0 context.Context, arg1 *types0.QuerySpendableBalanceByDenomRequest) (*types0.QuerySpendableBalanceByDenomResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SpendableBalanceByDenom", arg0, arg1)
	ret0, _ := ret[0].(*types0.QuerySpendableBalanceByDenomResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SpendableBalanceByDenom indicates an expected call of SpendableBalanceByDenom.
func (mr *MockBankKeeperMockRecorder) SpendableBalanceByDenom(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SpendableBalanceByDenom", reflect.TypeOf((*MockBankKeeper)(nil).SpendableBalanceByDenom), arg0, arg1)
}

// SpendableBalances mocks base method.
func (m *MockBankKeeper) SpendableBalances(arg0 context.Context, arg1 *types0.QuerySpendableBalancesRequest) (*types0.QuerySpendableBalancesResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SpendableBalances", arg0, arg1)
	ret0, _ := ret[0].(*types0.QuerySpendableBalancesResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SpendableBalances indicates an expected call of SpendableBalances.
func (mr *MockBankKeeperMockRecorder) SpendableBalances(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SpendableBalances", reflect.TypeOf((*MockBankKeeper)(nil).SpendableBalances), arg0, arg1)
}

// SpendableCoin mocks base method.
func (m *MockBankKeeper) SpendableCoin(ctx context.Context, addr types.AccAddress, denom string) types.Coin {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SpendableCoin", ctx, addr, denom)
	ret0, _ := ret[0].(types.Coin)
	return ret0
}

// SpendableCoin indicates an expected call of SpendableCoin.
func (mr *MockBankKeeperMockRecorder) SpendableCoin(ctx, addr, denom interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SpendableCoin", reflect.TypeOf((*MockBankKeeper)(nil).SpendableCoin), ctx, addr, denom)
}

// SpendableCoins mocks base method.
func (m *MockBankKeeper) SpendableCoins(ctx context.Context, addr types.AccAddress) types.Coins {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SpendableCoins", ctx, addr)
	ret0, _ := ret[0].(types.Coins)
	return ret0
}

// SpendableCoins indicates an expected call of SpendableCoins.
func (mr *MockBankKeeperMockRecorder) SpendableCoins(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SpendableCoins", reflect.TypeOf((*MockBankKeeper)(nil).SpendableCoins), ctx, addr)
}

// SupplyOf mocks base method.
func (m *MockBankKeeper) SupplyOf(arg0 context.Context, arg1 *types0.QuerySupplyOfRequest) (*types0.QuerySupplyOfResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SupplyOf", arg0, arg1)
	ret0, _ := ret[0].(*types0.QuerySupplyOfResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SupplyOf indicates an expected call of SupplyOf.
func (mr *MockBankKeeperMockRecorder) SupplyOf(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SupplyOf", reflect.TypeOf((*MockBankKeeper)(nil).SupplyOf), arg0, arg1)
}

// TotalSupply mocks base method.
func (m *MockBankKeeper) TotalSupply(arg0 context.Context, arg1 *types0.QueryTotalSupplyRequest) (*types0.QueryTotalSupplyResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TotalSupply", arg0, arg1)
	ret0, _ := ret[0].(*types0.QueryTotalSupplyResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TotalSupply indicates an expected call of TotalSupply.
func (mr *MockBankKeeperMockRecorder) TotalSupply(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TotalSupply", reflect.TypeOf((*MockBankKeeper)(nil).TotalSupply), arg0, arg1)
}

// UndelegateCoins mocks base method.
func (m *MockBankKeeper) UndelegateCoins(ctx context.Context, moduleAccAddr, delegatorAddr types.AccAddress, amt types.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UndelegateCoins", ctx, moduleAccAddr, delegatorAddr, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// UndelegateCoins indicates an expected call of UndelegateCoins.
func (mr *MockBankKeeperMockRecorder) UndelegateCoins(ctx, moduleAccAddr, delegatorAddr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UndelegateCoins", reflect.TypeOf((*MockBankKeeper)(nil).UndelegateCoins), ctx, moduleAccAddr, delegatorAddr, amt)
}

// UndelegateCoinsFromModuleToAccount mocks base method.
func (m *MockBankKeeper) UndelegateCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr types.AccAddress, amt types.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UndelegateCoinsFromModuleToAccount", ctx, senderModule, recipientAddr, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

// UndelegateCoinsFromModuleToAccount indicates an expected call of UndelegateCoinsFromModuleToAccount.
func (mr *MockBankKeeperMockRecorder) UndelegateCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UndelegateCoinsFromModuleToAccount", reflect.TypeOf((*MockBankKeeper)(nil).UndelegateCoinsFromModuleToAccount), ctx, senderModule, recipientAddr, amt)
}

// ValidateBalance mocks base method.
func (m *MockBankKeeper) ValidateBalance(ctx context.Context, addr types.AccAddress) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateBalance", ctx, addr)
	ret0, _ := ret[0].(error)
	return ret0
}

// ValidateBalance indicates an expected call of ValidateBalance.
func (mr *MockBankKeeperMockRecorder) ValidateBalance(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateBalance", reflect.TypeOf((*MockBankKeeper)(nil).ValidateBalance), ctx, addr)
}

// WithMintCoinsRestriction mocks base method.
func (m *MockBankKeeper) WithMintCoinsRestriction(arg0 types0.MintingRestrictionFn) keeper.BaseKeeper {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithMintCoinsRestriction", arg0)
	ret0, _ := ret[0].(keeper.BaseKeeper)
	return ret0
}

// WithMintCoinsRestriction indicates an expected call of WithMintCoinsRestriction.
func (mr *MockBankKeeperMockRecorder) WithMintCoinsRestriction(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithMintCoinsRestriction", reflect.TypeOf((*MockBankKeeper)(nil).WithMintCoinsRestriction), arg0)
}
