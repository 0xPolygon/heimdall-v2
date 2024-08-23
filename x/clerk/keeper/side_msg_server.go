package keeper

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/helper"
	hmModule "github.com/0xPolygon/heimdall-v2/module"
	heimdallTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

type sideMsgServer struct {
	Keeper
}

var (
	msgEventRecord = sdk.MsgTypeURL(&types.MsgEventRecordRequest{})
)

// NewSideMsgServerImpl returns an implementation of the clerk SideMsgServer interface
// for the provided Keeper.
func NewSideMsgServerImpl(keeper Keeper) types.SideMsgServer {
	return &sideMsgServer{Keeper: keeper}
}

// SideTxHandler returns a side handler for clerk type messages.
func (srv *sideMsgServer) SideTxHandler(methodName string) hmModule.SideTxHandler {
	switch methodName {
	case msgEventRecord:
		return srv.SideHandleMsgEventRecord
	default:
		return nil
	}
}

// PostTxHandler returns a post handler for clerk type messages.
func (srv *sideMsgServer) PostTxHandler(methodName string) hmModule.PostTxHandler {
	switch methodName {
	case msgEventRecord:
		return srv.PostHandleMsgEventRecord
	default:
		return nil
	}
}

func (srv *sideMsgServer) SideHandleMsgEventRecord(ctx sdk.Context, _msg sdk.Msg) (result hmModule.Vote) {
	msg, ok := _msg.(*types.MsgEventRecordRequest)
	if !ok {
		srv.Logger(ctx).Error("msg type mismatch for MsgEventRecordRequest")
		return hmModule.Vote_VOTE_NO
	}

	srv.Logger(ctx).Debug("✅ Validating External call for clerk msg",
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	// chainManager params
	params, err := srv.ChainKeeper.GetParams(ctx)
	if err != nil {
		srv.Logger(ctx).Error("failed to get chain manager params", "error", err)
		return hmModule.Vote_VOTE_NO
	}

	chainParams := params.ChainParams

	// get confirmed tx receipt
	receipt, err := srv.Keeper.contractCaller.GetConfirmedTxReceipt(common.HexToHash(msg.TxHash), params.GetMainChainTxConfirmations())
	if receipt == nil || err != nil {
		return hmModule.Vote_VOTE_NO
	}

	// get event log for clerk
	eventLog, err := srv.Keeper.contractCaller.DecodeStateSyncedEvent(chainParams.StateSenderAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		srv.Logger(ctx).Error("Error fetching log from tx hash")
		return hmModule.Vote_VOTE_NO
	}

	if receipt.BlockNumber.Uint64() != msg.BlockNumber {
		srv.Logger(ctx).Error("blockNumber in message doesn't match blockNumber in receipt", "MsgBlockNumber", msg.BlockNumber, "ReceiptBlockNumber", receipt.BlockNumber.Uint64())
		return hmModule.Vote_VOTE_NO
	}

	// check if message and event log matches
	if eventLog.Id.Uint64() != msg.ID {
		srv.Logger(ctx).Error("ID in message doesn't match with id in log", "msgId", msg.ID, "stateIdFromTx", eventLog.Id)
		return hmModule.Vote_VOTE_NO
	}

	// TODO HV2: ensure addresses/keys consistency (see https://polygon.atlassian.net/browse/POS-2622)
	msgContractAddr := common.HexToAddress(msg.ContractAddress)

	if !bytes.Equal(eventLog.ContractAddress.Bytes(), msgContractAddr.Bytes()) {
		srv.Logger(ctx).Error(
			"ContractAddress from event does not match with Msg ContractAddress",
			"EventContractAddress", eventLog.ContractAddress.String(),
			"MsgContractAddress", msg.ContractAddress,
		)

		return hmModule.Vote_VOTE_NO
	}

	if !bytes.Equal(eventLog.Data, msg.Data.GetHexBytes()) {
		if !(len(eventLog.Data) > helper.MaxStateSyncSize && bytes.Equal(msg.Data.HexBytes, []byte(""))) {
			srv.Logger(ctx).Error(
				"Data from event does not match with Msg Data",
				"EventData", hex.EncodeToString(eventLog.Data),
				"MsgData", msg.Data.String(),
			)

			return hmModule.Vote_VOTE_NO
		}
	}

	return hmModule.Vote_VOTE_YES
}

func (srv *sideMsgServer) PostHandleMsgEventRecord(ctx sdk.Context, _msg sdk.Msg, sideTxResult hmModule.Vote) {
	logger := srv.Logger(ctx)

	msg, ok := _msg.(*types.MsgEventRecordRequest)
	if !ok {
		logger.Error("msg type mismatch for MsgEventRecordRequest")
	}

	// Skip handler if clerk is not approved
	if sideTxResult != hmModule.Vote_VOTE_YES {
		logger.Debug("skipping new clerk since side-tx didn't get yes votes")
		return
	}

	// check for replay
	if srv.HasEventRecord(ctx, msg.ID) {
		logger.Debug("skipping new clerk record as it's already processed")
		return
	}

	logger.Debug("persisting clerk state", "sideTxResult", sideTxResult)

	// sequence id
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(heimdallTypes.DefaultLogIndexUnit))
	sequence.Add(sequence, new(big.Int).SetUint64(msg.LogIndex))

	// create event record
	record := types.NewEventRecord(
		msg.TxHash,
		msg.LogIndex,
		msg.ID,
		msg.ContractAddress,
		msg.Data,
		msg.ChainID,
		ctx.BlockTime(),
	)

	// save event into state
	if err := srv.SetEventRecord(ctx, record); err != nil {
		logger.Error("unable to update event record", "id", msg.ID, "error", err)
		return
	}

	// save record sequence
	srv.SetRecordSequence(ctx, sequence.String())

	// tx bytes
	txBytes := ctx.TxBytes()
	hash := heimdallTypes.TxHash{Hash: txBytes}

	// add events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRecord,
			sdk.NewAttribute(sdk.AttributeKeyAction, msg.Type()),                   // action
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory), // module name
			sdk.NewAttribute(heimdallTypes.AttributeKeyTxHash, hash.String()),      // tx hash
			sdk.NewAttribute(types.AttributeKeyRecordTxLogIndex, strconv.FormatUint(msg.LogIndex, 10)),
			sdk.NewAttribute(heimdallTypes.AttributeKeySideTxResult, sideTxResult.String()), // result
			sdk.NewAttribute(types.AttributeKeyRecordID, strconv.FormatUint(msg.ID, 10)),
			sdk.NewAttribute(types.AttributeKeyRecordContract, msg.ContractAddress),
		),
	})
}
