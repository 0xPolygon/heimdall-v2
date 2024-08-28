package util

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/helper"
	chainmanagertypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	clerktypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
	"github.com/cometbft/cometbft/libs/log"

	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	staketypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	cmtTypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

type BridgeEvent string

const (
	AccountDetailsURL       = "/auth/accounts/%v"
	LastNoAckURL            = "/checkpoints/last-no-ack"
	CheckpointParamsURL     = "/checkpoints/params"
	MilestoneParamsURL      = "/milestone/params"
	MilestoneCountURL       = "/milestone/count"
	ChainManagerParamsURL   = "/chainmanager/params"
	ProposersURL            = "/staking/proposer/%v"
	MilestoneProposersURL   = "/staking/milestoneProposer/%v"
	BufferedCheckpointURL   = "/checkpoints/buffer"
	LatestCheckpointURL     = "/checkpoints/latest"
	LatestMilestoneURL      = "/milestone/latest"
	CountCheckpointURL      = "/checkpoints/count"
	CurrentProposerURL      = "/staking/current-proposer"
	LatestSpanURL           = "/bor/latest-span"
	NextSpanInfoURL         = "/bor/prepare-next-span"
	NextSpanSeedURL         = "/bor/next-span-seed"
	DividendAccountRootURL  = "/topup/dividend-account-root"
	ValidatorURL            = "/staking/validator/%v"
	CurrentValidatorSetURL  = "staking/validator-set"
	StakingTxStatusURL      = "/staking/isoldtx"
	TopupTxStatusURL        = "/topup/isoldtx"
	ClerkTxStatusURL        = "/clerk/isoldtx"
	ClerkEventRecordURL     = "/clerk/event-record/%d"
	LatestSlashInfoBytesURL = "/slashing/latest_slash_info_bytes"
	TickSlashInfoListURL    = "/slashing/tick_slash_infos"
	SlashingTxStatusURL     = "/slashing/isoldtx"
	SlashingTickCountURL    = "/slashing/tick-count"

	CometBFTUnconfirmedTxsURL      = "/unconfirmed_txs"
	CometBFTUnconfirmedTxsCountURL = "/num_unconfirmed_txs"

	TransactionTimeout      = 1 * time.Minute
	CommitTimeout           = 2 * time.Minute
	TaskDelayBetweenEachVal = 10 * time.Second
	RetryTaskDelay          = 12 * time.Second
	RetryStateSyncTaskDelay = 24 * time.Second

	mempoolTxnCountDivisor = 1000

	// Bridge event types
	StakingEvent  BridgeEvent = "staking"
	TopupEvent    BridgeEvent = "topup"
	ClerkEvent    BridgeEvent = "clerk"
	SlashingEvent BridgeEvent = "slashing"

	BridgeDBFlag = "bridge-db"
)

// Logger returns logger singleton instance
func Logger() log.Logger {
	return log.NewNopLogger().With("module", "bridge")
}

