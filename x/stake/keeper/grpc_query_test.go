package keeper_test

import (
	"math/big"
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func (s *KeeperTestSuite) TestHandleQueryCurrentValidatorSet() {
	ctx, keeper, queryClient, require := s.ctx, s.stakeKeeper, s.queryClient, s.Require()

	req := &types.QueryCurrentValidatorSetRequest{}
	res, err := queryClient.CurrentValidatorSet(ctx, req)

	require.Error(err)

	validatorSet := testutil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)
	s.checkpointKeeper.EXPECT().GetAckCount(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	req = &types.QueryCurrentValidatorSetRequest{}
	res, err = queryClient.CurrentValidatorSet(ctx, req)

	require.NoError(err)

	require.NotNil(res)
	require.True(res.ValidatorSet.Equal(validatorSet))
}

func (s *KeeperTestSuite) TestHandleQuerySigner() {
	ctx, keeper, queryClient, require := s.ctx, s.stakeKeeper, s.queryClient, s.Require()

	req := &types.QuerySignerRequest{
		ValAddress: common.Address{}.String(),
	}

	res, err := queryClient.Signer(ctx, req)
	require.NotNil(err)

	testutil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)

	validators := keeper.GetAllValidators(ctx)

	req = &types.QuerySignerRequest{
		ValAddress: validators[0].Signer,
	}

	res, err = queryClient.Signer(ctx, req)
	require.NoError(err)

	// check response is not nil
	require.True(res.Validator.Equal(validators[0]))
}

func (s *KeeperTestSuite) TestHandleQueryValidator() {
	ctx, keeper, queryClient, require := s.ctx, s.stakeKeeper, s.queryClient, s.Require()

	req := &types.QueryValidatorRequest{
		Id: uint64(0),
	}

	res, err := queryClient.Validator(ctx, req)
	require.NotNil(err)
	require.Nil(res)

	req = &types.QueryValidatorRequest{
		Id: uint64(1),
	}

	res, err = queryClient.Validator(ctx, req)
	require.NotNil(err)
	require.Nil(res)

	testutil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)

	validators := keeper.GetAllValidators(ctx)

	req = &types.QueryValidatorRequest{
		Id: validators[0].ValId,
	}

	res, err = queryClient.Validator(ctx, req)

	require.NoError(err)

	require.True(res.Validator.Equal(validators[0]))
}

func (s *KeeperTestSuite) TestHandleQueryValidatorStatus() {
	ctx, keeper, queryClient, require := s.ctx, s.stakeKeeper, s.queryClient, s.Require()

	testutil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)
	s.checkpointKeeper.EXPECT().GetAckCount(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	validators := keeper.GetAllValidators(ctx)

	req := &types.QueryValidatorStatusRequest{
		ValAddress: validators[0].Signer,
	}
	res, err := queryClient.ValidatorStatus(ctx, req)
	require.NoError(err)

	require.NotNil(res)
	require.True(res.IsOld)

	req = &types.QueryValidatorStatusRequest{
		ValAddress: common.Address{}.String(),
	}
	res, err = queryClient.ValidatorStatus(ctx, req)
	require.Nil(err)
	require.False(res.IsOld)

}

func (s *KeeperTestSuite) TestHandleQueryStakingSequence() {
	ctx, keeper, queryClient, require := s.ctx, s.stakeKeeper, s.queryClient, s.Require()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	chainParams, err := s.cmKeeper.GetParams(ctx)
	require.NoError(err)

	txHash := hmTypes.TxHash{Hash: make([]byte, 32)}

	txReceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}

	logIndex := uint64(simulation.RandIntBetween(r1, 0, 100))

	req := &types.QueryStakingIsOldTxRequest{
		TxHash:   common.Bytes2Hex(txHash.Hash),
		LogIndex: logIndex,
	}

	sequence := new(big.Int).Mul(txReceipt.BlockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(logIndex))

	err = keeper.SetStakingSequence(ctx, sequence.String())
	require.NoError(err)

	s.contractCaller.On("GetConfirmedTxReceipt", common.BytesToHash(txHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

	res, err := queryClient.StakingIsOldTx(ctx, req)

	require.NoError(err)
	require.NotNil(res)
	require.True(res.IsOld)
}
