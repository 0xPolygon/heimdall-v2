package keeper_test

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func (suite *KeeperTestSuite) TestHandleMsgEventRecord() {
	t, ctx, ck, chainID := suite.T(), suite.ctx, suite.keeper, suite.chainID

	s := rand.NewSource(1)
	r := rand.New(s)

	ac := address.NewHexCodec()

	addrBz1, err := ac.StringToBytes(Address1)
	require.NoError(t, err)

	addrBz2, err := ac.StringToBytes(Address2)
	require.NoError(t, err)

	id := r.Uint64()
	logIndex := r.Uint64()
	blockNumber := r.Uint64()

	// successful message
	msg := types.NewMsgEventRecord(
		addrBz1,
		TxHash1,
		logIndex,
		blockNumber,
		id,
		addrBz2,
		hmTypes.HexBytes{
			HexBytes: make([]byte, 0),
		},
		chainID,
	)

	t.Run("Success", func(t *testing.T) {
		_, err := suite.msgServer.HandleMsgEventRecord(ctx, &msg)
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
				msg.ID,
				msg.ContractAddress,
				msg.Data,
				msg.ChainID,
				tempTime,
			),
		)
		require.NoError(t, err)

		_, err = suite.msgServer.HandleMsgEventRecord(ctx, &msg)
		require.Error(t, err)
		require.Equal(t, types.ErrEventRecordAlreadySynced, err)
	})

	t.Run("EventSizeExceed", func(t *testing.T) {
		const letterBytes = "abcdefABCDEF"
		b := hmTypes.HexBytes{
			HexBytes: make([]byte, helper.MaxStateSyncSize+3),
		}
		for i := range b.HexBytes {
			b.HexBytes[i] = letterBytes[rand.Intn(len(letterBytes))]
		}

		msg.Data = b

		err := msg.ValidateBasic()
		require.Error(t, err)
	})
}

func (suite *KeeperTestSuite) TestHandleMsgEventRecordSequence() {
	t, ctx, ck, chainID := suite.T(), suite.ctx, suite.keeper, suite.chainID

	s := rand.NewSource(1)
	r := rand.New(s)

	ac := address.NewHexCodec()

	addrBz1, err := ac.StringToBytes(Address1)
	require.NoError(t, err)

	addrBz2, err := ac.StringToBytes(Address2)
	require.NoError(t, err)

	msg := types.NewMsgEventRecord(
		addrBz1,
		TxHash1,
		r.Uint64(),
		r.Uint64(),
		r.Uint64(),
		addrBz2,
		hmTypes.HexBytes{
			HexBytes: make([]byte, 0),
		},
		chainID,
	)

	// sequence id
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
	ck.SetRecordSequence(ctx, sequence.String())

	_, err = suite.msgServer.HandleMsgEventRecord(ctx, &msg)
	require.Error(t, err)
	// require.Equal(t, common.ErrOldTx(types.ModuleName), err)
}

func (suite *KeeperTestSuite) TestHandleMsgEventRecordChainID() {
	t, ctx, ck := suite.T(), suite.ctx, suite.keeper

	s := rand.NewSource(1)
	r := rand.New(s)

	ac := address.NewHexCodec()

	addrBz1, err := ac.StringToBytes(Address1)
	require.NoError(t, err)

	addrBz2, err := ac.StringToBytes(Address2)
	require.NoError(t, err)

	id := r.Uint64()

	// wrong chain id
	msg := types.NewMsgEventRecord(
		addrBz1,
		TxHash1,
		r.Uint64(),
		r.Uint64(),
		id,
		addrBz2,
		hmTypes.HexBytes{
			HexBytes: make([]byte, 0),
		},
		"random chain id",
	)
	_, err = suite.msgServer.HandleMsgEventRecord(ctx, &msg)
	require.Error(t, err)
	// require.Equal(t, common.ErrInvalidBorChainID(types.ModuleName), err)

	// there should be no stored event record
	storedEventRecord, err := ck.GetEventRecord(ctx, id)
	require.Nil(t, storedEventRecord)
	require.Error(t, err)
}
