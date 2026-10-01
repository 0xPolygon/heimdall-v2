package keeper_test

import (
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	util "github.com/0xPolygon/heimdall-v2/common/hex"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	clerkKeeper "github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
	"github.com/0xPolygon/heimdall-v2/x/clerk/testutil"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

func (s *KeeperTestSuite) sideHandler(ctx sdk.Context, msg sdk.Msg) sidetxs.Vote {
	cfg := s.sideMsgCfg
	return cfg.GetSideHandler(msg)(ctx, msg)
}

func (s *KeeperTestSuite) postHandler(ctx sdk.Context, msg sdk.Msg, vote sidetxs.Vote) {
	cfg := s.sideMsgCfg

	_ = cfg.GetPostHandler(msg)(ctx, msg, vote)
}

func (s *KeeperTestSuite) TestSideHandler() {
	t, ctx, ck, sideHandler, contractCaller, chainId := s.T(), s.ctx, s.keeper, s.sideHandler, &s.contractCaller, s.chainId

	r := rand.New(rand.NewSource(1))
	ac := address.NewHexCodec()

	addrBz2, err := ac.StringToBytes(Address2)
	require.NoError(t, err)

	id := r.Uint64()
	logIndex := r.Uint64()
	blockNumber := r.Uint64()

	ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)

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

	txReceipt := &ethTypes.Receipt{
		BlockNumber: new(big.Int).SetUint64(blockNumber),
	}

	contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(txReceipt, nil)
	event := &statesender.StatesenderStateSynced{
		Id:              new(big.Int).SetUint64(msg.Id),
		ContractAddress: common.HexToAddress(msg.ContractAddress),
		Data:            msg.Data,
	}
	contractCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil)

	result := sideHandler(ctx, &msg)
	require.Equal(t, sidetxs.Vote_VOTE_YES, result)
}

func (s *KeeperTestSuite) TestSideHandleMsgEventRecord() {
	t, ctx, ck, sideHandler, contractCaller, chainId := s.T(), s.ctx, s.keeper, s.sideHandler, &s.contractCaller, s.chainId

	r := rand.New(rand.NewSource(1))
	ac := address.NewHexCodec()

	addrBz2, err := ac.StringToBytes(Address2)
	require.NoError(t, err)

	id := r.Uint64()

	t.Run("Success", func(t *testing.T) {
		logIndex := uint64(10)
		blockNumber := uint64(600)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber),
		}

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

		// mock external calls
		contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(txReceipt, nil)
		event := &statesender.StatesenderStateSynced{
			Id:              new(big.Int).SetUint64(msg.Id),
			ContractAddress: common.HexToAddress(msg.ContractAddress),
			Data:            msg.Data,
		}
		contractCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil)

		ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
		// execute handler
		result := sideHandler(ctx, &msg)
		require.Equal(t, sidetxs.Vote_VOTE_YES, result)

		// there should be no stored event record
		storedEventRecord, err := ck.GetEventRecord(ctx, id)
		require.Nil(t, storedEventRecord)
		require.Error(t, err)
	})

	t.Run("NoReceipt", func(t *testing.T) {
		logIndex := uint64(200)
		blockNumber := uint64(51)

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

		// mock external calls -- no receipt
		contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
		contractCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

		// execute handler
		ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
		result := sideHandler(ctx, &msg)
		require.Equal(t, sidetxs.Vote_VOTE_NO, result)
	})

	t.Run("NoLog", func(t *testing.T) {
		logIndex := uint64(100)
		blockNumber := uint64(510)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber + 1),
		}

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

		// mock external calls -- no receipt
		contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(txReceipt, nil)
		contractCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

		ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
		// execute handler
		result := sideHandler(ctx, &msg)
		require.Equal(t, sidetxs.Vote_VOTE_NO, result)
	})

	t.Run("EventDataExceed", func(t *testing.T) {
		id := uint64(111)
		logIndex := uint64(1)
		blockNumber := uint64(1000)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber),
		}

		const letterBytes = "abcdefABCDEF"
		b := make([]byte, helper.MaxStateSyncSize+3)
		for i := range b {
			b[i] = letterBytes[rand.Intn(len(letterBytes))]
		}

		// data created after trimming
		msg := types.NewMsgEventRecord(
			util.FormatAddress(Address1),
			TxHash1,
			logIndex,
			blockNumber,
			id,
			addrBz2,
			[]byte(""),
			chainId,
		)

		// mock external calls
		contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(txReceipt, nil)
		event := &statesender.StatesenderStateSynced{
			Id:              new(big.Int).SetUint64(msg.Id),
			ContractAddress: common.BytesToAddress([]byte(msg.ContractAddress)),
			Data:            b,
		}
		contractCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil)

		ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
		// execute handler
		result := sideHandler(ctx, &msg)
		require.Equal(t, sidetxs.Vote_VOTE_NO, result)

		// there should be no stored event record
		storedEventRecord, err := ck.GetEventRecord(ctx, id)
		require.Nil(t, storedEventRecord)
		require.Error(t, err)
	})

	t.Run("ContractAddressMismatch", func(t *testing.T) {
		s.contractCaller.Mock = mock.Mock{}

		logIndex := uint64(7)
		blockNumber := uint64(600)
		txReceipt := &ethTypes.Receipt{
			BlockNumber: new(big.Int).SetUint64(blockNumber),
		}

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

		// event has a different contract address than the msg
		event := &statesender.StatesenderStateSynced{
			Id:              new(big.Int).SetUint64(msg.Id),
			ContractAddress: common.HexToAddress(Address1),
			Data:            msg.Data,
		}

		contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(txReceipt, nil).Once()
		contractCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil).Once()

		ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
		result := sideHandler(ctx, &msg)
		require.Equal(t, sidetxs.Vote_VOTE_NO, result)
	})
}

