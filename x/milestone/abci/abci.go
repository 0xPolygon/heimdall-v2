package abci

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"slices"
	"sort"

	"cosmossdk.io/log"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	borKeeper "github.com/0xPolygon/heimdall-v2/x/bor/keeper"
	"github.com/0xPolygon/heimdall-v2/x/milestone/keeper"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

type GetBlockAuthorFunc func(ctx sdk.Context, blockNumber uint64) ([]common.Address, error)

func GenMilestoneProposition(ctx sdk.Context, borKeeper *borKeeper.Keeper, milestoneKeeper *keeper.Keeper, contractCaller helper.IContractCaller, getBlockAuthor GetBlockAuthorFunc) (*types.MilestoneProposition, error) {
	milestone, err := milestoneKeeper.GetLastMilestone(ctx)
	if err != nil && !errors.Is(err, types.ErrNoMilestoneFound) {
		return nil, err
	}

	propStartBlock := uint64(0)

	var lastMilestoneHash []byte
	var lastMilestoneBlockNumber uint64

	var latestHeader *ethTypes.Header

	if milestone != nil {
		propStartBlock = milestone.EndBlock + 1

		// Fetch the latest header, once and reuse it to avoid duplicate RPC calls and race conditions.
		latestHeader, err = contractCaller.GetBorChainBlock(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get the latest header: %w", err)
		}

		params, err := milestoneKeeper.GetParams(ctx)
		if err != nil {
			return nil, err
		}

		if isFastForwardMilestone(latestHeader.Number.Uint64(), milestone.EndBlock, params.FfMilestoneThreshold) {
			propStartBlock = getFastForwardMilestoneStartBlock(milestone.EndBlock, params.FfMilestoneBlockInterval)
		}

		lastMilestoneHash = milestone.Hash
		lastMilestoneBlockNumber = milestone.EndBlock
	}

	params, err := milestoneKeeper.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	parentHash, blockHashes, tds, authors, effectiveLatestHeader, err := getBlockInfo(ctx, contractCaller, propStartBlock, params.MaxMilestonePropositionLength, latestHeader, lastMilestoneHash, lastMilestoneBlockNumber)
	if err != nil {
		// Propagate ErrNoNewHeadersFound so the caller can handle it gracefully.
		// Other errors are also propagated.
		return nil, err
	}

	if err := validateMilestonePropositionFork(parentHash, lastMilestoneHash); err != nil {
		return nil, err
	}

	// Use the header getBlockInfo actually built the proposition from (it refreshes a stale cached
	// header internally), not the caller's possibly-stale latestHeader — otherwise the reported actual
	// head can fall behind the proposition's own end and fail validateLatestHead.
	latestBlockNumber, latestBlockHash := actualHeadFields(ctx, effectiveLatestHeader)

	if err := borKeeper.CanVoteProducers(ctx); err == nil {
		validIndex := 0
		for i := 0; i < len(authors); i++ {
			allowedAuthors, err := getBlockAuthor(ctx, propStartBlock+uint64(i))
			if err != nil {
				return nil, err
			}

			if slices.Contains(allowedAuthors, authors[i]) || propStartBlock+uint64(i) == 0 {
				validIndex = i + 1
			} else {
				break
			}
		}

		if validIndex == 0 {
			return nil, fmt.Errorf("no valid block author found")
		}

		return &types.MilestoneProposition{
			BlockHashes:       blockHashes[:validIndex],
			StartBlockNumber:  propStartBlock,
			ParentHash:        parentHash,
			BlockTds:          tds[:validIndex],
			LatestBlockNumber: latestBlockNumber,
			LatestBlockHash:   latestBlockHash,
		}, nil
	}

	return &types.MilestoneProposition{
		BlockHashes:       blockHashes,
		StartBlockNumber:  propStartBlock,
		ParentHash:        parentHash,
		BlockTds:          tds,
		LatestBlockNumber: latestBlockNumber,
		LatestBlockHash:   latestBlockHash,
	}, nil
}

// actualHeadFields returns the actual latest bor head (number, hash) to embed in a proposition for
// the pending-stall rotation to key on instead of the capped proposition tail. Emission
// is fork-gated on the vote extension's own height: VE validation rejects unknown fields, so these
// must not appear before the Ithaca height, by which point the network has done the
// coordinated upgrade the fork requires. Returns zero/nil when the fork is off or no latest header
// is available (e.g. no prior milestone).
func actualHeadFields(ctx sdk.Context, latestHeader *ethTypes.Header) (uint64, []byte) {
	if !helper.IsIthaca(ctx.BlockHeight()) || latestHeader == nil {
		return 0, nil
	}
	return latestHeader.Number.Uint64(), latestHeader.Hash().Bytes()
}

