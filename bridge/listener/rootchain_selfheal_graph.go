package listener

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
)

// stateSynced represents the StateSynced event.
type stateSynced struct {
	StateId         string `json:"stateId"`
	LogIndex        string `json:"logIndex"`
	TransactionHash string `json:"transactionHash"`
}

// newHeaderBlock represents the NewHeaderBlock event.
type newHeaderBlock struct {
	HeaderBlockId   string `json:"headerBlockId"`
	LogIndex        string `json:"logIndex"`
	TransactionHash string `json:"transactionHash"`
}

type stateSyncedsResponse struct {
	Data struct {
		StateSynceds []stateSynced `json:"stateSynceds"`
	} `json:"data"`
	Errors []graphqlError `json:"errors,omitempty"`
}

type newHeaderBlocksResponse struct {
	Data struct {
		NewHeaderBlocks []newHeaderBlock `json:"newHeaderBlocks"`
	} `json:"data"`
	Errors []graphqlError `json:"errors,omitempty"`
}

// nonceOnly is the projection used when only the nonce field is needed.
type nonceOnly struct {
	Nonce string `json:"nonce"`
}

// txAndLogIndex is the projection used to locate an L1 log by tx hash + log index.
type txAndLogIndex struct {
	TransactionHash string `json:"transactionHash"`
	LogIndex        string `json:"logIndex"`
	// EventName is set by the caller after unmarshaling (never present in the
	// subgraph JSON itself) to record which of the nonce-gated stake event
	// entities this hit came from — see pickStakeEventHit.
	EventName string `json:"-"`
}

// graphqlError is a single entry in a GraphQL response's top-level errors array.
// Captured to surface schema drift (e.g. signerChanges/unstakeInits not yet
// indexed) instead of silently treating it as "no events".
type graphqlError struct {
	Message string `json:"message"`
}

// stakeEventMaxNonceResponse holds the highest-nonce row for each of the three
// nonce-gated stake event entities returned by a single combined query.
type stakeEventMaxNonceResponse struct {
	Data struct {
		StakeUpdates  []nonceOnly `json:"stakeUpdates"`
		SignerChanges []nonceOnly `json:"signerChanges"`
		UnstakeInits  []nonceOnly `json:"unstakeInits"`
	} `json:"data"`
	Errors []graphqlError `json:"errors,omitempty"`
}

// stakeEventByNonceResponse holds tx hash + log index for whichever of the three
// nonce-gated stake event entities matched the queried (validatorId, nonce).
type stakeEventByNonceResponse struct {
	Data struct {
		StakeUpdates  []txAndLogIndex `json:"stakeUpdates"`
		SignerChanges []txAndLogIndex `json:"signerChanges"`
		UnstakeInits  []txAndLogIndex `json:"unstakeInits"`
	} `json:"data"`
	Errors []graphqlError `json:"errors,omitempty"`
}

func (rl *RootChainListener) querySubGraph(query []byte, ctx context.Context) (data []byte, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, rl.subGraphClient.graphUrl, bytes.NewBuffer(query))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := rl.subGraphClient.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			fmt.Println("Error closing response body:", err)
		}
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("self-healing: subgraph returned HTTP %d: %s", response.StatusCode, body)
	}
	return body, nil
}

// getLatestStateID returns the state ID from the latest StateSynced event
func (rl *RootChainListener) getLatestStateID(ctx context.Context) (*big.Int, error) {
	query := map[string]string{
		"query": `
		{
			stateSynceds(first : 1, orderBy : stateId, orderDirection : desc) {
				stateId
			}
		}
		`,
	}

	byteQuery, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	data, err := rl.querySubGraph(byteQuery, ctx)
	if err != nil {
		return nil, fmt.Errorf("self-healing: unable to fetch latest state id from graph with err: %w", err)
	}

	var response stateSyncedsResponse
	if err = json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("self-healing: unable to unmarshal graph response: %w", err)
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("self-healing: subgraph returned errors for latest state id query: %s", joinGraphQLErrors(response.Errors))
	}

	if len(response.Data.StateSynceds) == 0 {
		return big.NewInt(0), nil
	}

	stateId := big.NewInt(0)
	stateId.SetString(response.Data.StateSynceds[0].StateId, 10)
	rl.Logger.Info("Self-healing: fetched latest stateId from subgraph", "stateId", stateId)

	return stateId, nil
}

