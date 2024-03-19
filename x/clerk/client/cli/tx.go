package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	hexCodec "github.com/cosmos/cosmos-sdk/codec/address"

	"github.com/0xPolygon/heimdall-v2/types"
	clerkTypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

// TODO HV2 - check if we need this
// GetTxCmd returns the transaction commands for this module
func GetTxCmd(cdc *codec.Codec) *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        clerkTypes.ModuleName,
		Short:                      "Checkpoint transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	flags.AddQueryFlagsToCmd(txCmd)

	// TODO HV2 - check if this is needed
	// txCmd.AddCommand(
	// 	client.PostCommands(
	// 		CreateNewStateRecord(cdc),
	// 	)...,
	// )

	return txCmd
}

// CreateNewStateRecord send checkpoint transaction
func CreateNewStateRecord(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "record",
		Short: "new state record",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO HV2 - uncomment when we use cliCtx
			// cliCtx, err := client.GetClientQueryContext(cmd)
			_, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			// bor chain id
			borChainID := viper.GetString(FlagBorChainId)
			if borChainID == "" {
				return fmt.Errorf("BorChainID cannot be empty")
			}

			// get proposer
			proposer, err := hexCodec.NewHexCodec().StringToBytes(viper.GetString(FlagProposerAddress))
			if err != nil {
				return fmt.Errorf("error in parsing proposer address")
			}
			// TODO HV2 - check if this satisfies the condition (commented))
			// if proposer.Empty() {
			if proposer == nil {
				// TODO HV2 - uncomment when we have GetFromAddress updated and implemented in helper
				// proposer = helper.GetFromAddress(cliCtx)
				proposer = nil
				// `proposer = nil` is a placeholder
			}

			// tx hash
			txHashStr := viper.GetString(FlagTxHash)
			if txHashStr == "" {
				return fmt.Errorf("tx hash cannot be empty")
			}

			// tx hash
			recordIDStr := viper.GetString(FlagRecordID)
			if recordIDStr == "" {
				return fmt.Errorf("record id cannot be empty")
			}

			recordID, err := strconv.ParseUint(recordIDStr, 10, 64)
			if err != nil {
				return fmt.Errorf("record id cannot be empty")
			}

			// get contract Addr
			contractAddr, err := hexCodec.NewHexCodec().StringToBytes(viper.GetString(FlagContractAddress))
			if err != nil {
				return fmt.Errorf("error in parsing contract address")
			}
			// TODO HV2 - check if this satisfies the condition (commented))
			// if contractAddr.Empty() {
			if contractAddr == nil {
				return fmt.Errorf("contract Address cannot be empty")
			}

			// log index
			logIndexStr := viper.GetString(FlagLogIndex)
			if logIndexStr == "" {
				return fmt.Errorf("log index cannot be empty")
			}

			logIndex, err := strconv.ParseUint(logIndexStr, 10, 64)
			if err != nil {
				return fmt.Errorf("log index cannot be parsed")
			}

			// log index
			dataStr := viper.GetString(FlagData)
			if dataStr == "" {
				return fmt.Errorf("data cannot be empty")
			}

			// data, err := hexCodec.NewHexCodec().StringToBytes(dataStr)
			// if err != nil {
			// 	return fmt.Errorf("error in parsing data")
			// }

			// TODO HV2 - uncomment when we have setu and helper implemented
			// if util.GetBlockHeight(cliCtx) > helper.GetSpanOverrideHeight() && len(data) > helper.MaxStateSyncSize {
			// 	fmt.Sprintf(`Data is too large to process, Resetting to ""`, "id", recordIDStr)
			// 	data = hmTypes.HexToHexBytes("")
			// } else if len(data) > helper.LegacyMaxStateSyncSize {
			// 	fmt.Sprintf(`Data is too large to process, Resetting to ""`, "id", recordIDStr)
			// 	data = hmTypes.HexToHexBytes("")
			// }

			txHashBytes, err := hexCodec.NewHexCodec().StringToBytes(txHashStr)
			if err != nil {
				return fmt.Errorf("error in parsing tx hash")
			}
			// create new state record
			// TODO HV2 - uncomment when we use this msg in the return statement
			// msg := clerkTypes.NewMsgEventRecord(
			_ = clerkTypes.NewMsgEventRecord(
				proposer,
				types.HeimdallHash{Hash: txHashBytes},
				logIndex,
				viper.GetUint64(FlagBlockNumber),
				recordID,
				contractAddr,
				// TODO HV2 - uncomment when we have setu and helper implemented
				// data,
				types.HexBytes{},
				borChainID,
			)

			// uncomment when we have BroadcastMsgsWithCLI implemented in helper
			// return helper.BroadcastMsgsWithCLI(cliCtx, []sdk.Msg{msg})
			return nil
		},
	}
	cmd.Flags().StringP(FlagProposerAddress, "p", "", "--proposer=<proposer-address>")
	cmd.Flags().String(FlagTxHash, "", "--tx-hash=<tx-hash>")
	cmd.Flags().String(FlagLogIndex, "", "--log-index=<log-index>")
	cmd.Flags().String(FlagRecordID, "", "--id=<record-id>")
	cmd.Flags().String(FlagBorChainId, "", "--bor-chain-id=<bor-chain-id>")
	cmd.Flags().Uint64(FlagBlockNumber, 0, "--block-number=<block-number>")
	cmd.Flags().String(FlagContractAddress, "", "--contract-addr=<contract-addr>")
	cmd.Flags().String(FlagData, "", "--data=<data>")

	if err := cmd.MarkFlagRequired(FlagRecordID); err != nil {
		fmt.Errorf("CreateNewStateRecord | MarkFlagRequired | FlagRecordID", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagTxHash); err != nil {
		fmt.Errorf("CreateNewStateRecord | MarkFlagRequired | FlagTxHash", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagLogIndex); err != nil {
		fmt.Errorf("CreateNewStateRecord | MarkFlagRequired | FlagLogIndex", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagBorChainId); err != nil {
		fmt.Errorf("CreateNewStateRecord | MarkFlagRequired | FlagBorChainId", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagBlockNumber); err != nil {
		fmt.Errorf("CreateNewStateRecord | MarkFlagRequired | FlagBlockNumber", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagContractAddress); err != nil {
		fmt.Errorf("CreateNewStateRecord | MarkFlagRequired | FlagContractAddress", "Error", err)
	}

	if err := cmd.MarkFlagRequired(FlagData); err != nil {
		fmt.Errorf("CreateNewStateRecord | MarkFlagRequired | FlagData", "Error", err)
	}

	return cmd
}