func isFastForwardMilestone(latestHeaderNumber, latestMilestoneEndBlock, ffMilestoneThreshold uint64) bool {
	return latestHeaderNumber > latestMilestoneEndBlock && latestHeaderNumber-latestMilestoneEndBlock > ffMilestoneThreshold
}

func getFastForwardMilestoneStartBlock(latestMilestoneEndBlock, ffMilestoneBlockInterval uint64) uint64 {
	return latestMilestoneEndBlock + ffMilestoneBlockInterval
}

func GetMajorityMilestoneProposition(
	ctx sdk.Context,
	validatorSet *stakeTypes.ValidatorSet,
	extVoteInfo []abciTypes.ExtendedVoteInfo,
	majorityVP int64,
	logger log.Logger,
	lastEndBlock *uint64,
	lastEndBlockHash []byte,
) (*types.MilestoneProposition, []byte, string, map[uint64]struct{}, error) {
	ac := address.HexCodec{}

	// Track voting power per block number
	blockVotingPower := make(map[uint64]int64)
	blockHashVotes := make(map[uint64]map[string]int64) // block -> (hash + td) -> voting power
	blockToHashAndTd := make(map[uint64][]byte)
	validatorVotes := make(map[string]map[uint64][]byte) // validator -> block -> (hash + td)
	validatorAddresses := make(map[string][]byte)
	valAddressToVotingPower := make(map[string]int64)
	parentHashes := make(map[string]struct{})
	parentHashToVotingPower := make(map[string]int64)

	// Track which validators we've already processed to prevent duplicate votes
	processedValidators := make(map[string]bool)

	getParentChildKey := func(parent, child string) string {
		return fmt.Sprintf("%s-%s", parent, child)
	}

	// First pass - collect all votes
	for _, vote := range extVoteInfo {
		if vote.BlockIdFlag != cmtTypes.BlockIDFlagCommit {
			continue
		}

		voteExtension := new(sidetxs.VoteExtension)
		if err := voteExtension.Unmarshal(vote.VoteExtension); err != nil {
			return nil, nil, "", nil, fmt.Errorf("error while unmarshalling vote extension: %w", err)
		}

		if voteExtension.MilestoneProposition == nil {
			continue
		}

		valAddr, err := ac.BytesToString(vote.Validator.Address)
		if err != nil {
			return nil, nil, "", nil, err
		}

		// Skip if we've already processed a vote from this validator
		if processedValidators[valAddr] {
			logger.Debug("Skipping duplicate vote from validator", "validator", valAddr)
			continue
		}

		// Mark this validator as processed
		processedValidators[valAddr] = true

		_, validator := validatorSet.GetByAddress(valAddr)
		if validator == nil {
			if ShouldErrorOnValidatorNotFound(ctx.BlockHeight()) {
				return nil, nil, "", nil, errors.New(helper.ErrFailedToGetValidator(valAddr))
			}
			continue
		}

		validatorAddresses[valAddr] = vote.Validator.Address
		valAddressToVotingPower[valAddr] = validator.VotingPower
		validatorVotes[valAddr] = make(map[uint64][]byte)

		prop := voteExtension.MilestoneProposition
		for i, blockHash := range prop.BlockHashes {
			blockTd := prop.BlockTds[i]
			var buf bytes.Buffer
			if err := binary.Write(&buf, binary.LittleEndian, blockTd); err != nil {
				return nil, nil, "", nil, fmt.Errorf("failed to convert td to binary: %w", err)
			}

			// Hash Bytes + Td Bytes
			tdBytes := [8]byte(buf.Bytes()) // enforce 8 bytes
			blockHashAndTd := append(blockHash, tdBytes[:]...)

			blockNum := prop.StartBlockNumber + uint64(i)

			// Record this validator's vote for this block
			validatorVotes[valAddr][blockNum] = blockHashAndTd

			// Initialize maps if needed
			if _, ok := blockVotingPower[blockNum]; !ok {
				blockVotingPower[blockNum] = 0
				blockHashVotes[blockNum] = make(map[string]int64)
			}

			// Record block hash -> voting power
			hashStr := common.Bytes2Hex(blockHashAndTd)
			blockHashVotes[blockNum][hashStr] += validator.VotingPower

			// Track the hash that currently has the most votes for this block
			// Use a deterministic comparison to break ties
			if blockHashVotes[blockNum][hashStr] > blockVotingPower[blockNum] ||
				(blockHashVotes[blockNum][hashStr] == blockVotingPower[blockNum] &&
					hashStr < common.Bytes2Hex(blockToHashAndTd[blockNum])) {
				blockVotingPower[blockNum] = blockHashVotes[blockNum][hashStr]
				blockToHashAndTd[blockNum] = blockHashAndTd
			}

			key := getParentChildKey(common.Bytes2Hex(prop.ParentHash), common.Bytes2Hex(blockHashAndTd))
			parentHashToVotingPower[key] += validator.VotingPower
		}
		parentHashes[common.Bytes2Hex(prop.ParentHash)] = struct{}{}
	}

	// Find blocks with majority support - use a slice for deterministic ordering
	blockNumbers := make([]uint64, 0, len(blockVotingPower))
	for blockNum := range blockVotingPower {
		blockNumbers = append(blockNumbers, blockNum)
	}
	sort.Slice(blockNumbers, func(i, j int) bool {
		return blockNumbers[i] < blockNumbers[j]
	})

	var majorityBlocks []uint64
	for _, blockNum := range blockNumbers {
		if blockVotingPower[blockNum] >= majorityVP {
			majorityBlocks = append(majorityBlocks, blockNum)
		}
	}

	if len(majorityBlocks) == 0 {
		logger.Debug("No blocks found with majority support")
		return nil, nil, "", nil, nil
	}

	startBlock := uint64(0)

	// Check if we have a block that starts exactly from the (last end block + 1)
	if lastEndBlock != nil {
		startBlock = *lastEndBlock + 1

		if majorityBlocks[0] > startBlock {
			startBlock = majorityBlocks[0]
		}
	}

	// Check if startBlock is in majorityBlocks
	startBlockFound := false
	for _, blockNum := range majorityBlocks {
		if blockNum == startBlock {
			startBlockFound = true
			break
		}
	}

	if !startBlockFound {
		logger.Debug("No blocks with majority support starting at requested block",
			"requestedStartBlock", startBlock)
		return nil, nil, "", nil, nil
	}

	// Validate the proposition's parent against the last milestone's end block hash — the only
	// legitimate parent. Parent hashes are not bound by ValidateMilestoneProposition, so a byzantine
	// slice can vote real blocks under a fabricated parent; under the 1/3 pending threshold that bogus
	// parent can clear majority alongside the honest one. The check is hardfork-gated so already-live
	// behavior is preserved and only the gate boundary ever changes behavior, never a finalized height:
	//   - Ithaca: key by the block we will actually return (startBlock), because earlier overlapping
	//     blocks may also have majority support and must not decide the parent;
	//   - Zurich (already live): the deployed direct lookup keyed on majorityBlocks[0];
	//   - pre-Zurich: the legacy tournament over proposed parents.
	// Ithaca is checked first since it activates after Zurich.
	lastEndBlockHashHex := common.Bytes2Hex(lastEndBlockHash)
	if helper.IsIthaca(ctx.BlockHeight()) {
		key := getParentChildKey(lastEndBlockHashHex, common.Bytes2Hex(blockToHashAndTd[startBlock]))
		if parentHashToVotingPower[key] < majorityVP {
			logger.Debug("No parent hash with majority support matching the last end block hash",
				"lastEndBlockHash", lastEndBlockHashHex)
			return nil, nil, "", nil, nil
		}
	} else if helper.IsZurichHardfork(ctx.BlockHeight()) {
		key := getParentChildKey(lastEndBlockHashHex, common.Bytes2Hex(blockToHashAndTd[majorityBlocks[0]]))
		if parentHashToVotingPower[key] < majorityVP {
			logger.Debug("Parent hash does not match last end block hash",
				"lastEndBlockHash", lastEndBlockHashHex)
			return nil, nil, "", nil, nil
		}
	} else {
		var majorityParentHash string
		isParentHashMajority := false
		for parentHash := range parentHashes {
			key := getParentChildKey(parentHash, common.Bytes2Hex(blockToHashAndTd[majorityBlocks[0]]))
			if parentHashToVotingPower[key] >= majorityVP {
				isParentHashMajority = true
				majorityParentHash = parentHash
				break
			}
		}
		if !isParentHashMajority || majorityParentHash != lastEndBlockHashHex {
			logger.Debug("Parent hash does not match last end block hash",
				"majorityParentHash", majorityParentHash,
				"lastEndBlockHash", lastEndBlockHashHex)
			return nil, nil, "", nil, nil
		}
	}

	// Find the first continuous range starting from startBlock
	endBlock := startBlock
	for i := 0; i < len(majorityBlocks); i++ {
		if majorityBlocks[i] == startBlock {
			// Find continuous blocks after startBlock
			for j := i + 1; j < len(majorityBlocks); j++ {
				if majorityBlocks[j] == endBlock+1 {
					endBlock = majorityBlocks[j]
				} else {
					break
				}
			}
			break
		}
	}

	blockCount := endBlock - startBlock + 1
	blockHashesAndTds := make([][]byte, 0, blockCount)
	for i := startBlock; i <= endBlock; i++ {
		blockHashesAndTds = append(blockHashesAndTds, blockToHashAndTd[i])
	}

	// Find validators who support the entire winning range
	var supportingValidatorList []string
	supportingValidatorIDs := make(map[uint64]struct{})

	for valAddr, blocks := range validatorVotes {
		supports := true
		for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
			hash, hasBlock := blocks[blockNum]
			if !hasBlock || !bytes.Equal(hash, blockToHashAndTd[blockNum]) {
				supports = false
				break
			}
		}
		if supports {
			supportingValidatorList = append(supportingValidatorList, valAddr)
			_, validator := validatorSet.GetByAddress(valAddr)
			if validator != nil {
				supportingValidatorIDs[validator.ValId] = struct{}{}
			}
		}
	}

	// Sort validators deterministically
	sort.Strings(supportingValidatorList)

	// Verify that we still have a 2/3 majority after filtering
	totalSupportingPower := int64(0)
	for _, valAddr := range supportingValidatorList {
		totalSupportingPower += valAddressToVotingPower[valAddr]
	}

	if totalSupportingPower < majorityVP {
		logger.Debug("After filtering validators, no range has 2/3 majority support",
			"totalSupportingPower", totalSupportingPower,
			"requiredPower", majorityVP)
		return nil, nil, "", nil, nil
	}

	// Additional sort by voting power (stable to preserve string order when tied)
	sort.SliceStable(supportingValidatorList, func(i, j int) bool {
		return valAddressToVotingPower[supportingValidatorList[i]] > valAddressToVotingPower[supportingValidatorList[j]]
	})

	if len(supportingValidatorList) == 0 {
		return nil, nil, "", nil, fmt.Errorf("no validators support the winning range")
	}

	// Generate aggregated proposers hash from supporting validators
	var aggregatedProposersHash []byte
	for _, valAddr := range supportingValidatorList {
		aggregatedProposersHash = crypto.Keccak256(
			aggregatedProposersHash,
			[]byte{'|'},
			validatorAddresses[valAddr],
		)
	}

	// splitting back blockHashAndTd
	blockHashes := make([][]byte, 0, len(blockHashesAndTds))
	blockTds := make([]uint64, 0, len(blockHashesAndTds))
	for _, blockHashAndTd := range blockHashesAndTds {
		tdBytes := blockHashAndTd[len(blockHashAndTd)-8:] // the last 8 bytes are the TD
		blockTds = append(blockTds, binary.LittleEndian.Uint64(tdBytes))

		blockHashes = append(blockHashes, blockHashAndTd[:len(blockHashAndTd)-8])
	}

	// Create a final proposition
	proposition := &types.MilestoneProposition{
		BlockHashes:      blockHashes,
		StartBlockNumber: startBlock,
		BlockTds:         blockTds,
	}

	logger.Debug("Found majority milestone proposition",
		"startBlock", startBlock,
		"endBlock", endBlock,
		"blockCount", blockCount,
		"supportingValidators", len(supportingValidatorList))

	return proposition, aggregatedProposersHash, supportingValidatorList[0], supportingValidatorIDs, nil
}

