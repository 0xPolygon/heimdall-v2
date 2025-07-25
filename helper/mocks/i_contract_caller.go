// Code generated by mockery v2.53.4. DO NOT EDIT.

package mocks

import (
	context "context"
	big "math/big"

	common "github.com/ethereum/go-ethereum/common"

	erc20 "github.com/0xPolygon/heimdall-v2/contracts/erc20"

	mock "github.com/stretchr/testify/mock"

	rootchain "github.com/0xPolygon/heimdall-v2/contracts/rootchain"

	slashmanager "github.com/0xPolygon/heimdall-v2/contracts/slashmanager"

	stakemanager "github.com/0xPolygon/heimdall-v2/contracts/stakemanager"

	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"

	stakinginfo "github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"

	statereceiver "github.com/0xPolygon/heimdall-v2/contracts/statereceiver"

	statesender "github.com/0xPolygon/heimdall-v2/contracts/statesender"

	types "github.com/ethereum/go-ethereum/core/types"

	validatorset "github.com/0xPolygon/heimdall-v2/contracts/validatorset"
)

// IContractCaller is an autogenerated mock type for the IContractCaller type
type IContractCaller struct {
	mock.Mock
}

// ApproveTokens provides a mock function with given fields: _a0, _a1, _a2, _a3
func (_m *IContractCaller) ApproveTokens(_a0 *big.Int, _a1 common.Address, _a2 common.Address, _a3 *erc20.Erc20) error {
	ret := _m.Called(_a0, _a1, _a2, _a3)

	if len(ret) == 0 {
		panic("no return value specified for ApproveTokens")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*big.Int, common.Address, common.Address, *erc20.Erc20) error); ok {
		r0 = rf(_a0, _a1, _a2, _a3)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CheckIfBlocksExist provides a mock function with given fields: end
func (_m *IContractCaller) CheckIfBlocksExist(end uint64) (bool, error) {
	ret := _m.Called(end)

	if len(ret) == 0 {
		panic("no return value specified for CheckIfBlocksExist")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64) (bool, error)); ok {
		return rf(end)
	}
	if rf, ok := ret.Get(0).(func(uint64) bool); ok {
		r0 = rf(end)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(uint64) error); ok {
		r1 = rf(end)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CurrentAccountStateRoot provides a mock function with given fields: stakingInfoInstance
func (_m *IContractCaller) CurrentAccountStateRoot(stakingInfoInstance *stakinginfo.Stakinginfo) ([32]byte, error) {
	ret := _m.Called(stakingInfoInstance)

	if len(ret) == 0 {
		panic("no return value specified for CurrentAccountStateRoot")
	}

	var r0 [32]byte
	var r1 error
	if rf, ok := ret.Get(0).(func(*stakinginfo.Stakinginfo) ([32]byte, error)); ok {
		return rf(stakingInfoInstance)
	}
	if rf, ok := ret.Get(0).(func(*stakinginfo.Stakinginfo) [32]byte); ok {
		r0 = rf(stakingInfoInstance)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([32]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(*stakinginfo.Stakinginfo) error); ok {
		r1 = rf(stakingInfoInstance)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CurrentHeaderBlock provides a mock function with given fields: rootChainInstance, childBlockInterval
func (_m *IContractCaller) CurrentHeaderBlock(rootChainInstance *rootchain.Rootchain, childBlockInterval uint64) (uint64, error) {
	ret := _m.Called(rootChainInstance, childBlockInterval)

	if len(ret) == 0 {
		panic("no return value specified for CurrentHeaderBlock")
	}

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(*rootchain.Rootchain, uint64) (uint64, error)); ok {
		return rf(rootChainInstance, childBlockInterval)
	}
	if rf, ok := ret.Get(0).(func(*rootchain.Rootchain, uint64) uint64); ok {
		r0 = rf(rootChainInstance, childBlockInterval)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(*rootchain.Rootchain, uint64) error); ok {
		r1 = rf(rootChainInstance, childBlockInterval)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CurrentSpanNumber provides a mock function with given fields: validatorSet
func (_m *IContractCaller) CurrentSpanNumber(validatorSet *validatorset.Validatorset) *big.Int {
	ret := _m.Called(validatorSet)

	if len(ret) == 0 {
		panic("no return value specified for CurrentSpanNumber")
	}

	var r0 *big.Int
	if rf, ok := ret.Get(0).(func(*validatorset.Validatorset) *big.Int); ok {
		r0 = rf(validatorSet)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	return r0
}

// CurrentStateCounter provides a mock function with given fields: stateSenderInstance
func (_m *IContractCaller) CurrentStateCounter(stateSenderInstance *statesender.Statesender) *big.Int {
	ret := _m.Called(stateSenderInstance)

	if len(ret) == 0 {
		panic("no return value specified for CurrentStateCounter")
	}

	var r0 *big.Int
	if rf, ok := ret.Get(0).(func(*statesender.Statesender) *big.Int); ok {
		r0 = rf(stateSenderInstance)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	return r0
}

// DecodeNewHeaderBlockEvent provides a mock function with given fields: _a0, _a1, _a2
func (_m *IContractCaller) DecodeNewHeaderBlockEvent(_a0 string, _a1 *types.Receipt, _a2 uint64) (*rootchain.RootchainNewHeaderBlock, error) {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for DecodeNewHeaderBlockEvent")
	}

	var r0 *rootchain.RootchainNewHeaderBlock
	var r1 error
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) (*rootchain.RootchainNewHeaderBlock, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) *rootchain.RootchainNewHeaderBlock); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rootchain.RootchainNewHeaderBlock)
		}
	}

	if rf, ok := ret.Get(1).(func(string, *types.Receipt, uint64) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DecodeSignerUpdateEvent provides a mock function with given fields: _a0, _a1, _a2
func (_m *IContractCaller) DecodeSignerUpdateEvent(_a0 string, _a1 *types.Receipt, _a2 uint64) (*stakinginfo.StakinginfoSignerChange, error) {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for DecodeSignerUpdateEvent")
	}

	var r0 *stakinginfo.StakinginfoSignerChange
	var r1 error
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) (*stakinginfo.StakinginfoSignerChange, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) *stakinginfo.StakinginfoSignerChange); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*stakinginfo.StakinginfoSignerChange)
		}
	}

	if rf, ok := ret.Get(1).(func(string, *types.Receipt, uint64) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DecodeSlashedEvent provides a mock function with given fields: _a0, _a1, _a2
func (_m *IContractCaller) DecodeSlashedEvent(_a0 string, _a1 *types.Receipt, _a2 uint64) (*stakinginfo.StakinginfoSlashed, error) {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for DecodeSlashedEvent")
	}

	var r0 *stakinginfo.StakinginfoSlashed
	var r1 error
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) (*stakinginfo.StakinginfoSlashed, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) *stakinginfo.StakinginfoSlashed); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*stakinginfo.StakinginfoSlashed)
		}
	}

	if rf, ok := ret.Get(1).(func(string, *types.Receipt, uint64) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DecodeStateSyncedEvent provides a mock function with given fields: _a0, _a1, _a2
func (_m *IContractCaller) DecodeStateSyncedEvent(_a0 string, _a1 *types.Receipt, _a2 uint64) (*statesender.StatesenderStateSynced, error) {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for DecodeStateSyncedEvent")
	}

	var r0 *statesender.StatesenderStateSynced
	var r1 error
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) (*statesender.StatesenderStateSynced, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) *statesender.StatesenderStateSynced); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*statesender.StatesenderStateSynced)
		}
	}

	if rf, ok := ret.Get(1).(func(string, *types.Receipt, uint64) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DecodeUnJailedEvent provides a mock function with given fields: _a0, _a1, _a2
func (_m *IContractCaller) DecodeUnJailedEvent(_a0 string, _a1 *types.Receipt, _a2 uint64) (*stakinginfo.StakinginfoUnJailed, error) {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for DecodeUnJailedEvent")
	}

	var r0 *stakinginfo.StakinginfoUnJailed
	var r1 error
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) (*stakinginfo.StakinginfoUnJailed, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) *stakinginfo.StakinginfoUnJailed); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*stakinginfo.StakinginfoUnJailed)
		}
	}

	if rf, ok := ret.Get(1).(func(string, *types.Receipt, uint64) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DecodeValidatorExitEvent provides a mock function with given fields: _a0, _a1, _a2
func (_m *IContractCaller) DecodeValidatorExitEvent(_a0 string, _a1 *types.Receipt, _a2 uint64) (*stakinginfo.StakinginfoUnstakeInit, error) {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for DecodeValidatorExitEvent")
	}

	var r0 *stakinginfo.StakinginfoUnstakeInit
	var r1 error
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) (*stakinginfo.StakinginfoUnstakeInit, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) *stakinginfo.StakinginfoUnstakeInit); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*stakinginfo.StakinginfoUnstakeInit)
		}
	}

	if rf, ok := ret.Get(1).(func(string, *types.Receipt, uint64) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DecodeValidatorJoinEvent provides a mock function with given fields: _a0, _a1, _a2
func (_m *IContractCaller) DecodeValidatorJoinEvent(_a0 string, _a1 *types.Receipt, _a2 uint64) (*stakinginfo.StakinginfoStaked, error) {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for DecodeValidatorJoinEvent")
	}

	var r0 *stakinginfo.StakinginfoStaked
	var r1 error
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) (*stakinginfo.StakinginfoStaked, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) *stakinginfo.StakinginfoStaked); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*stakinginfo.StakinginfoStaked)
		}
	}

	if rf, ok := ret.Get(1).(func(string, *types.Receipt, uint64) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DecodeValidatorStakeUpdateEvent provides a mock function with given fields: _a0, _a1, _a2
func (_m *IContractCaller) DecodeValidatorStakeUpdateEvent(_a0 string, _a1 *types.Receipt, _a2 uint64) (*stakinginfo.StakinginfoStakeUpdate, error) {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for DecodeValidatorStakeUpdateEvent")
	}

	var r0 *stakinginfo.StakinginfoStakeUpdate
	var r1 error
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) (*stakinginfo.StakinginfoStakeUpdate, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) *stakinginfo.StakinginfoStakeUpdate); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*stakinginfo.StakinginfoStakeUpdate)
		}
	}

	if rf, ok := ret.Get(1).(func(string, *types.Receipt, uint64) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DecodeValidatorTopupFeesEvent provides a mock function with given fields: _a0, _a1, _a2
func (_m *IContractCaller) DecodeValidatorTopupFeesEvent(_a0 string, _a1 *types.Receipt, _a2 uint64) (*stakinginfo.StakinginfoTopUpFee, error) {
	ret := _m.Called(_a0, _a1, _a2)

	if len(ret) == 0 {
		panic("no return value specified for DecodeValidatorTopupFeesEvent")
	}

	var r0 *stakinginfo.StakinginfoTopUpFee
	var r1 error
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) (*stakinginfo.StakinginfoTopUpFee, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(string, *types.Receipt, uint64) *stakinginfo.StakinginfoTopUpFee); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*stakinginfo.StakinginfoTopUpFee)
		}
	}

	if rf, ok := ret.Get(1).(func(string, *types.Receipt, uint64) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBalance provides a mock function with given fields: address
func (_m *IContractCaller) GetBalance(address common.Address) (*big.Int, error) {
	ret := _m.Called(address)

	if len(ret) == 0 {
		panic("no return value specified for GetBalance")
	}

	var r0 *big.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(common.Address) (*big.Int, error)); ok {
		return rf(address)
	}
	if rf, ok := ret.Get(0).(func(common.Address) *big.Int); ok {
		r0 = rf(address)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	if rf, ok := ret.Get(1).(func(common.Address) error); ok {
		r1 = rf(address)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockNumberFromTxHash provides a mock function with given fields: _a0
func (_m *IContractCaller) GetBlockNumberFromTxHash(_a0 common.Hash) (*big.Int, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetBlockNumberFromTxHash")
	}

	var r0 *big.Int
	var r1 error
	if rf, ok := ret.Get(0).(func(common.Hash) (*big.Int, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(common.Hash) *big.Int); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBorChainBlock provides a mock function with given fields: _a0, _a1
func (_m *IContractCaller) GetBorChainBlock(_a0 context.Context, _a1 *big.Int) (*types.Header, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for GetBorChainBlock")
	}

	var r0 *types.Header
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *big.Int) (*types.Header, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *big.Int) *types.Header); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Header)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *big.Int) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBorChainBlockAuthor provides a mock function with given fields: _a0
func (_m *IContractCaller) GetBorChainBlockAuthor(_a0 *big.Int) (*common.Address, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetBorChainBlockAuthor")
	}

	var r0 *common.Address
	var r1 error
	if rf, ok := ret.Get(0).(func(*big.Int) (*common.Address, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(*big.Int) *common.Address); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*common.Address)
		}
	}

	if rf, ok := ret.Get(1).(func(*big.Int) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBorChainBlockInfoInBatch provides a mock function with given fields: ctx, start, end
func (_m *IContractCaller) GetBorChainBlockInfoInBatch(ctx context.Context, start int64, end int64) ([]*types.Header, []uint64, []common.Address, error) {
	ret := _m.Called(ctx, start, end)

	if len(ret) == 0 {
		panic("no return value specified for GetBorChainBlockInfoInBatch")
	}

	var r0 []*types.Header
	var r1 []uint64
	var r2 []common.Address
	var r3 error
	if rf, ok := ret.Get(0).(func(context.Context, int64, int64) ([]*types.Header, []uint64, []common.Address, error)); ok {
		return rf(ctx, start, end)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int64, int64) []*types.Header); ok {
		r0 = rf(ctx, start, end)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*types.Header)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, int64, int64) []uint64); ok {
		r1 = rf(ctx, start, end)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]uint64)
		}
	}

	if rf, ok := ret.Get(2).(func(context.Context, int64, int64) []common.Address); ok {
		r2 = rf(ctx, start, end)
	} else {
		if ret.Get(2) != nil {
			r2 = ret.Get(2).([]common.Address)
		}
	}

	if rf, ok := ret.Get(3).(func(context.Context, int64, int64) error); ok {
		r3 = rf(ctx, start, end)
	} else {
		r3 = ret.Error(3)
	}

	return r0, r1, r2, r3
}

