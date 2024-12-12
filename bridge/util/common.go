package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	addressCodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/pkg/errors"

	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	clerktypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

type BridgeEvent string

const (
	AccountDetailsURL      = "/cosmos/auth/v1beta1/accounts/%v"
	AccountParamsURL       = "/cosmos/auth/v1beta1/params"
	LastNoAckURL           = "/checkpoints/last-no-ack"
	CheckpointParamsURL    = "/checkpoints/params"
	MilestoneCountURL      = "/milestone/count"
	ChainManagerParamsURL  = "/chainmanager/params"
	ProposersURL           = "/stake/proposer/%v"
	MilestoneProposersURL  = "/milestone/proposer/%v"
	BufferedCheckpointURL  = "/checkpoints/buffer"
	LatestCheckpointURL    = "/checkpoints/latest"
	LatestMilestoneURL     = "/milestone/latest"
	CountCheckpointURL     = "/checkpoints/count"
	CurrentProposerURL     = "/checkpoint/proposers/current"
	LatestSpanURL          = "/bor/span-latest"
	NextSpanInfoURL        = "/bor/span-prepare/%v"
	NextSpanSeedURL        = "/bor/span-seed"
	DividendAccountRootURL = "/topup/dividend-account-root"
	ValidatorURL           = "/stake/validator/%v"
	CurrentValidatorSetURL = "/stake/validator-set"
	StakingTxStatusURL     = "/stake/is-old-tx"
	TopupTxStatusURL       = "/topup/isoldtx"
	ClerkTxStatusURL       = "/clerk/isoldtx"
	ClerkEventRecordURL    = "/clerk/event-record/%d"
	/* HV2 - not adding slashing
	LatestSlashInfoBytesURL = "/slashing/latest_slash_info_bytes"
	TickSlashInfoListURL    = "/slashing/tick_slash_infos"
	SlashingTxStatusURL     = "/slashing/isoldtx"
	SlashingTickCountURL    = "/slashing/tick-count"
	*/

	CometBFTUnconfirmedTxsURL      = "/unconfirmed_txs"
	CometBFTUnconfirmedTxsCountURL = "/num_unconfirmed_txs"

	TransactionTimeout      = 1 * time.Minute
	CommitTimeout           = 2 * time.Minute
	TaskDelayBetweenEachVal = 10 * time.Second
	RetryTaskDelay          = 12 * time.Second
	RetryStateSyncTaskDelay = 24 * time.Second

	mempoolTxnCountDivisor = 1000

	// Bridge event types
	StakingEvent BridgeEvent = "staking"
	TopupEvent   BridgeEvent = "topup"
	ClerkEvent   BridgeEvent = "clerk"
	/* HV2 - not adding slashing
	SlashingEvent BridgeEvent = "slashing"
	*/

	BridgeDBFlag = "bridge-db"
)

// Logger returns logger singleton instance
func Logger() log.Logger {
	return log.NewNopLogger().With("module", "bridge")
}

// IsProposer checks if we are proposer
func IsProposer() (bool, error) {
	logger := Logger()
	var (
		proposers []staketypes.Validator
		count     = uint64(1)
	)

	result, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(fmt.Sprintf(ProposersURL, strconv.FormatUint(count, 10))))
	if err != nil {
		logger.Error("Error fetching proposers", "url", ProposersURL, "error", err)
		return false, err
	}

	err = json.Unmarshal(result, &proposers)
	if err != nil {
		logger.Error("error unmarshalling proposer slice", "error", err)
		return false, err
	}

	ac := addressCodec.NewHexCodec()
	signerBytes, err := ac.StringToBytes(proposers[0].Signer)
	if err != nil {
		logger.Error("Error converting signer string to bytes", "error", err)
		return false, err
	}
	if bytes.Equal(signerBytes, helper.GetAddress()) {
		return true, nil
	}

	return false, nil
}