// GetMajorityActualHead tallies the actual latest bor head reported in vote extensions
// and returns the (number, hash) with the greatest summed voting power that reaches minMajorityVP,
// with found=true. Each validator reports a single latest head; during a stall honest validators
// converge on the same head, so the most-voted head is the real tip, and a >1/3 byzantine minority
// reporting a fabricated head cannot outvote that stronger honest agreement. found is false when no
// head clears minMajorityVP — the caller then skips rotation rather than falling back to the
// truncated proposition tail. Deterministic: uses canonical voting power from the validator set,
// dedupes per validator, and breaks equal-power ties on the lexicographically smaller key, mirroring
// GetMajorityMilestoneProposition.
//
// Heads beyond maxBlock (the last span's end — an honest producer cannot advance past the scheduled
// runway) are dropped before the tally, so a colluding minority cannot push the agreed head past
// chain state, keeping the downstream stall tracking from being poisoned with an out-of-range head.
func GetMajorityActualHead(
	ctx sdk.Context,
	validatorSet *stakeTypes.ValidatorSet,
	extVoteInfo []abciTypes.ExtendedVoteInfo,
	minMajorityVP int64,
	maxBlock uint64,
) (uint64, []byte, bool, error) {
	tally, err := tallyActualHeads(ctx, validatorSet, extVoteInfo, maxBlock)
	if err != nil {
		return 0, nil, false, err
	}
	num, hash, found := tally.majority(minMajorityVP)
	return num, hash, found, nil
}

