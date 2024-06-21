package keeper_test

import (
	"math/big"
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func (s *KeeperTestSuite) TestHandleQueryCurrentValidatorSet() {
	ctx, keeper, queryClient, require := s.ctx, s.stakeKeeper, s.queryClient, s.Require()

	req := &types.QueryCurrentValidatorSetRequest{}
	res, err := queryClient.CurrentValidatorSet(ctx, req)

	require.NoError(err)
	require.Equal(len(res.ValidatorSet.Validators), 0)

	validatorSet := testutil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)

	req = &types.QueryCurrentValidatorSetRequest{}
	res, err = queryClient.CurrentValidatorSet(ctx, req)

	require.NoError(err)

	require.NotNil(res)
	require.Equal(res.ValidatorSet.Proposer.GetSigner(), validatorSet.Proposer.GetSigner())
	require.Equal(len(res.ValidatorSet.Validators), len(validatorSet.Validators))
	require.Equal(res.ValidatorSet.TotalVotingPower, validatorSet.TotalVotingPower)
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

	require.Equal(res.Validator.Signer, validators[0].Signer)
	require.Equal(res.Validator.StartEpoch, validators[0].StartEpoch)
	require.Equal(res.Validator.EndEpoch, validators[0].EndEpoch)
	require.Equal(res.Validator.PubKey.Compare(validators[0].PubKey), 0)
	require.Equal(res.Validator.ProposerPriority, validators[0].ProposerPriority)
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

	require.Equal(res.Validator.Signer, validators[0].Signer)
	require.Equal(res.Validator.StartEpoch, validators[0].StartEpoch)
	require.Equal(res.Validator.EndEpoch, validators[0].EndEpoch)
	require.Equal(res.Validator.PubKey.Compare(validators[0].PubKey), 0)
	require.Equal(res.Validator.ProposerPriority, validators[0].ProposerPriority)
}

func (s *KeeperTestSuite) TestHandleQueryValidatorStatus() {
	ctx, keeper, queryClient, require := s.ctx, s.stakeKeeper, s.queryClient, s.Require()

	testutil.LoadRandomValidatorSet(require, 4, keeper, ctx, false, 10)
	validators := keeper.GetAllValidators(ctx)

	req := &types.QueryValidatorStatusRequest{
		ValAddress: validators[0].Signer,
	}
	res, err := queryClient.ValidatorStatus(ctx, req)
	require.NoError(err)

	require.NotNil(res)
	require.True(res.Status)

	req = &types.QueryValidatorStatusRequest{
		ValAddress: common.Address{}.String(),
	}
	res, err = queryClient.ValidatorStatus(ctx, req)
	require.Nil(err)
	require.False(res.Status)

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

	req := &types.QueryStakingSequenceRequest{
		TxHash:   common.Bytes2Hex(txHash.Hash),
		LogIndex: logIndex,
	}

	sequence := new(big.Int).Mul(txReceipt.BlockNumber, big.NewInt(types.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(logIndex))

	err = keeper.SetStakingSequence(ctx, sequence.String())
	require.NoError(err)

	s.contractCaller.On("GetConfirmedTxReceipt", common.BytesToHash(txHash.Hash), chainParams.MainChainTxConfirmations).Return(txReceipt, nil)

	res, err := queryClient.StakingSequence(ctx, req)

	require.NoError(err)
	require.NotNil(res)
	require.True(res.Status)
}