// GetBorChainBlockTd provides a mock function with given fields: ctx, blockHash
func (_m *IContractCaller) GetBorChainBlockTd(ctx context.Context, blockHash common.Hash) (uint64, error) {
	ret := _m.Called(ctx, blockHash)

	if len(ret) == 0 {
		panic("no return value specified for GetBorChainBlockTd")
	}

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) (uint64, error)); ok {
		return rf(ctx, blockHash)
	}
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) uint64); ok {
		r0 = rf(ctx, blockHash)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, common.Hash) error); ok {
		r1 = rf(ctx, blockHash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBorTxReceipt provides a mock function with given fields: _a0
func (_m *IContractCaller) GetBorTxReceipt(_a0 common.Hash) (*types.Receipt, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetBorTxReceipt")
	}

	var r0 *types.Receipt
	var r1 error
	if rf, ok := ret.Get(0).(func(common.Hash) (*types.Receipt, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(common.Hash) *types.Receipt); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Receipt)
		}
	}

	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCheckpointSign provides a mock function with given fields: txHash
func (_m *IContractCaller) GetCheckpointSign(txHash common.Hash) ([]byte, []byte, []byte, error) {
	ret := _m.Called(txHash)

	if len(ret) == 0 {
		panic("no return value specified for GetCheckpointSign")
	}

	var r0 []byte
	var r1 []byte
	var r2 []byte
	var r3 error
	if rf, ok := ret.Get(0).(func(common.Hash) ([]byte, []byte, []byte, error)); ok {
		return rf(txHash)
	}
	if rf, ok := ret.Get(0).(func(common.Hash) []byte); ok {
		r0 = rf(txHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(common.Hash) []byte); ok {
		r1 = rf(txHash)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]byte)
		}
	}

	if rf, ok := ret.Get(2).(func(common.Hash) []byte); ok {
		r2 = rf(txHash)
	} else {
		if ret.Get(2) != nil {
			r2 = ret.Get(2).([]byte)
		}
	}

	if rf, ok := ret.Get(3).(func(common.Hash) error); ok {
		r3 = rf(txHash)
	} else {
		r3 = ret.Error(3)
	}

	return r0, r1, r2, r3
}

