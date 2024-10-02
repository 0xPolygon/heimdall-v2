package heimdalld

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	"cosmossdk.io/x/tx/signing"
	v034gov "github.com/0xPolygon/heimdall-v2/cmd/heimdalld/cmd/migration/gov/v034"
	v036gov "github.com/0xPolygon/heimdall-v2/cmd/heimdalld/cmd/migration/gov/v036"
	v036params "github.com/0xPolygon/heimdall-v2/cmd/heimdalld/cmd/migration/params/v036"
	milestoneTypes "github.com/0xPolygon/heimdall-v2/x/milestone/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govTypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govTypesV1Beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	paramTypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cobra"
)

// TODO HV2: Initially in heimdall v2 we used HexBytes, HeimdallHash and TxHash
// types which were removed in favor of using bytes in proto definitions.
// Because default encoding for bytes in proto is base64 instead of hex encoding like in heimdall v1,
// it could be breaking change for anyone querying the node API.
func MigrateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate [genesis-file]",
		Short: "Migrate application state",
		Long:  `Run migrations to update the application state (e.g., for a chain upgrade) based on the provided genesis file.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runMigrate,
	}
}

func runMigrate(cmd *cobra.Command, args []string) error {
	genesisFileV1 := args[0]

	logger.Info("Starting migration...")

	if _, err := os.Stat(genesisFileV1); os.IsNotExist(err) {
		return fmt.Errorf("genesis file does not exist: %s", genesisFileV1)
	}

	// TODO: This should be done in root command PreRunE?
	interfaceRegistry, err := codecTypes.NewInterfaceRegistryWithOptions(codecTypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          address.HexCodec{},
			ValidatorAddressCodec: address.HexCodec{},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create interface registry: %w", err)
	}

	appCodec = codec.NewProtoCodec(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	authTypes.RegisterInterfaces(interfaceRegistry)
	govTypes.RegisterInterfaces(interfaceRegistry)
	paramTypes.RegisterInterfaces(interfaceRegistry)

	legacyAmino = codec.NewLegacyAmino()
	v036gov.RegisterLegacyAminoCodec(legacyAmino)
	v036params.RegisterLegacyAminoCodec(legacyAmino)

	logger.Info("Loading genesis file...", "file", genesisFileV1)

	genesisData, err := loadJSONFromFile(genesisFileV1)
	if err != nil {
		return fmt.Errorf("failed to load genesis file: %w", err)
	}

	if err := performMigrations(genesisData); err != nil {
		logger.Error("Migration failed", "error", err)
		return err
	}

	dir := filepath.Dir(genesisFileV1)
	base := filepath.Base(genesisFileV1)
	genesisFileV2 := filepath.Join(dir, fmt.Sprintf("migrated_%s", base))
	if err := saveJSONToFile(genesisData, genesisFileV2); err != nil {
		return fmt.Errorf("failed to save migrated genesis file: %w", err)
	}

	logger.Info("Migration completed successfully")

	return nil
}

func performMigrations(genesisData map[string]interface{}) error {
	logger.Info("Performing custom migrations...")

	if err := addMissingCometBFTConsensusParams(genesisData); err != nil {
		return err
	}

	if err := removeUnusedTendermintConsensusParams(genesisData); err != nil {
		return err
	}

	// Bank module should always be before auth module, because it gets accounts balances from the auth module state
	// they are deleted from the genesis during auth module migration
	if err := migrateBankModule(genesisData); err != nil {
		return err
	}

	if err := migrateAuthModule(genesisData); err != nil {
		return err
	}

	if err := migrateGovModule(genesisData); err != nil {
		return err
	}

	if err := migrateClerkModule(genesisData); err != nil {
		return err
	}

	if err := migrateBorModule(genesisData); err != nil {
		return err
	}

	if err := migrateCheckpointModule(genesisData); err != nil {
		return err
	}

	if err := migrateTopupModule(genesisData); err != nil {
		return err
	}

	if err := migrateMilestoneModule(genesisData); err != nil {
		return err
	}

	if err := migrateChainmanagerModule(genesisData); err != nil {
		return err
	}

	if err := migrateStakeModule(genesisData); err != nil {
		return err
	}

	return nil
}

func migrateStakeModule(genesisData map[string]interface{}) error {
	logger.Info("Migrating stake module...")

	// TODO: TotalVotingPower is never assigned during InitGenesis, maybe at end of the initilization we should call GetTotalVotingPower
	// because it gets calculated if its zero
	if err := renameProperty(genesisData, "app_state", "staking", "stake"); err != nil {
		return fmt.Errorf("failed to rename staking module: %w", err)
	}

	stakeModule, ok := genesisData["app_state"].(map[string]interface{})["stake"]
	if !ok {
		return fmt.Errorf("stake module not found in app_state")
	}

	stakeData, ok := stakeModule.(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to cast stake module data")
	}

	if err := renameProperty(stakeData, ".", "current_val_set", "current_validator_set"); err != nil {
		return fmt.Errorf("failed to rename current_val_set field: %w", err)
	}

	// TODO: There are couple of places where we iterate and migrate validators, we should refactor this to a single function
	validators, ok := stakeData["validators"].([]interface{})
	if !ok {
		return fmt.Errorf("failed to find validators in stake module")
	}
	for i, validator := range validators {
		validatorMap, ok := validator.(map[string]interface{})
		if !ok {
			return fmt.Errorf("failed to cast validator data at index %d", i)
		}

		if err := migrateValidator(validatorMap); err != nil {
			return fmt.Errorf("failed to migrate validator at index %d: %w", i, err)
		}
	}

	currentValidatorSet, ok := stakeData["current_validator_set"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to find current_validator_set in stake module")
	}

	validators, ok = currentValidatorSet["validators"].([]interface{})
	if !ok {
		return fmt.Errorf("failed to find validators in current_validator_set")
	}
	for i, validator := range validators {
		validatorMap, ok := validator.(map[string]interface{})
		if !ok {
			return fmt.Errorf("failed to cast validator data at index %d", i)
		}

		if err := migrateValidator(validatorMap); err != nil {
			return fmt.Errorf("failed to migrate validator at index %d: %w", i, err)
		}
	}

	proposer, ok := currentValidatorSet["proposer"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to find proposer in current_validator_set")
	}

	if err := migrateValidator(proposer); err != nil {
		return fmt.Errorf("failed to migrate proposer: %w", err)
	}

	logger.Info("Stake module migration completed successfully")

	return nil
}

func migrateChainmanagerModule(genesisData map[string]interface{}) error {
	logger.Info("Migrating chainmanager module...")

	chainmanagerModule, ok := genesisData["app_state"].(map[string]interface{})["chainmanager"]
	if !ok {
		return fmt.Errorf("chainmanager module not found in app_state")
	}

	chainmanagerData, ok := chainmanagerModule.(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to cast chainmanager module data")
	}

	if err := renameProperty(chainmanagerData, "params", "mainchain_tx_confirmations", "main_chain_tx_confirmations"); err != nil {
		return fmt.Errorf("failed to rename mainchain_tx_confirmations field: %w", err)
	}

	if err := renameProperty(chainmanagerData, "params", "maticchain_tx_confirmations", "bor_chain_tx_confirmations"); err != nil {
		return fmt.Errorf("failed to rename mainchain_tx_timeout field: %w", err)
	}

	logger.Info("Chainmanager module migration completed successfully")

	return nil
}

func migrateMilestoneModule(genesisData map[string]interface{}) error {
	logger.Info("Migrating milestone module...")

	params := milestoneTypes.DefaultParams()
	milestoneState := milestoneTypes.NewGenesisState(&params)

	milestoneStateMarshled := appCodec.MustMarshalJSON(&milestoneState)

	genesisData["app_state"].(map[string]interface{})["milestone"] = json.RawMessage(milestoneStateMarshled)

	logger.Info("Milestone module migration completed successfully")

	return nil
}

func migrateTopupModule(genesisData map[string]interface{}) error {
	logger.Info("Migrating topup module...")

	topupModule, ok := genesisData["app_state"].(map[string]interface{})["topup"]
	if !ok {
		return fmt.Errorf("topup module not found in app_state")
	}

	topupData, ok := topupModule.(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to cast topup module data")
	}

	if err := renameProperty(topupData, ".", "tx_sequences", "topup_sequences"); err != nil {
		return fmt.Errorf("failed to rename topup_sequences field: %w", err)
	}

	logger.Info("Topup module migration completed successfully")

	return nil
}

func migrateGovModule(genesisData map[string]interface{}) error {
	logger.Info("Migrating gov module...")

	govModule, ok := genesisData["app_state"].(map[string]interface{})["gov"]
	if !ok {
		return fmt.Errorf("gov module not found in app_state")
	}

	govJSON, err := json.Marshal(govModule)
	if err != nil {
		return fmt.Errorf("failed to marshal gov module: %w", err)
	}

	oldGovState := v036gov.GenesisState{}

	legacyAmino.MustUnmarshalJSON(govJSON, &oldGovState)

	newDeposits := make([]*govTypes.Deposit, len(oldGovState.Deposits))
	for i, oldDeposit := range oldGovState.Deposits {
		newDeposits[i] = &govTypes.Deposit{
			ProposalId: oldDeposit.ProposalID,
			Depositor:  oldDeposit.Depositor.String(),
			Amount:     oldDeposit.Amount,
		}
	}

	newVotes := make([]*govTypes.Vote, len(oldGovState.Votes))
	for i, oldVote := range oldGovState.Votes {
		newVotes[i] = &govTypes.Vote{
			ProposalId: oldVote.ProposalID,
			Voter:      oldVote.Voter.String(),
			Options:    govTypes.NewNonSplitVoteOption(migrateVoteOption(oldVote.Option)),
		}
	}

	newProposals := make([]*govTypes.Proposal, len(oldGovState.Proposals))
	for i, oldProposal := range oldGovState.Proposals {

		newProposals[i] = &govTypes.Proposal{
			Id:       oldProposal.ProposalID,
			Messages: []*codecTypes.Any{migrateContent(oldProposal.Content)},
			Title:    oldProposal.GetTitle(),
			Summary:  oldProposal.GetDescription(),
			Status:   govTypes.ProposalStatus(oldProposal.Status),
			FinalTallyResult: &govTypes.TallyResult{
				YesCount:        oldProposal.FinalTallyResult.Yes,
				AbstainCount:    oldProposal.FinalTallyResult.Abstain,
				NoCount:         oldProposal.FinalTallyResult.No,
				NoWithVetoCount: oldProposal.FinalTallyResult.NoWithVeto,
			},
			SubmitTime:      &oldProposal.SubmitTime,
			DepositEndTime:  &oldProposal.DepositEndTime,
			TotalDeposit:    oldProposal.TotalDeposit,
			VotingStartTime: &oldProposal.VotingStartTime,
			VotingEndTime:   &oldProposal.VotingEndTime,
		}
	}

	defaultParams := govTypes.DefaultParams()

	params := govTypes.NewParams(
		oldGovState.DepositParams.MinDeposit,
		defaultParams.ExpeditedMinDeposit,
		oldGovState.DepositParams.MaxDepositPeriod,
		oldGovState.VotingParams.VotingPeriod,
		*defaultParams.ExpeditedVotingPeriod,
		oldGovState.TallyParams.Quorum,
		oldGovState.TallyParams.Threshold,
		defaultParams.ExpeditedThreshold,
		oldGovState.TallyParams.Veto,
		defaultParams.MinInitialDepositRatio,
		defaultParams.ProposalCancelRatio,
		defaultParams.ProposalCancelDest,
		defaultParams.BurnProposalDepositPrevote,
		defaultParams.BurnVoteQuorum,
		defaultParams.BurnVoteVeto,
		defaultParams.MinDepositRatio,
	)

	newGovState := govTypes.GenesisState{
		StartingProposalId: oldGovState.StartingProposalID,
		Deposits:           newDeposits,
		Votes:              newVotes,
		Proposals:          newProposals,
		Params:             &params,
		Constitution:       "This chain has no constitution.", // TODO HV2: This should be updated with the actual constitution
	}

	newGovStateMarshaled := appCodec.MustMarshalJSON(&newGovState)

	genesisData["app_state"].(map[string]interface{})["gov"] = json.RawMessage(newGovStateMarshaled)

	logger.Info("Gov module migration completed successfully")

	return nil
}

func migrateContent(oldContent v036gov.Content) *codecTypes.Any {
	authority := authTypes.NewModuleAddress(v036gov.ModuleName).String()

	var protoProposal proto.Message

	switch oldContent := oldContent.(type) {
	case v036gov.TextProposal:
		{
			protoProposal = &govTypesV1Beta1.TextProposal{
				Title:       oldContent.Title,
				Description: oldContent.Description,
			}
			contentAny, err := codecTypes.NewAnyWithValue(protoProposal)
			if err != nil {
				panic(err)
			}
			return contentAny
		}
	case v036params.ParameterChangeProposal:
		{
			newChanges := make([]paramTypes.ParamChange, len(oldContent.Changes))
			for i, oldChange := range oldContent.Changes {
				newChanges[i] = paramTypes.ParamChange{
					Subspace: oldChange.Subspace,
					Key:      oldChange.Key,
					Value:    oldChange.Value,
				}
			}

			protoProposal = &paramTypes.ParameterChangeProposal{
				Description: oldContent.Description,
				Title:       oldContent.Title,
				Changes:     newChanges,
			}
		}
	default:
		panic(fmt.Errorf("%T is not a valid proposal content type", oldContent))
	}

	any, err := codecTypes.NewAnyWithValue(protoProposal)
	if err != nil {
		panic(fmt.Errorf("failed to create Any type for proposal content: %w", err))
	}

	msg := govTypes.NewMsgExecLegacyContent(any, authority)

	msgAny, err := codecTypes.NewAnyWithValue(msg)
	if err != nil {
		panic(fmt.Errorf("failed to create Any type for proposal content: %w", err))
	}

	return msgAny
}

func migrateVoteOption(oldVoteOption v034gov.VoteOption) govTypes.VoteOption {
	switch oldVoteOption {
	case v034gov.OptionEmpty:
		return govTypes.OptionEmpty

	case v034gov.OptionYes:
		return govTypes.OptionYes

	case v034gov.OptionAbstain:
		return govTypes.OptionAbstain

	case v034gov.OptionNo:
		return govTypes.OptionNo

	case v034gov.OptionNoWithVeto:
		return govTypes.OptionNoWithVeto

	default:
		panic(fmt.Errorf("'%s' is not a valid vote option", oldVoteOption))
	}
}

func migrateBankModule(genesisData map[string]interface{}) error {
	logger.Info("Migrating bank module...")

	bankModule, ok := genesisData["app_state"].(map[string]interface{})["bank"]
	if !ok {
		return fmt.Errorf("bank module not found in app_state")
	}

	bankData, ok := bankModule.(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to cast bank module data")
	}

	authModule, ok := genesisData["app_state"].(map[string]interface{})["auth"]
	if !ok {
		return fmt.Errorf("auth module not found in app_state")
	}

	authData, ok := authModule.(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to cast auth module data")
	}

	accounts, ok := authData["accounts"].([]interface{})
	if !ok {
		return fmt.Errorf("accounts not found or invalid format")
	}

	balances, err := migrateAuthAccountsToBankBalances(accounts)
	if err != nil {
		return err
	}

	sendEnabled, ok := bankData["send_enabled"].(bool)
	if !ok {
		return fmt.Errorf("send_enabled not found in bank module")
	}

	totalSupply, err := getTotalSupply(genesisData)
	if err != nil {
		return err
	}

	newBankGenesis := bankTypes.GenesisState{
		Params:        bankTypes.NewParams(sendEnabled),
		Balances:      balances,
		Supply:        []types.Coin{totalSupply},
		DenomMetadata: []bankTypes.Metadata{},
		SendEnabled:   []bankTypes.SendEnabled{},
	}

	marshaledGenesisState := appCodec.MustMarshalJSON(&newBankGenesis)

	genesisData["app_state"].(map[string]interface{})["bank"] = json.RawMessage(marshaledGenesisState)

	logger.Info("Bank module migration completed successfully")

	return nil
}

func getTotalSupply(genesisData map[string]interface{}) (types.Coin, error) {
	supplyModule, ok := genesisData["app_state"].(map[string]interface{})["supply"]
	if !ok {
		return types.Coin{}, fmt.Errorf("supply module not found in app_state")
	}

	supplyData, ok := supplyModule.(map[string]interface{})["supply"].(map[string]interface{})
	if !ok {
		return types.Coin{}, fmt.Errorf("failed to cast supply module data")
	}

	totalSupply, ok := supplyData["total"].([]interface{})
	if !ok {
		return types.Coin{}, fmt.Errorf("total supply not found in supply module")
	}

	coin, ok := totalSupply[0].(map[string]interface{})
	if !ok {
		return types.Coin{}, fmt.Errorf("invalid coin format")
	}

	denom, ok := coin["denom"].(string)
	if !ok {
		return types.Coin{}, fmt.Errorf("denom not found in total supply")
	}

	amountStr, ok := coin["amount"].(string)
	if !ok {
		return types.Coin{}, fmt.Errorf("amount not found in total supply")
	}

	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return types.Coin{}, fmt.Errorf("failed to parse amount: %s", amountStr)
	}

	return types.NewCoin(denom, amount), nil
}

func migrateAuthAccountsToBankBalances(authAccounts []interface{}) ([]bankTypes.Balance, error) {

	addressCoins := map[string]types.Coins{}

	for i, account := range authAccounts {
		accountMap, ok := account.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid account format at index %d", i)
		}

		address, _ := accountMap["address"].(string)
		coins, _ := accountMap["coins"].([]interface{})

		for _, coin := range coins {
			coinMap, ok := coin.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid coin format at index %d", i)
			}

			denom, _ := coinMap["denom"].(string)
			amountStr, _ := coinMap["amount"].(string)

			amount, ok := math.NewIntFromString(amountStr)
			if !ok {
				return nil, fmt.Errorf("failed to parse amount at index %d: %s", i, amountStr)
			}

			if coins, ok := addressCoins[address]; ok {
				addressCoins[address] = append(coins, types.NewCoin(denom, amount))
			} else {
				addressCoins[address] = types.NewCoins(types.NewCoin(denom, amount))
			}
		}
	}

	var balances []bankTypes.Balance

	for address, coins := range addressCoins {
		balances = append(balances,
			bankTypes.Balance{
				Address: address,
				Coins:   coins,
			})
	}

	return balances, nil
}

func migrateAuthModule(genesisData map[string]interface{}) error {
	logger.Info("Migrating auth module...")

	authModule, ok := genesisData["app_state"].(map[string]interface{})["auth"]
	if !ok {
		return fmt.Errorf("auth module not found in app_state")
	}

	authData, ok := authModule.(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to cast auth module data")
	}

	baseAccounts, err := migrateAuthAccounts(authData)
	if err != nil {
		return err
	}

	params, err := migrateParams(authData)
	if err != nil {
		return err
	}

	genesisState := authTypes.GenesisState{
		Accounts: baseAccounts,
		Params:   *params,
	}

	marshaledGenesisState := appCodec.MustMarshalJSON(&genesisState)

	genesisData["app_state"].(map[string]interface{})["auth"] = json.RawMessage(marshaledGenesisState)

	logger.Info("Auth module migration completed successfully")

	return nil
}

func migrateAuthAccounts(authData map[string]interface{}) ([]*codecTypes.Any, error) {

	accounts, ok := authData["accounts"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("accounts not found or invalid format")
	}

	var baseAccounts authTypes.GenesisAccounts

	for i, account := range accounts {
		accountMap, ok := account.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid account format at index %d", i)
		}

		module_name, _ := accountMap["module_name"].(string)
		if module_name != "" {
			// TODO: We skip module accounts, because heimdall v2 will initialize them from zero anyways
			continue
		}

		address, _ := accountMap["address"].(string)
		accountNumberStr, _ := accountMap["account_number"].(string)
		sequenceNumberStr, _ := accountMap["sequence_number"].(string)

		accountNumber, err := strconv.ParseUint(accountNumberStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse account number at index %d: %w", i, err)
		}
		sequenceNumber, err := strconv.ParseUint(sequenceNumberStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sequence number at index %d: %w", i, err)
		}

		addr, err := types.AccAddressFromHex(address)
		if err != nil {
			return nil, fmt.Errorf("failed to parse address at index %d: %w", i, err)
		}

		baseAccounts = append(baseAccounts, authTypes.NewBaseAccount(addr, nil, accountNumber, sequenceNumber))
	}

	packedAccounts, err := authTypes.PackAccounts(baseAccounts)
	if err != nil {
		return nil, fmt.Errorf("failed to pack accounts: %w", err)
	}

	return packedAccounts, nil
}

func migrateParams(authData map[string]interface{}) (*authTypes.Params, error) {
	paramsData, ok := authData["params"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("params not found in auth module")
	}

	params := authTypes.Params{
		MaxMemoCharacters:      parseUint(paramsData["max_memo_characters"]),
		TxSigLimit:             parseUint(paramsData["tx_sig_limit"]),
		TxSizeCostPerByte:      parseUint(paramsData["tx_size_cost_per_byte"]),
		SigVerifyCostED25519:   parseUint(paramsData["sig_verify_cost_ed25519"]),
		SigVerifyCostSecp256k1: parseUint(paramsData["sig_verify_cost_secp256k1"]),
		MaxTxGas:               parseUint(paramsData["max_tx_gas"]),
		TxFees:                 paramsData["tx_fees"].(string),
	}

	return &params, nil
}

func parseUint(value interface{}) uint64 {
	switch v := value.(type) {
	case string:
		parsedValue, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("failed to parse string to uint64: %v", err))
		}
		return parsedValue
	case float64:
		return uint64(v)
	default:
		panic(fmt.Sprintf("unexpected type for uint64 parsing: %T", value))
	}
}

func migrateCheckpointModule(genesisData map[string]interface{}) error {
	logger.Info("Migrating checkpoint module...")

	checkpointModule, ok := genesisData["app_state"].(map[string]interface{})["checkpoint"]
	if !ok {
		return fmt.Errorf("bor module not found in app_state")
	}

	checkpointData, ok := checkpointModule.(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to cast checkpoint module data")
	}

	params, ok := checkpointData["params"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("checkpoint params not found")
	}

	// Get the current checkpoint_buffer_time (which is in nanoseconds)
	checkpointBufferTimeStr, ok := params["checkpoint_buffer_time"].(string)
	if !ok {
		return fmt.Errorf("checkpoint_buffer_time not found or invalid format")
	}

	// Convert the checkpoint_buffer_time from string to int64 (nanoseconds)
	checkpointBufferTimeNs, err := strconv.ParseInt(checkpointBufferTimeStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse checkpoint_buffer_time: %w", err)
	}

	// Convert nanoseconds to time.Duration
	checkpointBufferTimeDuration := time.Duration(checkpointBufferTimeNs)

	// Convert the duration to a human-readable format (e.g., seconds)
	checkpointBufferTimeReadable := fmt.Sprintf("%ds", int64(checkpointBufferTimeDuration.Seconds()))

	// Update the checkpoint_buffer_time with the human-readable value
	params["checkpoint_buffer_time"] = checkpointBufferTimeReadable

	if err := renameProperty(params, ".", "child_chain_block_interval", "child_block_interval"); err != nil {
		return fmt.Errorf("failed to rename child_chain_block_interval field: %w", err)
	}

	checkpoints, ok := checkpointData["checkpoints"].([]interface{})
	if !ok {
		return fmt.Errorf("checkpoints not found in checkpoint module")
	}

	for i, checkpoint := range checkpoints {
		checkpointMap, ok := checkpoint.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid checkpoint format at index %d", i)
		}

		rootHashHex, ok := checkpointMap["root_hash"].(string)
		if !ok || !strings.HasPrefix(rootHashHex, "0x") {
			return fmt.Errorf("invalid or missing root_hash at index %d", i)
		}

		rootHashHex = rootHashHex[2:]

		rootHashBytes, err := hex.DecodeString(rootHashHex)
		if err != nil {
			return fmt.Errorf("failed to decode root_hash at index %d: %w", i, err)
		}

		rootHashBase64 := base64.StdEncoding.EncodeToString(rootHashBytes)

		checkpointMap["root_hash"] = rootHashBase64
	}

	logger.Info("Checkpoint module migration completed successfully")

	return nil
}

func migrateBorModule(genesisData map[string]interface{}) error {
	logger.Info("Migrating bor module...")

	borModule, ok := genesisData["app_state"].(map[string]interface{})["bor"]
	if !ok {
		return fmt.Errorf("bor module not found in app_state")
	}

	borData, ok := borModule.(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to cast bor module data")
	}

	spans, ok := borData["spans"].([]interface{})
	if !ok {
		return fmt.Errorf("failed to find spans in bor module")
	}

	for _, span := range spans {
		spanMap, ok := span.(map[string]interface{})
		if !ok {
			return fmt.Errorf("failed to cast span data")
		}

		if err := renameProperty(spanMap, ".", "bor_chain_id", "chain_id"); err != nil {
			return fmt.Errorf("failed to rename bor_chain_id field: %w", err)
		}

		if err := renameProperty(spanMap, ".", "span_id", "id"); err != nil {
			return fmt.Errorf("failed to rename bor_chain_id field: %w", err)
		}

		producers, ok := spanMap["selected_producers"].([]interface{})
		if !ok {
			return fmt.Errorf("failed to find selected_producers in span")
		}

		for i, producer := range producers {
			producerMap, ok := producer.(map[string]interface{})
			if !ok {
				return fmt.Errorf("failed to cast producer data")
			}

			if err := migrateValidator(producerMap); err != nil {
				return fmt.Errorf("failed to migrate producer at index %d: %w", i, err)
			}
		}

		validatorSet, ok := spanMap["validator_set"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("failed to find validator_set in span")
		}

		proposer, ok := validatorSet["proposer"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("failed to find proposer in validator_set")
		}

		if err := migrateValidator(proposer); err != nil {
			return fmt.Errorf("failed to migrate proposer: %w", err)
		}

		validators, ok := validatorSet["validators"].([]interface{})
		if !ok {
			return fmt.Errorf("failed to find validators in validator_set")
		}
		for i, validator := range validators {
			validatorMap, ok := validator.(map[string]interface{})
			if !ok {
				return fmt.Errorf("failed to cast validator data at index %d", i)
			}

			if err := migrateValidator(validatorMap); err != nil {
				return fmt.Errorf("failed to migrate validator at index %d: %w", i, err)
			}
		}
	}

	logger.Info("Bor module migration completed successfully")

	return nil
}

func migrateValidator(validator map[string]interface{}) error {
	if err := renameProperty(validator, ".", "power", "voting_power"); err != nil {
		return fmt.Errorf("failed to rename power field: %w", err)
	}

	if err := renameProperty(validator, ".", "accum", "proposer_priority"); err != nil {
		return fmt.Errorf("failed to rename accum field: %w", err)
	}

	if err := renameProperty(validator, ".", "ID", "val_id"); err != nil {
		return fmt.Errorf("failed to rename ID field: %w", err)
	}

	pubKeyStr, ok := validator["pubKey"].(string)
	if !ok {
		return fmt.Errorf("public key not found")
	}

	migratedKey, err := migratePubKey(pubKeyStr)
	if err != nil {
		return fmt.Errorf("failed to migrate pubKey: %w", err)
	}

	validator["pubKey"] = json.RawMessage(migratedKey)

	return nil
}

func migratePubKey(pubKeyStr string) (json.RawMessage, error) {

	pubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(pubKeyStr, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex public key: %w", err)
	}

	secpKey := secp256k1.PubKey{Key: pubKeyBytes}

	anyPubKey, err := codecTypes.NewAnyWithValue(&secpKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Any type for pubKey: %w", err)
	}

	return appCodec.MustMarshalJSON(anyPubKey), nil
}

func migrateClerkModule(genesisData map[string]interface{}) error {
	logger.Info("Migrating clerk module...")

	clerkModule, ok := genesisData["app_state"].(map[string]interface{})["clerk"]
	if !ok {
		return fmt.Errorf("clerk module not found in app_state")
	}

	clerkData, ok := clerkModule.(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to cast clerk module data")
	}

	eventRecords, ok := clerkData["event_records"]
	if !ok || eventRecords == nil {
		return fmt.Errorf("event_records not found in clerk module")
	}

	records, ok := eventRecords.([]interface{})
	if !ok {
		return fmt.Errorf("invalid event_records format")
	}

	for i, record := range records {
		recMap, ok := record.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid record format at index %d", i)
		}

		dataHex, ok := recMap["data"].(string)
		if !ok {
			return fmt.Errorf("data field not found or invalid at index %d", i)
		}

		dataHex = strings.TrimPrefix(dataHex, "0x")

		decodedData, err := hex.DecodeString(dataHex)
		if err != nil {
			return fmt.Errorf("failed to decode hex data at index %d: %w", i, err)
		}

		base64Data := base64.StdEncoding.EncodeToString(decodedData)

		recMap["data"] = base64Data
	}

	logger.Info("Clerk module migration completed successfully")

	return nil
}

// Changes are according to genesis.json generated by CometBFT 0.38.5
func addMissingCometBFTConsensusParams(genesisData map[string]interface{}) error {
	logger.Info("Adding missing CometBFT consensus parameters...")

	if err := addProperty(genesisData, "consensus_params.evidence", "max_age_num_blocks", "100000"); err != nil {
		return err
	}

	if err := addProperty(genesisData, "consensus_params.evidence", "max_age_duration", "172800000000000"); err != nil {
		return err
	}

	if err := addProperty(genesisData, "consensus_params.evidence", "max_bytes", "1048576"); err != nil {
		return err
	}

	return nil
}

// Changes are according to genesis.json generated by CometBFT 0.38.5
func removeUnusedTendermintConsensusParams(genesisData map[string]interface{}) error {
	logger.Info("Removing unused Tendermint consensus parameters...")

	if err := deleteProperty(genesisData, "consensus_params.evidence", "max_age"); err != nil {
		return err
	}

	if err := deleteProperty(genesisData, "consensus_params.block", "time_iota_ms"); err != nil {
		return err
	}

	return nil
}

func loadJSONFromFile(filename string) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if len(fileContent) == 0 {
		return nil, fmt.Errorf("file is empty")
	}

	if err := json.Unmarshal(fileContent, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return data, nil
}

func saveJSONToFile(data map[string]interface{}, filename string) error {
	logger.Info("Saving migrated genesis file...", "file", filename)

	fileContent, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filename, fileContent, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func traversePath(data map[string]interface{}, path string) (map[string]interface{}, error) {
	if path == "." {
		return data, nil
	}

	keys := strings.Split(path, ".")
	current := data

	for _, key := range keys {
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
			continue
		}
		return nil, fmt.Errorf("invalid path: %s", path)
	}

	return current, nil
}

func renameProperty(data map[string]interface{}, path string, oldKey string, newKey string) error {
	current, err := traversePath(data, path)
	if err != nil {
		return err
	}

	if val, exists := current[oldKey]; exists {
		current[newKey] = val
		delete(current, oldKey)
		return nil
	}

	return fmt.Errorf("property %s not found", oldKey)
}

func deleteProperty(data map[string]interface{}, path string, key string) error {
	current, err := traversePath(data, path)
	if err != nil {
		return err
	}

	if _, exists := current[key]; exists {
		delete(current, key)
		return nil
	}

	return fmt.Errorf("property %s not found", key)
}

func addProperty(data map[string]interface{}, path string, key string, value interface{}) error {
	current, err := traversePath(data, path)
	if err != nil {
		return err
	}
	current[key] = value
	return nil
}

var appCodec *codec.ProtoCodec
var legacyAmino *codec.LegacyAmino