// IsProposer checks if we are proposer
func IsProposer(cliCtx client.Context) (bool, error) {
	logger := Logger()
	var (
		proposers []staketypes.Validator
		// count     = uint64(1)
	)

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		result, err := helper.FetchFromAPI(cliCtx,
			helper.GetHeimdallServerEndpoint(fmt.Sprintf(ProposersURL, strconv.FormatUint(count, 10))),
		)
		if err != nil {
			logger.Error("Error fetching proposers", "url", ProposersURL, "error", err)
			return false, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var result struct{ Result []byte }
	var err error

	err = jsoniter.ConfigFastest.Unmarshal(result.Result, &proposers)
	if err != nil {
		logger.Error("error unmarshalling proposer slice", "error", err)
		return false, err
	}

	if bytes.Equal([]byte(proposers[0].Signer), helper.GetAddress()) {
		return true, nil
	}

	return false, nil
}

// IsMilestoneProposer checks if we are milestone proposer
func IsMilestoneProposer(cliCtx client.Context) (bool, error) {
	logger := Logger()

	var (
		proposers []staketypes.Validator
		// count     = uint64(1)
	)

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		result, err := helper.FetchFromAPI(cliCtx,
			helper.GetHeimdallServerEndpoint(fmt.Sprintf(MilestoneProposersURL, strconv.FormatUint(count, 10))),
		)
		if err != nil {
			logger.Error("Error fetching milestone proposers", "url", MilestoneProposersURL, "error", err)
			return false, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var result struct{ Result []byte }
	var err error

	err = jsoniter.ConfigFastest.Unmarshal(result.Result, &proposers)
	if err != nil {
		logger.Error("error unmarshalling milestone proposer slice", "error", err)
		return false, err
	}

	if len(proposers) == 0 {
		logger.Error("length of proposer list is 0")
		return false, errors.Errorf("Length of proposer list is 0")
	}

	if bytes.Equal([]byte(proposers[0].Signer), helper.GetAddress()) {
		return true, nil
	}

	return false, nil
}

// IsInProposerList checks if we are in current proposer
func IsInProposerList(cliCtx client.Context, count uint64) (bool, error) {
	logger := Logger()

	logger.Debug("Skipping proposers", "count", strconv.FormatUint(count+1, 10))

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		response, err := helper.FetchFromAPI(
			cliCtx,
			helper.GetHeimdallServerEndpoint(fmt.Sprintf(ProposersURL, strconv.FormatUint(count+1, 10))),
		)
		if err != nil {
			logger.Error("Unable to send request for next proposers", "url", ProposersURL, "error", err)
			return false, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var response struct{ Result []byte }

	// unmarshall data from buffer
	var proposers []staketypes.Validator
	if err := jsoniter.ConfigFastest.Unmarshal(response.Result, &proposers); err != nil {
		logger.Error("Error unmarshalling validator data ", "error", err)
		return false, err
	}

	logger.Debug("Fetched proposers list", "numberOfProposers", count+1)

	for i := 1; i <= int(count) && i < len(proposers); i++ {
		if bytes.Equal([]byte(proposers[i].Signer), helper.GetAddress()) {
			return true, nil
		}
	}

	return false, nil
}

// IsInMilestoneProposerList checks if we are in current proposer
func IsInMilestoneProposerList(cliCtx client.Context, count uint64) (bool, error) {
	logger := Logger()

	logger.Debug("Skipping proposers", "count", strconv.FormatUint(count, 10))

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		response, err := helper.FetchFromAPI(
			cliCtx,
			helper.GetHeimdallServerEndpoint(fmt.Sprintf(MilestoneProposersURL, strconv.FormatUint(count, 10))),
		)
		if err != nil {
			logger.Error("Unable to send request for next proposers", "url", MilestoneProposersURL, "error", err)
			return false, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var response struct{ Result []byte }

	// unmarshall data from buffer
	var proposers []staketypes.Validator
	if err := jsoniter.ConfigFastest.Unmarshal(response.Result, &proposers); err != nil {
		logger.Error("Error unmarshalling validator data ", "error", err)
		return false, err
	}

	logger.Debug("Fetched proposers list", "numberOfProposers", count)

	for _, proposer := range proposers {
		if bytes.Equal([]byte(proposer.Signer), helper.GetAddress()) {
			return true, nil
		}
	}

	return false, nil
}

// CalculateTaskDelay calculates delay required for current validator to propose the tx
// It solves for multiple validators sending same transaction.
func CalculateTaskDelay(cliCtx client.Context, event interface{}) (bool, time.Duration) {
	logger := Logger()

	defer LogElapsedTimeForStateSyncedEvent(event, "CalculateTaskDelay", time.Now())

	// calculate validator position
	valPosition := 0
	isCurrentValidator := false

	validatorSet, err := GetValidatorSet(cliCtx)
	if err != nil {
		logger.Error("Error getting current validatorset data ", "error", err)
		return isCurrentValidator, 0
	}

	logger.Info("Fetched current validatorset list", "currentValidatorcount", len(validatorSet.Validators))

	for i, validator := range validatorSet.Validators {
		if bytes.Equal([]byte(validator.Signer), helper.GetAddress()) {
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
func IsCurrentProposer(cliCtx client.Context) (bool, error) {
	logger := Logger()

	var proposer staketypes.Validator

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		result, err := helper.FetchFromAPI(cliCtx, helper.GetHeimdallServerEndpoint(CurrentProposerURL))
		if err != nil {
			logger.Error("Error fetching proposers", "error", err)
			return false, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var result struct{ Result []byte }
	var err error

	if err = jsoniter.ConfigFastest.Unmarshal(result.Result, &proposer); err != nil {
		logger.Error("error unmarshalling validator", "error", err)
		return false, err
	}

	logger.Debug("Current proposer fetched", "validator", proposer.String())

	if bytes.Equal([]byte(proposer.Signer), helper.GetAddress()) {
		return true, nil
	}

	logger.Debug("We are not the current proposer")

	return false, nil
}

// IsEventSender check if we are the EventSender
func IsEventSender(cliCtx client.Context, validatorID uint64) bool {
	logger := Logger()

	var validator staketypes.Validator

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		result, err := helper.FetchFromAPI(cliCtx,
			helper.GetHeimdallServerEndpoint(fmt.Sprintf(ValidatorURL, strconv.FormatUint(validatorID, 10))),
		)
		if err != nil {
			logger.Error("Error fetching proposers", "error", err)
			return false
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var result struct{ Result []byte }
	var err error

	if err = jsoniter.ConfigFastest.Unmarshal(result.Result, &validator); err != nil {
		logger.Error("error unmarshalling proposer slice", "error", err)
		return false
	}

	logger.Debug("Current event sender received", "validator", validator.String())

	return bytes.Equal([]byte(validator.Signer), helper.GetAddress())
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

// WaitForOneEvent subscribes to a websocket event for the given
// event time and returns upon receiving it one time, or
// when the timeout duration has expired.
//
// This handles subscribing and unsubscribing under the hood
func WaitForOneEvent(tx cmtTypes.Tx, client *rpchttp.HTTP) (cmtTypes.TMEventData, error) {
	logger := Logger()

	ctx, cancel := context.WithTimeout(context.Background(), CommitTimeout)
	defer cancel()

	// subscriber
	subscriber := hex.EncodeToString(tx.Hash())

	// query
	query := cmtTypes.EventQueryTxFor(tx).String()

	// register for the next event of this type
	eventCh, err := client.Subscribe(ctx, subscriber, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to subscribe")
	}

	// make sure to unregister after the test is over
	defer func() {
		if err := client.UnsubscribeAll(ctx, subscriber); err != nil {
			logger.Error("WaitForOneEvent | UnsubscribeAll", "Error", err)
		}
	}()

	select {
	case event := <-eventCh:
		return event.Data, nil
	case <-ctx.Done():
		return nil, errors.New("timed out waiting for event")
	}
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
	logger := Logger()

	url := helper.GetHeimdallServerEndpoint(fmt.Sprintf(AccountDetailsURL, address))

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		// call account rest api
		response, err := helper.FetchFromAPI(cliCtx, url)
		if err != nil {
			return nil, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var response struct{ Result []byte }
	var err error

	var account sdk.AccountI
	if err = cliCtx.Codec.UnmarshalJSON(response.Result, account); err != nil {
		logger.Error("Error unmarshalling account details", "url", url)
		return nil, err
	}

	return account, nil
}

// GetChainmanagerParams return chain manager params
func GetChainmanagerParams(cliCtx client.Context) (*chainmanagertypes.Params, error) {
	logger := Logger()

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		response, err := helper.FetchFromAPI(
			cliCtx,
			helper.GetHeimdallServerEndpoint(ChainManagerParamsURL),
		)
		if err != nil {
			logger.Error("Error fetching chainmanager params", "err", err)
			return nil, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var response struct{ Result []byte }
	var err error

	var params chainmanagertypes.Params
	if err = jsoniter.ConfigFastest.Unmarshal(response.Result, &params); err != nil {
		logger.Error("Error unmarshalling chainmanager params", "url", ChainManagerParamsURL, "err", err)
		return nil, err
	}

	return &params, nil
}

// GetCheckpointParams return params
func GetCheckpointParams(cliCtx client.Context) (*checkpointTypes.Params, error) {
	logger := Logger()

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		response, err := helper.FetchFromAPI(
			cliCtx,
			helper.GetHeimdallServerEndpoint(CheckpointParamsURL),
		)
		if err != nil {
			logger.Error("Error fetching Checkpoint params", "err", err)
			return nil, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var response struct{ Result []byte }

	var params checkpointTypes.Params
	if err := jsoniter.ConfigFastest.Unmarshal(response.Result, &params); err != nil {
		logger.Error("Error unmarshalling Checkpoint params", "url", CheckpointParamsURL)
		return nil, err
	}

	return &params, nil
}

// GetCheckpointParams return params
func GetMilestoneParams(cliCtx client.Context) (*milestoneTypes.Params, error) {
	logger := Logger()

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		response, err := helper.FetchFromAPI(
			cliCtx,
			helper.GetHeimdallServerEndpoint(MilestoneParamsURL),
		)

		if err != nil {
			logger.Error("Error fetching Milestone params", "err", err)
			return nil, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var response struct{ Result []byte }

	var params milestoneTypes.Params
	if err := json.Unmarshal(response.Result, &params); err != nil {
		logger.Error("Error unmarshalling Checkpoint params", "url", MilestoneParamsURL)
		return nil, err
	}

	return &params, nil
}

// GetBufferedCheckpoint return checkpoint from bueffer
func GetBufferedCheckpoint(cliCtx client.Context) (*checkpointTypes.Checkpoint, error) {
	logger := Logger()

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		response, err := helper.FetchFromAPI(
			cliCtx,
			helper.GetHeimdallServerEndpoint(BufferedCheckpointURL),
		)
		if err != nil {
			logger.Debug("Error fetching buffered checkpoint", "err", err)
			return nil, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var response struct{ Result []byte }

	var checkpoint checkpointTypes.Checkpoint
	if err := jsoniter.ConfigFastest.Unmarshal(response.Result, &checkpoint); err != nil {
		logger.Error("Error unmarshalling buffered checkpoint", "url", BufferedCheckpointURL, "err", err)
		return nil, err
	}

	return &checkpoint, nil
}

// GetLatestCheckpoint return last successful checkpoint
func GetLatestCheckpoint(cliCtx client.Context) (*checkpointTypes.Checkpoint, error) {
	logger := Logger()

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		response, err := helper.FetchFromAPI(
			cliCtx,
			helper.GetHeimdallServerEndpoint(LatestCheckpointURL),
		)
		if err != nil {
			logger.Debug("Error fetching latest checkpoint", "err", err)
			return nil, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var response struct{ Result []byte }
	var err error

	var checkpoint checkpointTypes.Checkpoint
	if err = jsoniter.ConfigFastest.Unmarshal(response.Result, &checkpoint); err != nil {
		logger.Error("Error unmarshalling latest checkpoint", "url", LatestCheckpointURL, "err", err)
		return nil, err
	}

	return &checkpoint, nil
}

// GetLatestMilestone return last successful milestone
func GetLatestMilestone(cliCtx client.Context) (*milestoneTypes.Milestone, error) {
	logger := Logger()

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		response, err := helper.FetchFromAPI(
			cliCtx,
			helper.GetHeimdallServerEndpoint(LatestMilestoneURL),
		)
		if err != nil {
			logger.Debug("Error fetching latest milestone", "err", err)
			return nil, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var response struct{ Result []byte }
	var err error

	var milestone milestoneTypes.Milestone
	if err = json.Unmarshal(response.Result, &milestone); err != nil {
		logger.Error("Error unmarshalling latest milestone", "url", LatestMilestoneURL, "err", err)
		return nil, err
	}

	return &milestone, nil
}

// GetCheckpointParams return params
func GetMilestoneCount(cliCtx client.Context) (*milestoneTypes.MilestoneCount, error) {
	logger := Logger()

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		response, err := helper.FetchFromAPI(
			cliCtx,
			helper.GetHeimdallServerEndpoint(MilestoneCountURL),
		)
		if err != nil {
			logger.Error("Error fetching Milestone count", "err", err)
			return nil, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var response struct{ Result []byte }

	var count milestoneTypes.MilestoneCount
	if err := json.Unmarshal(response.Result, &count); err != nil {
		logger.Error("Error unmarshalling milestone Count", "url", MilestoneCountURL)
		return nil, err
	}

	return &count, nil
}

// AppendPrefix returns publickey in uncompressed format
func AppendPrefix(signerPubKey []byte) []byte {
	// append prefix - "0x04" as heimdall uses publickey in uncompressed format. Refer below link
	// https://superuser.com/questions/1465455/what-is-the-size-of-public-key-for-ecdsa-spec256r1
	prefix := make([]byte, 1)
	prefix[0] = byte(0x04)
	signerPubKey = append(prefix[:], signerPubKey[:]...)

	return signerPubKey
}

// GetValidatorNonce fetches validator nonce and height
func GetValidatorNonce(cliCtx client.Context, validatorID uint64) (uint64, int64, error) {
	logger := Logger()

	var validator staketypes.Validator

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		result, err := helper.FetchFromAPI(cliCtx,
			helper.GetHeimdallServerEndpoint(fmt.Sprintf(ValidatorURL, strconv.FormatUint(validatorID, 10))),
		)
		if err != nil {
			logger.Error("Error fetching validator data", "error", err)
			return 0, 0, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var result struct {
		Height int64
		Result []byte
	}
	var err error

	if err = jsoniter.ConfigFastest.Unmarshal(result.Result, &validator); err != nil {
		logger.Error("error unmarshalling validator data", "error", err)
		return 0, 0, err
	}

	logger.Debug("Validator data received ", "validator", validator.String())

	return validator.Nonce, result.Height, nil
}

// GetValidatorSet fetches the current validator set
func GetValidatorSet(cliCtx client.Context) (*staketypes.ValidatorSet, error) {
	logger := Logger()

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		response, err := helper.FetchFromAPI(cliCtx, helper.GetHeimdallServerEndpoint(CurrentValidatorSetURL))
		if err != nil {
			logger.Error("Unable to send request for current validatorset", "url", CurrentValidatorSetURL, "error", err)
			return nil, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var response struct{ Result []byte }
	var err error

	var validatorSet staketypes.ValidatorSet
	if err = jsoniter.ConfigFastest.Unmarshal(response.Result, &validatorSet); err != nil {
		logger.Error("Error unmarshalling current validatorset data ", "error", err)
		return nil, err
	}

	return &validatorSet, nil
}

// GetBlockHeight return last successful checkpoint
func GetBlockHeight(cliCtx client.Context) int64 {
	// logger := Logger()

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		response, err := helper.FetchFromAPI(
			cliCtx,
			helper.GetHeimdallServerEndpoint(CountCheckpointURL),
		)
		if err != nil {
			logger.Debug("Error fetching latest block height", "err", err)
			return 0
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var response struct{ Height int64 }

	return response.Height
}

// GetClerkEventRecord return last successful checkpoint
func GetClerkEventRecord(cliCtx client.Context, stateId int64) (*clerktypes.EventRecord, error) {
	logger := Logger()

	// TODO HV2 - uncomment the following fn once it is uncommented in helper.
	/*
		response, err := helper.FetchFromAPI(
			cliCtx,
			helper.GetHeimdallServerEndpoint(fmt.Sprintf(ClerkEventRecordURL, stateId)),
		)
		if err != nil {
			logger.Error("Error fetching event record by state ID", "error", err)
			return nil, err
		}
	*/

	// TODO HV2 - This is a place holder, remove when the above function is uncommented.
	var response struct{ Result []byte }
	var err error

	var eventRecord clerktypes.EventRecord
	if err = jsoniter.ConfigFastest.Unmarshal(response.Result, &eventRecord); err != nil {
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

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		logger.Error("Error fetching mempool txs count", "error", err)
		return 0
	}

	// a minimal response of the unconfirmed txs
	var response CometBFTUnconfirmedTxs

	err = jsoniter.ConfigFastest.Unmarshal(body, &response)
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
