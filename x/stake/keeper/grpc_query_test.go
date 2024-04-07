package keeper_test

import (
	"github.com/0xPolygon/heimdall-v2/x/stake/testutil"
	"github.com/0xPolygon/heimdall-v2/x/stake/types"
	"github.com/ethereum/go-ethereum/common"
)

func (s *KeeperTestSuite) TestHandleQueryCurrentValidatorSet() {
	ctx, keeper, queryClient := s.ctx, s.stakeKeeper, s.queryClient
	require := s.Require()

	req := &types.QueryCurrentValidatorSetRequest{}
	res, err := queryClient.CurrentValidatorSet(ctx, req)
	// check no error found
	require.NoError(err)
	require.Equal(len(res.ValidatorSet.Validators), 0)

	//Set the validator set
	validatorSet := testutil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)

	req = &types.QueryCurrentValidatorSetRequest{}
	res, err = queryClient.CurrentValidatorSet(ctx, req)
	// check no error found
	require.NoError(err)

	// check response is not nil
	require.NotNil(res)
	require.Equal(res.ValidatorSet.Proposer.GetSigner(), validatorSet.Proposer.GetSigner())
	require.Equal(len(res.ValidatorSet.Validators), len(validatorSet.Validators))
	require.Equal(res.ValidatorSet.TotalVotingPower, validatorSet.TotalVotingPower)
}

func (s *KeeperTestSuite) TesthandleQuerySigner() {
	ctx, keeper, queryClient := s.ctx, s.stakeKeeper, s.queryClient
	require := s.Require()

	req := &types.QuerySignerRequest{
		ValAddress: common.Address{}.String(),
	}

	res, err := queryClient.Signer(ctx, req)
	// check no error found
	require.NotNil(err)

	testutil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)

	validators := keeper.GetAllValidators(ctx)

	req = &types.QuerySignerRequest{
		ValAddress: validators[0].Signer,
	}

	res, err = queryClient.Signer(ctx, req)
	// check no error found
	require.NoError(err)

	// check response is not nil
	require.Equal(res.Validator.Signer, validators[0].Signer)
	require.Equal(res.Validator.StartEpoch, validators[0].StartEpoch)
	require.Equal(res.Validator.EndEpoch, validators[0].EndEpoch)
	require.Equal(res.Validator.PubKey.Compare(validators[0].PubKey), 0)
	require.Equal(res.Validator.ProposerPriority, validators[0].ProposerPriority)
}

func (s *KeeperTestSuite) TesthandleQueryValidator() {
	ctx, keeper, queryClient := s.ctx, s.stakeKeeper, s.queryClient
	require := s.Require()
	req := &types.QueryValidatorRequest{
		Id: uint64(0),
	}

	res, err := queryClient.Validator(ctx, req)
	// check no error found
	require.NotNil(err)
	require.Nil(res)

	req = &types.QueryValidatorRequest{
		Id: uint64(1),
	}

	res, err = queryClient.Validator(ctx, req)
	// check no error found
	require.NotNil(err)
	require.Nil(res)

	testutil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)

	validators := keeper.GetAllValidators(ctx)

	req = &types.QueryValidatorRequest{
		Id: validators[0].ValId,
	}

	res, err = queryClient.Validator(ctx, req)
	// check no error found
	require.NoError(err)

	// check response is not nil
	require.Equal(res.Validator.Signer, validators[0].Signer)
	require.Equal(res.Validator.StartEpoch, validators[0].StartEpoch)
	require.Equal(res.Validator.EndEpoch, validators[0].EndEpoch)
	require.Equal(res.Validator.PubKey.Compare(validators[0].PubKey), 0)
	require.Equal(res.Validator.ProposerPriority, validators[0].ProposerPriority)
}

func (s *KeeperTestSuite) TestHandleQueryValidatorStatus() {
	ctx, keeper, queryClient := s.ctx, s.stakeKeeper, s.queryClient
	require := s.Require()
	testutil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	validators := keeper.GetAllValidators(ctx)

	req := &types.QueryValidatorStatusRequest{
		ValAddress: validators[0].Signer,
	}
	res, err := queryClient.ValidatorStatus(ctx, req)
	// check no error found
	require.NoError(err)

	// check response is not nil
	require.NotNil(res)
	require.True(res.Status)

	req = &types.QueryValidatorStatusRequest{
		ValAddress: common.Address{}.String(),
	}
	res, err = queryClient.ValidatorStatus(ctx, req)
	// check no error found
	require.Nil(err)
	require.False(res.Status)

}

// TODO H2 Recheck it
func (s *KeeperTestSuite) TestHandleCurrentQueryProposer() {
	ctx, keeper, queryClient := s.ctx, s.stakeKeeper, s.queryClient
	require := s.Require()
	validatorSet := testutil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)
	require.NotNil(validatorSet)

	req := &types.QueryCurrentProposerRequest{}

	res, err := queryClient.CurrentProposer(ctx, req)
	// check no error found
	require.NoError(err)
	require.NotNil(res)
}

// TODO HV2 Please look into this.
// func (s *KeeperTestSuite) TestHandleQueryMilestoneProposer() {
// 	ctx, keeper, queryClient := s.ctx, s.stakeKeeper, s.queryClient
// 	require := s.Require()
// 	testutil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)

// 	path := []string{types.QueryMilestoneProposer}

// 	route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryMilestoneProposer)

// 	req := abci.RequestQuery{
// 		Path: route,
// 		Data: app.Codec().MustMarshalJSON(types.NewQueryProposerParams(uint64(2))),
// 	}
// 	res, err := querier(ctx, path, req)
// 	// check no error found
// 	require.NoError(t, err)

// 	// check response is not nil
// 	require.NotNil(t, res)
// }

// func (s *KeeperTestSuite) TestHandleQueryCurrentProposer() {
// 	ctx, keeper, queryClient := s.ctx, s.stakeKeeper, s.queryClient
// 	require := s.Require()
// 	testutil.LoadValidatorSet(require, 4, keeper, ctx, false, 10)

// 	path := []string{types.QueryCurrentProposer}

// 	route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryCurrentProposer)

// 	req := abci.RequestQuery{
// 		Path: route,
// 		Data: []byte{},
// 	}
// 	res, err := querier(ctx, path, req)
// 	// check no error found
// 	require.NoError(t, err)

// 	// check response is not nil
// 	require.NotNil(t, res)
// }

// func (s *KeeperTestSuite) TestHandleQueryStakingSequence() {
// 	ctx, keeper, queryClient := s.ctx, s.stakeKeeper, s.queryClient
// 	s1 := rand.NewSource(time.Now().UnixNano())
// 	r1 := rand.New(s1)

// 	txHash := hmTypes.TxHash{make([]byte, 20)}

// 	txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}

// 	logIndex := uint64(simulation.RandIntBetween(r1, 0, 100))

// 	msg := types.NewQueryStakingSequenceParams(txHash.String(), logIndex)

// 	sequence := new(big.Int).Mul(txreceipt.BlockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
// 	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

// 	keeper.SetStakingSequence(ctx, sequence.String())

// 	path := []string{types.QueryStakingSequence}

// 	route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryStakingSequence)

// 	req := abci.RequestQuery{
// 		Path: route,
// 		Data: app.Codec().MustMarshalJSON(msg),
// 	}

// 	res, err := querier(ctx, path, req)
// 	// check no error found
// 	require.NoError(t, err)

// 	// check response is not nil
// 	require.NotNil(t, res)
// 	require.Equal(t, sequence.String(), string(res))
// }
