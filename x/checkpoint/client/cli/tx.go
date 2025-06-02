package cli

import (
	"bytes"
	"fmt"
	"strconv"

	"cosmossdk.io/core/address"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	codec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/0xPolygon/heimdall-v2/common/cli"
	"github.com/0xPolygon/heimdall-v2/helper"
	chainmanagerTypes "github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
	"github.com/0xPolygon/heimdall-v2/x/checkpoint/types"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

var logger = helper.Logger.With("module", "checkpoint/client/cli")

// NewTxCmd returns a root CLI command handler for all x/checkpoint transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Checkpoint module commands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	ac := codec.NewHexCodec()

	txCmd.AddCommand(
		SendCheckpointCmd(ac),
		SendCheckpointAckCmd(),
	)

	return txCmd
}

// SendCheckpointCmd returns a CLI command handler to create a `MsgCheckpoint` transaction.
func SendCheckpointCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-checkpoint",
		Short: "send checkpoint to cometBFT and ethereum",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// bor chain id
			borChainID := viper.GetString(FlagBorChainID)
			if borChainID == "" {
				return fmt.Errorf("bor chain id cannot be empty")
			}

			if viper.GetBool(FlagAutoConfigure) {
				stakeQueryClient := stakeTypes.NewQueryClient(clientCtx)
				checkpointQueryClient := types.NewQueryClient(clientCtx)
				proposer, err := stakeQueryClient.GetCurrentProposer(cmd.Context(), &stakeTypes.QueryCurrentProposerRequest{})
				if err != nil {
					return err
				}

				signerBytes, err := ac.StringToBytes(proposer.Validator.Signer)
				if err != nil {
					return fmt.Errorf("the validator signer address is invalid: %w", err)
				}

				if !bytes.Equal(signerBytes, helper.GetAddress()) {
					return fmt.Errorf("please wait for your turn to propose checkpoint. Checkpoint proposer: %v", proposer.Validator.Signer)
				}

				nextCheckpoint, err := checkpointQueryClient.GetNextCheckpoint(cmd.Context(), &types.QueryNextCheckpointRequest{})
				if err != nil {
					return err
				}

				msg := types.NewMsgCheckpointBlock(proposer.Validator.Signer, nextCheckpoint.Checkpoint.StartBlock, nextCheckpoint.Checkpoint.EndBlock, nextCheckpoint.Checkpoint.RootHash, nextCheckpoint.Checkpoint.AccountRootHash, borChainID)

				return cli.BroadcastMsg(clientCtx, proposer.Validator.Signer, msg, logger)
			}

			// get and check the proposer
			proposer := viper.GetString(FlagProposerAddress)
			if proposer == "" {
				proposer, err = helper.GetAddressString()
				if err != nil {
					return fmt.Errorf("the proposer address is invalid: %w", err)
				}
			}

			//	start block
			startBlockStr := viper.GetString(FlagStartBlock)
			if startBlockStr == "" {
				return fmt.Errorf("start block cannot be empty")
			}

			startBlock, err := strconv.ParseUint(startBlockStr, 10, 64)
			if err != nil {
				return err
			}

			//	end block
			endBlockStr := viper.GetString(FlagEndBlock)
			if endBlockStr == "" {
				return fmt.Errorf("end block cannot be empty")
			}

			endBlock, err := strconv.ParseUint(endBlockStr, 10, 64)
			if err != nil {
				return err
			}

			// root hash
			rootHashStr := viper.GetString(FlagRootHash)
			if rootHashStr == "" {
				return fmt.Errorf("root hash cannot be empty")
			}

			// account Root Hash
			accountRootHashStr := viper.GetString(FlagAccountRootHash)
			if accountRootHashStr == "" {
				return fmt.Errorf("account root hash cannot be empty")
			}

			msg := types.NewMsgCheckpointBlock(proposer, startBlock, endBlock, common.FromHex(rootHashStr), common.FromHex(accountRootHashStr), borChainID)

			return cli.BroadcastMsg(clientCtx, proposer, msg, logger)
		},
	}

	cmd.Flags().StringP(FlagProposerAddress, "p", "", "--proposer=<proposer-address>")
	cmd.Flags().String(FlagStartBlock, "", "--start-block=<start-block-number>")
	cmd.Flags().String(FlagEndBlock, "", "--end-block=<end-block-number>")
	cmd.Flags().StringP(FlagRootHash, "r", "", "--root-hash=<root-hash>")
	cmd.Flags().String(FlagAccountRootHash, "", "--account-root=<account-root>")
	cmd.Flags().String(FlagBorChainID, "", "--bor-chain-id=<bor-chain-id>")
	cmd.Flags().String(flags.FlagChainID, "", "--chain-id=<chain-id>")
	cmd.Flags().Bool(FlagAutoConfigure, false, "--auto-configure=true/false")

	return cmd
}