// getCurrentStateID returns the current state ID handled by the polygon chain
func (rl *RootChainListener) getCurrentStateID(ctx context.Context) (*big.Int, error) {
	rootChainContext, err := rl.getRootChainContext()
	if err != nil {
		return nil, err
	}

	stateReceiverInstance, err := rl.contractCaller.GetStateReceiverInstance(
		rootChainContext.ChainmanagerParams.ChainParams.StateReceiverAddress,
	)
	if err != nil {
		return nil, err
	}

	stateId, err := stateReceiverInstance.LastStateId(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, err
	}

	return stateId, nil
}

// queryStateSyncedHit queries the subgraph for the (txHash, logIndex) of the
// StateSynced event with the given state ID.
func (rl *RootChainListener) queryStateSyncedHit(ctx context.Context, stateId int64) (stateSynced, error) {
	query := map[string]string{
		"query": `
		{
			stateSynceds(where: {stateId: ` + strconv.Itoa(int(stateId)) + `}) {
				logIndex
				transactionHash
			}
		}
		`,
	}

	byteQuery, err := json.Marshal(query)
	if err != nil {
		return stateSynced{}, err
	}

	data, err := rl.querySubGraph(byteQuery, ctx)
	if err != nil {
		return stateSynced{}, fmt.Errorf("self-healing: unable to fetch latest stateId from graph with err: %w", err)
	}

	var response stateSyncedsResponse
	if err = json.Unmarshal(data, &response); err != nil {
		return stateSynced{}, fmt.Errorf("self-healing: unable to unmarshal graph response: %w", err)
	}

	if len(response.Errors) > 0 {
		return stateSynced{}, fmt.Errorf("self-healing: subgraph returned errors for state synced query: %s", joinGraphQLErrors(response.Errors))
	}

	if len(response.Data.StateSynceds) == 0 {
		return stateSynced{}, fmt.Errorf("self-healing: no state synced event found for state id %d", stateId)
	}

	return response.Data.StateSynceds[0], nil
}

// getStateSynced fetches and validates the StateSynced log for stateId. The
// receipt fetch is wrapped in ExponentialBackoff so transient L1 RPC blips
// don't kill self-heal recovery for this state ID; validateReceiptLog and
// confirmStateSyncedId run once, unwrapped, afterward — those checks are
// deterministic, so retrying them would only waste cycles on a mismatch
// that retrying can't fix.
func (rl *RootChainListener) getStateSynced(ctx context.Context, stateId int64) (*types.Log, error) {
	hit, err := rl.queryStateSyncedHit(ctx, stateId)
	if err != nil {
		return nil, err
	}

	rootChainContext, err := rl.getRootChainContext()
	if err != nil {
		return nil, fmt.Errorf("self-healing: unable to fetch chain manager params: %w", err)
	}
	expectedAddr := common.HexToAddress(rootChainContext.ChainmanagerParams.ChainParams.StateSenderAddress)

	expectedTopic, ok := rl.eventTopicByName(helper.StateSyncedEvent)
	if !ok {
		return nil, fmt.Errorf("self-healing: no known topic for event %q", helper.StateSyncedEvent)
	}

	var receipt *types.Receipt
	if err = helper.ExponentialBackoff(func() error {
		receipt, err = rl.contractCaller.MainChainClient.TransactionReceipt(ctx, common.HexToHash(hit.TransactionHash))
		return err
	}, 3, time.Second); err != nil {
		return nil, fmt.Errorf("self-healing: failed to fetch L1 receipt for tx %s: %w", hit.TransactionHash, err)
	}

	log, err := validateReceiptLog(receipt, expectedAddr, expectedTopic, hit.TransactionHash, hit.LogIndex)
	if err != nil {
		return nil, fmt.Errorf("self-healing: %w", err)
	}

	if err := rl.confirmStateSyncedId(receipt, rootChainContext.ChainmanagerParams.ChainParams.StateSenderAddress, hit.LogIndex, stateId); err != nil {
		return nil, fmt.Errorf("self-healing: %w", err)
	}

	rl.Logger.Info("Self-healing: retrieved log for StateSynced event", "stateId", stateId, "logIndex", hit.LogIndex, "txHash", hit.TransactionHash)
	return log, nil
}