// GetConfirmedTxReceipt provides a mock function with given fields: _a0, _a1
func (_m *IContractCaller) GetConfirmedTxReceipt(_a0 common.Hash, _a1 uint64) (*types.Receipt, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for GetConfirmedTxReceipt")
	}

	var r0 *types.Receipt
	var r1 error
	if rf, ok := ret.Get(0).(func(common.Hash, uint64) (*types.Receipt, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(common.Hash, uint64) *types.Receipt); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Receipt)
		}
	}

	if rf, ok := ret.Get(1).(func(common.Hash, uint64) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetHeaderInfo provides a mock function with given fields: headerID, rootChainInstance, childBlockInterval
func (_m *IContractCaller) GetHeaderInfo(headerID uint64, rootChainInstance *rootchain.Rootchain, childBlockInterval uint64) (common.Hash, uint64, uint64, uint64, string, error) {
	ret := _m.Called(headerID, rootChainInstance, childBlockInterval)

	if len(ret) == 0 {
		panic("no return value specified for GetHeaderInfo")
	}

	var r0 common.Hash
	var r1 uint64
	var r2 uint64
	var r3 uint64
	var r4 string
	var r5 error
	if rf, ok := ret.Get(0).(func(uint64, *rootchain.Rootchain, uint64) (common.Hash, uint64, uint64, uint64, string, error)); ok {
		return rf(headerID, rootChainInstance, childBlockInterval)
	}
	if rf, ok := ret.Get(0).(func(uint64, *rootchain.Rootchain, uint64) common.Hash); ok {
		r0 = rf(headerID, rootChainInstance, childBlockInterval)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(common.Hash)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64, *rootchain.Rootchain, uint64) uint64); ok {
		r1 = rf(headerID, rootChainInstance, childBlockInterval)
	} else {
		r1 = ret.Get(1).(uint64)
	}

	if rf, ok := ret.Get(2).(func(uint64, *rootchain.Rootchain, uint64) uint64); ok {
		r2 = rf(headerID, rootChainInstance, childBlockInterval)
	} else {
		r2 = ret.Get(2).(uint64)
	}

	if rf, ok := ret.Get(3).(func(uint64, *rootchain.Rootchain, uint64) uint64); ok {
		r3 = rf(headerID, rootChainInstance, childBlockInterval)
	} else {
		r3 = ret.Get(3).(uint64)
	}

	if rf, ok := ret.Get(4).(func(uint64, *rootchain.Rootchain, uint64) string); ok {
		r4 = rf(headerID, rootChainInstance, childBlockInterval)
	} else {
		r4 = ret.Get(4).(string)
	}

	if rf, ok := ret.Get(5).(func(uint64, *rootchain.Rootchain, uint64) error); ok {
		r5 = rf(headerID, rootChainInstance, childBlockInterval)
	} else {
		r5 = ret.Error(5)
	}

	return r0, r1, r2, r3, r4, r5
}

