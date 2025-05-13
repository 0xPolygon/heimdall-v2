package heimdalld

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"

	db "github.com/cometbft/cometbft-db"
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
	cmd.Flags().Uint64P("endpoint", "e", 26657, "Cometbft RPC endpoint")

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

	chainId, err := cmd.Flags().GetString("chain-id")
	if err != nil {
		return err
	}
	if chainId == "" {
		return fmt.Errorf("non-empty chain ID is required")
	}

	host, err := cmd.Flags().GetString("host")
	if err != nil {
		return fmt.Errorf("error parsing host flag: %w", err)
	}

	endpoint, err := cmd.Flags().GetUint64("endpoint")
	if err != nil {
		return fmt.Errorf("error parsing endpoint flag: %w", err)
	}

	// Get VoteExtension from the block at the specified height.
	voteExts, err := getVEs(height, host, endpoint)
	if err != nil {
		return fmt.Errorf("error getting VEs: %w", err)
	}
	if voteExts == nil {
		return fmt.Errorf("no VEs found for block height %d", height)
	}

	// Decode and print the extended commit info.
	if err := decodeAndPrintExtendedCommitInfo(height, voteExts); err != nil {
		return fmt.Errorf("error decoding and printing extended commit info: %w", err)
	}
	return nil
}

func getVEs(height int64, host string, endpoint uint64) (*abci.ExtendedCommitInfo, error) {
	// 1) Try the RPC endpoint first.
	voteExt, err1 := getVEsFromEndpoint(height, host, endpoint)
	if err1 == nil {
		return voteExt, nil
	}
	fmt.Printf("Error fetching VEs from endpoint %d: %v\n", endpoint, err1)

	// 2) Fallback to the local block store.
	voteExt, err2 := getVEsFromBlockStore(height)
	if err2 == nil {
		return voteExt, nil
	}

	// 3) Both failed, report both the errors.
	return nil, fmt.Errorf("failed to fetch VEs:\n- endpoint (port %d): %w\n- block store: %w", endpoint, err1, err2)
}

func getVEsFromEndpoint(height int64, host string, endpoint uint64) (*abci.ExtendedCommitInfo, error) {
	if endpoint < 1 || endpoint > 65535 {
		return nil, fmt.Errorf("invalid RPC port: %d", endpoint)
	}
	u := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, strconv.FormatUint(endpoint, 10)),
		Path:   "/block",
	}
	q := u.Query()
	q.Set("height", strconv.FormatInt(height, 10))
	u.RawQuery = q.Encode()
	resp, err := http.Get(u.String())
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

func getVEsFromBlockStore(height int64) (*abci.ExtendedCommitInfo, error) {
	homeDir := viper.GetString(flags.FlagHome)
	if homeDir == "" {
		return nil, fmt.Errorf("home directory not set")
	}

	db, err := db.NewGoLevelDB("blockstore", path.Join(homeDir, "data"))
	if err != nil {
		return nil, err
	}
	blockStore := store.NewBlockStore(db)
	block := blockStore.LoadBlock(height)
	if block == nil {
		return nil, fmt.Errorf("block at height %d not found", height)
	}

	ves := block.Data.Txs[0]
	if ves == nil {
		return nil, fmt.Errorf("no vote extensions found in the block")
	}

	var voteExt abci.ExtendedCommitInfo
	if err := voteExt.Unmarshal(ves); err != nil {
		return nil, err
	}

	return &voteExt, nil
}