func (s *KeeperTestSuite) TestPostHandler() {
	t, ctx, postHandler, chainId := s.T(), s.ctx, s.postHandler, s.chainId

	r := rand.New(rand.NewSource(1))
	ac := address.NewHexCodec()

	addrBz2, err := ac.StringToBytes(Address2)
	require.NoError(t, err)

	id := r.Uint64()
	logIndex := r.Uint64()
	blockNumber := r.Uint64()

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

	// post-handler should fail
	postHandler(ctx, &msg, sidetxs.Vote_VOTE_YES)
}

func (s *KeeperTestSuite) TestPostHandleMsgEventRecord() {
	t, ctx, ck, postHandler, chainId := s.T(), s.ctx, s.keeper, s.postHandler, s.chainId

	r := rand.New(rand.NewSource(1))
	ac := address.NewHexCodec()

	addrBz2, err := ac.StringToBytes(Address2)
	require.NoError(t, err)

	id := r.Uint64()
	logIndex := r.Uint64()
	blockNumber := r.Uint64()

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

	t.Run("NoResult", func(t *testing.T) {
		// post-handler should fail
		postHandler(ctx, &msg, sidetxs.Vote_VOTE_NO)

		// there should be no stored event record
		storedEventRecord, err := ck.GetEventRecord(ctx, id)
		require.Nil(t, storedEventRecord)
		require.Error(t, err)
	})

	t.Run("YesResult", func(t *testing.T) {
		// post-handler should succeed
		postHandler(ctx, &msg, sidetxs.Vote_VOTE_YES)

		// sequence id
		blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
		sequence := new(big.Int).Mul(blockNumber, big.NewInt(hmTypes.DefaultLogIndexUnit))
		sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

		// check the sequence
		hasSequence := ck.HasRecordSequence(ctx, sequence.String())
		require.True(t, hasSequence, "Sequence should be stored correctly")

		// there should be no stored event record
		storedEventRecord, err := ck.GetEventRecord(ctx, id)
		require.NotNil(t, storedEventRecord)
		require.NoError(t, err)
		require.Equal(t, id, storedEventRecord.Id)
		require.Equal(t, logIndex, storedEventRecord.LogIndex)
	})

	t.Run("Replay", func(t *testing.T) {
		id := r.Uint64()
		logIndex := r.Uint64()
		blockNumber := r.Uint64()

		_ = types.NewMsgEventRecord(
			util.FormatAddress(Address1),
			TxHash1,
			logIndex,
			blockNumber,
			id,
			addrBz2,
			make([]byte, 0),
			chainId,
		)

		// post-handler should succeed
		postHandler(ctx, &msg, sidetxs.Vote_VOTE_YES)

		// post-handler should prevent replay attack
		postHandler(ctx, &msg, sidetxs.Vote_VOTE_YES)
	})
}