// IsMilestoneProposer checks if we are the milestone proposer
func IsMilestoneProposer() (bool, error) {
	logger := Logger()

	var (
		proposers []staketypes.Validator
		count     = uint64(1)
	)

	result, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(fmt.Sprintf(MilestoneProposersURL, strconv.FormatUint(count, 10))))
	if err != nil {
		logger.Error("Error fetching milestone proposers", "url", MilestoneProposersURL, "error", err)
		return false, err
	}

	err = json.Unmarshal(result, &proposers)
	if err != nil {
		logger.Error("error unmarshalling milestone proposer slice", "error", err)
		return false, err
	}

	if len(proposers) == 0 {
		logger.Error("length of proposer list is 0")
		return false, errors.Errorf("Length of proposer list is 0")
	}

	ac := addressCodec.NewHexCodec()
	signerBytes, err := ac.StringToBytes(proposers[0].Signer)
	if err != nil {
		logger.Error("Error converting signer string to bytes", "error", err)
		return false, err
	}
	if bytes.Equal(signerBytes, helper.GetAddress()) {
		return true, nil
	}

	return false, nil
}

// IsInProposerList checks if we are in the current proposers list
func IsInProposerList(count uint64) (bool, error) {
	logger := Logger()

	logger.Debug("Skipping proposers", "count", strconv.FormatUint(count+1, 10))

	response, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(fmt.Sprintf(ProposersURL, strconv.FormatUint(count+1, 10))))
	if err != nil {
		logger.Error("Unable to send request for next proposers", "url", ProposersURL, "error", err)
		return false, err
	}

	// unmarshall data from buffer
	var proposers []staketypes.Validator
	if err := json.Unmarshal(response, &proposers); err != nil {
		logger.Error("Error unmarshalling validator data ", "error", err)
		return false, err
	}

	logger.Debug("Fetched proposers list", "numberOfProposers", count+1)

	ac := addressCodec.NewHexCodec()

	for i := 1; i <= int(count) && i < len(proposers); i++ {
		signerBytes, err := ac.StringToBytes(proposers[i].Signer)
		if err != nil {
			logger.Error("Error converting signer string to bytes", "error", err)
			return false, err
		}
		if bytes.Equal(signerBytes, helper.GetAddress()) {
			return true, nil
		}
	}

	return false, nil
}

// IsInMilestoneProposerList checks if we are in the current milestone proposers list
func IsInMilestoneProposerList(count uint64) (bool, error) {
	logger := Logger()

	logger.Debug("Skipping proposers", "count", strconv.FormatUint(count, 10))

	response, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(fmt.Sprintf(MilestoneProposersURL, strconv.FormatUint(count, 10))))
	if err != nil {
		logger.Error("Unable to send request for next proposers", "url", MilestoneProposersURL, "error", err)
		return false, err
	}

	// unmarshall data from buffer
	var proposers []staketypes.Validator
	if err := json.Unmarshal(response, &proposers); err != nil {
		logger.Error("Error unmarshalling validator data ", "error", err)
		return false, err
	}

	logger.Debug("Fetched proposers list", "numberOfProposers", count)

	ac := addressCodec.NewHexCodec()

	for i := 1; i <= int(count) && i < len(proposers); i++ {
		signerBytes, err := ac.StringToBytes(proposers[i].Signer)
		if err != nil {
			logger.Error("Error converting signer string to bytes", "error", err)
			return false, err
		}
		if bytes.Equal(signerBytes, helper.GetAddress()) {
			return true, nil
		}
	}

	return false, nil
}

// CalculateTaskDelay calculates delay required for current validator to propose the tx
// It solves for multiple validators sending same transaction.
func CalculateTaskDelay(event interface{}) (bool, time.Duration) {
	logger := Logger()

	defer LogElapsedTimeForStateSyncedEvent(event, "CalculateTaskDelay", time.Now())

	// calculate validator position
	valPosition := 0
	isCurrentValidator := false

	validatorSet, err := GetValidatorSet()
	if err != nil {
		logger.Error("Error getting current validatorset data ", "error", err)
		return false, 0
	}

	logger.Info("Fetched current validator set list", "currentValidatorCount", len(validatorSet.Validators))

	ac := addressCodec.NewHexCodec()
	for i, validator := range validatorSet.Validators {
		signerBytes, err := ac.StringToBytes(validator.Signer)
		if err != nil {
			logger.Error("Error converting signer string to bytes", "error", err)
			return false, 0
		}
		if bytes.Equal(signerBytes, helper.GetAddress()) {
			valPosition = i + 1
			isCurrentValidator = true

			break
		}
	}

	// Change calculation later as per the discussion
	// Currently it will multiply delay for every 1000 unconfirmed txns in mempool
	// For example if the current default delay is 12 Seconds
	// Then for upto 1000 txns it will stay as 12 only
	// For 1000-2000 It will be 24 seconds
	// For 2000-3000 it will be 36 seconds
	// Basically for every 1000 txns it will increase the factor by 1.

	mempoolFactor := GetUnconfirmedTxnCount(event) / mempoolTxnCountDivisor

	// calculate delay
	taskDelay := time.Duration(valPosition) * TaskDelayBetweenEachVal * time.Duration(mempoolFactor+1)

	return isCurrentValidator, taskDelay
}