// SendCheckpointAckCmd returns a CLI command handler for creating a MsgCpAck transaction.
func SendCheckpointAckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-ack",
		Short: "send acknowledgement for checkpoint in buffer",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// get and check the proposer
			proposer := viper.GetString(FlagProposerAddress)
			if proposer == "" {
				proposer, err = helper.GetAddressString()
				if err != nil {
					return fmt.Errorf("the proposer address is invalid: %w", err)
				}
			}

			headerBlockStr := viper.GetString(FlagHeaderNumber)
			if headerBlockStr == "" {
				return fmt.Errorf("header number cannot be empty")
			}

			headerBlock, err := strconv.ParseUint(headerBlockStr, 10, 64)
			if err != nil {
				return err
			}

			txHashStr := viper.GetString(FlagCheckpointTxHash)
			if txHashStr == "" {
				return fmt.Errorf("checkpoint tx hash cannot be empty")
			}

			txHash := common.HexToHash(txHashStr)

			// get header block details
			contractCaller, err := helper.NewContractCaller()
			if err != nil {
				return err
			}

			// fetch params
			queryClient := chainmanagerTypes.NewQueryClient(clientCtx)
			cmParams, err := queryClient.GetChainManagerParams(cmd.Context(), &chainmanagerTypes.QueryParamsRequest{})
			if err != nil {
				return err
			}

			// get main tx receipt
			receipt, err := contractCaller.GetConfirmedTxReceipt(txHash, cmParams.Params.MainChainTxConfirmations)
			if err != nil || receipt == nil {
				return fmt.Errorf("transaction %s is not confirmed yet, please wait for some time and try again", txHash)
			}

			// decode the new header block event
			res, err := contractCaller.DecodeNewHeaderBlockEvent(
				cmParams.Params.ChainParams.RootChainAddress,
				receipt,
				uint64(viper.GetInt64(FlagCheckpointLogIndex)),
			)
			if err != nil {
				return fmt.Errorf("invalid transaction for header block. Error: %w", err)
			}

			msg := types.NewMsgCpAck(proposer, headerBlock, res.Proposer.String(), res.Start.Uint64(), res.End.Uint64(), res.Root[:])

			return cli.BroadcastMsg(clientCtx, proposer, &msg, logger)
		},
	}

	cmd.Flags().StringP(FlagProposerAddress, "p", "", "--proposer=<proposer-address>")
	cmd.Flags().String(FlagHeaderNumber, "", "--header=<header-index>")
	cmd.Flags().StringP(FlagCheckpointTxHash, "t", "", "--txhash=<checkpoint-txhash>")
	cmd.Flags().String(FlagCheckpointLogIndex, "", "--log-index=<log-index>")
	cmd.Flags().String(flags.FlagChainID, "", "--chain-id=<chain-id>")

	if err := cmd.MarkFlagRequired(FlagHeaderNumber); err != nil {
		logger.Error("SendCheckpointACKTx | MarkFlagRequired | FlagHeaderNumber", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagCheckpointTxHash); err != nil {
		logger.Error("SendCheckpointACKTx | MarkFlagRequired | FlagCheckpointTxHash", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagCheckpointLogIndex); err != nil {
		logger.Error("SendCheckpointACKTx | MarkFlagRequired | FlagCheckpointLogIndex", "Error", err)
	}

	return cmd
}
