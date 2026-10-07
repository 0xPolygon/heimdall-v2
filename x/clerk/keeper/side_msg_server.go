package keeper

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics/api"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	heimdallTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

type sideMsgServer struct {
	*Keeper
}

var msgEventRecord = sdk.MsgTypeURL(&types.MsgEventRecord{})

// NewSideMsgServerImpl returns an implementation of the clerk SideMsgServer interface
// for the provided Keeper.
func NewSideMsgServerImpl(keeper *Keeper) sidetxs.SideMsgServer {
	return &sideMsgServer{Keeper: keeper}
}

// SideTxHandler returns a side handler for clerk type messages.
func (srv *sideMsgServer) SideTxHandler(methodName string) sidetxs.SideTxHandler {
	switch methodName {
	case msgEventRecord:
		return srv.SideHandleMsgEventRecord
	default:
		return nil
	}
}

// PostTxHandler returns a post-handler for clerk type messages.
func (srv *sideMsgServer) PostTxHandler(methodName string) sidetxs.PostTxHandler {
	switch methodName {
	case msgEventRecord:
		return srv.PostHandleMsgEventRecord
	default:
		return nil
	}
}

func (srv *sideMsgServer) SideHandleMsgEventRecord(ctx sdk.Context, m sdk.Msg) (result sidetxs.Vote) {
	var err error
	startTime := time.Now()
	defer recordClerkMetric(api.SideHandleMsgEventRecordMethod, api.SideType, startTime, &err)

	msg, ok := m.(*types.MsgEventRecord)
	if !ok {
		srv.Logger(ctx).Error(helper.ErrTypeMismatch("MsgEventRecord"))
		return sidetxs.Vote_VOTE_NO
	}
	if helper.IsLugano(ctx.BlockHeight()) {
		if err = msg.ValidateTxHash(); err != nil {
			return sidetxs.Vote_VOTE_NO
		}
	}

	srv.Logger(ctx).Debug(helper.LogValidatingExternalCall("ClerkEventRecord"),
		"txHash", msg.TxHash,
		"logIndex", msg.LogIndex,
		"blockNumber", msg.BlockNumber,
	)

	// check if the event record exists
	if exists := srv.HasEventRecord(ctx, msg.Id); exists {
		srv.Logger(ctx).Info("Msg event record already present in clerk side handler, voting NO")
		return sidetxs.Vote_VOTE_NO
	}

	// chainManager params
	params, err := srv.ChainKeeper.GetParams(ctx)
	if err != nil {
		srv.Logger(ctx).Error(heimdallTypes.ErrMsgFailedToGetChainManagerParams, heimdallTypes.LogKeyError, err)
		return sidetxs.Vote_VOTE_NO
	}

	chainParams := params.ChainParams

	// check chain id
	if !helper.ValidateChainID(msg.ChainId, chainParams.BorChainId, srv.Logger(ctx), "clerk") {
		return sidetxs.Vote_VOTE_NO
	}

	// sequence id
	sequence := helper.CalculateSequence(ctx.BlockHeight(), msg.BlockNumber, msg.LogIndex)

	// check if the event has already been processed
	if srv.HasRecordSequence(ctx, sequence) {
		srv.Logger(ctx).Error(helper.LogEventAlreadyProcessedIn("clerk"), heimdallTypes.LogKeySequence, sequence)
		return sidetxs.Vote_VOTE_NO
	}

	// get and validate confirmed tx receipt
	receipt := helper.FetchAndValidateReceipt(
		ctx,
		srv.contractCaller,
		helper.ReceiptValidationParams{
			TxHash:         common.HexToHash(msg.TxHash).Bytes(),
			MsgBlockNumber: msg.BlockNumber,
			Confirmations:  params.GetMainChainTxConfirmations(),
			ModuleName:     "clerk",
		},
		srv.Logger(ctx),
	)
	if receipt == nil {
		return sidetxs.Vote_VOTE_NO
	}

	// get event log for clerk
	eventLog, err := srv.contractCaller.DecodeStateSyncedEvent(chainParams.StateSenderAddress, receipt, msg.LogIndex)
	if err != nil || eventLog == nil {
		srv.Logger(ctx).Error(heimdallTypes.ErrMsgErrorFetchingLog)
		return sidetxs.Vote_VOTE_NO
	}

	// check if the message and the event log match
	if eventLog.Id.Uint64() != msg.Id {
		srv.Logger(ctx).Error(heimdallTypes.ErrMsgIDMismatch, heimdallTypes.LogKeyMsgID, msg.Id, "stateIdFromTx", eventLog.Id)
		return sidetxs.Vote_VOTE_NO
	}

	ac := address.NewHexCodec()
	msgContractAddrBytes, err := ac.StringToBytes(msg.ContractAddress)
	if err != nil {
		srv.Logger(ctx).Error(
			"Could not generate bytes from msg contract address",
			"MsgContractAddress", msg.ContractAddress,
		)
		return sidetxs.Vote_VOTE_NO
	}
	eventLogContractAddrBytes, err := ac.StringToBytes(eventLog.ContractAddress.String())
	if err != nil {
		srv.Logger(ctx).Error(
			"Could not generate bytes from event logs contract address",
			"EventContractAddress", eventLog.ContractAddress.String(),
		)
		return sidetxs.Vote_VOTE_NO
	}

	if !bytes.Equal(eventLogContractAddrBytes, msgContractAddrBytes) {
		srv.Logger(ctx).Error(
			"ContractAddress from event does not match with Msg ContractAddress",
			"EventContractAddress", eventLog.ContractAddress.String(),
			"MsgContractAddress", msg.ContractAddress,
		)

		return sidetxs.Vote_VOTE_NO
	}

	if !bytes.Equal(eventLog.Data, msg.Data) {
		if len(eventLog.Data) <= helper.MaxStateSyncSize || !bytes.Equal(msg.Data, []byte("")) {
			srv.Logger(ctx).Error(
				"Data from event does not match with Msg Data",
				"EventData", hex.EncodeToString(eventLog.Data),
				"MsgData", string(msg.Data),
			)

			return sidetxs.Vote_VOTE_NO
		}
	}

	return sidetxs.Vote_VOTE_YES
}