// confirmStateSyncedId decodes the StateSynced event at logIndex and confirms
// its id matches the state ID that was requested — structural validation
// alone doesn't rule out a subgraph hit pointing at a different, genuine
// StateSynced event.
func (rl *RootChainListener) confirmStateSyncedId(receipt *types.Receipt, stateSenderAddress, logIndex string, stateId int64) error {
	idx, err := strconv.ParseUint(logIndex, 10, 64)
	if err != nil {
		metrics.SelfHealValidationRejected.Inc()
		return fmt.Errorf("invalid log index %q: %w", logIndex, err)
	}

	decoded, err := rl.contractCaller.DecodeStateSyncedEvent(stateSenderAddress, receipt, idx)
	if err != nil {
		metrics.SelfHealValidationRejected.Inc()
		return fmt.Errorf("failed to decode StateSynced event: %w", err)
	}

	if decoded.Id.Cmp(big.NewInt(stateId)) != 0 {
		metrics.SelfHealValidationRejected.Inc()
		return fmt.Errorf("decoded stateId %s does not match requested %d", decoded.Id.String(), stateId)
	}

	return nil
}

// getMaxL1NonceForValidator returns the highest nonce across StakeUpdate,
// SignerChange, and UnstakeInit for a validator in one round-trip. The L1
// StakingInfo contract shares a single nonce counter across these three event
// types, so the max is the validator's authoritative L1 nonce.
func (rl *RootChainListener) getMaxL1NonceForValidator(ctx context.Context, validatorId uint64) (uint64, error) {
	idStr := strconv.FormatUint(validatorId, 10)
	query := map[string]string{
		"query": `
		{
			stakeUpdates(first:1, orderBy: nonce, orderDirection: desc, where: {validatorId: ` + idStr + `}) { nonce }
			signerChanges(first:1, orderBy: nonce, orderDirection: desc, where: {validatorId: ` + idStr + `}) { nonce }
			unstakeInits(first:1, orderBy: nonce, orderDirection: desc, where: {validatorId: ` + idStr + `}) { nonce }
		}
		`,
	}

	byteQuery, err := json.Marshal(query)
	if err != nil {
		return 0, err
	}

	data, err := rl.querySubGraph(byteQuery, ctx)
	if err != nil {
		return 0, fmt.Errorf("self-healing: unable to fetch max nonce from graph: %w", err)
	}

	var response stakeEventMaxNonceResponse
	if err = json.Unmarshal(data, &response); err != nil {
		return 0, fmt.Errorf("self-healing: unable to unmarshal max nonce response: %w", err)
	}
	if len(response.Errors) > 0 {
		return 0, fmt.Errorf("self-healing: subgraph returned errors for max nonce query: %s", joinGraphQLErrors(response.Errors))
	}

	maxNonce, err := maxNonceFromResponse(response)
	if err != nil {
		return 0, err
	}
	rl.Logger.Info("Self-healing: fetched latest nonce from subgraph", "validatorId", validatorId, "latestNonce", maxNonce)
	return maxNonce, nil
}

