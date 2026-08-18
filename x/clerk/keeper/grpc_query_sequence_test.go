package keeper_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"

	"github.com/0xPolygon/heimdall-v2/helper"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/testutil"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

// The sequence-existence queries must derive the lookup key with the same gated helper the
// write path uses, so a stored record is found and an unstored log index is reported missing.
func (s *KeeperTestSuite) TestGRPCRecordSequenceQueries() {
	ctx, ck, queryClient, require := s.ctx, s.keeper, s.queryClient, s.Require()

	receipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
	const storedLogIndex = uint64(5)
	storedSeq := helper.CalculateSequence(sdk.UnwrapSDKContext(ctx).BlockHeight(), receipt.BlockNumber.Uint64(), storedLogIndex)
	ck.SetRecordSequence(ctx, storedSeq)

	ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).AnyTimes()
	cc := &s.contractCaller
	cc.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(receipt, nil)

	wantSeq := receipt.BlockNumber.Uint64()*uint64(hmTypes.DefaultLogIndexUnit) + storedLogIndex

	s.Run("GetRecordSequence returns the stored sequence", func() {
		res, err := queryClient.GetRecordSequence(ctx, &types.RecordSequenceRequest{TxHash: TxHash1, LogIndex: storedLogIndex})
		require.NoError(err)
		require.NotNil(res)
		require.Equal(wantSeq, res.Sequence)
	})
	s.Run("GetRecordSequence rejects an unstored sequence", func() {
		res, err := queryClient.GetRecordSequence(ctx, &types.RecordSequenceRequest{TxHash: TxHash1, LogIndex: storedLogIndex + 1})
		require.Error(err)
		require.Nil(res)
	})
	s.Run("IsClerkTxOld is true for a stored sequence", func() {
		res, err := queryClient.IsClerkTxOld(ctx, &types.RecordSequenceRequest{TxHash: TxHash1, LogIndex: storedLogIndex})
		require.NoError(err)
		require.NotNil(res)
		require.True(res.IsOld)
	})
	s.Run("IsClerkTxOld errors for an unstored sequence", func() {
		res, err := queryClient.IsClerkTxOld(ctx, &types.RecordSequenceRequest{TxHash: TxHash1, LogIndex: storedLogIndex + 1})
		require.Error(err)
		require.Nil(res)
	})
}