// actualHeadTally accumulates voting power per reported (number, hash) actual bor head, keyed on
// number ++ hash so distinct heads (including same-height forks) tally separately.
type actualHeadTally struct {
	power  map[string]int64
	number map[string]uint64
	hash   map[string][]byte
}

func (t *actualHeadTally) add(number uint64, hash []byte, vp int64) {
	var numBuf [8]byte
	binary.LittleEndian.PutUint64(numBuf[:], number)
	key := common.Bytes2Hex(append(numBuf[:], hash...))
	t.power[key] += vp
	t.number[key] = number
	t.hash[key] = hash
}

// majority returns the actual head with the greatest summed voting power that reaches minMajorityVP,
// with found=true. Greatest voting power wins, not the highest block number: during a genuine stall
// honest validators converge on the same head, so the most-voted head is the real tip, and a >1/3
// byzantine minority reporting a fabricated higher head cannot outvote the honest agreement (which
// also stops it from resetting the stall clock by rotating a fake head's hash). Ties on equal summed
// power break on the lexicographically smaller tally key, keeping the choice deterministic across
// validators.
func (t *actualHeadTally) majority(minMajorityVP int64) (uint64, []byte, bool) {
	keys := make([]string, 0, len(t.power))
	for k := range t.power {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	found := false
	var bestPower int64
	var bestNum uint64
	var bestHash []byte
	for _, k := range keys {
		if t.power[k] < minMajorityVP {
			continue
		}
		if !found || t.power[k] > bestPower {
			found = true
			bestPower = t.power[k]
			bestNum = t.number[k]
			bestHash = t.hash[k]
		}
	}
	return bestNum, bestHash, found
}

// tallyActualHeads sums canonical voting power per reported actual head across the vote extensions,
// deduping repeated votes per validator. Heads beyond maxBlock are dropped: an honest producer
// cannot advance past its span end, so such a head is fabricated and must not enter the tally.
func tallyActualHeads(ctx sdk.Context, validatorSet *stakeTypes.ValidatorSet, extVoteInfo []abciTypes.ExtendedVoteInfo, maxBlock uint64) (*actualHeadTally, error) {
	ac := address.HexCodec{}
	tally := &actualHeadTally{power: map[string]int64{}, number: map[string]uint64{}, hash: map[string][]byte{}}
	processed := make(map[string]bool)

	for _, vote := range extVoteInfo {
		number, hash, ok, err := decodeActualHeadVote(vote)
		if err != nil {
			return nil, err
		}
		if !ok || number > maxBlock {
			continue
		}
		vp, ok, err := resolveVoterPower(ctx, validatorSet, ac, vote, processed)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		tally.add(number, hash, vp)
	}
	return tally, nil
}

// decodeActualHeadVote extracts the actual latest-head fields from a committed vote extension. ok is
// false for non-committed votes, missing fields, or a malformed latest-head hash (skip them).
func decodeActualHeadVote(vote abciTypes.ExtendedVoteInfo) (uint64, []byte, bool, error) {
	if vote.BlockIdFlag != cmtTypes.BlockIDFlagCommit {
		return 0, nil, false, nil
	}
	ve := new(sidetxs.VoteExtension)
	if err := ve.Unmarshal(vote.VoteExtension); err != nil {
		return 0, nil, false, fmt.Errorf("error while unmarshalling vote extension: %w", err)
	}
	if ve.MilestoneProposition == nil || len(ve.MilestoneProposition.LatestBlockHash) == 0 {
		return 0, nil, false, nil
	}
	if len(ve.MilestoneProposition.LatestBlockHash) != common.HashLength {
		return 0, nil, false, nil
	}
	return ve.MilestoneProposition.LatestBlockNumber, ve.MilestoneProposition.LatestBlockHash, true, nil
}

// resolveVoterPower returns the canonical voting power of a vote's validator, deduping repeats. ok is
// false when the vote should be skipped (duplicate validator, or absent validator when non-fatal).
func resolveVoterPower(ctx sdk.Context, validatorSet *stakeTypes.ValidatorSet, ac address.HexCodec, vote abciTypes.ExtendedVoteInfo, processed map[string]bool) (int64, bool, error) {
	valAddr, err := ac.BytesToString(vote.Validator.Address)
	if err != nil {
		return 0, false, err
	}
	if processed[valAddr] {
		return 0, false, nil
	}
	processed[valAddr] = true

	_, validator := validatorSet.GetByAddress(valAddr)
	if validator == nil {
		if ShouldErrorOnValidatorNotFound(ctx.BlockHeight()) {
			return 0, false, errors.New(helper.ErrFailedToGetValidator(valAddr))
		}
		return 0, false, nil
	}
	return validator.VotingPower, true, nil
}

var ErrNoNewHeadersFound = errors.New("no new headers found for milestone proposition")

func getBlockInfo(ctx sdk.Context, contractCaller helper.IContractCaller, startBlockNum, maxBlocksInProposition uint64, latestHeader *ethTypes.Header, lastMilestoneHash []byte, lastMilestoneBlock uint64) ([]byte, [][]byte, []uint64, []common.Address, *ethTypes.Header, error) {
	// Reuse the provided latestHeader if available, otherwise fetch it.
	var err error
	if latestHeader == nil {
		latestHeader, err = contractCaller.GetBorChainBlock(ctx, nil)
		if err != nil || latestHeader == nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("failed to get the latest header: %w", err)
		}
	}

	latestBlockNum := latestHeader.Number.Uint64()

	// Check if there are any new blocks available to fetch.
	// If the cached latestHeader is stale (latestBlockNum < startBlockNum), refresh it once in case Bor produced it in the meantime.
	// This handles the case where Heimdall blocks faster than Bor.
	if latestBlockNum < startBlockNum {
		latestHeader, err = contractCaller.GetBorChainBlock(ctx, nil)
		if err != nil || latestHeader == nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("failed to refresh the latest header: %w", err)
		}
		latestBlockNum = latestHeader.Number.Uint64()
		// If still not available, return ErrNoNewHeadersFound since Bor hasn't produced the block yet.
		// GenMilestoneProposition will propagate this, and app/abci.go will handle it gracefully.
		if latestBlockNum < startBlockNum {
			return nil, nil, nil, nil, nil, ErrNoNewHeadersFound
		}
	}

	// Calculate how many blocks are actually available to fetch from the Bor chain.
	availableBlocks := latestBlockNum - startBlockNum + 1

	// Only fetch the minimum of available blocks and max blocks in proposition
	// This optimizes RPC calls when synced (e.g., only 1-2 blocks are actually available to fetch).
	blocksToFetch := min(availableBlocks, maxBlocksInProposition)

	milestoneEnd := startBlockNum + blocksToFetch - 1

	headers, tds, authors, err := contractCaller.GetBorChainBlockInfoInBatch(ctx, int64(startBlockNum), int64(milestoneEnd))
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to get block batch info: %w", err)
	}

	if len(headers) == 0 {
		return nil, nil, nil, nil, nil, ErrNoNewHeadersFound
	}

	result := make([][]byte, 0, len(headers))

	var parentHash []byte
	if len(headers) > 0 && len(lastMilestoneHash) > 0 {
		parentHash = headers[0].ParentHash.Bytes()
		if startBlockNum-lastMilestoneBlock > 1 {
			header, err := contractCaller.GetBorChainBlock(ctx, big.NewInt(int64(lastMilestoneBlock+1)))
			if err != nil {
				return nil, nil, nil, nil, nil, fmt.Errorf("failed to get header for parent hash: %w", err)
			}

			parentHash = header.ParentHash.Bytes()
		}
	}

	for _, h := range headers {
		result = append(result, h.Hash().Bytes())
	}

	return parentHash, result, tds, authors, latestHeader, nil
}

