package keeper

import (
	"math/big"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	hmModule "github.com/0xPolygon/heimdall-v2/module"
	type2 "github.com/0xPolygon/heimdall-v2/types"
	types "github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

type sideMsgServer struct {
	Keeper
}

var (
	msgeventrecord = sdk.MsgTypeURL(&types.MsgEventRecord{})
)

// NewSideMsgServerImpl returns an implementation of the clerk MsgServer interface
// for the provided Keeper.
func NewSideMsgServerImpl(keeper Keeper) types.SideMsgServer {
	return &sideMsgServer{Keeper: keeper}
}

// SideTxHandler returns a side handler for "topup" type messages.
func (srv *sideMsgServer) SideTxHandler(methodName string) hmModule.SideTxHandler {
	switch methodName {
	case msgeventrecord:
		return srv.SideHandleMsgEventRecord
	default:
		return nil
	}
}

// PostTxHandler returns a side handler for "bank" type messages.
func (srv *sideMsgServer) PostTxHandler(methodName string) hmModule.PostTxHandler {
	switch methodName {
	case msgeventrecord:
		return srv.PostHandleMsgEventRecord
	default:
		return nil
	}
}

func (srv *sideMsgServer) SideHandleMsgEventRecord(ctx sdk.Context, _msg sdk.Msg) (result hmModule.Vote) {
	msg, ok := _msg.(*types.MsgEventRecord)
	if !ok {
		srv.Logger(ctx).Error("msg type mismatched")
		return hmModule.Vote_VOTE_NO
	}

	srv.Logger(ctx).Debug("✅ Validating External call for clerk msg",
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	// TODO HV2 - uncomment when contractCaller is implemented
	// // chainManager params
	// params, err := srv.ChainKeeper.GetParams(ctx)
	// if err != nil {
	// 	srv.Logger(ctx).Error("failed to get chain manager params", "error", err)
	// 	return hmModule.Vote_VOTE_NO
	// }

	// TODO HV2 - uncomment when contractCaller is implemented
	// chainParams := params.ChainParams
	// _ = params.ChainParams

	// TODO HV2 - uncomment when contractCaller is implemented
	// get confirmed tx receipt
	/*
		receipt, err := contractCaller.GetConfirmedTxReceipt(msg.TxHash, params.MainchainTxConfirmations)
		if receipt == nil || err != nil {
			return hmModule.Vote_VOTE_NO
		}
	*/

	// TODO HV2 - uncomment when contractCaller is implemented
	// get event log for topup
	/*
		eventLog, err := contractCaller.DecodeStateSyncedEvent(chainParams.StateSenderAddress.EthAddress(), receipt, msg.LogIndex)
		if err != nil || eventLog == nil {
			srv.Logger(ctx).Error("Error fetching log from txhash")
			return hmModule.Vote_VOTE_NO
		}
	*/

	// TODO HV2 - the following commented code depends on the results of the above code, uncomment when contractCaller is implemented
	/*
		if receipt.BlockNumber.Uint64() != msg.BlockNumber {
			srv.Logger(ctx).Error("BlockNumber in message doesn't match blocknumber in receipt", "MsgBlockNumber", msg.BlockNumber, "ReceiptBlockNumber", receipt.BlockNumber.Uint64())
			return hmModule.Vote_VOTE_NO
		}

		// check if message and event log matches
		if eventLog.Id.Uint64() != msg.ID {
			srv.Logger(ctx).Error("ID in message doesn't match with id in log", "msgId", msg.ID, "stateIdFromTx", eventLog.Id)
			return hmModule.Vote_VOTE_NO
		}

		if !bytes.Equal(eventLog.ContractAddress.Bytes(), msg.ContractAddress.Bytes()) {
			srv.Logger(ctx).Error(
				"ContractAddress from event does not match with Msg ContractAddress",
				"EventContractAddress", eventLog.ContractAddress.String(),
				"MsgContractAddress", msg.ContractAddress,
			)

			return hmModule.Vote_VOTE_NO
		}

		if !bytes.Equal(eventLog.Data, msg.Data.GetHexBytes()) {
			if ctx.BlockHeight() > helper.GetSpanOverrideHeight() {
				if !(len(eventLog.Data) > helper.MaxStateSyncSize && bytes.Equal(msg.Data.GetHexBytes(), hmModule.HexToHexBytes(""))) {
					srv.Logger(ctx).Error(
						"Data from event does not match with Msg Data",
						"EventData", hmModule.BytesToHexBytes(eventLog.Data),
						"MsgData", hmModule.BytesToHexBytes(msg.Data),
					)

					return hmModule.Vote_VOTE_NO
				}
			} else {
				if !(len(eventLog.Data) > helper.LegacyMaxStateSyncSize && bytes.Equal(msg.Data, hmModule.HexToHexBytes(""))) {
					srv.Logger(ctx).Error(
						"Data from event does not match with Msg Data",
						"EventData", hmModule.BytesToHexBytes(eventLog.Data),
						"MsgData", hmModule.BytesToHexBytes(msg.Data),
					)

					return hmModule.Vote_VOTE_NO
				}
			}
		}
	*/

	return hmModule.Vote_VOTE_YES
}

func (srv *sideMsgServer) PostHandleMsgEventRecord(ctx sdk.Context, _msg sdk.Msg, sideTxResult hmModule.Vote) {
	msg, ok := _msg.(*types.MsgEventRecord)
	if !ok {
		srv.Logger(ctx).Error("msg type mismatched")
	}

	// Skip handler if clerk is not approved
	if sideTxResult != hmModule.Vote_VOTE_YES {
		srv.Logger(ctx).Debug("Skipping new clerk since side-tx didn't get yes votes")
		return
	}

	// check for replay
	if srv.HasEventRecord(ctx, msg.ID) {
		srv.Logger(ctx).Debug("Skipping new clerk record as it's already processed")
		return
	}

	srv.Logger(ctx).Debug("Persisting clerk state", "sideTxResult", sideTxResult)

	// sequence id
	blockNumber := new(big.Int).SetUint64(msg.BlockNumber)
	sequence := new(big.Int).Mul(blockNumber, big.NewInt(type2.DefaultLogIndexUnit))
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
		srv.Logger(ctx).Error("Unable to update event record", "id", msg.ID, "error", err)
		return
	}

	// save record sequence
	srv.SetRecordSequence(ctx, sequence.String())

	// TX bytes
	txBytes := ctx.TxBytes()
	hash := type2.TxHash{Hash: txBytes}

	// add events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRecord,
			sdk.NewAttribute(sdk.AttributeKeyAction, msg.Type()),                   // action
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory), // module name
			sdk.NewAttribute(type2.AttributeKeyTxHash, hash.String()),              // tx hash
			sdk.NewAttribute(types.AttributeKeyRecordTxLogIndex, strconv.FormatUint(msg.LogIndex, 10)),
			sdk.NewAttribute(type2.AttributeKeySideTxResult, sideTxResult.String()), // result
			sdk.NewAttribute(types.AttributeKeyRecordID, strconv.FormatUint(msg.ID, 10)),
			sdk.NewAttribute(types.AttributeKeyRecordContract, msg.ContractAddress),
		),
	})
}
