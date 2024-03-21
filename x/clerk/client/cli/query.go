package cli

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	keeper "github.com/0xPolygon/heimdall-v2/x/clerk/keeper"
	clerkTypes "github.com/0xPolygon/heimdall-v2/x/clerk/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(cdc *codec.Codec) *cobra.Command {
	// Group supply queries under a subcommand
	queryCmds := &cobra.Command{
		Use:                        clerkTypes.ModuleName,
		Short:                      "Querying commands for the clerk module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	flags.AddQueryFlagsToCmd(queryCmds)

	// TODO HV2 - check if this is needed
	// // clerk query command
	// queryCmds.AddCommand(
	// 	client.GetCommands(
	// 		GetStateRecord(cdc),
	// 	)...,
	// )

	return queryCmds
}

// GetStateRecord shows the state record
func GetStateRecord() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "record",
		Short: "show state record",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			recordIDStr := viper.GetString(FlagRecordID)
			if recordIDStr == "" {
				return fmt.Errorf("record id cannot be empty")
			}

			recordID, err := strconv.ParseUint(recordIDStr, 10, 64)
			if err != nil {
				return err
			}

			// get query params
			queryParams, err := cliCtx.Codec.MarshalJSON(&clerkTypes.QueryRecordParams{RecordID: recordID})
			if err != nil {
				return err
			}

			// fetch state record
			res, _, err := cliCtx.QueryWithData(
				fmt.Sprintf("custom/%s/%s", clerkTypes.QuerierRoute, keeper.QueryRecord),
				queryParams,
			)

			if err != nil {
				return err
			}

			if len(res) == 0 {
				return errors.New("record not found")
			}

			fmt.Println(string(res))
			return nil
		},
	}

	cmd.Flags().Uint64(FlagRecordID, 0, "--id=<record ID here>")

	if err := cmd.MarkFlagRequired(FlagRecordID); err != nil {
		_ = fmt.Errorf("GetStateRecord | MarkFlagRequired | FlagRecordID Error: %w", err)
	}

	return cmd
}

// IsOldTx Check whether the transaction is old
func IsOldTx() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "is-old-tx",
		Short: "Check whether the transaction is old",
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			// tx hash
			txHash := viper.GetString(FlagTxHash)
			if txHash == "" {
				return fmt.Errorf("tx hash cannot be empty")
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

			// get query params
			queryParams, err := cliCtx.Codec.MarshalJSON(&clerkTypes.QueryRecordSequenceParams{TxHash: txHash, LogIndex: logIndex})
			if err != nil {
				return err
			}

			seqNo, _, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", clerkTypes.QuerierRoute, keeper.QueryRecordSequence), queryParams)
			if err != nil {
				return err
			}

			// error if no tx status found
			if len(seqNo) == 0 {
				fmt.Printf("false")
				return nil
			}

			return nil
		},
	}

	cmd.Flags().Uint64(FlagLogIndex, 0, "--log-index=<log index here>")
	cmd.Flags().Uint64(FlagTxHash, 0, "--tx-hash=<tx hash here>")

	if err := cmd.MarkFlagRequired(FlagLogIndex); err != nil {
		_ = fmt.Errorf("IsOldTx | MarkFlagRequired | FlagLogIndex Error: %w", err)
	}

	if err := cmd.MarkFlagRequired(FlagTxHash); err != nil {
		_ = fmt.Errorf("IsOldTx | MarkFlagRequired | FlagTxHash Error: %w", err)
	}

	return cmd
}