// GetLastChildBlock provides a mock function with given fields: rootChainInstance
func (_m *IContractCaller) GetLastChildBlock(rootChainInstance *rootchain.Rootchain) (uint64, error) {
	ret := _m.Called(rootChainInstance)

	if len(ret) == 0 {
		panic("no return value specified for GetLastChildBlock")
	}

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(*rootchain.Rootchain) (uint64, error)); ok {
		return rf(rootChainInstance)
	}
	if rf, ok := ret.Get(0).(func(*rootchain.Rootchain) uint64); ok {
		r0 = rf(rootChainInstance)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(*rootchain.Rootchain) error); ok {
		r1 = rf(rootChainInstance)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetMainChainBlock provides a mock function with given fields: _a0
func (_m *IContractCaller) GetMainChainBlock(_a0 *big.Int) (*types.Header, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetMainChainBlock")
	}

	var r0 *types.Header
	var r1 error
	if rf, ok := ret.Get(0).(func(*big.Int) (*types.Header, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(*big.Int) *types.Header); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Header)
		}
	}

	if rf, ok := ret.Get(1).(func(*big.Int) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetMainTxReceipt provides a mock function with given fields: _a0
func (_m *IContractCaller) GetMainTxReceipt(_a0 common.Hash) (*types.Receipt, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetMainTxReceipt")
	}

	var r0 *types.Receipt
	var r1 error
	if rf, ok := ret.Get(0).(func(common.Hash) (*types.Receipt, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(common.Hash) *types.Receipt); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Receipt)
		}
	}

	if rf, ok := ret.Get(1).(func(common.Hash) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRootChainInstance provides a mock function with given fields: rootChainAddress
func (_m *IContractCaller) GetRootChainInstance(rootChainAddress string) (*rootchain.Rootchain, error) {
	ret := _m.Called(rootChainAddress)

	if len(ret) == 0 {
		panic("no return value specified for GetRootChainInstance")
	}

	var r0 *rootchain.Rootchain
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*rootchain.Rootchain, error)); ok {
		return rf(rootChainAddress)
	}
	if rf, ok := ret.Get(0).(func(string) *rootchain.Rootchain); ok {
		r0 = rf(rootChainAddress)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*rootchain.Rootchain)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(rootChainAddress)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRootHash provides a mock function with given fields: start, end, checkpointLength
