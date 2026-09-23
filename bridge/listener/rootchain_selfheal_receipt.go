package listener

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xPolygon/heimdall-v2/helper"
)

// fetchAndValidateStakeEventLog pulls the L1 receipt for the given hit and
// runs validateReceiptLog against the StakingInfo address and the hit's own
// event name from ChainManager params. The receipt fetch is wrapped in
// ExponentialBackoff so transient L1 RPC blips don't kill per-validator
// recovery for a full self-heal cycle. validateReceiptLog is intentionally
// outside the retry — its checks are deterministic, retrying them would
// waste cycles on permanent failures.
func (rl *RootChainListener) fetchAndValidateStakeEventLog(ctx context.Context, hit *txAndLogIndex) (*types.Log, error) {
	rootChainContext, err := rl.getRootChainContext()
	if err != nil {
		return nil, fmt.Errorf("self-healing: unable to fetch chain manager params: %w", err)
	}
	expectedAddr := common.HexToAddress(rootChainContext.ChainmanagerParams.ChainParams.StakingInfoAddress)

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
	return log, nil
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
// the receipt must actually be for the requested transaction, the tx must
// have succeeded, the indexed log must exist, it must come from the expected
// contract, and its topic must be the specific event that was asked for —
// not just any event that contract can emit. Shared by every self-heal path
// that resolves a subgraph hit (txHash, logIndex) to an L1 log — self-heal
// must apply at least the same scrutiny as the live listener path, since a
// subgraph or RPC that returns a mismatched log is the same class of
// untrusted response either way.
func validateReceiptLog(receipt *types.Receipt, expectedAddr common.Address, expectedTopic common.Hash, txHash, logIndex string) (*types.Log, error) {
	if receipt == nil {
		return nil, fmt.Errorf("nil receipt for tx %s", txHash)
	}
	expectedHash := common.HexToHash(txHash)
	if receipt.TxHash != expectedHash {
		return nil, fmt.Errorf("receipt tx hash %s does not match requested tx %s", receipt.TxHash.Hex(), txHash)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("tx %s reverted (status=%d)", txHash, receipt.Status)
	}
	log := findLogByIndex(receipt.Logs, logIndex)
	if log == nil {
		return nil, fmt.Errorf("no log found for log index %s in tx %s", logIndex, txHash)
	}
	if log.Removed {
		return nil, fmt.Errorf("log at index %s in tx %s is removed (reorg'd)", logIndex, txHash)
	}
	if log.TxHash != expectedHash {
		return nil, fmt.Errorf("log tx hash %s does not match requested tx %s", log.TxHash.Hex(), txHash)
	}
	if log.Address != expectedAddr {
		return nil, fmt.Errorf("log address %s does not match expected contract %s", log.Address.Hex(), expectedAddr.Hex())
	}
	if len(log.Topics) == 0 || log.Topics[0] != expectedTopic {
		return nil, fmt.Errorf("log topic does not match expected event (tx %s, log index %s)", txHash, logIndex)
	}
	return log, nil
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

// findLogByIndex returns the receipt log whose decimal-string index equals the
// given target. The subgraph stores logIndex as a decimal string; comparing
// strings avoids parsing each call. receipt.Logs is []*types.Log — a
// malformed response can contain a nil entry, so each one is checked before
// dereferencing.
func findLogByIndex(logs []*types.Log, target string) *types.Log {
	for _, log := range logs {
		if log == nil {
			continue
		}
		if strconv.Itoa(int(log.Index)) == target {
			return log
		}
	}
	return nil
}