// IsCurrentProposer checks if we are current proposer
func IsCurrentProposer() (bool, error) {
	logger := Logger()

	var proposer staketypes.Validator

	result, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(CurrentProposerURL))
	if err != nil {
		logger.Error("Error fetching proposers", "error", err)
		return false, err
	}

	if err = json.Unmarshal(result, &proposer); err != nil {
		logger.Error("error unmarshalling validator", "error", err)
		return false, err
	}

	logger.Debug("Current proposer fetched", "validator", proposer.String())

	ac := addressCodec.NewHexCodec()
	signerBytes, err := ac.StringToBytes(proposer.Signer)
	if err != nil {
		logger.Error("Error converting signer string to bytes", "error", err)
		return false, err
	}
	if bytes.Equal(signerBytes, helper.GetAddress()) {
		return true, nil
	}

	logger.Debug("We are not the current proposer")

	return false, nil
}

// IsEventSender checks if the validatorID belongs to the event sender
func IsEventSender(validatorID uint64) bool {
	logger := Logger()

	var validator staketypes.Validator

	result, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(fmt.Sprintf(ValidatorURL, strconv.FormatUint(validatorID, 10))))
	if err != nil {
		logger.Error("Error fetching proposers", "error", err)
		return false
	}

	if err = json.Unmarshal(result, &validator); err != nil {
		logger.Error("error unmarshalling proposer slice", "error", err)
		return false
	}

	logger.Debug("Current event sender received", "validator", validator.String())

	ac := addressCodec.NewHexCodec()
	signerBytes, err := ac.StringToBytes(validator.Signer)
	if err != nil {
		logger.Error("Error converting signer string to bytes", "error", err)
		return false
	}
	return bytes.Equal(signerBytes, helper.GetAddress())
}

// CreateURLWithQuery receives the uri and parameters in key value form
// it will return the new url with the given query from the parameter
func CreateURLWithQuery(uri string, param map[string]interface{}) (string, error) {
	urlObj, err := url.Parse(uri)
	if err != nil {
		return uri, err
	}

	query := urlObj.Query()
	for k, v := range param {
		query.Set(k, fmt.Sprintf("%v", v))
	}

	urlObj.RawQuery = query.Encode()

	return urlObj.String(), nil
}

// IsCatchingUp checks if the heimdall node you are connected to is fully synced or not
// returns true when synced
func IsCatchingUp(cliCtx client.Context) bool {
	resp, err := helper.GetNodeStatus(cliCtx)
	if err != nil {
		return true
	}

	return resp.SyncInfo.CatchingUp
}

// GetAccount returns heimdall auth account
func GetAccount(cliCtx client.Context, address string) (sdk.AccountI, error) {
	var account sdk.AccountI
	cmt := helper.GetConfig().CometBFTRPCUrl
	rpc, err := client.NewClientFromNode(cmt)
	if err != nil {
		panic(err)
	}
	cliCtx = cliCtx.WithClient(rpc)

	queryClient := authtypes.NewQueryClient(cliCtx)
	res, err := queryClient.Account(context.Background(), &authtypes.QueryAccountRequest{Address: address})
	if err != nil {
		return nil, err
	}

	if err := cliCtx.InterfaceRegistry.UnpackAny(res.Account, &account); err != nil {
		return nil, err
	}

	return account, nil
}

