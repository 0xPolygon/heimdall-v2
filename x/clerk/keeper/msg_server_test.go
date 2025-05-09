package keeper_test

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	util "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/helper"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/testutil"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func (s *KeeperTestSuite) TestHandleMsgEventRecord() {
	t, ctx, ck, msgServer, chainId := s.T(), s.ctx, s.keeper, s.msgServer, s.chainId

	r := rand.New(rand.NewSource(1))
	ac := address.NewHexCodec()

	addrBz2, err := ac.StringToBytes(Address2)
	require.NoError(t, err)

	id := r.Uint64()
	logIndex := r.Uint64()
	blockNumber := r.Uint64()

	// successful message
	msg := types.NewMsgEventRecord(
		util.FormatAddress(Address1),
		TxHash1,
		logIndex,
		blockNumber,
		id,
		addrBz2,
		make([]byte, 0),
		chainId,
	)

	t.Run("Success", func(t *testing.T) {
		ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
		_, err := msgServer.HandleMsgEventRecord(ctx, &msg)
		require.NoError(t, err)

		// there should be no stored event record
		storedEventRecord, err := ck.GetEventRecord(ctx, id)
		require.Nil(t, storedEventRecord)
		require.Error(t, err)
	})

	t.Run("ExistingRecord", func(t *testing.T) {
		// store event record in keeper
		tempTime := time.Now()
		err := ck.SetEventRecord(ctx,
			types.NewEventRecord(
				msg.TxHash,
				msg.LogIndex,
				msg.Id,
				msg.ContractAddress,
				msg.Data,
				msg.ChainId,
				tempTime,
			),
		)
		require.NoError(t, err)

		ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
		_, err = msgServer.HandleMsgEventRecord(ctx, &msg)
		require.Error(t, err)
		require.Equal(t, types.ErrEventRecordAlreadySynced, err)
	})

	t.Run("EventSizeExceed", func(t *testing.T) {
		const letterBytes = "abcdefABCDEF"
		b := make([]byte, helper.MaxStateSyncSize+3)
		for i := range b {
			b[i] = letterBytes[rand.Intn(len(letterBytes))]
		}

		msg.Data = b

		err := msg.ValidateBasic()
		require.Error(t, err)
	})
}

func (s *KeeperTestSuite) TestHandleMsgEventRecordSequence() {
	t, ctx, ck, msgServer, chainId := s.T(), s.ctx, s.keeper, s.msgServer, s.chainId

	r := rand.New(rand.NewSource(1))
	ac := address.NewHexCodec()

	addrBz2, err := ac.StringToBytes(Address2)
	require.NoError(t, err)

	msg := types.NewMsgEventRecord(
		util.FormatAddress(Address1),
		TxHash1,
		r.Uint64(),
		r.Uint64(),
		r.Uint64(),
		addrBz2,
		make([]byte, 0),
		chainId,
	)

	// sequence id
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
	ck.SetRecordSequence(ctx, sequence.String())

	ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
	_, err = msgServer.HandleMsgEventRecord(ctx, &msg)
	require.Error(t, err)
}

func (s *KeeperTestSuite) TestHandleMsgEventRecordChainID() {
	t, ctx, ck, msgServer := s.T(), s.ctx, s.keeper, s.msgServer

	r := rand.New(rand.NewSource(1))
	ac := address.NewHexCodec()

	addrBz2, err := ac.StringToBytes(Address2)
	require.NoError(t, err)

	id := r.Uint64()

	// wrong chain id
	msg := types.NewMsgEventRecord(
		util.FormatAddress(Address1),
		TxHash1,
		r.Uint64(),
		r.Uint64(),
		id,
		addrBz2,
		make([]byte, 0),
		"random chain id",
	)

	ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
	_, err = msgServer.HandleMsgEventRecord(ctx, &msg)
	require.Error(t, err)

	// there should be no stored event record
	storedEventRecord, err := ck.GetEventRecord(ctx, id)
	require.Nil(t, storedEventRecord)
	require.Error(t, err)
}
