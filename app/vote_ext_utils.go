package app

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	cryptoenc "github.com/cometbft/cometbft/crypto/encoding"
	"github.com/cometbft/cometbft/libs/protoio"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	stakeKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// ValidateVoteExtensions verifies the vote extension correctness
// It checks the signature of the proposer (during PrepareProposal) or validators (during ProcessProposal)
// Also, it checks if the vote extensions are enabled, valid and have >2/3 voting power
// It returns an error in case the validation fails
func ValidateVoteExtensions(ctx sdk.Context, reqHeight int64, extVoteInfo []abciTypes.ExtendedVoteInfo, round int32, stakeKeeper stakeKeeper.Keeper) error {
	currentHeight := ctx.BlockHeight()

	// check if VEs are enabled
	mustAddSpecialTransaction(ctx, reqHeight+1)

	// Fetch validatorSet from previous block
	// TODO HV2: Heimdall as of now uses validator set from current height.
	//  Should we be taking into account the validator set from currentHeight-1 or currentHeight-2? Discuss with PoS team
	validatorSet, err := stakeKeeper.GetValidatorSet(ctx)
	if err != nil {
		return err
	}
	if len(validatorSet.Validators) == 0 {
		return errors.New("no validators found in validator set")
	}

	// calculate total voting power
	var totalVP int64
	for _, v := range validatorSet.Validators {
		totalVP += v.VotingPower
	}

	sumVP := math.NewInt(0)

	for _, vote := range extVoteInfo {
		// make sure the BlockIdFlag is valid
		if vote.BlockIdFlag == cmtTypes.BlockIDFlagUnknown {
			return fmt.Errorf("received vote with unknown block ID flag at height %d", currentHeight)
		}
		// if not BlockIDFlagCommit, skip that vote, as it doesn't have relevant information
		if vote.BlockIdFlag != cmtTypes.BlockIDFlagCommit {
			continue
		}
		if len(vote.ExtensionSignature) == 0 {
			return fmt.Errorf("received empty vote extension signature at height %d", currentHeight)
		}

		codec := address.HexCodec{}
		valAddrStr, err := codec.BytesToString(vote.Validator.Address)
		if err != nil {
			return err
		}

		validator, err := stakeKeeper.GetValidatorInfo(ctx, valAddrStr)
		if err != nil {
			return fmt.Errorf("failed to get validator %s: %w", valAddrStr, err)
		}

		cmtPubKeyProto, err := validator.CmtConsPublicKey()
		if err != nil {
			return fmt.Errorf("failed to get validator %s public key: %w", valAddrStr, err)
		}

		cmtPubKey, err := cryptoenc.PubKeyFromProto(cmtPubKeyProto)
		if err != nil {
			return fmt.Errorf("failed to convert validator %s public key: %w", valAddrStr, err)
		}

		cve := cmtTypes.CanonicalVoteExtension{
			Extension: vote.VoteExtension,
			Height:    currentHeight - 1, // the vote extension was signed in the previous height
			Round:     int64(round),
			ChainId:   ctx.ChainID(),
		}

		marshalDelimitedFn := func(msg proto.Message) ([]byte, error) {
			var buf bytes.Buffer
			if _, err := protoio.NewDelimitedWriter(&buf).WriteMsg(msg); err != nil {
				return nil, err
			}

			return buf.Bytes(), nil
		}
		extSignBytes, err := marshalDelimitedFn(&cve)
		if err != nil {
			return fmt.Errorf("failed to encode CanonicalVoteExtension: %w", err)
		}

		if !cmtPubKey.VerifySignature(extSignBytes, vote.ExtensionSignature) {
			return fmt.Errorf("failed to verify validator %X vote extension signature", valAddrStr)
		}

		sumVP = sumVP.Add(math.NewInt(validator.VotingPower))

	}

	// Ensure we have at least 2/3 voting power for the submitted vote extensions in each side tx
	if sumVP.Int64() <= (2*totalVP)/3 {
		return fmt.Errorf("insufficient cumulative voting power received to verify vote extensions; got: %d, expected: >=%d", sumVP.Int64(), totalVP)
	}

	return nil
}

