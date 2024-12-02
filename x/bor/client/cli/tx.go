package cli

import (
	"fmt"
	"strconv"

	"cosmossdk.io/core/address"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	codec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/0xPolygon/heimdall-v2/x/bor/types"
)

// NewTxCmd returns a root CLI command handler for all x/bor transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Bor transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	ac := codec.NewHexCodec()

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

			_, err = ac.StringToBytes(proposer)
			if err != nil {
				return fmt.Errorf("proposer address is invalid: %v", err)
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
			res, err := queryClient.GetParams(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}
			spanDuration := res.Params.SpanDuration

			// fetch next span seed
			nextSpanSeedResponse, err := queryClient.GetNextSpanSeed(cmd.Context(), &types.QueryNextSpanSeedRequest{
				Id: spanID,
			})
			if err != nil {
				return err
			}
			seed := common.HexToHash(nextSpanSeedResponse.Seed)
			msg := types.NewMsgProposeSpan(spanID, proposer, startBlock, startBlock+spanDuration-1, borChainID, seed.Bytes())

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringP(FlagProposerAddress, "p", "", "--proposer=<proposer-address>")
	cmd.Flags().String(FlagSpanId, "", "--span-id=<span-id>")
	cmd.Flags().String(FlagBorChainId, "", "--bor-chain-id=<bor-chain-id>")
	cmd.Flags().String(FlagStartBlock, "", "--start-block=<start-block-number>")

	if err := cmd.MarkFlagRequired(FlagBorChainId); err != nil {
		fmt.Printf("PostSendProposeSpanTx | MarkFlagRequired | FlagBorChainId Error: %v", err)
	}

	if err := cmd.MarkFlagRequired(FlagStartBlock); err != nil {
		fmt.Printf("PostSendProposeSpanTx | MarkFlagRequired | FlagStartBlock Error: %v", err)
	}

	return cmd
}
