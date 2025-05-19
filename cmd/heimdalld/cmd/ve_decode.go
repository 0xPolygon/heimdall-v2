package heimdalld

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"

	cometbftDB "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/store"
	"github.com/cosmos/cosmos-sdk/client/flags"
	goproto "github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	app "github.com/0xPolygon/heimdall-v2/app"
	"github.com/0xPolygon/heimdall-v2/sidetxs"
	checkpointTypes "github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
)

// veDecodeCmd returns the ve-decode command.
func veDecodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ve-decode",
		Short: "Decode VEs for a specific block height",
		Long:  `This command decodes the vote extensions of a specific block height provided by the user.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runVeDecode,
	}

	cmd.Flags().String("chain-id", "", "Heimdall-v2 network chain id")
	cmd.Flags().String("host", "localhost", "RPC host")
	cmd.Flags().Uint64("cometbft-port", 26657, "Cometbft RPC endpoint")

	if err := cmd.MarkFlagRequired("chain-id"); err != nil {
		panic(err)
	}

	return cmd
}

func runVeDecode(cmd *cobra.Command, args []string) error {
	height, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid height: %w", err)
	}
	if height < 1 {
		return fmt.Errorf("block height number must be greater than VoteExtEnableHeight (1)")
	}

	chainID, err := cmd.Flags().GetString("chain-id")
	if err != nil {
		return err
	}
	if chainID == "" {
		return fmt.Errorf("non-empty chain ID is required")
	}

	host, err := cmd.Flags().GetString("host")
	if err != nil {
		return fmt.Errorf("error parsing host flag: %w", err)
	}

	port, err := cmd.Flags().GetUint64("cometbft-port")
	if err != nil {
		return fmt.Errorf("error parsing cometbft-port flag: %w", err)
	}

	// Fetch vote extension info
	extInfo, err := getVEs(height, host, port)
	if err != nil {
		return fmt.Errorf("error getting vote extensions: %w", err)
	}

	// Encode to JSON and print
	out, err := BuildCommitJSON(height, extInfo)
	if err != nil {
		return fmt.Errorf("error marshalling to JSON: %w", err)
	}
	fmt.Println("Vote Extension:")
	fmt.Println(string(out))
	fmt.Println()

	// Print summary
	summary, err := BuildSummaryJSON(height, extInfo)
	if err != nil {
		return fmt.Errorf("error marshalling summary to JSON: %w", err)
	}
	fmt.Println("Summary:")
	fmt.Println(string(summary))

	return nil
}

func getVEs(height int64, host string, endpoint uint64) (*abci.ExtendedCommitInfo, error) {
	// 1) Try the RPC endpoint first.
	voteExt, err1 := GetVEsFromEndpoint(height, host, endpoint)
	if err1 == nil {
		return voteExt, nil
	}

	// 2) Fallback to the local block store.
	voteExt, err2 := GetVEsFromBlockStore(height)
	if err2 == nil {
		return voteExt, nil
	}

	// 3) Both failed, report both the errors.
	return nil, fmt.Errorf("failed to fetch VEs:\n- endpoint (port %d): %w\n- block store: %w", endpoint, err1, err2)
}

func GetVEsFromEndpoint(height int64, host string, endpoint uint64) (*abci.ExtendedCommitInfo, error) {
	if endpoint < 1 || endpoint > 65535 {
		return nil, fmt.Errorf("invalid RPC port: %d", endpoint)
	}
	url := fmt.Sprintf("http://%s:%d/block?height=%d", host, endpoint, height)

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch block: %s", resp.Status)
	}

	type BlockResponse struct {
		Result struct {
			Block struct {
				Data struct {
					Txs []string `json:"txs"`
				} `json:"data"`
			} `json:"block"`
		} `json:"result"`
	}

	var br BlockResponse

	if err := json.NewDecoder(resp.Body).Decode(&br); err != nil {
		return nil, err
	}

	if len(br.Result.Block.Data.Txs) == 0 {
		return nil, fmt.Errorf("no vote extensions found in the block")
	}

	veB64Str := br.Result.Block.Data.Txs[0]
	veBytes, err := base64.StdEncoding.DecodeString(veB64Str)
	if err != nil {
		return nil, err
	}

	var voteExt abci.ExtendedCommitInfo
	if err := goproto.Unmarshal(veBytes, &voteExt); err != nil {
		return nil, err
	}
	return &voteExt, nil
}

func GetVEsFromBlockStore(height int64) (*abci.ExtendedCommitInfo, error) {
	homeDir := viper.GetString(flags.FlagHome)
	if homeDir == "" {
		return nil, fmt.Errorf("home directory not set")
	}

	cometbftdb, err := cometbftDB.NewGoLevelDB("blockstore", path.Join(homeDir, "data"))
	if err != nil {
		return nil, err
	}
	blockStore := store.NewBlockStore(cometbftdb)
	block := blockStore.LoadBlock(height)
	if block == nil {
		return nil, fmt.Errorf("block at height %d not found", height)
	}

	ves := block.Data.Txs[0] //nolint:staticcheck
	if ves == nil {
		return nil, fmt.Errorf("no vote extensions found in the block")
	}

	var voteExt abci.ExtendedCommitInfo
	if err := voteExt.Unmarshal(ves); err != nil {
		return nil, err
	}

	return &voteExt, nil
}

// BuildCommitJSON builds a JSON representation for the given ExtendedCommitInfo and block height.
func BuildCommitJSON(height int64, ext *abci.ExtendedCommitInfo) ([]byte, error) {
	data := CommitData{
		Height: height,
		Round:  ext.Round,
		Votes:  make([]VoteData, len(ext.Votes)),
	}

	for i, v := range ext.Votes {
		// Unmarshal sideTx extension
		var stxs sidetxs.VoteExtension
		if err := goproto.Unmarshal(v.VoteExtension, &stxs); err != nil {
			return nil, err
		}

		vote := VoteData{
			ValidatorAddr: "0x" + hex.EncodeToString(v.Validator.Address),
			Power:         v.Validator.Power,
			ExtSignature:  "0x" + hex.EncodeToString(v.ExtensionSignature),
			BlockIDFlag:   v.BlockIdFlag.String(),
		}

		// SideTxResponses
		for _, r := range stxs.SideTxResponses {
			vote.SideTxs = append(vote.SideTxs, SideTxData{
				TxHash: "0x" + hex.EncodeToString(r.TxHash),
				Result: r.Result.String(),
			})
		}

		// Milestone
		if mp := stxs.MilestoneProposition; mp != nil {
			hashes := make([]string, len(mp.BlockHashes))
			for j, bh := range mp.BlockHashes {
				hashes[j] = "0x" + hex.EncodeToString(bh)
			}
			vote.Milestone = &MilestoneData{
				BlockHashes:      hashes,
				StartBlockNumber: mp.StartBlockNumber,
				ParentHash:       "0x" + hex.EncodeToString(mp.ParentHash),
			}
		}

		// Non-RP vote extension: dummy vs checkpoint
		if isDummy, _ := IsDummyNonRpVoteExtension(height, v.NonRpVoteExtension); isDummy {
			vote.NonRpData = "0x" + hex.EncodeToString(v.NonRpVoteExtension)
		} else {
			msg, err := GetCheckpointMsg(v.NonRpVoteExtension)
			if err != nil {
				return nil, fmt.Errorf("error unpacking checkpoint: %w", err)
			}
			vote.NonRpData = CheckpointData{
				Proposer:        msg.Proposer,
				StartBlock:      msg.StartBlock,
				EndBlock:        msg.EndBlock,
				RootHash:        "0x" + hex.EncodeToString(msg.RootHash),
				AccountRootHash: "0x" + hex.EncodeToString(msg.AccountRootHash),
				BorChainID:      msg.BorChainId,
			}
		}

		vote.NonRpSignature = "0x" + hex.EncodeToString(v.NonRpExtensionSignature)
		data.Votes[i] = vote
	}

	return json.MarshalIndent(data, "", "  ")
}

// BuildSummaryJSON builds a JSON summary from ExtendedCommitInfo.
func BuildSummaryJSON(height int64, ext *abci.ExtendedCommitInfo) ([]byte, error) {
	var totalPower int64
	for _, v := range ext.Votes {
		totalPower += v.Validator.Power
	}

	format := func(vp int64) string {
		pct := float64(vp) / float64(totalPower) * 100
		return fmt.Sprintf("%d (%.2f%%)", vp, pct)
	}

	milestoneVP := make(map[string]int64)
	sideTxVP := make(map[string]map[string]int64)
	nonRpVP := make(map[string]int64)

	for _, v := range ext.Votes {
		power := v.Validator.Power

		var stxs sidetxs.VoteExtension
		if err := goproto.Unmarshal(v.VoteExtension, &stxs); err != nil {
			return nil, err
		}
		if mp := stxs.MilestoneProposition; mp != nil {
			for _, h := range mp.BlockHashes {
				milestoneVP["0x"+hex.EncodeToString(h)] += power
			}
		}
		for _, r := range stxs.SideTxResponses {
			txKey := "0x" + hex.EncodeToString(r.TxHash)
			if sideTxVP[txKey] == nil {
				sideTxVP[txKey] = make(map[string]int64)
			}
			sideTxVP[txKey][r.Result.String()] += power
		}

		var key string
		isDummy, err := IsDummyNonRpVoteExtension(height, v.NonRpVoteExtension)
		if err != nil {
			return nil, fmt.Errorf("error checking dummy non-RP extension: %w", err)
		}
		if isDummy {
			key = "0x" + hex.EncodeToString(v.NonRpVoteExtension)
		} else {
			msg, err := GetCheckpointMsg(v.NonRpVoteExtension)
			if err != nil {
				return nil, fmt.Errorf("error unpacking checkpoint message: %w", err)
			}
			checkpointData := CheckpointData{
				Proposer:        msg.Proposer,
				StartBlock:      msg.StartBlock,
				EndBlock:        msg.EndBlock,
				RootHash:        "0x" + hex.EncodeToString(msg.RootHash),
				AccountRootHash: "0x" + hex.EncodeToString(msg.AccountRootHash),
				BorChainID:      msg.BorChainId,
			}
			b, err := json.Marshal(checkpointData)
			if err != nil {
				return nil, err
			}
			key = string(b)
		}
		nonRpVP[key] += power
	}

	summary := SummaryData{
		Milestone: make(map[string]string),
		SideTx:    make(map[string]map[string]string),
		NonRp:     make(map[string]string),
	}

	for h, vp := range milestoneVP {
		summary.Milestone[h] = format(vp)
	}
	for tx, results := range sideTxVP {
		summary.SideTx[tx] = make(map[string]string)
		for res, vp := range results {
			summary.SideTx[tx][res] = format(vp)
		}
	}
	for extKey, vp := range nonRpVP {
		summary.NonRp[extKey] = format(vp)
	}

	return json.MarshalIndent(summary, "", "  ")
}

func IsDummyNonRpVoteExtension(height int64, nonRpVoteExt []byte) (bool, error) {
	chainID := viper.GetString(flags.FlagChainID)
	if chainID == "" {
		return false, fmt.Errorf("chain ID not set")
	}
	dummyVoteExt, err := app.GetDummyNonRpVoteExtension(height-1, chainID)
	if err != nil {
		return false, err
	}
	return bytes.Equal(nonRpVoteExt, dummyVoteExt), nil
}

func GetCheckpointMsg(nonRpVoteExt []byte) (*checkpointTypes.MsgCheckpoint, error) {
	// Skip leading marker byte
	checkpointMsg, err := checkpointTypes.UnpackCheckpointSideSignBytes(nonRpVoteExt[1:])
	if err != nil {
		return nil, err
	}

	return checkpointMsg, nil
}

// Helper structs for JSON encoding and decoding.

// CommitData represents the JSON output for an extended commit.
type CommitData struct {
	Height int64      `json:"height"`
	Round  int32      `json:"round"`
	Votes  []VoteData `json:"votes"`
}

// VoteData contains per-validator vote extension details.
type VoteData struct {
	ValidatorAddr  string         `json:"validator_address"`
	Power          int64          `json:"power"`
	SideTxs        []SideTxData   `json:"side_tx_responses"`
	Milestone      *MilestoneData `json:"milestone_proposition,omitempty"`
	ExtSignature   string         `json:"extension_signature"`
	BlockIDFlag    string         `json:"block_id_flag"`
	NonRpData      any            `json:"non_rp_vote_extension"`
	NonRpSignature string         `json:"non_rp_extension_signature"`
}

// SideTxData describes a single side transaction response.
type SideTxData struct {
	TxHash string `json:"tx_hash"`
	Result string `json:"result"`
}

// MilestoneData represents proposed milestone information.
type MilestoneData struct {
	BlockHashes      []string `json:"block_hashes"`
	StartBlockNumber uint64   `json:"start_block_number"`
	ParentHash       string   `json:"parent_hash"`
}

// CheckpointData holds the decoded checkpoint data from non-RP vote extension details.
type CheckpointData struct {
	Proposer        string `json:"proposer"`
	StartBlock      uint64 `json:"start_block"`
	EndBlock        uint64 `json:"end_block"`
	RootHash        string `json:"root_hash"`
	AccountRootHash string `json:"account_root_hash"`
	BorChainID      string `json:"bor_chain_id"`
}

// SummaryData is the JSON shape for the summary.
type SummaryData struct {
	Milestone map[string]string            `json:"milestone_voting_power"`
	SideTx    map[string]map[string]string `json:"side_tx_voting_power"`
	NonRp     map[string]string            `json:"non_rp_voting_power"`
}
