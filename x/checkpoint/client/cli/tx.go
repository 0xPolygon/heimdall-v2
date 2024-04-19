package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"cosmossdk.io/core/address"

	"github.com/0xPolygon/heimdall-v2/helper"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
)

var logger = helper.Logger.With("module", "checkpoint/client/cli")

// NewTxCmd returns a root CLI command handler for all x/checkpoint transaction commands.
func NewTxCmd(valAddrCodec, ac address.Codec) *cobra.Command {
	checkpointTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Staking transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	checkpointTxCmd.AddCommand(
		CheckpointAdjustCmd(valAddrCodec),
		CheckpointCmd(valAddrCodec),
		//CheckpointAckCmd(valAddrCodec),
		CheckpointNoAckCmd(),
	)

	return checkpointTxCmd
}

// CheckpointAdjustCmd returns a CLI command handler for creating a NewMsgCheckpointAdjust tx
func CheckpointAdjustCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkpoint-adjust",
		Short: "adjusts previous checkpoint transaction according to ethereum chain (details to be provided for checkpoint on ethereum chain)",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// header index
			headerIndex, err := cmd.Flags().GetUint64(FlagHeaderNumber)
			if err != nil {
				return err
			}

			// get from
			from := clientCtx.GetFromAddress().String()

			// get proposer
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

			//	end block
			endBlock, err := cmd.Flags().GetUint64(FlagEndBlock)
			if err != nil {
				return err
			}

			// root hash
			rootHashStr, err := cmd.Flags().GetString(FlagRootHash)
			if err != nil {
				return err
			}

			if rootHashStr == "" {
				return fmt.Errorf("root hash cannot be empty")
			}

			msg := types.NewMsgCheckpointAdjust(headerIndex, startBlock, endBlock, proposer, from, hmTypes.HexToHeimdallHash(rootHashStr))

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	cmd.Flags().String(FlagHeaderNumber, "", "--header=<header-index>")
	cmd.Flags().StringP(FlagProposerAddress, "p", "", "--proposer=<proposer-address>")
	cmd.Flags().Uint64(FlagStartBlock, 0, "--start-block=<start-block-number>")
	cmd.Flags().Uint64(FlagEndBlock, 0, "--end-block=<end-block-number>")
	cmd.Flags().StringP(FlagRootHash, "r", "", "--root-hash=<root-hash>")

	if err := cmd.MarkFlagRequired(FlagHeaderNumber); err != nil {
		logger.Error("SendCheckpointAdjust | MarkFlagRequired | FlagHeaderNumber", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagRootHash); err != nil {
		logger.Error("SendCheckpointAdjust | MarkFlagRequired | FlagRootHash", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagProposerAddress); err != nil {
		logger.Error("SendCheckpointAdjust | MarkFlagRequired | FlagProposerAddress", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagStartBlock); err != nil {
		logger.Error("SendCheckpointAdjust | MarkFlagRequired | FlagStartBlock", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagEndBlock); err != nil {
		logger.Error("SendCheckpointAdjust | MarkFlagRequired | FlagEndBlock", "Error", err)
	}

	//TODO HV2 Please check this, do we require it or not?
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewEditValidatorCmd returns a CLI command handler for creating a MsgEditValidator transaction.
func CheckpointCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-checkpoint",
		Short: "send checkpoint to tendermint and ethereum chain ",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// bor chain id
			borChainID, err := cmd.Flags().GetString(FlagBorChainID)
			if err != nil {
				return err
			}

			if borChainID == "" {
				return fmt.Errorf("bor chain id cannot be empty")
			}

			// get proposer
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

			//	end block
			endBlock, err := cmd.Flags().GetUint64(FlagEndBlock)
			if err != nil {
				return err
			}

			// root hash
			rootHashStr, err := cmd.Flags().GetString(FlagRootHash)
			if err != nil {
				return err
			}

			//account root hash
			accountRootHashStr, err := cmd.Flags().GetString(FlagAccountRootHash)
			if err != nil {
				return err
			}

			msg := types.NewMsgCheckpointBlock(
				proposer,
				startBlock,
				endBlock,
				hmTypes.HexToHeimdallHash(rootHashStr),
				hmTypes.HexToHeimdallHash(accountRootHashStr),
				borChainID,
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	cmd.Flags().StringP(FlagProposerAddress, "p", "", "--proposer=<proposer-address>")
	cmd.Flags().Uint64(FlagStartBlock, 0, "--start-block=<start-block-number>")
	cmd.Flags().Uint64(FlagEndBlock, 0, "--end-block=<end-block-number>")
	cmd.Flags().StringP(FlagRootHash, "r", "", "--root-hash=<root-hash>")
	cmd.Flags().String(FlagAccountRootHash, "", "--account-root=<account-root>")
	cmd.Flags().String(FlagBorChainID, "", "--bor-chain-id=<bor-chain-id>")
	cmd.Flags().Bool(FlagAutoConfigure, false, "--auto-configure=true/false")

	if err := cmd.MarkFlagRequired(FlagRootHash); err != nil {
		logger.Error("SendCheckpointTx | MarkFlagRequired | FlagRootHash", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagAccountRootHash); err != nil {
		logger.Error("SendCheckpointTx | MarkFlagRequired | FlagAccountRootHash", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagBorChainID); err != nil {
		logger.Error("SendCheckpointTx | MarkFlagRequired | FlagBorChainID", "Error", err)
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

/*

// TODO HV2 Please implement it later

// CheckpointAckCmd returns a CLI command handler for creating a MsgCheckpointAck tx
func CheckpointAckCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-ack",
		Short: "send acknowledgement for checkpoint in buffer",

		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// get from
			from := clientCtx.GetFromAddress().String()

			// header index
			headerBlock, err := cmd.Flags().GetUint64(FlagHeaderNumber)
			if err != nil {
				return err
			}

			txHashStr, err := cmd.Flags().GetString(FlagCheckpointTxHash)
			if err != nil {
				return err
			}

			if txHashStr == "" {
				return fmt.Errorf("checkpoint tx hash cannot be empty")
			}

			txHash := hmTypes.BytesToHeimdallHash(common.FromHex(txHashStr))

			// header index
			checkpointLogIndex, err := cmd.Flags().GetUint64(FlagCheckpointLogIndex)
			if err != nil {
				return err
			}

			//
			// Get header details
			//

			contractCallerObj, err := helper.NewContractCaller()
			if err != nil {
				return err
			}

			chainmanagerParams, err := util.GetChainmanagerParams(cliCtx)
			if err != nil {
				return err
			}

			// get main tx receipt
			receipt, err := contractCallerObj.GetConfirmedTxReceipt(txHash.EthHash(), chainmanagerParams.MainchainTxConfirmations)
			if err != nil || receipt == nil {
				return errors.New("Transaction is not confirmed yet. Please wait for sometime and try again")
			}

			// decode new header block event
			res, err := contractCallerObj.DecodeNewHeaderBlockEvent(
				chainmanagerParams.ChainParams.RootChainAddress.EthAddress(),
				receipt,
				checkpointLogIndex,
			)
			if err != nil {
				return errors.New("Invalid transaction for header block")
			}

			// draft new checkpoint no-ack msg
			msg := types.NewMsgCheckpointAck(
				from, // ack tx sender
				headerBlock,
				res.Proposer.String(),
				res.Start.Uint64(),
				res.End.Uint64(),
				hmTypes.BytesToHeimdallHash(res.Root[:]),
				txHash,
				checkpointLogIndex,
			)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	cmd.Flags().StringP(FlagProposerAddress, "p", "", "--proposer=<proposer-address>")
	cmd.Flags().Uint64(FlagHeaderNumber, 0, "--header=<header-index>")
	cmd.Flags().StringP(FlagCheckpointTxHash, "t", "", "--txhash=<checkpoint-txhash>")
	cmd.Flags().String(FlagCheckpointLogIndex, "", "--log-index=<log-index>")

	if err := cmd.MarkFlagRequired(FlagHeaderNumber); err != nil {
		logger.Error("SendCheckpointACKTx | MarkFlagRequired | FlagHeaderNumber", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagCheckpointTxHash); err != nil {
		logger.Error("SendCheckpointACKTx | MarkFlagRequired | FlagCheckpointTxHash", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagCheckpointLogIndex); err != nil {
		logger.Error("SendCheckpointACKTx | MarkFlagRequired | FlagCheckpointLogIndex", "Error", err)
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

*/

// CheckpointNoAckCmd returns a CLI command handler for creating a Msg
func CheckpointNoAckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-noack",
		Short: "send no-acknowledgement for last proposer",

		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress().String()

			msg := types.NewMsgCheckpointNoAck(from)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
