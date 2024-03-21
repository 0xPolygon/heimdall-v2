package keeper_test

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/0xPolygon/heimdall-v2/helper"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
	hexCodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func (suite *KeeperTestSuite) TestHandleMsgEventRecord() {
	t, app, ctx, chainID, r := suite.T(), suite.app, suite.ctx, suite.chainID, suite.r

	// TODO HV2 - let's use real data (in this whole file)
	addr1 := sdk.AccAddress([]byte("addr1"))

	id := r.Uint64()
	logIndex := r.Uint64()
	blockNumber := r.Uint64()

	txHashBytes, err := hexCodec.NewHexCodec().StringToBytes("123")
	require.NoError(t, err)
	// successful message
	msg := types.NewMsgEventRecord(
		addr1,
		hmTypes.HeimdallHash{Hash: txHashBytes},
		logIndex,
		blockNumber,
		id,
		addr1,
		hmTypes.HexBytes{
			HexBytes: make([]byte, 0),
		},
		chainID,
	)

	t.Run("Success", func(t *testing.T) {
		_, err := suite.msgServer.HandleMsgEventRecord(ctx, &msg)
		require.NoError(t, err)
		// TODO HV2 - the above check seems enough, we can remove the below commented lines
		// require.True(t, result.IsOK(), "expected msg record to be ok, got %v", result)

		// there should be no stored event record
		storedEventRecord, err := app.ClerkKeeper.GetEventRecord(ctx, id)
		require.Nil(t, storedEventRecord)
		require.Error(t, err)
	})

	t.Run("ExistingRecord", func(t *testing.T) {
		// store event record in keeper
		tempTime := time.Now()
		err := app.ClerkKeeper.SetEventRecord(ctx,
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
		// TODO HV2 - the above check seems enough, we can remove the below commented lines
		// require.False(t, result.IsOK(), "should fail due to existent event record but succeeded")
		// require.Equal(t, types.CodeEventRecordAlreadySynced, result.Code)
	})

	t.Run("EventSizeExceed", func(t *testing.T) {
		// TODO HV2 - uncomment when mock contract caller is implemented
		// suite.contractCaller = mocks.IContractCaller{}

		const letterBytes = "abcdefABCDEF"
		b := hmTypes.HexBytes{
			HexBytes: make([]byte, helper.LegacyMaxStateSyncSize+3),
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
	t, app, ctx, chainID, r := suite.T(), suite.app, suite.ctx, suite.chainID, suite.r

	addr1 := sdk.AccAddress([]byte("addr1"))

	txHashBytes, err := hexCodec.NewHexCodec().StringToBytes("123")
	require.NoError(t, err)

	msg := types.NewMsgEventRecord(
		addr1,
		hmTypes.HeimdallHash{Hash: txHashBytes},
		r.Uint64(),
		r.Uint64(),
		r.Uint64(),
		addr1,
		hmTypes.HexBytes{
			HexBytes: make([]byte, 0),
		},
		chainID,
	)

	// sequence id
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))
	app.ClerkKeeper.SetRecordSequence(ctx, sequence.String())

	_, err = suite.msgServer.HandleMsgEventRecord(ctx, &msg)
	require.Error(t, err)
	// TODO HV2 - the above check seems enough, we can remove the below commented lines
	// require.False(t, result.IsOK(), "should fail due to existent sequence but succeeded")
	// require.Equal(t, common.CodeOldTx, result.Code)
}

func (suite *KeeperTestSuite) TestHandleMsgEventRecordChainID() {
	t, app, ctx, r := suite.T(), suite.app, suite.ctx, suite.r

	addr1 := sdk.AccAddress([]byte("addr1"))

	id := r.Uint64()

	txHashBytes, err := hexCodec.NewHexCodec().StringToBytes("123")
	require.NoError(t, err)

	// wrong chain id
	msg := types.NewMsgEventRecord(
		addr1,
		hmTypes.HeimdallHash{Hash: txHashBytes},
		r.Uint64(),
		r.Uint64(),
		id,
		addr1,
		hmTypes.HexBytes{
			HexBytes: make([]byte, 0),
		},
		"random chain id",
	)
	_, err = suite.msgServer.HandleMsgEventRecord(ctx, &msg)
	require.Error(t, err)
	// TODO HV2 - the above check seems enough, we can remove the below commented lines
	// require.False(t, result.IsOK(), "error invalid bor chain id %v", result.Code)
	// require.Equal(t, common.CodeInvalidBorChainID, result.Code)

	// there should be no stored event record
	storedEventRecord, err := app.ClerkKeeper.GetEventRecord(ctx, id)
	require.Nil(t, storedEventRecord)
	require.Error(t, err)
}