// normalizeContractAddress re-derives a contract address string from its decoded bytes,
// instead of trusting the attacker-controlled string verbatim: heimdall's own decode
// (ToLower, then hex-decode) can disagree with Bor's independent, non-ToLower'd decode of
// certain non-canonical strings, silently misrouting a state sync. Every consumer of the
// stored EventRecord must decode the same address heimdall validated against the L1 event.
func normalizeContractAddress(contractAddress string) (string, error) {
	ac := address.NewHexCodec()
	contractAddressBytes, err := ac.StringToBytes(contractAddress)
	if err != nil {
		return "", err
	}
	return ac.BytesToString(contractAddressBytes)
}

// contractAddressForRecord returns the contract address to persist for msg: the raw string
// below Lugano, or its normalized form at/after it. See normalizeContractAddress for why.
//
// Gated on ctx.BlockHeight()-1, not ctx.BlockHeight(): this runs inside the post-handler,
// which executes one height after the tx's own inclusion/side-handler height H (at H+1), so
// -1 recovers H -- the same height contractAddressForEvent uses for this same tx's emitted
// event, and the same convention persistApprovedEventRecord's own tx-hash gate already uses
// a few lines below. Without it, a tx included at LuganoHeight-1 would be normalized here
// (post-handler sees H+1=LuganoHeight) but not in the event HandleMsgEventRecord already
// emitted for it at H, producing a one-block window where the persisted record and its own
// emitted event disagree on the contract address for the identical transaction.
func (srv *sideMsgServer) contractAddressForRecord(ctx sdk.Context, msg *types.MsgEventRecord) (string, error) {
	if !helper.IsLugano(ctx.BlockHeight() - 1) {
		return msg.ContractAddress, nil
	}
	contractAddress, err := normalizeContractAddress(msg.ContractAddress)
	if err != nil {
		srv.Logger(ctx).Error("could not normalize contract address", "id", msg.Id, heimdallTypes.LogKeyError, err)
		return "", err
	}
	return contractAddress, nil
}