func validateMilestonePropositionFork(parentHash []byte, lastMilestoneHash []byte) error {
	if len(parentHash) > 0 && len(lastMilestoneHash) > 0 {
		if !bytes.Equal(parentHash, lastMilestoneHash) {
			return fmt.Errorf("first block parent hash does not match last milestone hash")
		}
	}
	return nil
}

func ValidateMilestoneProposition(ctx sdk.Context, milestoneKeeper *keeper.Keeper, milestoneProp *types.MilestoneProposition) error {
	if milestoneProp == nil {
		return nil
	}

	params, err := milestoneKeeper.GetParams(ctx)
	if err != nil {
		return err
	}

	if len(milestoneProp.BlockHashes) > int(params.MaxMilestonePropositionLength) {
		return fmt.Errorf("too many blocks in proposition")
	}

	if len(milestoneProp.BlockHashes) == 0 {
		return fmt.Errorf("no blocks in proposition")
	}

	if len(milestoneProp.BlockHashes) != len(milestoneProp.BlockTds) {
		return fmt.Errorf("len mismatch between hashes and tds: %d != %d", len(milestoneProp.BlockHashes), len(milestoneProp.BlockTds))
	}

	duplicateBlockHashes := make(map[string]struct{})
	for _, blockHash := range milestoneProp.BlockHashes {
		if len(blockHash) != common.HashLength {
			return fmt.Errorf("invalid block hash length")
		}
		duplicateBlockHashes[string(blockHash)] = struct{}{}
	}

	if len(duplicateBlockHashes) != len(milestoneProp.BlockHashes) {
		return fmt.Errorf("duplicate block hashes found")
	}

	// Older binaries treat the Ithaca latest-head fields as unknown protobuf fields and reject the
	// entire vote extension. Reject them explicitly on upgraded binaries before activation too, so a
	// byzantine validator cannot make old and new validators disagree during the mixed-version rollout.
	if !helper.IsIthaca(ctx.BlockHeight()) {
		if milestoneProp.LatestBlockNumber != 0 || len(milestoneProp.LatestBlockHash) != 0 {
			return fmt.Errorf("latest block fields set before Ithaca")
		}
		return nil
	}

	return validateLatestHead(milestoneProp)
}

