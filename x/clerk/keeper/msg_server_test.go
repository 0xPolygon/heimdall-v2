package keeper_test

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
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

		err = msg.ValidateBasic()
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

func (s *KeeperTestSuite) TestHandleMsgEventRecordTxHashActivation() {
	ctx, msgServer, chainID := s.ctx, s.msgServer, s.chainId
	require := s.Require()
	originalLuganoHeight := helper.GetLuganoHeight()
	helper.SetLuganoHeight(100)
	s.T().Cleanup(func() { helper.SetLuganoHeight(originalLuganoHeight) })

	ac := address.NewHexCodec()
	contractAddress, err := ac.StringToBytes(Address2)
	require.NoError(err)

	msg := types.NewMsgEventRecord(
		util.FormatAddress(Address1),
		"0x00"+TxHash1[2:],
		1,
		1,
		1,
		contractAddress,
		nil,
		chainID,
	)

	s.keeper.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
	_, err = msgServer.HandleMsgEventRecord(ctx.WithBlockHeight(99), &msg)
	require.NoError(err)

	_, err = msgServer.HandleMsgEventRecord(ctx.WithBlockHeight(100), &msg)
	require.ErrorIs(err, types.ErrInvalidTxHash)
}

// TestHandleMsgEventRecord_ContractAddressEventNormalized reproduces the same Heimdall vs
// Bor address-decoder divergence as TestPostHandleMsgEventRecord_ContractAddressNormalized,
// but on the plain (non-side-tx) message handler: HandleMsgEventRecord never persists an
// EventRecord, but it does emit one, and that emitted event must be height-gated exactly like
// PostHandleMsgEventRecord's -- otherwise the two handlers would emit different contract
// addresses for the same tx pre-Lugano, which is an unexplained inconsistency for anything
// reading the event stream even though it can't affect app hash.
func (s *KeeperTestSuite) TestHandleMsgEventRecord_ContractAddressEventNormalized() {
	t, ck, msgServer, chainId := s.T(), s.keeper, s.msgServer, s.chainId
	ctx := s.ctx.WithBlockHeight(1)
	ac := address.NewHexCodec()

	orig := helper.GetLuganoHeight()
	t.Cleanup(func() { helper.SetLuganoHeight(orig) })

	realAddr := common.HexToAddress(util.FormatAddress(Address1))
	craftedContractAddress := realAddr.Hex() + contractAddressHomoglyph

	gateBytes, err := ac.StringToBytes(craftedContractAddress)
	require.NoError(t, err)
	require.Equal(t, realAddr, common.BytesToAddress(gateBytes))
	require.NotEqual(t, realAddr, common.HexToAddress(craftedContractAddress))

	msg := types.MsgEventRecord{
		From:            util.FormatAddress(Address1),
		TxHash:          TxHash1,
		LogIndex:        1,
		BlockNumber:     1,
		ContractAddress: craftedContractAddress,
		Data:            make([]byte, 0),
		Id:              99,
		ChainId:         chainId,
	}

	recordContractAttr := func(eventCtx sdk.Context) string {
		var recordEvent sdk.Event
		var foundEvent bool
		for _, e := range eventCtx.EventManager().Events() {
			if e.Type == types.EventTypeRecord {
				recordEvent, foundEvent = e, true
				break
			}
		}
		require.True(t, foundEvent, "HandleMsgEventRecord must emit an EventTypeRecord event")
		attr, foundAttr := recordEvent.GetAttribute(types.AttributeKeyRecordContract)
		require.True(t, foundAttr, "emitted event must carry the contract attribute")
		return attr.Value
	}

	s.Run("PreLugano_EventLeaksRawAddress", func() {
		helper.SetLuganoHeight(0)
		eventCtx := ctx.WithEventManager(sdk.NewEventManager())
		ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
		_, err := msgServer.HandleMsgEventRecord(eventCtx, &msg)
		require.NoError(t, err)
		require.Equal(t, craftedContractAddress, recordContractAttr(eventCtx), "pre-fork, the raw attacker string is emitted verbatim, matching PostHandleMsgEventRecord's pre-fork behavior")
	})

	s.Run("AtLugano_EventNormalizesAndAgreesWithBor", func() {
		helper.SetLuganoHeight(1)
		eventCtx := ctx.WithEventManager(sdk.NewEventManager())
		ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
		_, err := msgServer.HandleMsgEventRecord(eventCtx, &msg)
		require.NoError(t, err)
		require.Equal(t, realAddr, common.HexToAddress(recordContractAttr(eventCtx)), "at/after Lugano, the emitted event's contract attribute must agree with what Bor decodes, not the raw attacker string")
	})
}