// tallyVotes is a helper function to tally votes received for the side txs
// It returns lists of txs which got >2/3+ YES, NO and UNSPECIFIED votes respectively
func tallyVotes(extVoteInfo []abci.ExtendedVoteInfo, logger log.Logger, validators []*stakeTypes.Validator, currentHeight int64) ([][]byte, [][]byte, [][]byte, error) {
	logger.Debug("Tallying votes")

	// calculate total voting power
	var totalVP int64
	for _, v := range validators {
		totalVP += v.VotingPower
	}

	voteByTxHash, err := aggregateVotes(extVoteInfo, currentHeight)
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
		if voteMap[sidetxs.Vote_VOTE_YES] > (totalVP * 2 / 3) {
			// approved
			logger.Debug("Approved side-tx", "txHash", txHash)

			// append to approved tx slice
			approvedTxs = append(approvedTxs, []byte(txHash))
		} else if voteMap[sidetxs.Vote_VOTE_NO] > (totalVP * 2 / 3) {
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
func aggregateVotes(extVoteInfo []abci.ExtendedVoteInfo, currentHeight int64) (map[string]map[sidetxs.Vote]int64, error) {
	voteByTxHash := make(map[string]map[sidetxs.Vote]int64)  // track votes for a side tx
	validatorToTxMap := make(map[string]map[string]struct{}) // ensure a validator doesn't procure conflicting votes for a side tx

	for _, vote := range extVoteInfo {

		var ve sidetxs.ConsolidatedSideTxResponse
		// if not BlockIDFlagCommit, skip that vote, as it doesn't have relevant information
		if vote.BlockIdFlag != cmtTypes.BlockIDFlagCommit {
			continue
		}

		err := proto.Unmarshal(vote.VoteExtension, &ve)
		if err != nil {
			return nil, err
		}
		if ve.Height != currentHeight-1 {
			return nil, fmt.Errorf("invalid height received for vote extension")
		}
		addr, err := address.NewHexCodec().BytesToString(vote.Validator.Address)
		if err != nil {
			return nil, err
		}

		// iterate through vote extensions and accumulate voting power for YES/NO/UNSPECIFIED votes
		for _, res := range ve.SideTxResponses {
			txHashStr := string(res.TxHash)

			// TODO HV2: (once slashing is enabled) do we slash in case a validator maliciously adds conflicting votes ?
			// Given that we also check for duplicate votes during VerifyVoteExtension, is this redundant ?
			if _, hasVoted := validatorToTxMap[addr][txHashStr]; !hasVoted {

				if voteByTxHash[txHashStr] == nil {
					voteByTxHash[txHashStr] = make(map[sidetxs.Vote]int64)
				}

				if !isVoteValid(res.Result) {
					return nil, fmt.Errorf("invalid vote received for side tx %s", txHashStr)
				}

				voteByTxHash[txHashStr][res.Result] += vote.Validator.Power

				// validator's vote received; mark it to avoid duplicated votes
				if validatorToTxMap[addr] == nil {
					validatorToTxMap[addr] = make(map[string]struct{})
				}
				validatorToTxMap[addr][txHashStr] = struct{}{}
			}

		}

	}

	return voteByTxHash, nil
}

// checkDuplicateVotes detects duplicate votes by a validator for a side tx
func checkDuplicateVotes(sideTxResponses []*sidetxs.SideTxResponse) (bool, []byte) {
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

// mustAddSpecialTransaction indicates whether the proposer must include V.E from previous height in the block proposal as a special transaction.
// Since we are using a hard fork approach for the heimdall migration, vote extensions will be enabled from v2 genesis' initial height.
// We can use this function in case further checks are needed. Anyway, VoteExtensionsEnableHeight wil be set to 1 (first block)
func mustAddSpecialTransaction(ctx sdk.Context, height int64) {
	enableHeight := ctx.ConsensusParams().Abci.VoteExtensionsEnableHeight
	if enableHeight == 0 || height <= enableHeight {
		panic("mustAddSpecialTransaction should not be called before VoteExtensionsEnableHeight")
	}
}

func isVoteValid(v sidetxs.Vote) bool {
	return v == sidetxs.Vote_UNSPECIFIED || v == sidetxs.Vote_VOTE_YES || v == sidetxs.Vote_VOTE_NO
}