// contractAddressForEvent returns the contract address to emit in a clerk event for a
// handler that never persists state. Height-gated the same way as contractAddressForRecord,
// even though this handler has no app-hash to protect: HandleMsgEventRecord runs identically
// on every validator for the same tx as PostHandleMsgEventRecord's approved side-tx, and
// emitting a different contract address for the two events on the same tx pre-Lugano would be
// an unexplained inconsistency for anything reading the event stream (indexers, explorers)
// rather than stored state. Falls back to the raw string on decode failure rather than failing
// a handler with no state to roll back (ValidateBasic already rejects an undecodable address
// earlier in the tx pipeline, so that fallback should be unreachable in practice).
func contractAddressForEvent(height int64, contractAddress string) string {
	if !helper.IsLugano(height) {
		return contractAddress
	}
	if normalized, err := normalizeContractAddress(contractAddress); err == nil {
		return normalized
	}
	return contractAddress
}

func (srv *sideMsgServer) PostHandleMsgEventRecord(ctx sdk.Context, m sdk.Msg, sideTxResult sidetxs.Vote) error {
	var err error
	startTime := time.Now()
	defer recordClerkMetric(api.PostHandleMsgEventRecordMethod, api.PostType, startTime, &err)

	logger := srv.Logger(ctx)
	msg, ok := m.(*types.MsgEventRecord)
	if !ok {
		err = errors.New(helper.ErrTypeMismatch("MsgEventRecord"))
		logger.Error(err.Error())
		return err
	}

	if !helper.IsSideTxApproved(sideTxResult) {
		logger.Debug(helper.ErrSkippingMsg("ClerkEventRecord"))
		return nil
	}

	record, err := srv.persistApprovedEventRecord(ctx, msg)
	if err != nil {
		return err
	}

	emitEventRecord(ctx, msg, record.Contract, sideTxResult)
	return nil
}

func (srv *sideMsgServer) persistApprovedEventRecord(ctx sdk.Context, msg *types.MsgEventRecord) (*types.EventRecord, error) {
	if helper.IsLugano(ctx.BlockHeight() - 1) {
		if err := msg.ValidateTxHash(); err != nil {
			return nil, err
		}
	}

	if srv.HasEventRecord(ctx, msg.Id) {
		srv.Logger(ctx).Debug("Skipping new clerk record as it's already processed")
		return nil, errors.New("clerk record already processed")
	}
	srv.Logger(ctx).Debug("Persisting clerk state")

	sequence := helper.CalculateSequence(ctx.BlockHeight(), msg.BlockNumber, msg.LogIndex)
	contractAddress, err := srv.contractAddressForRecord(ctx, msg)
	if err != nil {
		return nil, err
	}

	record := types.NewEventRecord(
		msg.TxHash,
		msg.LogIndex,
		msg.Id,
		contractAddress,
		msg.Data,
		msg.ChainId,
		ctx.BlockTime(),
	)
	if err := srv.SetEventRecord(ctx, record); err != nil {
		srv.Logger(ctx).Error("Unable to update event record", "id", msg.Id, heimdallTypes.LogKeyError, err)
		return nil, err
	}

	if helper.IsZurichHardfork(ctx.BlockHeight()) {
		if err := srv.AddPendingVisibilityEvent(ctx, record.Id); err != nil {
			srv.Logger(ctx).Error("Unable to add pending visibility event", "id", record.Id, heimdallTypes.LogKeyError, err)
			return nil, err
		}
	}

	srv.SetRecordSequence(ctx, sequence)
	return &record, nil
}

func emitEventRecord(ctx sdk.Context, msg *types.MsgEventRecord, contractAddress string, sideTxResult sidetxs.Vote) {
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRecord,
			sdk.NewAttribute(sdk.AttributeKeyAction, msg.Type()),
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(heimdallTypes.AttributeKeyTxHash, common.Bytes2Hex(ctx.TxBytes())),
			sdk.NewAttribute(types.AttributeKeyRecordTxLogIndex, strconv.FormatUint(msg.LogIndex, 10)),
			sdk.NewAttribute(heimdallTypes.AttributeKeySideTxResult, sideTxResult.String()),
			sdk.NewAttribute(types.AttributeKeyRecordID, strconv.FormatUint(msg.Id, 10)),
			sdk.NewAttribute(types.AttributeKeyRecordContract, contractAddress),
		),
	})
}

// recordClerkMetric records metrics for side and post-handlers.
func recordClerkMetric(method string, apiType string, start time.Time, err *error) {
	success := *err == nil
	api.RecordAPICallWithStart(api.ClerkSubsystem, method, apiType, success, start)
}
