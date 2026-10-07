package keeper_test

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmosTestutil "github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	cmTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	checkpointKeeper "github.com/0xPolygon/heimdall-v2/x/checkpoint/keeper"
	chSim "github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
)

const testLuganoHeight = int64(100)

func (s *KeeperTestSuite) activateLuganoAt(height int64) {
	orig := helper.GetLuganoHeight()
	helper.SetLuganoHeight(height)
	s.T().Cleanup(func() { helper.SetLuganoHeight(orig) })
}

func (s *KeeperTestSuite) TestValidateCheckpointWindow() {
	require, keeper := s.Require(), s.checkpointKeeper
	s.activateLuganoAt(testLuganoHeight)

	params, err := keeper.GetParams(s.ctx)
	require.NoError(err)

	avg, max := params.AvgCheckpointLength, params.MaxCheckpointLength

	tests := []struct {
		name       string
		height     int64
		start, end uint64
		wantErr    bool
	}{
		{name: "single block window before fork", height: testLuganoHeight - 1, start: 1000, end: 1000},
		{name: "over max window before fork", height: testLuganoHeight - 1, start: 1000, end: 1000 + max},
		{name: "single block window at fork", height: testLuganoHeight, start: 1000, end: 1000, wantErr: true},
		{name: "single block window after fork", height: testLuganoHeight + 1, start: 1000, end: 1000, wantErr: true},
		{name: "one over avg at fork", height: testLuganoHeight, start: 1000, end: 1000 + avg, wantErr: true},
		{name: "over max at fork", height: testLuganoHeight, start: 1000, end: 1000 + max, wantErr: true},
		{name: "avg window at fork", height: testLuganoHeight, start: 1000, end: 1000 + avg - 1},
		{name: "multiple of avg at fork", height: testLuganoHeight, start: 1000, end: 1000 + 3*avg - 1},
		{name: "max window at fork", height: testLuganoHeight, start: 1000, end: 1000 + max - 1},
	}

	for _, tt := range tests {
		err := keeper.ValidateCheckpointWindow(s.ctx, tt.height, tt.start, tt.end)
		if tt.wantErr {
			require.ErrorIs(err, types.ErrInvalidCheckpointLength, tt.name)
			continue
		}
		require.NoError(err, tt.name)
	}
}

func (s *KeeperTestSuite) TestHandleMsgCheckpointWindow() {
	require, keeper, msgServer := s.Require(), s.checkpointKeeper, s.msgServer
	s.activateLuganoAt(testLuganoHeight)

	params, err := keeper.GetParams(s.ctx)
	require.NoError(err)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	s.stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	s.topupKeeper.EXPECT().GetAllDividendAccounts(gomock.Any()).AnyTimes().Return(chSim.RandDividendAccounts(), nil)

	dividendAccounts, err := s.topupKeeper.GetAllDividendAccounts(s.ctx)
	require.NoError(err)
	accRootHash, err := hmTypes.GetAccountRootHash(dividendAccounts)
	require.NoError(err)

	msgFor := func(end uint64) *types.MsgCheckpoint {
		return types.NewMsgCheckpointBlock(
			validatorSet.Proposer.Signer, 0, end, chSim.RandomBytes(), accRootHash, TestBorChainID,
		)
	}

	tests := []struct {
		name    string
		height  int64
		end     uint64
		wantErr bool
	}{
		{name: "one over avg before fork", height: testLuganoHeight - 1, end: params.AvgCheckpointLength},
		{name: "one over avg at fork", height: testLuganoHeight, end: params.AvgCheckpointLength, wantErr: true},
		{name: "over max at fork", height: testLuganoHeight, end: params.MaxCheckpointLength, wantErr: true},
		{name: "avg window at fork", height: testLuganoHeight, end: params.AvgCheckpointLength - 1},
		{name: "one over avg after fork", height: testLuganoHeight + 1, end: params.AvgCheckpointLength, wantErr: true},
	}

	for _, tt := range tests {
		ctx := s.ctx.WithBlockHeight(tt.height)

		res, err := msgServer.Checkpoint(ctx, msgFor(tt.end))
		if tt.wantErr {
			require.ErrorIs(err, types.ErrInvalidCheckpointLength, tt.name)
			continue
		}
		require.NoError(err, tt.name)
		require.NotNil(res, tt.name)
	}
}

