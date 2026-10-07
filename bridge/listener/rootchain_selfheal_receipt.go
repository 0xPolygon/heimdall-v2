package listener

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/metrics"
)

// fetchAndValidateStakeEventLog pulls the L1 receipt for the given hit, runs
// validateReceiptLog against the StakingInfo address and the hit's own event
// name from ChainManager params, then decodes the event and confirms its
// (validatorId, nonce) are actually the ones that were requested —
// structural validation alone doesn't rule out a subgraph hit pointing at a
// different validator's or a different nonce's genuine stake event. The
// receipt fetch is wrapped in ExponentialBackoff so transient L1 RPC blips
// don't kill per-validator recovery for a full self-heal cycle.
// validateReceiptLog and the identity check are intentionally outside the
// retry — their checks are deterministic, retrying them would waste cycles
// on permanent failures.
func (rl *RootChainListener) fetchAndValidateStakeEventLog(ctx context.Context, hit *txAndLogIndex, validatorId, nonce uint64) (*types.Log, error) {
	rootChainContext, err := rl.getRootChainContext()
	if err != nil {
		return nil, fmt.Errorf("self-healing: unable to fetch chain manager params: %w", err)
	}
	stakingInfoAddress := rootChainContext.ChainmanagerParams.ChainParams.StakingInfoAddress
	expectedAddr := common.HexToAddress(stakingInfoAddress)

	expectedTopic, ok := rl.eventTopicByName(hit.EventName)
	if !ok {
		return nil, fmt.Errorf("self-healing: no known topic for event %q", hit.EventName)
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

	if err := rl.confirmStakeEventIdentity(receipt, stakingInfoAddress, hit, validatorId, nonce); err != nil {
		return nil, fmt.Errorf("self-healing: %w", err)
	}

	return log, nil
}

// confirmStakeEventIdentity decodes the stake event named by hit.EventName at
// hit.LogIndex and confirms its (validatorId, nonce) match what was
// requested. Each of the three nonce-gated stake event types has its own ABI
// shape, so each is decoded separately.
func (rl *RootChainListener) confirmStakeEventIdentity(receipt *types.Receipt, stakingInfoAddress string, hit *txAndLogIndex, validatorId, nonce uint64) error {
	idx, err := strconv.ParseUint(hit.LogIndex, 10, 64)
	if err != nil {
		metrics.SelfHealValidationRejected.Inc()
		return fmt.Errorf("invalid log index %q: %w", hit.LogIndex, err)
	}

	gotValidatorId, gotNonce, err := rl.decodeStakeEventFields(receipt, stakingInfoAddress, hit.EventName, idx)
	if err != nil {
		metrics.SelfHealValidationRejected.Inc()
		return err
	}

	// nonce is a non-indexed field on SignerChange/UnstakeInit, so it's only
	// populated by ABI-unpacking the log's Data — which helper.UnpackLog
	// skips entirely (no error) when Data is empty, leaving Nonce nil.
	// validatorId is indexed on every one of these events and is always
	// populated regardless, but guard both since Cmp panics on a nil
	// receiver either way.
	if gotValidatorId == nil || gotNonce == nil {
		metrics.SelfHealValidationRejected.Inc()
		return fmt.Errorf("decoded %s event at tx %s log index %s is missing validatorId/nonce",
			hit.EventName, hit.TransactionHash, hit.LogIndex)
	}

	// Compare the decoded *big.Int fields directly — converting to uint64
	// first would silently truncate an oversized on-chain value, letting it
	// alias the requested (validatorId, nonce) after wraparound.
	if gotValidatorId.Cmp(new(big.Int).SetUint64(validatorId)) != 0 || gotNonce.Cmp(new(big.Int).SetUint64(nonce)) != 0 {
		metrics.SelfHealValidationRejected.Inc()
		return fmt.Errorf("decoded (validatorId=%s, nonce=%s) does not match requested (validatorId=%d, nonce=%d)",
			gotValidatorId.String(), gotNonce.String(), validatorId, nonce)
	}

	return nil
}

// decodeStakeEventFields decodes the stake event named eventName at idx and
// returns its (validatorId, nonce) fields. Each of the three nonce-gated
// stake event types has its own ABI shape, so each is decoded separately.
func (rl *RootChainListener) decodeStakeEventFields(receipt *types.Receipt, stakingInfoAddress, eventName string, idx uint64) (*big.Int, *big.Int, error) {
	switch eventName {
	case helper.StakeUpdateEvent:
		decoded, err := rl.contractCaller.DecodeValidatorStakeUpdateEvent(stakingInfoAddress, receipt, idx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode StakeUpdate event: %w", err)
		}
		return decoded.ValidatorId, decoded.Nonce, nil
	case helper.SignerChangeEvent:
		decoded, err := rl.contractCaller.DecodeSignerUpdateEvent(stakingInfoAddress, receipt, idx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode SignerChange event: %w", err)
		}
		return decoded.ValidatorId, decoded.Nonce, nil
	case helper.UnstakeInitEvent:
		decoded, err := rl.contractCaller.DecodeValidatorExitEvent(stakingInfoAddress, receipt, idx)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode UnstakeInit event: %w", err)
		}
		return decoded.ValidatorId, decoded.Nonce, nil
	default:
		return nil, nil, fmt.Errorf("unrecognized stake event name %q", eventName)
	}
}