// validateLatestHead checks the optional actual-head fields. They are emitted together
// post-fork or both absent pre-fork, so a partially-populated pair is rejected. When present, the
// head cannot be behind the proposition's own last block. Assumes BlockHashes is non-empty (the
// caller checks that).
func validateLatestHead(milestoneProp *types.MilestoneProposition) error {
	if len(milestoneProp.LatestBlockHash) == 0 {
		if milestoneProp.LatestBlockNumber != 0 {
			return fmt.Errorf("latest block number set without latest block hash")
		}
		return nil
	}
	if len(milestoneProp.LatestBlockHash) != common.HashLength {
		return fmt.Errorf("invalid latest block hash length")
	}
	// StartBlockNumber is attacker-influenced; compute the proposition end overflow-safe before comparing.
	offset := uint64(len(milestoneProp.BlockHashes) - 1)
	if milestoneProp.StartBlockNumber > math.MaxUint64-offset {
		return fmt.Errorf("proposition start block overflow")
	}
	propEnd := milestoneProp.StartBlockNumber + offset
	if milestoneProp.LatestBlockNumber < propEnd {
		return fmt.Errorf("latest block number %d behind proposition end %d", milestoneProp.LatestBlockNumber, propEnd)
	}
	// When the head is the proposition's own last block, both reference the same block, so their
	// hashes must match. Beyond propEnd the head is legitimately outside the proposition window and
	// the cross-check doesn't apply. Closes a vote-power split where a validator feeds the milestone
	// and actual-head tallies different hashes at the same height.
	if milestoneProp.LatestBlockNumber == propEnd &&
		!bytes.Equal(milestoneProp.LatestBlockHash, milestoneProp.BlockHashes[len(milestoneProp.BlockHashes)-1]) {
		return fmt.Errorf("latest block hash does not match proposition tail at height %d", propEnd)
	}
	return nil
}

func ShouldErrorOnValidatorNotFound(height int64) bool {
	return height >= helper.GetTallyFixHeight() || height < helper.GetDisableValSetCheckHeight()
}