func (s *KeeperTestSuite) TestSideHandleMsgCheckpointWindow() {
	require, keeper, contractCaller := s.Require(), s.checkpointKeeper, s.contractCaller
	s.activateLuganoAt(testLuganoHeight)

	params, err := keeper.GetParams(s.ctx)
	require.NoError(err)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	s.stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	s.cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)
	s.topupKeeper.EXPECT().GetAllDividendAccounts(gomock.Any()).AnyTimes().Return(chSim.RandDividendAccounts(), nil)

	dividendAccounts, err := s.topupKeeper.GetAllDividendAccounts(s.ctx)
	require.NoError(err)
	accRootHash, err := hmTypes.GetAccountRootHash(dividendAccounts)
	require.NoError(err)

	chainParams, err := s.cmKeeper.GetParams(s.ctx)
	require.NoError(err)

	rootHash := chSim.RandomBytes()
	msgFor := func(end uint64) *types.MsgCheckpoint {
		return types.NewMsgCheckpointBlock(
			validatorSet.Proposer.Signer, 0, end, rootHash, accRootHash,
			chainParams.ChainParams.BorChainId,
		)
	}

	mockBorFor := func(end uint64) {
		contractCaller.Mock = mock.Mock{}
		contractCaller.On("CheckIfBlocksExist", mock.Anything, end+chainParams.BorChainTxConfirmations).Return(true, nil)
		contractCaller.On("GetRootHash", mock.Anything, uint64(0), end, params.MaxCheckpointLength).Return(rootHash, nil)
	}

	// Windows sized off 2*avg keep the shared root-hash cache keys distinct from the
	// ones the other side-handler tests populate.
	misaligned, aligned := 2*params.AvgCheckpointLength, 2*params.AvgCheckpointLength-1

	s.Run("misaligned window is accepted before the fork", func() {
		mockBorFor(misaligned)

		ctx := s.ctx.WithBlockHeight(testLuganoHeight - 1)
		require.Equal(sidetxs.Vote_VOTE_YES, s.sideHandler(ctx, msgFor(misaligned)))
	})

	s.Run("misaligned window is rejected at the fork", func() {
		mockBorFor(misaligned)

		// The preceding case left this window's root hash in the package-level cache, so a
		// YES here would mean the gate never ran.
		ctx := s.ctx.WithBlockHeight(testLuganoHeight)
		require.Equal(sidetxs.Vote_VOTE_NO, s.sideHandler(ctx, msgFor(misaligned)))
	})

	s.Run("aligned window is accepted at the fork", func() {
		mockBorFor(aligned)

		ctx := s.ctx.WithBlockHeight(testLuganoHeight)
		require.Equal(sidetxs.Vote_VOTE_YES, s.sideHandler(ctx, msgFor(aligned)))
	})

	s.Run("misaligned window is still rejected after the fork", func() {
		mockBorFor(misaligned)

		ctx := s.ctx.WithBlockHeight(testLuganoHeight + 1)
		require.Equal(sidetxs.Vote_VOTE_NO, s.sideHandler(ctx, msgFor(misaligned)))
	})
}

func (s *KeeperTestSuite) TestPostHandleMsgCheckpointWindow() {
	require, keeper := s.Require(), s.checkpointKeeper
	s.activateLuganoAt(testLuganoHeight)

	params, err := keeper.GetParams(s.ctx)
	require.NoError(err)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	s.stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	s.stakeKeeper.EXPECT().GetCurrentProposer(gomock.Any()).AnyTimes().Return(validatorSet.Proposer)
	s.cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)

	rootHash := chSim.RandomBytes()
	msg := types.NewMsgCheckpointBlock(
		validatorSet.Proposer.Signer, 0, params.AvgCheckpointLength, rootHash, rootHash, TestBorChainID,
	)

	// The post handler runs one height after the block carrying the tx, so it is the tx
	// height that decides the gate.
	postForTxHeight := func(txHeight int64) error {
		ctx := s.ctx.WithBlockHeight(txHeight + 1)
		return s.sideMsgCfg.GetPostHandler(msg)(ctx, msg, sidetxs.Vote_VOTE_YES)
	}

	s.Run("misaligned window is buffered when the tx predates the fork", func() {
		require.NoError(postForTxHeight(testLuganoHeight - 1))

		buffered, err := keeper.GetCheckpointFromBuffer(s.ctx)
		require.NoError(err)
		require.Equal(params.AvgCheckpointLength, buffered.EndBlock)

		require.NoError(keeper.FlushCheckpointBuffer(s.ctx))
	})

	s.Run("misaligned window is not buffered when the tx is at or after the fork", func() {
		for _, txHeight := range []int64{testLuganoHeight, testLuganoHeight + 1} {
			require.ErrorIs(postForTxHeight(txHeight), types.ErrInvalidCheckpointLength, "txHeight=%d", txHeight)

			buffered, err := keeper.GetCheckpointFromBuffer(s.ctx)
			require.NoError(err)
			require.Equal(types.Checkpoint{}, buffered)
		}
	})
}

