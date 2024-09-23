package app

import (
	"encoding/json"
	"sort"

	"cosmossdk.io/log"
	mod "github.com/0xPolygon/heimdall-v2/module"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

/* TODO HV2: uncomment when stake is merged
// ValidateVoteExtensions is a helper function for verifying vote extension
// signatures by a proposer during PrepareProposal and validators during ProcessProposal.
// It returns an error if any signature is invalid or if unexpected vote extensions and/or signatures are found or less than 2/3
// power is received.
func ValidateVoteExtensions(ctx sdk.Context,
	currentHeight int64,
	chainID string,
	extVoteInfo []abci.ExtendedVoteInfo,
	round int32,
	stakeKeeper stakekeeper.StakeKeeper) error {
	cp := ctx.ConsensusParams()
	extsEnabled := cp.Abci != nil && currentHeight >= cp.Abci.VoteExtensionsEnableHeight && cp.Abci.VoteExtensionsEnableHeight != 0

	marshalDelimitedFn := func(msg proto.Message) ([]byte, error) {
		var buf bytes.Buffer
		if err := protoio.NewDelimitedWriter(&buf).WriteMsg(msg); err != nil {
			return nil, err
		}

		return buf.Bytes(), nil
	}

	// Fetch validators from previous block
	// TODO HV2: Heimdall as of now uses validator set from currentHeight. But should we be taking into account the validator set from currentHeight - 1/ currentHeight - 2 ?
	validators := stakeKeeper.GetValidators(ctx, currentHeight-1)
	if len(validators) == 0 {
		return errors.New("No validators found")
	}

	// calculate total voting power
	// var totalVP int64
	// for _, v := range validators {
	// 	totalVP += v.Power
	// }

	sumVP := math.NewInt(0)

	for _, vote := range extVoteInfo {
		if !extsEnabled {
			if len(vote.VoteExtension) > 0 {
				return fmt.Errorf("vote extensions disabled; received non-empty vote extension at height %d", currentHeight)
			}
			if len(vote.ExtensionSignature) > 0 {
				return fmt.Errorf("vote extensions disabled; received non-empty vote extension signature at height %d", currentHeight)
			}

			continue
		}

		if len(vote.ExtensionSignature) == 0 {
			return fmt.Errorf("vote extensions enabled; received empty vote extension signature at height %d", currentHeight)
		}

		valAddrStr := hex.EncodeToString(vote.Validator.Address)
		valAddr, err := sdk.ConsAddressFromHex(valAddrStr)

		if err != nil {
			return err
		}

		validator, err := keeper.GetValidatorByConsAddr(ctx, valAddr)
		if err != nil {
			return fmt.Errorf("failed to get validator %X: %w", valAddr, err)
		}

		cmtPubKeyProto, err := validator.CmtConsPublicKey()
		if err != nil {
			return fmt.Errorf("failed to get validator %X public key: %w", valAddr, err)
		}

		cmtPubKey, err := cryptoenc.PubKeyFromProto(cmtPubKeyProto)
		if err != nil {
			return fmt.Errorf("failed to convert validator %X public key: %w", valAddr, err)
		}

		cve := cmtproto.CanonicalVoteExtension{
			Extension: vote.VoteExtension,
			Height:    currentHeight - 1, // the vote extension was signed in the previous height
			Round:     int64(round),
			ChainId:   chainID,
		}

		extSignBytes, err := marshalDelimitedFn(&cve)
		if err != nil {
			return fmt.Errorf("failed to encode CanonicalVoteExtension: %w", err)
		}

		if !cmtPubKey.VerifySignature(extSignBytes, vote.ExtensionSignature) {
			return fmt.Errorf("failed to verify validator %X vote extension signature", valAddr)
		}

		sumVP = sumVP.Add(validator.Power)

	}

	// Ensure we have at least 2/3 voting power that submitted valid vote
	// extensions for each side tx msg.
	if sumVP < 2/3*(totalVP)+1 {
		return fmt.Errorf("insufficient cumulative voting power received to verify vote extensions; got: %s, expected: >=%s", sumVP, totalVP)
	}

	return nil
}
*/

// tallyVotes is a helper function to tally votes received for the side txs
// It returns lists of txs which got >2/3+ YES, NO and SKIP votes
//
// nolint:unused
func tallyVotes(extVoteInfo []abci.ExtendedVoteInfo, logger log.Logger, validators []abci.Validator) ([][]byte, [][]byte, [][]byte, error) {
	logger.Debug("Tallying votes")

	// calculate total voting power
	var totalVP int64
	for _, v := range validators {
		totalVP += v.Power
	}

	voteByTxHash, err := aggregateVotes(extVoteInfo)
	if err != nil {
		return nil, nil, nil, err
	}

	// check for vote majority
	txHashList := make([]string, 0, len(voteByTxHash))
	for txHash := range voteByTxHash {
		txHashList = append(txHashList, txHash)
	}

	sort.Strings(txHashList)

	approvedTxs, rejectedTxs, skippedTxs := make([][]byte, 0, len(txHashList)), make([][]byte, 0, len(txHashList)), make([][]byte, 0, len(txHashList))

	for _, txHash := range txHashList {
		voteMap := voteByTxHash[txHash]
		if voteMap[mod.Vote_VOTE_YES] >= (totalVP*2/3 + 1) {
			// approved
			logger.Debug("Approved side-tx", "txHash", txHash)

			// append to approved tx slice
			approvedTxs = append(approvedTxs, []byte(txHash))
		} else if voteMap[mod.Vote_VOTE_NO] >= (totalVP*2/3 + 1) {
			// rejected
			logger.Debug("Rejected side-tx", "txHash", txHash)

			// append to rejected tx slice
			rejectedTxs = append(rejectedTxs, []byte(txHash))
		} else {
			// skipped
			logger.Debug("Skipped side-tx", "txHash", txHash)

			// append to rejected tx slice
			skippedTxs = append(skippedTxs, []byte(txHash))
		}
	}

	return approvedTxs, rejectedTxs, skippedTxs, nil
}

// aggregateVotes collates votes received for a side tx
func aggregateVotes(extVoteInfo []abci.ExtendedVoteInfo) (map[string]map[mod.Vote]int64, error) {
	voteByTxHash := make(map[string]map[mod.Vote]int64)      // track votes for a side tx
	validatorToTxMap := make(map[string]map[string]struct{}) // ensure a validator doesn't procure conflicting votes for a side tx

	var ve mod.CanonicalSideTxResponse

	for _, vote := range extVoteInfo {
		if err := json.Unmarshal(vote.VoteExtension, &ve); err != nil {
			return nil, err
		}

		addrStr := string(vote.Validator.Address[:])

		// iterate through vote extensions and accumulate voting power for YES/NO/SKIP votes
		for _, res := range ve.SideTxResponses {
			txHashStr := string(res.TxHash[:])

			// TODO HV2: do we slash in case a validator maliciously adds conflicting votes ?
			// Given that we also check for duplicate votes during VerifyVoteExtension, is this redundant ?
			if _, hasVoted := validatorToTxMap[addrStr][txHashStr]; !hasVoted {
				voteByTxHash[string(res.TxHash[:])][res.Result] += vote.Validator.Power

				// validator's vote received; mark it avoid duplicate votes
				validatorToTxMap[addrStr][txHashStr] = struct{}{}
			}

		}

	}

	return voteByTxHash, nil
}

// checkDuplicateVotes detects duplicate votes by a validator for a side tx
func checkDuplicateVotes(sideTxResponses []*mod.SideTxResponse) (bool, []byte) {
	// track votes of the validator
	txVoteMap := make(map[string]struct{})

	for _, res := range sideTxResponses {
		if _, ok := txVoteMap[string(res.TxHash)]; ok {
			return true, res.TxHash
		}

		txVoteMap[string(res.TxHash)] = struct{}{}
	}

	return false, nil
}

// canAddVE indicates whether the proposer can include V.E in the block proposal from previous height
func canAddVE(ctx sdk.Context, height int64) bool {
	return height >= ctx.ConsensusParams().Abci.VoteExtensionsEnableHeight+1
}