func (s *KeeperTestSuite) TestPostHandleMsgEventRecord_InvalidMsgTypeReturnsError() {
	ctx := s.ctx
	require := s.Require()

	postHandler := clerkKeeper.NewSideMsgServerImpl(&s.keeper).(interface {
		PostHandleMsgEventRecord(sdk.Context, sdk.Msg, sidetxs.Vote) error
	})

	require.NotPanics(func() {
		err := postHandler.PostHandleMsgEventRecord(ctx, nil, sidetxs.Vote_VOTE_YES)
		require.Error(err)
		require.Contains(err.Error(), "MsgEventRecord")
	})
}

func (s *KeeperTestSuite) TestPostHandleMsgEventRecord_ReplayReturnsError() {
	ctx, ck, chainId := s.ctx, s.keeper, s.chainId
	require := s.Require()

	postHandler := clerkKeeper.NewSideMsgServerImpl(&s.keeper).(interface {
		PostHandleMsgEventRecord(sdk.Context, sdk.Msg, sidetxs.Vote) error
	})

	ac := address.NewHexCodec()
	addrBz2, err := ac.StringToBytes(Address2)
	require.NoError(err)

	msg := types.NewMsgEventRecord(
		util.FormatAddress(Address1),
		TxHash1,
		1,
		1,
		1,
		addrBz2,
		make([]byte, 0),
		chainId,
	)

	err = postHandler.PostHandleMsgEventRecord(ctx, &msg, sidetxs.Vote_VOTE_YES)
	require.NoError(err)

	err = postHandler.PostHandleMsgEventRecord(ctx, &msg, sidetxs.Vote_VOTE_YES)
	require.Error(err)
	require.Contains(err.Error(), "already processed")

	storedEventRecord, getErr := ck.GetEventRecord(ctx, msg.Id)
	require.NoError(getErr)
	require.NotNil(storedEventRecord)
}

func (s *KeeperTestSuite) TestSideHandleMsgEventRecordTxHashActivation() {
	ctx, ck, contractCaller := s.ctx, s.keeper, &s.contractCaller
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
		600,
		1,
		contractAddress,
		nil,
		s.chainId,
	)

	receipt := &ethTypes.Receipt{BlockNumber: new(big.Int).SetUint64(msg.BlockNumber)}
	event := &statesender.StatesenderStateSynced{
		Id:              new(big.Int).SetUint64(msg.Id),
		ContractAddress: common.HexToAddress(msg.ContractAddress),
		Data:            msg.Data,
	}
	ck.ChainKeeper.(*testutil.MockChainKeeper).EXPECT().GetParams(gomock.Any()).Return(chainmanagertypes.DefaultParams(), nil).Times(1)
	contractCaller.On("GetConfirmedTxReceipt", mock.Anything, mock.Anything, mock.Anything).Return(receipt, nil).Once()
	contractCaller.On("DecodeStateSyncedEvent", mock.Anything, mock.Anything, mock.Anything).Return(event, nil).Once()

	require.Equal(sidetxs.Vote_VOTE_YES, s.sideHandler(ctx.WithBlockHeight(99), &msg))
	require.Equal(sidetxs.Vote_VOTE_NO, s.sideHandler(ctx.WithBlockHeight(100), &msg))
}