// The message handler runs at the tx's own height and the post handler one height later;
// both gates have to reach the same verdict for the same transaction, or a window can be
// accepted into the block and then dropped on the way to the buffer.
func (s *KeeperTestSuite) TestMessageAndPostHandlerAgreeOnTxHeight() {
	require, keeper, msgServer := s.Require(), s.checkpointKeeper, s.msgServer
	s.activateLuganoAt(testLuganoHeight)

	params, err := keeper.GetParams(s.ctx)
	require.NoError(err)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	s.stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	s.stakeKeeper.EXPECT().GetCurrentProposer(gomock.Any()).AnyTimes().Return(validatorSet.Proposer)
	s.cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)
	s.topupKeeper.EXPECT().GetAllDividendAccounts(gomock.Any()).AnyTimes().Return(chSim.RandDividendAccounts(), nil)

	dividendAccounts, err := s.topupKeeper.GetAllDividendAccounts(s.ctx)
	require.NoError(err)
	accRootHash, err := hmTypes.GetAccountRootHash(dividendAccounts)
	require.NoError(err)

	msg := types.NewMsgCheckpointBlock(
		validatorSet.Proposer.Signer, 0, params.AvgCheckpointLength, chSim.RandomBytes(), accRootHash, TestBorChainID,
	)

	for _, txHeight := range []int64{testLuganoHeight - 1, testLuganoHeight, testLuganoHeight + 1} {
		_, msgErr := msgServer.Checkpoint(s.ctx.WithBlockHeight(txHeight), msg)
		postErr := s.sideMsgCfg.GetPostHandler(msg)(s.ctx.WithBlockHeight(txHeight+1), msg, sidetxs.Vote_VOTE_YES)

		if txHeight < testLuganoHeight {
			require.NoError(msgErr, "txHeight=%d", txHeight)
			require.NoError(postErr, "txHeight=%d", txHeight)
			require.NoError(keeper.FlushCheckpointBuffer(s.ctx))
			continue
		}
		require.ErrorIs(msgErr, types.ErrInvalidCheckpointLength, "txHeight=%d", txHeight)
		require.ErrorIs(postErr, types.ErrInvalidCheckpointLength, "txHeight=%d", txHeight)
	}
}