func (_m *IContractCaller) GetRootHash(start uint64, end uint64, checkpointLength uint64) ([]byte, error) {
	ret := _m.Called(start, end, checkpointLength)

	if len(ret) == 0 {
		panic("no return value specified for GetRootHash")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, uint64, uint64) ([]byte, error)); ok {
		return rf(start, end, checkpointLength)
	}
	if rf, ok := ret.Get(0).(func(uint64, uint64, uint64) []byte); ok {
		r0 = rf(start, end, checkpointLength)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(uint64, uint64, uint64) error); ok {
		r1 = rf(start, end, checkpointLength)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSlashManagerInstance provides a mock function with given fields: slashManagerAddress
func (_m *IContractCaller) GetSlashManagerInstance(slashManagerAddress string) (*slashmanager.Slashmanager, error) {
	ret := _m.Called(slashManagerAddress)

	if len(ret) == 0 {
		panic("no return value specified for GetSlashManagerInstance")
	}

	var r0 *slashmanager.Slashmanager
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*slashmanager.Slashmanager, error)); ok {
		return rf(slashManagerAddress)
	}
	if rf, ok := ret.Get(0).(func(string) *slashmanager.Slashmanager); ok {
		r0 = rf(slashManagerAddress)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*slashmanager.Slashmanager)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(slashManagerAddress)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSpanDetails provides a mock function with given fields: id, validatorSet
