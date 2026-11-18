package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	cmTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/testutil"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakeSim "github.com/0xPolygon/heimdall-v2/x/stake/testutil"
)

// ExtendVote(H) reads the state committed at H-1, so it never sees a buffer that PreBlocker(H) installs.
// A YES voted on at H must not replace such a buffer at H+1, even once H+1 is past its expiry.
func (s *KeeperTestSuite) TestPostHandleMsgCheckpointBufferInstalledAfterVote() {
	require, keeper := s.Require(), s.checkpointKeeper
	const voteHeight = int64(10)

	params, err := keeper.GetParams(s.ctx)
	require.NoError(err)
	bufferTime := params.CheckpointBufferTime

	chainParams := cmTypes.DefaultParams()
	s.cmKeeper.EXPECT().GetParams(gomock.Any()).AnyTimes().Return(chainParams, nil)
	s.topupKeeper.EXPECT().GetAllDividendAccounts(gomock.Any()).AnyTimes().Return([]hmTypes.DividendAccount{
		{User: "0x000000000000000000000000000000000000dEaD", FeeAmount: "1000"},
	}, nil)
	validatorSet := stakeSim.GetRandomValidatorSet(2)
	s.stakeKeeper.EXPECT().GetValidatorSet(gomock.Any()).AnyTimes().Return(validatorSet, nil)

	accountRoot := common.Hex2Bytes("b1c33eede53d5c39b5a2da9f1b5fe2bb49f7dbbbd98ac238b43294381eeb838a")
	// The Bor root cache is package-global, so keep this window apart from other tests'.
	window := testutil.GenRandCheckpoint(0, 383, 1)
	borRoot := window.RootHash
	newMsg := func(root []byte) *types.MsgCheckpoint {
		return types.NewMsgCheckpointBlock(validatorSet.Proposer.Signer, window.StartBlock, window.EndBlock, root, accountRoot, chainParams.ChainParams.BorChainId)
	}
	installed, replacement := newMsg(testutil.RandomBytes()), newMsg(borRoot)

	s.contractCaller.Mock = mock.Mock{}
	s.contractCaller.On("CheckIfBlocksExist", mock.Anything, window.EndBlock+chainParams.BorChainTxConfirmations).Return(true, nil)
	s.contractCaller.On("GetRootHash", mock.Anything, window.StartBlock, window.EndBlock, uint64(1024)).Return(borRoot, nil)

	orig := helper.GetPurleyHeight()
	s.T().Cleanup(func() { helper.SetPurleyHeight(orig) })
	helper.SetPurleyHeight(1)

	postHandler := s.sideMsgCfg.GetPostHandler(replacement)
	at := func(height int64, t time.Time) sdk.Context { return s.ctx.WithBlockHeight(height).WithBlockTime(t) }
	bufferRoot := func() []byte {
		got, err := keeper.GetCheckpointFromBuffer(s.ctx)
		require.NoError(err)
		return got.RootHash
	}

	tVote := time.Unix(1_700_000_000, 0)
	require.NoError(keeper.FlushCheckpointBuffer(s.ctx))

	// ExtendVote(H): the buffer is still empty.
	require.Equal(sidetxs.Vote_VOTE_YES, s.sideHandler(at(voteHeight, tVote), replacement))
	// PreBlocker(H): the checkpoint voted on at H-1 is installed.
	require.NoError(postHandler(at(voteHeight, tVote), installed, sidetxs.Vote_VOTE_YES))
	// FinalizeBlock(H): the msg handler sees that buffer unexpired.
	_, err = s.msgServer.Checkpoint(at(voteHeight, tVote), replacement)
	require.ErrorIs(err, types.ErrNoAck)
	// PreBlocker(H+1), a full buffer time later.
	require.Error(postHandler(at(voteHeight+1, tVote.Add(bufferTime)), replacement, sidetxs.Vote_VOTE_YES))
	require.Equal(installed.RootHash, bufferRoot())

	// A buffer installed at H-1 was visible to the vote at H and is replaced once expired.
	require.NoError(keeper.FlushCheckpointBuffer(s.ctx))
	require.NoError(postHandler(at(voteHeight-1, tVote), installed, sidetxs.Vote_VOTE_YES))
	tExpired := tVote.Add(bufferTime)
	require.Equal(sidetxs.Vote_VOTE_YES, s.sideHandler(at(voteHeight, tExpired), replacement))
	require.NoError(postHandler(at(voteHeight+1, tExpired.Add(time.Second)), replacement, sidetxs.Vote_VOTE_YES))
	require.Equal(replacement.RootHash, bufferRoot())
}