// A window approved under one set of params is re-checked by the post handler against
// whatever params the store holds a height later, so a governance update landing in
// between discards the approved submission. Deterministic on every validator, and the
// same shape as the continuity re-check above it, which the ack post handler's own
// AddCheckpoint can invalidate within the same PreBlocker. The bridge re-proposes under
// the new params on the next turn.
func (s *KeeperTestSuite) TestPostHandlerUsesParamsAtItsOwnHeight() {
	require, keeper := s.Require(), s.checkpointKeeper
	s.activateLuganoAt(testLuganoHeight)

	params, err := keeper.GetParams(s.ctx)
	require.NoError(err)
	avg := params.AvgCheckpointLength

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	s.stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	s.stakeKeeper.EXPECT().GetCurrentProposer(gomock.Any()).AnyTimes().Return(validatorSet.Proposer)
	s.cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)

	rootHash := chSim.RandomBytes()
	// Exactly avg blocks: accepted under avg, rejected once avg doubles.
	msg := types.NewMsgCheckpointBlock(
		validatorSet.Proposer.Signer, 0, avg-1, rootHash, rootHash, TestBorChainID,
	)
	post := func() error {
		return s.sideMsgCfg.GetPostHandler(msg)(s.ctx.WithBlockHeight(testLuganoHeight+1), msg, sidetxs.Vote_VOTE_YES)
	}

	require.NoError(keeper.ValidateCheckpointWindow(s.ctx, testLuganoHeight, 0, avg-1),
		"the window has to be valid at the height it was proposed at")

	s.Run("unchanged params buffer the approved window", func() {
		require.NoError(post())

		buffered, err := keeper.GetCheckpointFromBuffer(s.ctx)
		require.NoError(err)
		require.Equal(avg-1, buffered.EndBlock)
		require.NoError(keeper.FlushCheckpointBuffer(s.ctx))
	})

	s.Run("params doubled in between discard it", func() {
		doubled := params
		doubled.AvgCheckpointLength = 2 * avg
		require.NoError(keeper.SetParams(s.ctx, doubled))

		require.ErrorIs(post(), types.ErrInvalidCheckpointLength)

		buffered, err := keeper.GetCheckpointFromBuffer(s.ctx)
		require.NoError(err)
		require.Equal(types.Checkpoint{}, buffered)
	})
}

func (s *KeeperTestSuite) TestQueryNextCheckpointWindow() {
	require, keeper, contractCaller := s.Require(), s.checkpointKeeper, s.contractCaller
	s.activateLuganoAt(testLuganoHeight)

	params, err := keeper.GetParams(s.ctx)
	require.NoError(err)

	validatorSet := stakeSim.GetRandomValidatorSet(2)
	s.stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)
	s.cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(cmTypes.DefaultParams(), nil)
	s.topupKeeper.EXPECT().GetAllDividendAccounts(gomock.Any()).AnyTimes().Return(chSim.RandDividendAccounts(), nil)

	contractCaller.Mock = mock.Mock{}
	contractCaller.On("GetRootHash", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(chSim.RandomBytes(), nil)

	// The CLI submits this window verbatim, so it has to pass both ValidateBasic and the
	// handlers' own length check.
	queryServer := checkpointKeeper.NewQueryServer(keeper)

	res, err := queryServer.GetNextCheckpoint(s.ctx, &types.QueryNextCheckpointRequest{})
	require.NoError(err)
	require.Equal(params.AvgCheckpointLength-1, res.Checkpoint.EndBlock)
	require.NoError(res.Checkpoint.ValidateBasic())
	require.NoError(keeper.ValidateCheckpointWindow(
		s.ctx, testLuganoHeight, res.Checkpoint.StartBlock, res.Checkpoint.EndBlock,
	))

	// An avg of 1 is a legal param set. The window still has to be submittable.
	s.Run("avg of one still yields a submittable window", func() {
		single := params
		single.AvgCheckpointLength = 1
		require.NoError(keeper.SetParams(s.ctx, single))

		res, err := queryServer.GetNextCheckpoint(s.ctx, &types.QueryNextCheckpointRequest{})
		require.NoError(err)
		require.Greater(res.Checkpoint.EndBlock, res.Checkpoint.StartBlock)
		require.NoError(res.Checkpoint.ValidateBasic())
		require.NoError(keeper.ValidateCheckpointWindow(
			s.ctx, testLuganoHeight, res.Checkpoint.StartBlock, res.Checkpoint.EndBlock,
		))
	})
}

func (s *KeeperTestSuite) TestValidateCheckpointWindowParamsUnavailable() {
	require := s.Require()
	s.activateLuganoAt(testLuganoHeight)

	// A keeper over a store that never went through InitGenesis has no params, so the
	// window bounds can't be derived and the check has to fail closed.
	key := storetypes.NewKVStoreKey("checkpoint_without_params")
	testCtx := cosmosTestutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_without_params"))
	encCfg := moduletestutil.MakeTestEncodingConfig()

	keeper := checkpointKeeper.NewKeeper(
		encCfg.Codec,
		runtime.NewKVStoreService(key),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		s.stakeKeeper,
		s.cmKeeper,
		s.topupKeeper,
		s.contractCaller,
	)

	err := keeper.ValidateCheckpointWindow(testCtx.Ctx, testLuganoHeight, 0, 255)
	require.ErrorIs(err, types.ErrCheckpointParams)
}
