package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"cosmossdk.io/core/address"

	"github.com/0xPolygon/heimdall-v2/helper"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/milestone/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
)

var logger = helper.Logger.With("module", "x/milestone")

// NewTxCmd returns a root CLI command handler for all x/milestone transaction commands.
func NewTxCmd(valAddrCodec, ac address.Codec) *cobra.Command {
	milestoneTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "milestone transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	milestoneTxCmd.AddCommand(
		MilestoneCmd(valAddrCodec),
		MilestoneTimeoutCmd(),
	)

	return milestoneTxCmd
}

// MilestoneCmd returns a CLI command handler for creating a MsgMilestone transaction.
func MilestoneCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-milestone",
		Short: "propose a milestone on heimdall",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			borChainID, err := cmd.Flags().GetString(FlagBorChainID)
			if err != nil {
				return err
			}

			if borChainID == "" {
				return fmt.Errorf("bor chain id cannot be empty")
			}

			milestoneID, err := cmd.Flags().GetString(FlagMilestoneID)
			if err != nil {
				return err
			}

			if milestoneID == "" {
				return fmt.Errorf("milestone ID cannot be empty")
			}

			proposer, err := cmd.Flags().GetString(FlagProposerAddress)
			if err != nil {
				return err
			}

			_, err = ac.StringToBytes(proposer)
			if err != nil {
				return err
			}

			startBlock, err := cmd.Flags().GetUint64(FlagStartBlock)
			if err != nil {
				return err
			}

			endBlock, err := cmd.Flags().GetUint64(FlagEndBlock)
			if err != nil {
				return err
			}

			hashStr, err := cmd.Flags().GetString(FlagHash)
			if err != nil {
				return err
			}

			msg := types.NewMsgMilestoneBlock(
				proposer,
				startBlock,
				endBlock,
				hmTypes.HexToHeimdallHash(hashStr),
				borChainID,
				milestoneID,
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	cmd.Flags().StringP(FlagProposerAddress, "p", "", "--proposer=<proposer-address>")
	cmd.Flags().Uint64(FlagStartBlock, 0, "--start-block=<start-block-number>")
	cmd.Flags().Uint64(FlagEndBlock, 0, "--end-block=<end-block-number>")
	cmd.Flags().StringP(FlagHash, "r", "", "--root-hash=<root-hash>")
	cmd.Flags().String(FlagBorChainID, "", "--bor-chain-id=<bor-chain-id>")
	cmd.Flags().String(FlagMilestoneID, "", "--milestone-id=<milestone-id>")

	if err := cmd.MarkFlagRequired(FlagProposerAddress); err != nil {
		logger.Error("SendMilestoneTx | MarkFlagRequired | FlagProposerAddress", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagStartBlock); err != nil {
		logger.Error("SendMilestoneTx | MarkFlagRequired | FlagStartBlock", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagEndBlock); err != nil {
		logger.Error("SendMilestoneTx | MarkFlagRequired | FlagEndBlock", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagHash); err != nil {
		logger.Error("SendMilestoneTx | MarkFlagRequired | FlagHash", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagBorChainID); err != nil {
		logger.Error("SendMilestoneTx | MarkFlagRequired | FlagBorChainID", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagMilestoneID); err != nil {
		logger.Error("SendMilestoneTx | MarkFlagRequired | FlagMilestoneID", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagBorChainID); err != nil {
		logger.Error("SendMilestoneTx | MarkFlagRequired | FlagBorChainID", "Error", err)
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// MilestoneTimeoutCmd returns a CLI command handler for creating a MsgMilestoneTimeout
func MilestoneTimeoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-milestone-timeout",
		Short: "send milestone-timeout",

		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress().String()

			msg := types.NewMsgMilestoneTimeout(from)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
