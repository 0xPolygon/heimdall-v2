package cli

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/0xPolygon/heimdall-v2/common/cli"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

var logger = helper.Logger.With("module", "bor/client/cli")

// NewTxCmd returns a root CLI command handler for all x/bor transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Bor transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewSpanProposalCmd(),
		NewProducerDowntimeCmd(),
	)

	return txCmd
}

// NewSpanProposalCmd returns a CLI command handler for creating a MsgSpanProposal transaction.
func NewSpanProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-span",
		Short: "send propose span tx",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			borChainID := viper.GetString(FlagBorChainId)
			if borChainID == "" {
				return fmt.Errorf("BorChainID cannot be empty")
			}

			// get proposer
			proposer := viper.GetString(FlagProposerAddress)
			if proposer == "" {
				proposer = clientCtx.GetFromAddress().String()
			}

			addressCodec := addresscodec.NewHexCodec()
			_, err = addressCodec.StringToBytes(proposer)
			if err != nil {
				return fmt.Errorf("proposer address is invalid: %w", err)
			}

			// get start block
			startBlockStr := viper.GetString(FlagStartBlock)
			if startBlockStr == "" {
				return fmt.Errorf("start block cannot be empty")
			}

			startBlock, err := strconv.ParseUint(startBlockStr, 10, 64)
			if err != nil {
				return err
			}

			// get span id
			spanIDStr := viper.GetString(FlagSpanId)
			if spanIDStr == "" {
				return fmt.Errorf("span id cannot be empty")
			}

			spanID, err := strconv.ParseUint(spanIDStr, 10, 64)
			if err != nil {
				return err
			}

			// fetch params
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.GetBorParams(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}
			spanDuration := res.Params.SpanDuration

			// fetch the next span seed
			nextSpanSeedResponse, err := queryClient.GetNextSpanSeed(cmd.Context(), &types.QueryNextSpanSeedRequest{
				Id: spanID,
			})
			if err != nil {
				return err
			}
			seed := common.HexToHash(nextSpanSeedResponse.Seed)
			msg := types.NewMsgProposeSpan(spanID, proposer, startBlock, startBlock+spanDuration-1, borChainID, seed.Bytes(), nextSpanSeedResponse.SeedAuthor)

			return cli.BroadcastMsg(clientCtx, proposer, msg, logger)
		},
	}

	cmd.Flags().StringP(FlagProposerAddress, "p", "", "--proposer=<proposer-address>")
	cmd.Flags().String(FlagSpanId, "", "--span-id=<span-id>")
	cmd.Flags().String(FlagBorChainId, "", "--bor-chain-id=<bor-chain-id>")
	cmd.Flags().String(FlagStartBlock, "", "--start-block=<start-block-number>")
	cmd.Flags().String(flags.FlagChainID, "", "--chain-id=<chain-id>")

	if err := cmd.MarkFlagRequired(FlagBorChainId); err != nil {
		fmt.Printf("PostSendProposeSpanTx | MarkFlagRequired | FlagBorChainId Error: %v", err)
	}

	if err := cmd.MarkFlagRequired(FlagStartBlock); err != nil {
		fmt.Printf("PostSendProposeSpanTx | MarkFlagRequired | FlagStartBlock Error: %v", err)
	}

	return cmd
}

func NewProducerDowntimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "producer-downtime",
		Short: "Set producer downtime",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			producerAddress := viper.GetString(FlagProducerAddress)
			if producerAddress == "" {
				producerAddress = clientCtx.GetFromAddress().String()
			}

			addressCodec := addresscodec.NewHexCodec()
			_, err = addressCodec.StringToBytes(producerAddress)
			if err != nil {
				return fmt.Errorf("producer address is invalid: %w", err)
			}

			startTimeUTC := viper.GetInt(FlagStartTimestampUTC)
			if startTimeUTC <= 0 {
				return fmt.Errorf("start timestamp utc is invalid")
			}

			endTimeUTC := viper.GetInt(FlagEndTimestampUTC)
			if endTimeUTC <= 0 {
				return fmt.Errorf("end timestamp utc is invalid")
			}

			node, err := clientCtx.GetNode()
			if err != nil {
				return err
			}

			status, err := node.Status(clientCtx.CmdContext)
			if err != nil {
				return fmt.Errorf("failed to get node status: %w", err)
			}

			if startTimeUTC < int(status.SyncInfo.LatestBlockTime.Unix()) {
				return fmt.Errorf("start timestamp utc cannot be in the past")
			}

			if endTimeUTC <= startTimeUTC {
				return fmt.Errorf("end timestamp utc must be greater than start timestamp utc")
			}

			averageBlockTime, err := calculateAverageBlocktime(clientCtx, node)
			if err != nil {
				return fmt.Errorf("failed to calculate average block time: %w", err)
			}

			borClient := helper.GetBorClient()
			currentBlock, err := borClient.BlockNumber(clientCtx.CmdContext)
			if err != nil {
				return fmt.Errorf("failed to get latest bor block number: %w", err)
			}

			block, err := borClient.BlockByNumber(clientCtx.CmdContext, big.NewInt(int64(currentBlock)))
			if err != nil {
				return fmt.Errorf("failed to get latest bor block: %w", err)
			}

			startBlock := currentBlock + uint64((time.Unix(int64(startTimeUTC), 0).Sub(*block.AnnouncedAt).Seconds())/averageBlockTime)
			endBlock := currentBlock + uint64((time.Unix(int64(endTimeUTC), 0).Sub(*block.AnnouncedAt).Seconds())/averageBlockTime)

			msg := types.NewMsgSetProducerDowntime(producerAddress, startBlock, endBlock)

			return cli.BroadcastMsg(clientCtx, producerAddress, msg, logger)
		},
	}

	cmd.Flags().String(FlagProducerAddress, "", "--producer-address=<producer-address>")
	cmd.Flags().String(FlagStartTimestampUTC, "", "--start-timestamp-utc=<start-timestamp-utc>")
	cmd.Flags().String(FlagEndTimestampUTC, "", "--end-timestamp-utc=<end-timestamp-utc>")
	cmd.Flags().String(flags.FlagChainID, "", "--chain-id=<chain-id>")

	if err := cmd.MarkFlagRequired(FlagProducerAddress); err != nil {
		fmt.Printf("NewProducerDowntimeCmd | MarkFlagRequired | FlagProducerAddress Error: %v", err)
	}

	if err := cmd.MarkFlagRequired(FlagStartTimestampUTC); err != nil {
		fmt.Printf("NewProducerDowntimeCmd | MarkFlagRequired | FlagStartTimestampUTC Error: %v", err)
	}

	if err := cmd.MarkFlagRequired(FlagEndTimestampUTC); err != nil {
		fmt.Printf("NewProducerDowntimeCmd | MarkFlagRequired | FlagEndTimestampUTC Error: %v", err)
	}

	if err := cmd.MarkFlagRequired(flags.FlagChainID); err != nil {
		fmt.Printf("NewProducerDowntimeCmd | MarkFlagRequired | FlagChainID Error: %v", err)
	}

	return cmd
}

func calculateAverageBlocktime(clientCtx client.Context, node client.CometRPC) (float64, error) {
	borClient := helper.GetBorClient()
	currentBlock, err := borClient.BlockNumber(clientCtx.CmdContext)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest bor block number: %w", err)
	}

	blockTimesToGet := int64(100)
	blocksInBetween := int64(100)
	var averageBlockTime float64

	for i := int64(0); i < blockTimesToGet; i++ {
		blockNumber := big.NewInt(int64(currentBlock) - (blocksInBetween * i))
		block, err := borClient.BlockByNumber(clientCtx.CmdContext, blockNumber)
		if err != nil {
			return 0, err
		}

		prevBlockHeight := int64(currentBlock) - ((blocksInBetween * i) + 1)
		prevBlock, err := node.Block(clientCtx.CmdContext, &prevBlockHeight)
		if err != nil {
			return 0, err
		}

		blockTime := block.AnnouncedAt.Sub(prevBlock.Block.Time).Seconds()
		averageBlockTime += blockTime
	}

	averageBlockTime = averageBlockTime / float64(blockTimesToGet)

	return averageBlockTime, nil
}