func (_m *IContractCaller) GetSpanDetails(id *big.Int, validatorSet *validatorset.Validatorset) (*big.Int, *big.Int, *big.Int, error) {
	ret := _m.Called(id, validatorSet)

	if len(ret) == 0 {
		panic("no return value specified for GetSpanDetails")
	}

	var r0 *big.Int
	var r1 *big.Int
	var r2 *big.Int
	var r3 error
	if rf, ok := ret.Get(0).(func(*big.Int, *validatorset.Validatorset) (*big.Int, *big.Int, *big.Int, error)); ok {
		return rf(id, validatorSet)
	}
	if rf, ok := ret.Get(0).(func(*big.Int, *validatorset.Validatorset) *big.Int); ok {
		r0 = rf(id, validatorSet)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*big.Int)
		}
	}

	if rf, ok := ret.Get(1).(func(*big.Int, *validatorset.Validatorset) *big.Int); ok {
		r1 = rf(id, validatorSet)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*big.Int)
		}
	}

	if rf, ok := ret.Get(2).(func(*big.Int, *validatorset.Validatorset) *big.Int); ok {
		r2 = rf(id, validatorSet)
	} else {
		if ret.Get(2) != nil {
			r2 = ret.Get(2).(*big.Int)
		}
	}

	if rf, ok := ret.Get(3).(func(*big.Int, *validatorset.Validatorset) error); ok {
		r3 = rf(id, validatorSet)
	} else {
		r3 = ret.Error(3)
	}

	return r0, r1, r2, r3
}