func (s *KeeperTestSuite) TestPostHandleMsgEventRecordTxHashActivation() {
	ctx, ck := s.ctx, s.keeper
	require := s.Require()
	originalLuganoHeight := helper.GetLuganoHeight()
	helper.SetLuganoHeight(100)
	s.T().Cleanup(func() { helper.SetLuganoHeight(originalLuganoHeight) })

	postHandler := clerkKeeper.NewSideMsgServerImpl(&ck).(interface {
		PostHandleMsgEventRecord(sdk.Context, sdk.Msg, sidetxs.Vote) error
	})
	ac := address.NewHexCodec()
	contractAddress, err := ac.StringToBytes(Address2)
	require.NoError(err)

	newMessage := func(id uint64, txHash string) types.MsgEventRecord {
		return types.NewMsgEventRecord(
			util.FormatAddress(Address1), txHash, id, id, id, contractAddress, nil, s.chainId,
		)
	}

	oversizedHash := "0x00" + TxHash1[2:]
	legacyMsg := newMessage(1001, oversizedHash)
	// A tx included at H-1 is finalized by the post-handler at H.
	err = postHandler.PostHandleMsgEventRecord(ctx.WithBlockHeight(100), &legacyMsg, sidetxs.Vote_VOTE_YES)
	require.NoError(err)
	stored, err := ck.GetEventRecord(ctx, legacyMsg.Id)
	require.NoError(err)
	require.Equal(oversizedHash, stored.TxHash)

	invalidMsg := newMessage(1002, oversizedHash)
	err = postHandler.PostHandleMsgEventRecord(ctx.WithBlockHeight(101), &invalidMsg, sidetxs.Vote_VOTE_YES)
	require.ErrorIs(err, types.ErrInvalidTxHash)
	require.False(ck.HasEventRecord(ctx, invalidMsg.Id))

	uppercaseMsg := newMessage(1003, strings.ToUpper(TxHash1))
	err = postHandler.PostHandleMsgEventRecord(ctx.WithBlockHeight(101), &uppercaseMsg, sidetxs.Vote_VOTE_YES)
	require.NoError(err)
	stored, err = ck.GetEventRecord(ctx, uppercaseMsg.Id)
	require.NoError(err)
	require.Equal(strings.ToUpper(TxHash1), stored.TxHash)
}

// contractAddressHomoglyph is U+2126 OHM SIGN, 3 UTF-8 bytes, distinct from and easily
// confused with U+03A9 GREEK CAPITAL LETTER OMEGA (2 bytes) -- only the former's
// strings.ToLower mapping (to U+03C9 GREEK SMALL LETTER OMEGA, 2 bytes) changes byte
// length, so the exact code point matters and is spelled out explicitly here.
const contractAddressHomoglyph = "Ω"