// GetChainmanagerParams return chain manager params
func GetChainmanagerParams(cdc codec.Codec) (*chainmanagertypes.Params, error) {
	logger := Logger()

	response, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(ChainManagerParamsURL))
	if err != nil {
		logger.Error("Error fetching chainmanager params", "err", err)
		return nil, err
	}

	var params chainmanagertypes.QueryParamsResponse
	if err = cdc.UnmarshalJSON(response, &params); err != nil {
		logger.Error("Error unmarshalling chainmanager params", "url", ChainManagerParamsURL, "err", err)
		return nil, err
	}

	return &params.Params, nil
}

// GetCheckpointParams return checkpoint params
func GetCheckpointParams() (*checkpointTypes.Params, error) {
	logger := Logger()

	response, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(CheckpointParamsURL))
	if err != nil {
		logger.Error("Error fetching Checkpoint params", "err", err)
		return nil, err
	}

	var params checkpointTypes.Params
	if err := json.Unmarshal(response, &params); err != nil {
		logger.Error("Error unmarshalling Checkpoint params", "url", CheckpointParamsURL)
		return nil, err
	}

	return &params, nil
}

// GetBufferedCheckpoint return checkpoint from buffer
func GetBufferedCheckpoint() (*checkpointTypes.Checkpoint, error) {
	logger := Logger()

	response, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(BufferedCheckpointURL))
	if err != nil {
		logger.Debug("Error fetching buffered checkpoint", "err", err)
		return nil, err
	}

	var checkpoint checkpointTypes.Checkpoint
	if err := json.Unmarshal(response, &checkpoint); err != nil {
		logger.Error("Error unmarshalling buffered checkpoint", "url", BufferedCheckpointURL, "err", err)
		return nil, err
	}

	return &checkpoint, nil
}

// GetLatestCheckpoint return last successful checkpoint
func GetLatestCheckpoint() (*checkpointTypes.Checkpoint, error) {
	logger := Logger()

	response, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(LatestCheckpointURL))
	if err != nil {
		logger.Debug("Error fetching latest checkpoint", "err", err)
		return nil, err
	}

	var checkpoint checkpointTypes.Checkpoint
	if err = json.Unmarshal(response, &checkpoint); err != nil {
		logger.Error("Error unmarshalling latest checkpoint", "url", LatestCheckpointURL, "err", err)
		return nil, err
	}

	return &checkpoint, nil
}

// GetLatestMilestone return last successful milestone
func GetLatestMilestone() (*milestoneTypes.Milestone, error) {
	logger := Logger()

	response, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(LatestMilestoneURL))
	if err != nil {
		logger.Debug("Error fetching latest milestone", "err", err)
		return nil, err
	}

	var milestone milestoneTypes.Milestone
	if err = json.Unmarshal(response, &milestone); err != nil {
		logger.Error("Error unmarshalling latest milestone", "url", LatestMilestoneURL, "err", err)
		return nil, err
	}

	return &milestone, nil
}

// GetMilestoneCount return milestones count
func GetMilestoneCount() (*milestoneTypes.MilestoneCount, error) {
	logger := Logger()

	response, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(MilestoneCountURL))
	if err != nil {
		logger.Error("Error fetching Milestone count", "err", err)
		return nil, err
	}

	var count milestoneTypes.MilestoneCount
	if err := json.Unmarshal(response, &count); err != nil {
		logger.Error("Error unmarshalling milestone Count", "url", MilestoneCountURL)
		return nil, err
	}

	return &count, nil
}

// AppendPrefix returns PublicKey in uncompressed format
func AppendPrefix(signerPubKey []byte) []byte {
	// append prefix - "0x04" as heimdall uses publickey in uncompressed format. Refer below link
	// https://superuser.com/questions/1465455/what-is-the-size-of-public-key-for-ecdsa-spec256r1
	prefix := make([]byte, 1)
	prefix[0] = byte(0x04)
	signerPubKey = append(prefix[:], signerPubKey[:]...)

	return signerPubKey
}

