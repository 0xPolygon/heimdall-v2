package keeper_test

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
	hmModule "github.com/0xPolygon/heimdall-v2/module"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func (suite *KeeperTestSuite) sideHandler(ctx sdk.Context, msg sdk.Msg) hmModule.Vote {
	cfg := suite.sideMsgCfg
	return cfg.GetSideHandler(msg)(ctx, msg)
}

func (suite *KeeperTestSuite) postHandler(ctx sdk.Context, msg sdk.Msg, vote hmModule.Vote) {
	cfg := suite.sideMsgCfg

	cfg.GetPostHandler(msg)(ctx, msg, vote)
}

// Test cases

func (suite *KeeperTestSuite) TestSideHandler() {
	t, ctx, chainID := suite.T(), suite.ctx, suite.chainID

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

	// side handler
	result := suite.sideHandler(ctx, &msg)
	require.Equal(t, hmModule.Vote_VOTE_YES, result)
}

func (suite *KeeperTestSuite) TestSideHandleMsgEventRecord() {
	t, ctx, ck := suite.T(), suite.ctx, suite.keeper

	chainParams, err := ck.ChainKeeper.GetParams(suite.ctx)
	require.NoError(t, err)

	s := rand.NewSource(1)
	r := rand.New(s)

	ac := address.NewHexCodec()

	addrBz1, err := ac.StringToBytes(Address1)
	require.NoError(t, err)

	addrBz2, err := ac.StringToBytes(Address2)
	require.NoError(t, err)

	id := r.Uint64()

	t.Run("Success", func(t *testing.T) {
		logIndex := uint64(10)
		blockNumber := uint64(599)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber + 1),
		}
		txHash := "success hash"

		msg := types.NewMsgEventRecord(
			addrBz1,
			txHash,
			logIndex,
			blockNumber,
			id,
			addrBz2,
			hmTypes.HexBytes{
				HexBytes: make([]byte, 0),
			},
			suite.chainID,
		)

		// mock external calls
		suite.contractCaller.On("GetConfirmedTxReceipt", common.HexToAddress(txHash), chainParams.GetMainChainTxConfirmations()).Return(txReceipt, nil)
		event := &statesender.StatesenderStateSynced{
			Id:              new(big.Int).SetUint64(msg.ID),
			ContractAddress: common.BytesToAddress([]byte(msg.ContractAddress)),
			Data:            msg.Data.HexBytes,
		}
		suite.contractCaller.On("DecodeStateSyncedEvent", chainParams.ChainParams.StateSenderAddress, txReceipt, logIndex).Return(event, nil)

		// execute handler
		result := suite.sideHandler(ctx, &msg)
		require.Equal(t, hmModule.Vote_VOTE_YES, result)

		// there should be no stored event record
		storedEventRecord, err := ck.GetEventRecord(ctx, id)
		require.Nil(t, storedEventRecord)
		require.Error(t, err)
	})

	t.Run("NoReceipt", func(t *testing.T) {
		logIndex := uint64(200)
		blockNumber := uint64(51)
		txHash := "no receipt hash"

		msg := types.NewMsgEventRecord(
			addrBz1,
			txHash,
			logIndex,
			blockNumber,
			id,
			addrBz2,
			hmTypes.HexBytes{
				HexBytes: make([]byte, 0),
			},
			suite.chainID,
		)

		// mock external calls -- no receipt
		suite.contractCaller.On("GetConfirmedTxReceipt", common.HexToAddress(txHash), chainParams.GetMainChainTxConfirmations()).Return(nil, nil)
		suite.contractCaller.On("DecodeStateSyncedEvent", chainParams.ChainParams.StateSenderAddress, nil, logIndex).Return(nil, nil)

		// execute handler
		result := suite.sideHandler(ctx, &msg)
		require.Equal(t, hmModule.Vote_VOTE_SKIP, result)
	})

	t.Run("NoLog", func(t *testing.T) {
		logIndex := uint64(100)
		blockNumber := uint64(510)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber + 1),
		}
		txHash := "no log hash"

		msg := types.NewMsgEventRecord(
			addrBz1,
			txHash,
			logIndex,
			blockNumber,
			id,
			addrBz2,
			hmTypes.HexBytes{
				HexBytes: make([]byte, 0),
			},
			suite.chainID,
		)

		// mock external calls -- no receipt
		suite.contractCaller.On("GetConfirmedTxReceipt", common.HexToAddress(txHash), chainParams.GetMainChainTxConfirmations()).Return(txReceipt, nil)
		suite.contractCaller.On("DecodeStateSyncedEvent", chainParams.ChainParams.StateSenderAddress, txReceipt, logIndex).Return(nil, nil)

		// execute handler
		result := suite.sideHandler(ctx, &msg)
		require.Equal(t, hmModule.Vote_VOTE_SKIP, result)
	})

	t.Run("EventDataExceed", func(t *testing.T) {
		id := uint64(111)
		logIndex := uint64(1)
		blockNumber := uint64(1000)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber),
		}
		txHash := "success hash"

		const letterBytes = "abcdefABCDEF"
		b := make([]byte, helper.MaxStateSyncSize+3)
		for i := range b {
			b[i] = letterBytes[rand.Intn(len(letterBytes))]
		}

		// data created after trimming
		msg := types.NewMsgEventRecord(
			addrBz1,
			txHash,
			logIndex,
			blockNumber,
			id,
			addrBz2,
			hmTypes.HexBytes{
				HexBytes: []byte(""),
			},
			suite.chainID,
		)

		// mock external calls
		suite.contractCaller.On("GetConfirmedTxReceipt", common.HexToAddress(txHash), chainParams.GetMainChainTxConfirmations()).Return(txReceipt, nil)
		event := &statesender.StatesenderStateSynced{
			Id:              new(big.Int).SetUint64(msg.ID),
			ContractAddress: common.BytesToAddress([]byte(msg.ContractAddress)),
			Data:            b,
		}
		suite.contractCaller.On("DecodeStateSyncedEvent", chainParams.ChainParams.StateSenderAddress, txReceipt, logIndex).Return(event, nil)

		// execute handler
		result := suite.sideHandler(ctx, &msg)
		require.Equal(t, hmModule.Vote_VOTE_YES, result)

		// there should be no stored event record
		storedEventRecord, err := ck.GetEventRecord(ctx, id)
		require.Nil(t, storedEventRecord)
		require.Error(t, err)
	})
}