// maxNonceFromResponse returns the highest nonce across the three entity rows.
func maxNonceFromResponse(r stakeEventMaxNonceResponse) (uint64, error) {
	var maxNonce uint64
	for _, batch := range [][]nonceOnly{r.Data.StakeUpdates, r.Data.SignerChanges, r.Data.UnstakeInits} {
		if len(batch) == 0 {
			continue
		}
		n, err := strconv.ParseUint(batch[0].Nonce, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("self-healing: malformed nonce %q in subgraph response: %w", batch[0].Nonce, err)
		}
		if n > maxNonce {
			maxNonce = n
		}
	}
	return maxNonce, nil
}

// joinGraphQLErrors flattens the GraphQL errors array into a single line for log
// and error-message use. Empty messages render as "<no message>" so the caller
// always surfaces something.
func joinGraphQLErrors(errs []graphqlError) string {
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		if e.Message == "" {
			parts = append(parts, "<no message>")
			continue
		}
		parts = append(parts, e.Message)
	}
	return strings.Join(parts, "; ")
}

// getStakeEventRefByNonce queries the subgraph for the (txHash, logIndex)
// pointing at the L1 log carrying (validatorId, nonce) across the three
// nonce-gated stake event entities. Returns a non-nil hit on a match, or an
// error if the subgraph fails / returns errors / has no matching row. Does
// NOT fetch the L1 receipt — see fetchAndValidateStakeEventLog for that.
func (rl *RootChainListener) getStakeEventRefByNonce(ctx context.Context, validatorId, nonce uint64) (*txAndLogIndex, error) {
	idStr := strconv.FormatUint(validatorId, 10)
	nonceStr := strconv.FormatUint(nonce, 10)
	query := map[string]string{
		"query": `
		{
			stakeUpdates(where: {validatorId: ` + idStr + `, nonce: ` + nonceStr + `}) { transactionHash logIndex }
			signerChanges(where: {validatorId: ` + idStr + `, nonce: ` + nonceStr + `}) { transactionHash logIndex }
			unstakeInits(where: {validatorId: ` + idStr + `, nonce: ` + nonceStr + `}) { transactionHash logIndex }
		}
		`,
	}

	byteQuery, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	data, err := rl.querySubGraph(byteQuery, ctx)
	if err != nil {
		return nil, fmt.Errorf("self-healing: unable to fetch stake event from graph: %w", err)
	}

	var response stakeEventByNonceResponse
	if err = json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("self-healing: unable to unmarshal stake event response: %w", err)
	}
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("self-healing: subgraph returned errors for stake event query: %s", joinGraphQLErrors(response.Errors))
	}

	hit := pickStakeEventHit(response)
	if hit == nil {
		return nil, fmt.Errorf("self-healing: no stake event found for validator %d and nonce %d", validatorId, nonce)
	}
	return hit, nil
}

// getLatestCheckpointFromL1 returns the latest checkpoint from L1 using the subgraph
func (rl *RootChainListener) getLatestCheckpointFromL1(ctx context.Context) (*newHeaderBlock, error) {
	query := map[string]string{
		"query": `
		{
			newHeaderBlocks(first: 1, orderBy: headerBlockId, orderDirection: desc) {
				headerBlockId
				logIndex
				transactionHash
			}
		}
		`,
	}

	byteQuery, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	data, err := rl.querySubGraph(byteQuery, ctx)
	if err != nil {
		return nil, fmt.Errorf("self-healing: unable to fetch latest header block event from subgraph with err: %w", err)
	}

	var response newHeaderBlocksResponse
	if err = json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("self-healing: unable to unmarshal subgraph response: %w", err)
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("self-healing: subgraph returned errors for latest header block query: %s", joinGraphQLErrors(response.Errors))
	}

	if len(response.Data.NewHeaderBlocks) == 0 {
		return nil, fmt.Errorf("self-healing: no header block event found")
	}

	latestHeaderBlock := response.Data.NewHeaderBlocks[0]

	rl.Logger.Info("Self-healing: fetched latest header block event from subgraph", "headerBlockId", latestHeaderBlock.HeaderBlockId, "logIndex", latestHeaderBlock.LogIndex, "transactionHash", latestHeaderBlock.TransactionHash)

	return &latestHeaderBlock, nil
}