// GetStakeManagerInstance provides a mock function with given fields: stakingManagerAddress
func (_m *IContractCaller) GetStakeManagerInstance(stakingManagerAddress string) (*stakemanager.Stakemanager, error) {
	ret := _m.Called(stakingManagerAddress)

	if len(ret) == 0 {
		panic("no return value specified for GetStakeManagerInstance")
	}

	var r0 *stakemanager.Stakemanager
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*stakemanager.Stakemanager, error)); ok {
		return rf(stakingManagerAddress)
	}
	if rf, ok := ret.Get(0).(func(string) *stakemanager.Stakemanager); ok {
		r0 = rf(stakingManagerAddress)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*stakemanager.Stakemanager)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(stakingManagerAddress)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStakingInfoInstance provides a mock function with given fields: stakingInfoAddress
func (_m *IContractCaller) GetStakingInfoInstance(stakingInfoAddress string) (*stakinginfo.Stakinginfo, error) {
	ret := _m.Called(stakingInfoAddress)

	if len(ret) == 0 {
		panic("no return value specified for GetStakingInfoInstance")
	}

	var r0 *stakinginfo.Stakinginfo
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*stakinginfo.Stakinginfo, error)); ok {
		return rf(stakingInfoAddress)
	}
	if rf, ok := ret.Get(0).(func(string) *stakinginfo.Stakinginfo); ok {
		r0 = rf(stakingInfoAddress)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*stakinginfo.Stakinginfo)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(stakingInfoAddress)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStartBlockHeimdallSpanID provides a mock function with given fields: ctx, startBlock
func (_m *IContractCaller) GetStartBlockHeimdallSpanID(ctx context.Context, startBlock uint64) (uint64, error) {
	ret := _m.Called(ctx, startBlock)

	if len(ret) == 0 {
		panic("no return value specified for GetStartBlockHeimdallSpanID")
	}

	var r0 uint64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64) (uint64, error)); ok {
		return rf(ctx, startBlock)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64) uint64); ok {
		r0 = rf(ctx, startBlock)
	} else {
		r0 = ret.Get(0).(uint64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64) error); ok {
		r1 = rf(ctx, startBlock)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStateReceiverInstance provides a mock function with given fields: stateReceiverAddress
func (_m *IContractCaller) GetStateReceiverInstance(stateReceiverAddress string) (*statereceiver.Statereceiver, error) {
	ret := _m.Called(stateReceiverAddress)

	if len(ret) == 0 {
		panic("no return value specified for GetStateReceiverInstance")
	}

	var r0 *statereceiver.Statereceiver
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*statereceiver.Statereceiver, error)); ok {
		return rf(stateReceiverAddress)
	}
	if rf, ok := ret.Get(0).(func(string) *statereceiver.Statereceiver); ok {
		r0 = rf(stateReceiverAddress)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*statereceiver.Statereceiver)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(stateReceiverAddress)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetStateSenderInstance provides a mock function with given fields: stateSenderAddress
func (_m *IContractCaller) GetStateSenderInstance(stateSenderAddress string) (*statesender.Statesender, error) {
	ret := _m.Called(stateSenderAddress)

	if len(ret) == 0 {
		panic("no return value specified for GetStateSenderInstance")
	}

	var r0 *statesender.Statesender
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*statesender.Statesender, error)); ok {
		return rf(stateSenderAddress)
	}
	if rf, ok := ret.Get(0).(func(string) *statesender.Statesender); ok {
		r0 = rf(stateSenderAddress)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*statesender.Statesender)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(stateSenderAddress)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTokenInstance provides a mock function with given fields: tokenAddress
func (_m *IContractCaller) GetTokenInstance(tokenAddress string) (*erc20.Erc20, error) {
	ret := _m.Called(tokenAddress)

	if len(ret) == 0 {
		panic("no return value specified for GetTokenInstance")
	}

	var r0 *erc20.Erc20
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*erc20.Erc20, error)); ok {
		return rf(tokenAddress)
	}
	if rf, ok := ret.Get(0).(func(string) *erc20.Erc20); ok {
		r0 = rf(tokenAddress)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*erc20.Erc20)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(tokenAddress)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetValidatorInfo provides a mock function with given fields: valID, stakingInfoInstance