func (suite *KeeperTestSuite) TestPostHandler() {
	t, ctx, chainID := suite.T(), suite.ctx, suite.chainID

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

	// Post handler should fail
	suite.postHandler(ctx, &msg, hmModule.Vote_VOTE_YES)
}

func (suite *KeeperTestSuite) TestPostHandleMsgEventRecord() {
	t, ctx, ck := suite.T(), suite.ctx, suite.keeper

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
		suite.chainID,
	)

	t.Run("NoResult", func(t *testing.T) {
		// Post handler should fail
		suite.postHandler(ctx, &msg, hmModule.Vote_VOTE_NO)

		// there should be no stored event record
		storedEventRecord, err := ck.GetEventRecord(ctx, id)
		require.Nil(t, storedEventRecord)
		require.Error(t, err)
	})

	t.Run("YesResult", func(t *testing.T) {
		// Post handler should succeed
		suite.postHandler(ctx, &msg, hmModule.Vote_VOTE_YES)

		// sequence id
		blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
		sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
		sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

		// check sequence
		hasSequence := ck.HasRecordSequence(ctx, sequence.String())
		require.True(t, hasSequence, "Sequence should be stored correctly")

		// there should be no stored event record
		storedEventRecord, err := ck.GetEventRecord(ctx, id)
		require.NotNil(t, storedEventRecord)
		require.NoError(t, err)
		require.Equal(t, id, storedEventRecord.ID)
		require.Equal(t, logIndex, storedEventRecord.LogIndex)
	})

	t.Run("Replay", func(t *testing.T) {
		id := r.Uint64()
		logIndex := r.Uint64()
		blockNumber := r.Uint64()

		_ = types.NewMsgEventRecord(
			addrBz1,
			TxHash1,
			logIndex,
			blockNumber,
			id,
			addrBz2,
			hmTypes.HexBytes{
				HexBytes: make([]byte, 0),
			},
			suite.chainID,
		)

		// Post handler should succeed
		suite.postHandler(ctx, &msg, hmModule.Vote_VOTE_YES)

		// Post handler should prevent replay attack
		suite.postHandler(ctx, &msg, hmModule.Vote_VOTE_YES)
	})
}