// eventTopicByName finds the topic hash for a known event name. Self-heal's
// call volume is low (a handful of recoveries per cycle at most), so a linear
// scan over eventMap is fine — building a reverse index isn't worth it here.
func (rl *RootChainListener) eventTopicByName(name string) (common.Hash, bool) {
	for topic, event := range rl.eventMap {
		if event.Name == name {
			return topic, true
		}
	}
	return common.Hash{}, false
}

// validateReceiptLog verifies a fetched L1 receipt before its log is trusted:
// the receipt must actually be for the requested transaction and have a real
// block number, the tx must have succeeded, the indexed log must exist and
// its block/transaction metadata must agree with the receipt it supposedly
// came from, it must come from the expected contract, and its topic must be
// the specific event that was asked for — not just any event that contract
// can emit. Shared by every self-heal path that resolves a subgraph hit
// (txHash, logIndex) to an L1 log — self-heal must apply at least the same
// scrutiny as the live listener path, since a subgraph or RPC that returns a
// mismatched log is the same class of untrusted response either way.
//
// This only binds the log to the receipt's own metadata (shape). It doesn't
// decode the event and check its content matches the specific query key
// (headerBlockId / stateId / validatorId+nonce) that selected this hit in
// the first place — callers that resolve a hit to a specific value do that
// check themselves once they have the receipt to decode from.
func validateReceiptLog(receipt *types.Receipt, expectedAddr common.Address, expectedTopic common.Hash, txHash, logIndex string) (*types.Log, error) {
	if err := validateReceiptShape(receipt, txHash); err != nil {
		metrics.SelfHealValidationRejected.Inc()
		return nil, err
	}
	log, err := resolveAndValidateLog(receipt, txHash, logIndex)
	if err != nil {
		metrics.SelfHealValidationRejected.Inc()
		return nil, err
	}
	if err := validateLogIdentity(log, expectedAddr, expectedTopic, txHash, logIndex); err != nil {
		metrics.SelfHealValidationRejected.Inc()
		return nil, err
	}
	return log, nil
}

// validateReceiptShape confirms the receipt itself is well-formed and
// actually for the requested, successful transaction, before any of its
// logs are inspected.
func validateReceiptShape(receipt *types.Receipt, txHash string) error {
	if receipt == nil {
		return fmt.Errorf("nil receipt for tx %s", txHash)
	}
	if receipt.BlockNumber == nil || receipt.BlockNumber.Sign() == 0 || !receipt.BlockNumber.IsUint64() {
		return fmt.Errorf("receipt for tx %s has an invalid block number", txHash)
	}
	if receipt.TxHash != common.HexToHash(txHash) {
		return fmt.Errorf("receipt tx hash %s does not match requested tx %s", receipt.TxHash.Hex(), txHash)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("tx %s reverted (status=%d)", txHash, receipt.Status)
	}
	return nil
}

