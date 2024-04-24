package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"cosmossdk.io/core/address"
	"github.com/0xPolygon/heimdall-v2/x/bor/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewTxCmd returns a root CLI command handler for all x/bor transaction commands.
func NewTxCmd(ac address.Codec) *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Bor transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewSpanProposalCmd(ac),
	)

	return txCmd
}

// NewSpanProposalCmd returns a CLI command handler for creating a MsgSpanProposal transaction.
func NewSpanProposalCmd(ac address.Codec) *cobra.Command {
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

			// get start block
			startBlockStr := viper.GetString(FlagStartBlock)
			if startBlockStr == "" {
				return fmt.Errorf("Start block cannot be empty")
			}

			startBlock, err := strconv.ParseUint(startBlockStr, 10, 64)
			if err != nil {
				return err
			}

			// get span id
			spanIDStr := viper.GetString(FlagSpanId)
			if spanIDStr == "" {
				return fmt.Errorf("Span Id cannot be empty")
			}

			spanID, err := strconv.ParseUint(spanIDStr, 10, 64)
			if err != nil {
				return err
			}

			//
			// Query data
			//

			// fetch params
			res, _, err := clientCtx.QueryWithData(fmt.Sprintf("custom/%s/%s/%s", types.ModuleName, types.QueryParams), nil)
			if err != nil {
				return err
			}
			if len(res) == 0 {
				return errors.New("params not found")
			}

			var params types.Params
			if err := json.Unmarshal(res, &params); err != nil {
				return err
			}

			spanDuration := params.SpanDuration

			// fetch next span seed
			res, _, err = clientCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", types.ModuleName, types.QueryNextSpanSeed), nil)
			if err != nil {
				return err
			}

			if len(res) == 0 {
				return errors.New("next span seed not found")
			}

			var seed common.Hash
			if err := json.Unmarshal(res, &seed); err != nil {
				return err
			}

			msg := types.MsgProposeSpanRequest{
				SpanId:     spanID,
				Proposer:   proposer,
				StartBlock: startBlock,
				EndBlock:   startBlock + spanDuration - 1,
				ChainId:    borChainID,
				Seed:       seed.Bytes(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	cmd.Flags().StringP(FlagProposerAddress, "p", "", "--proposer=<proposer-address>")
	cmd.Flags().String(FlagSpanId, "", "--span-id=<span-id>")
	cmd.Flags().String(FlagBorChainId, "", "--bor-chain-id=<bor-chain-id>")
	cmd.Flags().String(FlagStartBlock, "", "--start-block=<start-block-number>")

	if err := cmd.MarkFlagRequired(FlagBorChainId); err != nil {
		fmt.Errorf(fmt.Sprintf("PostSendProposeSpanTx | MarkFlagRequired | FlagBorChainId Error: %v", err))
	}

	if err := cmd.MarkFlagRequired(FlagStartBlock); err != nil {
		fmt.Errorf("PostSendProposeSpanTx | MarkFlagRequired | FlagStartBlock Error: %v", err)
	}

	return cmd
}