// TestPostHandleMsgEventRecord_ContractAddressNormalized reproduces the reported Heimdall
// vs Bor address-decoder mismatch. Appending contractAddressHomoglyph to an otherwise-valid
// contract address makes heimdall's HexCodec.StringToBytes (ToLower, then decode) resolve
// to the real 20 address bytes -- so the side handler would vote YES -- while go-ethereum's
// common.HexToAddress (no ToLower, used independently by Bor) resolves the SAME raw string
// to a different, nibble-shifted address, because FromHex's odd/even-length zero-pad
// decision flips between the two decodes. Once persisted verbatim, Bor would route the
// state sync to the wrong address permanently (HasEventRecord blocks any correction).
func (s *KeeperTestSuite) TestPostHandleMsgEventRecord_ContractAddressNormalized() {
	ctx, ck, chainId := s.ctx.WithBlockHeight(1), s.keeper, s.chainId
	require := s.Require()

	orig := helper.GetLuganoHeight()
	s.T().Cleanup(func() { helper.SetLuganoHeight(orig) })

	postHandler := clerkKeeper.NewSideMsgServerImpl(&s.keeper).(interface {
		PostHandleMsgEventRecord(sdk.Context, sdk.Msg, sidetxs.Vote) error
	})

	ac := address.NewHexCodec()

	realAddr := common.HexToAddress(util.FormatAddress(Address1))
	craftedContractAddress := realAddr.Hex() + contractAddressHomoglyph // homoglyph appended after a full, valid 40-hex-char address

	// The crafted string must decode, gate-side, to the real address bytes -- otherwise the
	// side handler would never have voted YES for it in the first place.
	gateBytes, err := ac.StringToBytes(craftedContractAddress)
	require.NoError(err)
	require.Equal(realAddr, common.BytesToAddress(gateBytes), "crafted address must pass the gate's byte-equality check")

	// ... and it must decode differently on Bor's independent, non-ToLower'd path, or this
	// isn't the reported bug at all.
	require.NotEqual(realAddr, common.HexToAddress(craftedContractAddress), "crafted address must fool Bor's decoder")

	// Built as a raw struct, not via NewMsgEventRecord: that constructor round-trips the
	// contract address through BytesToString and can never produce a malformed value. An
	// attacker's raw tx bytes decode straight into this struct with no such sanitization.
	msg := types.MsgEventRecord{
		From:            util.FormatAddress(Address1),
		TxHash:          TxHash1,
		LogIndex:        1,
		BlockNumber:     1,
		ContractAddress: craftedContractAddress,
		Data:            make([]byte, 0),
		Id:              42,
		ChainId:         chainId,
	}

	s.Run("PreLugano_StoresRawAndDivergesFromBor", func() {
		helper.SetLuganoHeight(0)
		require.NoError(postHandler.PostHandleMsgEventRecord(ctx, &msg, sidetxs.Vote_VOTE_YES))

		stored, getErr := ck.GetEventRecord(ctx, msg.Id)
		require.NoError(getErr)
		require.Equal(craftedContractAddress, stored.Contract, "pre-fork, the raw attacker string is persisted verbatim")
		require.NotEqual(realAddr, common.HexToAddress(stored.Contract), "pre-fork, Bor would misroute the state sync")
	})

	s.Run("TxIncludedJustBeforeLugano_PostHandlerHeightAtActivation_StoresRaw", func() {
		msg.Id = 43 // fresh id: HasEventRecord would otherwise reject a replay
		helper.SetLuganoHeight(2)
		// Tx included at height 1 (= luganoHeight-1, still pre-Lugano) is finalized by the
		// post-handler at height 2 (= luganoHeight). Without the -1 adjustment in
		// contractAddressForRecord, the post-handler's own execution height reaching
		// activation would incorrectly normalize this tx, diverging from the raw address
		// HandleMsgEventRecord already emitted for the identical tx at its true (pre-Lugano)
		// inclusion height -- the exact bug this test guards against.
		postCtx := ctx.WithBlockHeight(2)
		require.NoError(postHandler.PostHandleMsgEventRecord(postCtx, &msg, sidetxs.Vote_VOTE_YES))

		stored, getErr := ck.GetEventRecord(postCtx, msg.Id)
		require.NoError(getErr)
		require.Equal(craftedContractAddress, stored.Contract,
			"a tx included just before Lugano must still be stored raw, even though the post-handler's own height already reached activation")
	})

	s.Run("AtLugano_NormalizesAndAgreesWithBor", func() {
		msg.Id = 46 // fresh id: HasEventRecord would otherwise reject a replay
		helper.SetLuganoHeight(1)
		// A tx included at H-1 is finalized by the post-handler at H: contractAddressForRecord
		// gates on ctx.BlockHeight()-1, so the post-handler call representing a tx included at
		// the activation height itself must run one height above it.
		eventCtx := ctx.WithBlockHeight(2).WithEventManager(sdk.NewEventManager())
		require.NoError(postHandler.PostHandleMsgEventRecord(eventCtx, &msg, sidetxs.Vote_VOTE_YES))

		stored, getErr := ck.GetEventRecord(eventCtx, msg.Id)
		require.NoError(getErr)
		require.Equal(realAddr, common.HexToAddress(stored.Contract), "at/after Lugano, the stored address must agree with what Bor decodes")

		// The emitted event must not leak the raw attacker string either -- an indexer or
		// relayer reading AttributeKeyRecordContract from the event stream, rather than
		// querying the stored record, must see the same normalized address.
		var recordEvent sdk.Event
		var foundEvent bool
		for _, e := range eventCtx.EventManager().Events() {
			if e.Type == types.EventTypeRecord {
				recordEvent, foundEvent = e, true
				break
			}
		}
		require.True(foundEvent, "PostHandleMsgEventRecord must emit an EventTypeRecord event")
		attr, foundAttr := recordEvent.GetAttribute(types.AttributeKeyRecordContract)
		require.True(foundAttr, "emitted event must carry the contract attribute")
		require.Equal(stored.Contract, attr.Value, "emitted event's contract attribute must match the normalized, persisted address")
	})

	s.Run("AtLugano_UndecodableAddressReturnsErrorAndIsNotStored", func() {
		undecodable := msg
		undecodable.Id = 44
		undecodable.ContractAddress = "not-a-hex-address"
		helper.SetLuganoHeight(1)

		err := postHandler.PostHandleMsgEventRecord(ctx.WithBlockHeight(2), &undecodable, sidetxs.Vote_VOTE_YES)
		require.Error(err)

		stored, getErr := ck.GetEventRecord(ctx, undecodable.Id)
		require.Error(getErr)
		require.Nil(stored)
	})
}