// resolveAndValidateLog finds the requested log index in the receipt and
// confirms its block/transaction metadata actually agrees with the receipt
// it supposedly came from.
func resolveAndValidateLog(receipt *types.Receipt, txHash, logIndex string) (*types.Log, error) {
	if err := receiptLogsWellFormed(receipt.Logs, txHash); err != nil {
		return nil, err
	}
	log := findLogByIndex(receipt.Logs, logIndex)
	if log == nil {
		return nil, fmt.Errorf("no log found for log index %s in tx %s", logIndex, txHash)
	}
	if log.Removed {
		return nil, fmt.Errorf("log at index %s in tx %s is removed (reorg'd)", logIndex, txHash)
	}
	if log.TxHash != common.HexToHash(txHash) {
		return nil, fmt.Errorf("log tx hash %s does not match requested tx %s", log.TxHash.Hex(), txHash)
	}
	if log.BlockNumber != receipt.BlockNumber.Uint64() || log.BlockHash != receipt.BlockHash || log.TxIndex != receipt.TransactionIndex {
		return nil, fmt.Errorf("log block/transaction metadata does not match its own receipt (tx %s, log index %s)", txHash, logIndex)
	}
	return log, nil
}

// validateLogIdentity confirms the log actually came from the expected
// contract and carries the specific event topic that was asked for.
func validateLogIdentity(log *types.Log, expectedAddr common.Address, expectedTopic common.Hash, txHash, logIndex string) error {
	if log.Address != expectedAddr {
		return fmt.Errorf("log address %s does not match expected contract %s", log.Address.Hex(), expectedAddr.Hex())
	}
	if len(log.Topics) == 0 || log.Topics[0] != expectedTopic {
		return fmt.Errorf("log topic does not match expected event (tx %s, log index %s)", txHash, logIndex)
	}
	return nil
}

// pickStakeEventHit returns the first non-empty entity row, tagged with the
// event name it came from. The shared L1 nonce counter ensures at most one
// entity holds a (validatorId, nonce) match.
func pickStakeEventHit(r stakeEventByNonceResponse) *txAndLogIndex {
	if len(r.Data.StakeUpdates) > 0 {
		hit := r.Data.StakeUpdates[0]
		hit.EventName = helper.StakeUpdateEvent
		return &hit
	}
	if len(r.Data.SignerChanges) > 0 {
		hit := r.Data.SignerChanges[0]
		hit.EventName = helper.SignerChangeEvent
		return &hit
	}
	if len(r.Data.UnstakeInits) > 0 {
		hit := r.Data.UnstakeInits[0]
		hit.EventName = helper.UnstakeInitEvent
		return &hit
	}
	return nil
}

// receiptLogsWellFormed rejects the whole receipt if receipt.Logs contains
// any nil entry, before findLogByIndex or any Decode*Event scan ever runs
// over it. Every helper.ContractCaller Decode*Event re-scans receipt.Logs
// from scratch with no nil check, so a nil anywhere in the slice is fatal to
// that later scan even when it sits well clear of the log this function is
// about to return as valid.
func receiptLogsWellFormed(logs []*types.Log, txHash string) error {
	for _, log := range logs {
		if log == nil {
			return fmt.Errorf("receipt for tx %s contains a malformed (nil) log entry", txHash)
		}
	}
	return nil
}

// findLogByIndex returns the receipt log whose decimal-string index equals the
// given target. The subgraph stores logIndex as a decimal string; comparing
// strings avoids parsing each call. Callers must run receiptLogsWellFormed
// first — this function assumes no entry in logs is nil.
func findLogByIndex(logs []*types.Log, target string) *types.Log {
	for _, log := range logs {
		if strconv.FormatUint(uint64(log.Index), 10) == target {
			return log
		}
	}
	return nil
}
