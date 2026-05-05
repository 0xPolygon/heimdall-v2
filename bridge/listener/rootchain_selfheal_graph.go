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

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

	return io.ReadAll(response.Body)
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

// getStateSynced returns the StateSynced event based on the given state ID
func (rl *RootChainListener) getStateSynced(ctx context.Context, stateId int64) (*types.Log, error) {
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
		return nil, err
	}

	data, err := rl.querySubGraph(byteQuery, ctx)
	if err != nil {
		return nil, fmt.Errorf("self-healing: unable to fetch latest stateId from graph with err: %w", err)
	}

	var response stateSyncedsResponse
	if err = json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("self-healing: unable to unmarshal graph response: %w", err)
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("self-healing: subgraph returned errors for state synced query: %s", joinGraphQLErrors(response.Errors))
	}

	if len(response.Data.StateSynceds) == 0 {
		return nil, fmt.Errorf("self-healing: no state synced event found for state id %d", stateId)
	}

	receipt, err := rl.contractCaller.MainChainClient.TransactionReceipt(ctx, common.HexToHash(response.Data.StateSynceds[0].TransactionHash))
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		if strconv.Itoa(int(log.Index)) == response.Data.StateSynceds[0].LogIndex {
			rl.Logger.Info("Self-healing: retrieved log for StateSynced event", "stateId", stateId, "logIndex", response.Data.StateSynceds[0].LogIndex, "txHash", response.Data.StateSynceds[0].TransactionHash)
			return log, nil
		}
	}

	return nil, fmt.Errorf("self-healing: no log found for given log index %s and state id %d", response.Data.StateSynceds[0].LogIndex, stateId)
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

// getStakeEventLogByNonce fetches the L1 log for whichever of the three
// nonce-gated stake event types carries (validatorId, nonce). The shared L1
// nonce counter guarantees at most one entity matches.
func (rl *RootChainListener) getStakeEventLogByNonce(ctx context.Context, validatorId, nonce uint64) (*types.Log, error) {
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

	receipt, err := rl.contractCaller.MainChainClient.TransactionReceipt(ctx, common.HexToHash(hit.TransactionHash))
	if err != nil {
		return nil, err
	}

	log := findLogByIndex(receipt.Logs, hit.LogIndex)
	if log == nil {
		return nil, fmt.Errorf("self-healing: no log found for log index %s, validator %d and nonce %d", hit.LogIndex, validatorId, nonce)
	}

	rl.Logger.Info("Self-healing: retrieved stake event log from Ethereum", "validatorId", validatorId, "nonce", nonce, "txHash", log.TxHash.Hex())
	return log, nil
}

// pickStakeEventHit returns the first non-empty entity row. The shared L1 nonce
// counter ensures at most one entity holds a (validatorId, nonce) match.
func pickStakeEventHit(r stakeEventByNonceResponse) *txAndLogIndex {
	if len(r.Data.StakeUpdates) > 0 {
		return &r.Data.StakeUpdates[0]
	}
	if len(r.Data.SignerChanges) > 0 {
		return &r.Data.SignerChanges[0]
	}
	if len(r.Data.UnstakeInits) > 0 {
		return &r.Data.UnstakeInits[0]
	}
	return nil
}

// findLogByIndex returns the receipt log whose decimal-string index equals the
// given target. The subgraph stores logIndex as a decimal string; comparing
// strings avoids parsing each call.
func findLogByIndex(logs []*types.Log, target string) *types.Log {
	for _, log := range logs {
		if strconv.Itoa(int(log.Index)) == target {
			return log
		}
	}
	return nil
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