func decodeAndPrintExtendedCommitInfo(height int64, info *abci.ExtendedCommitInfo) error {
	voteExts := make([]sidetxs.VoteExtension, len(info.Votes))
	for i, v := range info.Votes {
		if err := goproto.Unmarshal(v.VoteExtension, &voteExts[i]); err != nil {
			return err
		}
	}

	printHeader(height, info.Round)

	for i, v := range info.Votes {
		err := printVote(height, i+1, v, voteExts[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func printHeader(height int64, round int32) {
	fmt.Println()
	fmt.Println("================ Extended Commit Info ================")
	fmt.Printf("Height: %d\n", height)
	fmt.Printf("Round: %d\n", round)
	fmt.Println("======================================================")
	fmt.Println()
}

func printVote(height int64, index int, voteInfo abci.ExtendedVoteInfo, voteExt sidetxs.VoteExtension) error {
	fmt.Printf("Vote %d:\n", index)
	fmt.Println("------------------------------------------------------")

	printValidatorInfo(voteInfo)
	printVoteExtensionInfo(voteExt)
	err := printNonRpVoteExtAndSignatures(height, voteInfo)
	if err != nil {
		return err
	}

	fmt.Println("------------------------------------------------------")
	fmt.Println()

	return nil
}

func printValidatorInfo(v abci.ExtendedVoteInfo) {
	fmt.Printf("Validator: %s\n", hex.EncodeToString(v.Validator.Address))
	fmt.Printf("Power: %d\n", v.Validator.Power)
	fmt.Printf("BlockIdFlag: %s\n", v.BlockIdFlag.String())
}

func printVoteExtensionInfo(voteExt sidetxs.VoteExtension) {
	fmt.Println("VoteExtension:")
	fmt.Printf("  BlockHash: %s\n", hex.EncodeToString(voteExt.BlockHash))
	fmt.Printf("  Height: %d\n", voteExt.Height)

	if len(voteExt.SideTxResponses) == 0 {
		fmt.Println("  SideTxResponses: []")
	} else {
		fmt.Println("  SideTxResponses:")
		for j, resp := range voteExt.SideTxResponses {
			fmt.Printf("    Response %d:\n", j+1)
			fmt.Printf("      TxHash: %s\n", hex.EncodeToString(resp.TxHash))
			fmt.Printf("      Result: %s\n", resp.Result.String())
		}
	}

	if voteExt.MilestoneProposition != nil {
		mp := voteExt.MilestoneProposition
		fmt.Println("  MilestoneProposition:")
		for k, bh := range mp.BlockHashes {
			fmt.Printf("    BlockHash[%d]: %s\n", k, hex.EncodeToString(bh))
		}
		fmt.Printf("    StartBlockNumber: %d\n", mp.StartBlockNumber)
		fmt.Printf("    ParentHash: %s\n", hex.EncodeToString(mp.ParentHash))
	} else {
		fmt.Println("  MilestoneProposition: nil")
	}
}

func printNonRpVoteExtAndSignatures(height int64, v abci.ExtendedVoteInfo) error {
	fmt.Printf("ExtensionSignature: %s\n", hex.EncodeToString(v.ExtensionSignature))
	err := printNonRpVoteExtension(height, v.NonRpVoteExtension)
	if err != nil {
		return err
	}
	fmt.Printf("NonRpExtensionSignature: %s\n", hex.EncodeToString(v.NonRpExtensionSignature))

	return nil
}

func printNonRpVoteExtension(height int64, nonRpVoteExt []byte) error {
	dummy, err := isDummyNonRpVoteExtension(height, nonRpVoteExt)
	if err != nil {
		return err
	}
	if dummy {
		fmt.Printf("NonRpVoteExtension [DUMMY #HEIMDALL-VOTE-EXTENSION#]: %s\n", hex.EncodeToString(nonRpVoteExt))
	} else {
		msg, err := getCheckpointMsg(nonRpVoteExt)
		if err != nil {
			return err
		}
		fmt.Println("NonRpVoteExtension [CHECKPOINT MSG]:")
		fmt.Printf("  Proposer: %s\n", msg.Proposer)
		fmt.Printf("  StartBlock: %d\n", msg.StartBlock)
		fmt.Printf("  EndBlock: %d\n", msg.EndBlock)
		fmt.Printf("  RootHash: %s\n", hex.EncodeToString(msg.RootHash))
		fmt.Printf("  AccountRootHash: %s\n", hex.EncodeToString(msg.AccountRootHash))
		fmt.Printf("  BorChainId: %s\n", msg.BorChainId)
	}

	return nil
}

func isDummyNonRpVoteExtension(height int64, nonRpVoteExt []byte) (bool, error) {
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

func getCheckpointMsg(nonRpVoteExt []byte) (*checkpointTypes.MsgCheckpoint, error) {
	// Skip leading marker byte
	checkpointMsg, err := checkpointTypes.UnpackCheckpointSideSignBytes(nonRpVoteExt[1:])
	if err != nil {
		return nil, err
	}

	return checkpointMsg, nil
}
