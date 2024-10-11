package app

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abciTypes "github.com/cometbft/cometbft/abci/types"
	cryptoenc "github.com/cometbft/cometbft/crypto/encoding"
	"github.com/cometbft/cometbft/libs/protoio"
	cmtTypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/sidetxs"
	stakeKeeper "github.com/0xPolygon/heimdall-v2/x/stake/keeper"
)

// ValidateVoteExtensions verifies the vote extension correctness
// It checks the signature of each vote extension with its signer's public key
// Also, it checks if the vote extensions are enabled, valid and have >2/3 voting power
// It returns an error in case the validation fails
func ValidateVoteExtensions(ctx sdk.Context, reqHeight int64, proposerAddress []byte, extVoteInfo []abciTypes.ExtendedVoteInfo, round int32, stakeKeeper stakeKeeper.Keeper) error {

	// check if VEs are enabled
	panicOnVoteExtensionsDisabled(ctx, reqHeight+1)

	// check if reqHeight is the initial height
	if reqHeight == retrieveVoteExtensionsEnableHeight(ctx) {
		if len(extVoteInfo) != 0 {
			return fmt.Errorf("non-empty VEs received at initial height %d", reqHeight)
		}
		return nil
	}

	// Fetch validatorSet from previous block
	validatorSet, err := stakeKeeper.GetPreviousBlockValidatorSet(ctx)
	if err != nil {
		return err
	}
	if len(validatorSet.Validators) == 0 {
		return errors.New("no validators found in validator set")
	}

	var totalVotingPower = validatorSet.GetTotalVotingPower()
	sumVP := math.NewInt(0)

	// Map to track seen validator addresses
	seenValidators := make(map[string]struct{})

	ac := address.HexCodec{}
	proposerAdd, err := ac.BytesToString(proposerAddress)
	if err != nil {
		return err
	}

	for _, vote := range extVoteInfo {

		// make sure the BlockIdFlag is valid
		if !isBlockIdFlagValid(vote.BlockIdFlag) {
			return fmt.Errorf("received vote with invalid block ID %s flag at height %d", vote.BlockIdFlag.String(), reqHeight)
		}
		// if not BlockIDFlagCommit, skip that vote, as it doesn't have relevant information
		if vote.BlockIdFlag != cmtTypes.BlockIDFlagCommit {
			continue
		}

		if len(vote.ExtensionSignature) == 0 {
			return fmt.Errorf("received empty vote extension signature at height %d from validator %s", reqHeight, proposerAdd)
		}

		var consolidatedSideTxResponse sidetxs.ConsolidatedSideTxResponse
		if err = proto.Unmarshal(vote.VoteExtension, &consolidatedSideTxResponse); err != nil {
			return fmt.Errorf("error while unmarshalling vote extension: %w", err)
		}

		if consolidatedSideTxResponse.Height != reqHeight-1 {
			return fmt.Errorf("invalid height received for vote extension, expected %d, got %d", reqHeight-1, consolidatedSideTxResponse.Height)
		}

		txHash, err := validateSideTxResponses(consolidatedSideTxResponse.SideTxResponses)
		if err != nil {
			return fmt.Errorf("invalid sideTxResponses detected for validator %s and tx %s, error: %w", proposerAdd, common.Bytes2Hex(txHash), err)
		}

		// TODO HV2: See https://polygon.atlassian.net/browse/POS-2703
		valAddrStr := common.Bytes2Hex(vote.Validator.Address)

		// Check for duplicate votes by the same validator
		if _, found := seenValidators[valAddrStr]; found {
			return fmt.Errorf("duplicate vote detected from validator %s at height %d", valAddrStr, reqHeight)
		}
		// Add validator address to the map
		seenValidators[valAddrStr] = struct{}{}

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
			Height:    reqHeight - 1, // the vote extension was signed in the previous height
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
	majorityVP := totalVotingPower * 2 / 3
	if sumVP.Int64() <= majorityVP {
		return fmt.Errorf("insufficient cumulative voting power received to verify vote extensions; got: %d, expected: >=%d", sumVP.Int64(), majorityVP)
	}

	return nil
}

// tallyVotes tallies the votes received for the side tx
// It returns the lists of txs which got >2/3+ YES, NO and UNSPECIFIED votes respectively
func tallyVotes(extVoteInfo []abciTypes.ExtendedVoteInfo, logger log.Logger, totalVotingPower int64, currentHeight int64) ([][]byte, [][]byte, [][]byte, error) {
	logger.Debug("Tallying votes")

	voteByTxHash, err := aggregateVotes(extVoteInfo, currentHeight, logger)
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

	majorityVP := totalVotingPower * 2 / 3

	for _, txHash := range txHashList {
		voteMap := voteByTxHash[txHash]

		// calculate the total voting power in the voteMap
		power := voteMap[sidetxs.Vote_VOTE_YES] + voteMap[sidetxs.Vote_VOTE_NO] + voteMap[sidetxs.Vote_UNSPECIFIED]
		// ensure the total votes do not exceed the total voting power
		if power > totalVotingPower {
			logger.Error("the votes power exceeds the total voting power", "txHash", txHash, "power", power, "totalVotingPower", totalVotingPower)
			return nil, nil, nil, fmt.Errorf("votes power %d exceeds total voting power %d for txHash %s", power, totalVotingPower, txHash)
		}

		if voteMap[sidetxs.Vote_VOTE_YES] > majorityVP {
			// approved
			logger.Debug("Approved side-tx", "txHash", txHash)

			// append to approved tx slice
			approvedTxs = append(approvedTxs, common.Hex2Bytes(txHash))
		} else if voteMap[sidetxs.Vote_VOTE_NO] > majorityVP {
			// rejected
			logger.Debug("Rejected side-tx", "txHash", txHash)

			// append to rejected tx slice
			rejectedTxs = append(rejectedTxs, common.Hex2Bytes(txHash))
		} else {
			// skipped
			logger.Debug("Skipped side-tx", "txHash", txHash)

			// append to rejected tx slice
			skippedTxs = append(skippedTxs, common.Hex2Bytes(txHash))
		}
	}

	logger.Debug(fmt.Sprintf("Height %d: approved %d txs, rejected %d txs, skipped %d txs. ", currentHeight, len(approvedTxs), len(rejectedTxs), len(skippedTxs)))

	return approvedTxs, rejectedTxs, skippedTxs, nil
}

// aggregateVotes collates votes received for side txs
func aggregateVotes(extVoteInfo []abciTypes.ExtendedVoteInfo, currentHeight int64, logger log.Logger) (map[string]map[sidetxs.Vote]int64, error) {
	voteByTxHash := make(map[string]map[sidetxs.Vote]int64)  // track votes for a side tx
	validatorToTxMap := make(map[string]map[string]struct{}) // ensure a validator doesn't procure conflicting votes for a side tx
	var blockHash []byte                                     // store the block hash to make sure all votes are for the same block

	for _, vote := range extVoteInfo {

		var ve sidetxs.ConsolidatedSideTxResponse

		// make sure the BlockIdFlag is valid
		if !isBlockIdFlagValid(vote.BlockIdFlag) {
			return nil, fmt.Errorf("received vote with invalid block ID %s flag at height %d", vote.BlockIdFlag.String(), currentHeight-1)
		}
		// if not BlockIDFlagCommit, skip that vote, as it doesn't have relevant information
		if vote.BlockIdFlag != cmtTypes.BlockIDFlagCommit {
			continue
		}

		err := proto.Unmarshal(vote.VoteExtension, &ve)
		if err != nil {
			return nil, err
		}

		if ve.Height != currentHeight-1 {
			return nil, fmt.Errorf("invalid height received for vote extension, VeHeight should match CurrentHeight-1. VeHeight: %d, CurrentHeight: %d", ve.Height, currentHeight)
		}

		// blockHash consistency check
		if blockHash == nil {
			// store the block hash from the first vote
			blockHash = ve.BlockHash
		} else {
			ac := address.HexCodec{}
			valAddr, err := ac.BytesToString(vote.Validator.Address)
			if err != nil {
				return nil, err
			}
			// compare the current block hash with the stored block hash
			if !bytes.Equal(blockHash, ve.BlockHash) {
				logger.Error("invalid block hash found for vote extension",
					"expectedBlockHash", common.Bytes2Hex(blockHash),
					"receivedBlockHash", common.Bytes2Hex(ve.BlockHash),
					"validator", valAddr)
				return nil, fmt.Errorf("mismatching block hash for vote extension from validator %s", valAddr)
			}
		}

		addr, err := address.NewHexCodec().BytesToString(vote.Validator.Address)
		if err != nil {
			return nil, err
		}

		if validatorToTxMap[addr] != nil {
			return nil, fmt.Errorf("duplicate vote received from %s", addr)
		}
		validatorToTxMap[addr] = make(map[string]struct{})

		// iterate through vote extensions and accumulate voting power for YES/NO/UNSPECIFIED votes
		for _, res := range ve.SideTxResponses {
			txHashStr := common.Bytes2Hex(res.TxHash)

			// TODO HV2: (once slashing is enabled) we should slash in case a validator maliciously adds conflicting votes
			//  Given that we also check for duplicate votes during VerifyVoteExtensionHandler, is this redundant ?
			if _, hasVoted := validatorToTxMap[addr][txHashStr]; hasVoted {
				logger.Error("multiple votes received for side tx",
					"txHash", txHashStr, "validatorAddress", addr)
				return nil, fmt.Errorf("multiple votes received for side tx %s from validator %s", txHashStr, addr)
			}

			if !isVoteValid(res.Result) {
				return nil, fmt.Errorf("invalid vote %v received for side tx %s", res.Result, txHashStr)
			}

			if voteByTxHash[txHashStr] == nil {
				voteByTxHash[txHashStr] = make(map[sidetxs.Vote]int64)
			}

			voteByTxHash[txHashStr][res.Result] += vote.Validator.Power

			// validator's vote received; mark it to avoid duplicated votes
			validatorToTxMap[addr][txHashStr] = struct{}{}
		}

	}

	return voteByTxHash, nil
}

// validateSideTxResponses validates the SideTxResponses and returns the txHash of the first invalid tx detected, plus the error
func validateSideTxResponses(sideTxResponses []*sidetxs.SideTxResponse) ([]byte, error) {
	// track votes of the validator
	txVoteMap := make(map[string]struct{})

	for _, res := range sideTxResponses {
		// check txHash is well-formed
		if len(res.TxHash) != common.HashLength {
			return res.TxHash, errors.New(fmt.Sprintf("invalid tx hash received: %s", common.Bytes2Hex(res.TxHash)))
		}

		if !isVoteValid(res.Result) {
			return res.TxHash, errors.New(fmt.Sprintf("invalid vote result type %v received for side tx %s", res.Result, common.Bytes2Hex(res.TxHash)))
		}

		// check if the validator has already voted for the side tx
		if _, found := txVoteMap[string(res.TxHash)]; found {
			return res.TxHash, errors.New(fmt.Sprintf("duplicated votes detected for side tx %s", common.Bytes2Hex(res.TxHash)))
		}

		txVoteMap[string(res.TxHash)] = struct{}{}
	}

	return nil, nil
}

// panicOnVoteExtensionsDisabled indicates whether the proposer must include VEs from previous height in the block proposal as a special transaction.
// Since we are using a hard fork approach for the heimdall migration, VEs will be enabled from v2 genesis' initial height (v1 last height +1).
func panicOnVoteExtensionsDisabled(ctx sdk.Context, height int64) {
	// voteExtensionsEnableHeight is the height from which the vote extensions are enabled, and it's (v1_last_height +1)
	voteExtensionsEnableHeight := retrieveVoteExtensionsEnableHeight(ctx)
	if voteExtensionsEnableHeight == 0 {
		panic("VoteExtensions are disabled: VoteExtensionsEnableHeight is set to 0")
	}
	if height < voteExtensionsEnableHeight {
		panic(fmt.Sprintf("vote extensions are disabled: current height is %d, and VoteExtensionsEnableHeight is set to %d", height, voteExtensionsEnableHeight))
	}
}

func isVoteValid(v sidetxs.Vote) bool {
	return v == sidetxs.Vote_UNSPECIFIED || v == sidetxs.Vote_VOTE_YES || v == sidetxs.Vote_VOTE_NO
}

func isBlockIdFlagValid(flag cmtTypes.BlockIDFlag) bool {
	return flag == cmtTypes.BlockIDFlagAbsent || flag == cmtTypes.BlockIDFlagCommit || flag == cmtTypes.BlockIDFlagNil
}

// retrieveVoteExtensionsEnableHeight returns the height from which the vote extensions are enabled, which is equal to initial height of the v2 genesis
func retrieveVoteExtensionsEnableHeight(ctx sdk.Context) int64 {
	consensusParams := ctx.ConsensusParams()
	return consensusParams.GetAbci().GetVoteExtensionsEnableHeight()
}
