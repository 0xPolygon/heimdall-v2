package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/0xPolygon/heimdall-v2/common/cli"
	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
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
		NewBackfillSpans(),
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

// NewBackfillSpans returns a CLI command handler for creating a MsgBackfillSpans transaction.
func NewBackfillSpans() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backfill-spans",
		Short: "send backfill spans tx",
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

			// get latest span id
			queryClient := types.NewQueryClient(clientCtx)
			latestSpanResp, err := queryClient.GetLatestSpan(cmd.Context(), &types.QueryLatestSpanRequest{})
			if err != nil {
				return fmt.Errorf("failed to get latest span: %w", err)
			}

			if latestSpanResp == nil {
				return fmt.Errorf("no latest span found")
			}

			// contractCaller, err := helper.NewContractCaller()
			// if err != nil {
			// 	return err
			// }

			// borLastUsedSpanID, err := contractCaller.GetStartBlockHeimdallSpanID(clientCtx.CmdContext, latestSpanResp.Span.EndBlock+1)
			// if err != nil {
			// 	return fmt.Errorf("failed to get last used heimdall span id: %w", err)
			// }

			// if borLastUsedSpanID == 0 {
			// 	return fmt.Errorf("heimdall span id is 0, no backfill needed")
			// }

			borLastUsedSpanID := latestSpanResp.Span.Id

			borLastUsedSpan, err := queryClient.GetSpanById(cmd.Context(), &types.QuerySpanByIdRequest{
				Id: strconv.FormatUint(borLastUsedSpanID, 10),
			})
			if err != nil {
				return fmt.Errorf("failed to get last used heimdall span: %w", err)
			}

			if borLastUsedSpan == nil {
				return fmt.Errorf("no last used heimdall span found for id: %d", borLastUsedSpanID)
			}

			// calculate latest bor span id
			milestoneQueryClient := milestoneTypes.NewQueryClient(clientCtx)

			latestMilestoneResp, err := milestoneQueryClient.GetLatestMilestone(cmd.Context(), &milestoneTypes.QueryLatestMilestoneRequest{})
			if err != nil {
				return fmt.Errorf("failed to get latest milestone: %w", err)
			}
			if latestMilestoneResp == nil {
				return fmt.Errorf("no latest milestone found")
			}

			borSpanId, err := types.CalcCurrentBorSpanId(latestMilestoneResp.Milestone.EndBlock, borLastUsedSpan.Span)
			if err != nil {
				return fmt.Errorf("failed to calculate bor span id: %w", err)
			}

			msg := types.NewMsgBackfillSpans(proposer, borChainID, borLastUsedSpanID, borSpanId)

			return cli.BroadcastMsg(clientCtx, proposer, msg, logger)
		},
	}

	cmd.Flags().StringP(FlagProposerAddress, "p", "", "--proposer=<proposer-address>")
	cmd.Flags().String(FlagBorChainId, "", "--bor-chain-id=<bor-chain-id>")

	if err := cmd.MarkFlagRequired(FlagBorChainId); err != nil {
		fmt.Printf("PostSendProposeSpanTx | MarkFlagRequired | FlagBorChainId Error: %v", err)
	}

	return cmd
}