// GetValidatorNonce fetches validator nonce and height
func GetValidatorNonce(validatorID uint64) (uint64, error) {
	logger := Logger()

	var validator staketypes.Validator

	result, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(fmt.Sprintf(ValidatorURL, strconv.FormatUint(validatorID, 10))))
	if err != nil {
		logger.Error("Error fetching validator data", "error", err)
		return 0, err
	}

	if err = json.Unmarshal(result, &validator); err != nil {
		logger.Error("error unmarshalling validator data", "error", err)
		return 0, err
	}

	logger.Debug("Validator data received ", "validator", validator.String())

	return validator.Nonce, nil
}

// GetValidatorSet fetches the current validator set
func GetValidatorSet() (*staketypes.ValidatorSet, error) {
	logger := Logger()

	response, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(CurrentValidatorSetURL))
	if err != nil {
		logger.Error("Unable to send request for current validatorset", "url", CurrentValidatorSetURL, "error", err)
		return nil, err
	}

	var validatorSet staketypes.ValidatorSet
	if err = json.Unmarshal(response, &validatorSet); err != nil {
		logger.Error("Error unmarshalling current validatorset data ", "error", err)
		return nil, err
	}

	return &validatorSet, nil
}

// GetClerkEventRecord return last successful checkpoint
func GetClerkEventRecord(stateId int64) (*clerktypes.EventRecord, error) {
	logger := Logger()

	response, err := helper.FetchFromAPI(helper.GetHeimdallServerEndpoint(fmt.Sprintf(ClerkEventRecordURL, stateId)))
	if err != nil {
		logger.Error("Error fetching event record by state ID", "error", err)
		return nil, err
	}

	var eventRecord clerktypes.EventRecord
	if err = json.Unmarshal(response, &eventRecord); err != nil {
		logger.Error("Error unmarshalling event record", "error", err)
		return nil, err
	}

	return &eventRecord, nil
}

func GetUnconfirmedTxnCount(event interface{}) int {
	logger := Logger()

	defer LogElapsedTimeForStateSyncedEvent(event, "GetUnconfirmedTxnCount", time.Now())

	endpoint := helper.GetConfig().CometBFTRPCUrl + CometBFTUnconfirmedTxsCountURL

	resp, err := helper.Client.Get(endpoint)
	if err != nil || resp.StatusCode != http.StatusOK {
		logger.Error("Error fetching mempool txs count", "url", endpoint, "error", err)
		return 0
	}

	// Limit the number of bytes read from the response body
	limitedBody := http.MaxBytesReader(nil, resp.Body, helper.APIBodyLimit)

	body, err := io.ReadAll(limitedBody)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error("Error closing response body:", err)
		}
	}()

	if err != nil {
		logger.Error("Error fetching mempool txs count", "error", err)
		return 0
	}

	// a minimal response of the unconfirmed txs
	var response CometBFTUnconfirmedTxs

	err = json.Unmarshal(body, &response)
	if err != nil {
		logger.Error("Error unmarshalling response received from Heimdall Server", "error", err)
		return 0
	}

	count, _ := strconv.Atoi(response.Result.Total)

	return count
}

// LogElapsedTimeForStateSyncedEvent logs useful info for StateSynced events
func LogElapsedTimeForStateSyncedEvent(event interface{}, functionName string, startTime time.Time) {
	logger := Logger()

	if event == nil {
		return
	}

	var (
		typedEvent  statesender.StatesenderStateSynced
		timeElapsed = time.Since(startTime).Milliseconds()
	)

	switch e := event.(type) {
	case statesender.StatesenderStateSynced:
		typedEvent = e
	case *statesender.StatesenderStateSynced:
		if e == nil {
			return
		}

		typedEvent = *e
	default:
		return
	}

	logger.Info("StateSyncedEvent: "+functionName,
		"stateSyncId", typedEvent.Id,
		"timeElapsed", timeElapsed)
}

// IsPubKeyFirstByteValid checks the validity of the first byte of the public key.
// It must be 0x04 for uncompressed public keys
func IsPubKeyFirstByteValid(pubKey []byte) bool {
	prefix := make([]byte, 1)
	prefix[0] = byte(0x04)

	return bytes.Equal(prefix, pubKey[0:1])
}