func (_m *IContractCaller) GetValidatorInfo(valID uint64, stakingInfoInstance *stakinginfo.Stakinginfo) (staketypes.Validator, error) {
	ret := _m.Called(valID, stakingInfoInstance)

	if len(ret) == 0 {
		panic("no return value specified for GetValidatorInfo")
	}

	var r0 staketypes.Validator
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, *stakinginfo.Stakinginfo) (staketypes.Validator, error)); ok {
		return rf(valID, stakingInfoInstance)
	}
	if rf, ok := ret.Get(0).(func(uint64, *stakinginfo.Stakinginfo) staketypes.Validator); ok {
		r0 = rf(valID, stakingInfoInstance)
	} else {
		r0 = ret.Get(0).(staketypes.Validator)
	}

	if rf, ok := ret.Get(1).(func(uint64, *stakinginfo.Stakinginfo) error); ok {
		r1 = rf(valID, stakingInfoInstance)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetValidatorSetInstance provides a mock function with given fields: validatorSetAddress
func (_m *IContractCaller) GetValidatorSetInstance(validatorSetAddress string) (*validatorset.Validatorset, error) {
	ret := _m.Called(validatorSetAddress)

	if len(ret) == 0 {
		panic("no return value specified for GetValidatorSetInstance")
	}

	var r0 *validatorset.Validatorset
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*validatorset.Validatorset, error)); ok {
		return rf(validatorSetAddress)
	}
	if rf, ok := ret.Get(0).(func(string) *validatorset.Validatorset); ok {
		r0 = rf(validatorSetAddress)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*validatorset.Validatorset)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(validatorSetAddress)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetVoteOnHash provides a mock function with given fields: start, end, hash, milestoneID
func (_m *IContractCaller) GetVoteOnHash(start uint64, end uint64, hash string, milestoneID string) (bool, error) {
	ret := _m.Called(start, end, hash, milestoneID)

	if len(ret) == 0 {
		panic("no return value specified for GetVoteOnHash")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(uint64, uint64, string, string) (bool, error)); ok {
		return rf(start, end, hash, milestoneID)
	}
	if rf, ok := ret.Get(0).(func(uint64, uint64, string, string) bool); ok {
		r0 = rf(start, end, hash, milestoneID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(uint64, uint64, string, string) error); ok {
		r1 = rf(start, end, hash, milestoneID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsTxConfirmed provides a mock function with given fields: _a0, _a1
func (_m *IContractCaller) IsTxConfirmed(_a0 common.Hash, _a1 uint64) bool {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for IsTxConfirmed")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func(common.Hash, uint64) bool); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// SendCheckpoint provides a mock function with given fields: signedData, sigs, rootChainAddress, rootChainInstance
func (_m *IContractCaller) SendCheckpoint(signedData []byte, sigs [][3]*big.Int, rootChainAddress common.Address, rootChainInstance *rootchain.Rootchain) error {
	ret := _m.Called(signedData, sigs, rootChainAddress, rootChainInstance)

	if len(ret) == 0 {
		panic("no return value specified for SendCheckpoint")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func([]byte, [][3]*big.Int, common.Address, *rootchain.Rootchain) error); ok {
		r0 = rf(signedData, sigs, rootChainAddress, rootChainInstance)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StakeFor provides a mock function with given fields: _a0, _a1, _a2, _a3, _a4, _a5
func (_m *IContractCaller) StakeFor(_a0 common.Address, _a1 *big.Int, _a2 *big.Int, _a3 bool, _a4 common.Address, _a5 *stakemanager.Stakemanager) error {
	ret := _m.Called(_a0, _a1, _a2, _a3, _a4, _a5)

	if len(ret) == 0 {
		panic("no return value specified for StakeFor")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(common.Address, *big.Int, *big.Int, bool, common.Address, *stakemanager.Stakemanager) error); ok {
		r0 = rf(_a0, _a1, _a2, _a3, _a4, _a5)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIContractCaller creates a new instance of IContractCaller. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewIContractCaller(t interface {
	mock.TestingT
	Cleanup(func())
}) *IContractCaller {
	mock := &IContractCaller{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
