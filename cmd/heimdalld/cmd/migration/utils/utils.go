package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	v034gov "github.com/0xPolygon/heimdall-v2/cmd/heimdalld/cmd/migration/gov/v034"
	v036gov "github.com/0xPolygon/heimdall-v2/cmd/heimdalld/cmd/migration/gov/v036"
	v036params "github.com/0xPolygon/heimdall-v2/cmd/heimdalld/cmd/migration/params/v036"
	"github.com/cosmos/cosmos-sdk/codec"
	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankTypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govTypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govTypesV1Beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	paramTypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/gogoproto/proto"
)

// LoadJSONFromFile reads a JSON file and returns the data as a map.
func LoadJSONFromFile(filename string) (map[string]interface{}, error) {
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

// SaveJSONToFile writes data map to a file in JSON format.
func SaveJSONToFile(data map[string]interface{}, filename string) error {
	fileContent, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filename, fileContent, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// RenameProperty renames a property in the data map.
func RenameProperty(data map[string]interface{}, path string, oldKey string, newKey string) error {
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

// DeleteProperty deletes a property in the data map.
func DeleteProperty(data map[string]interface{}, path string, key string) error {
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

// AddProperty adds a property to the data map.
func AddProperty(data map[string]interface{}, path string, key string, value interface{}) error {
	current, err := traversePath(data, path)
	if err != nil {
		return err
	}
	current[key] = value
	return nil
}

// traversePath traverses the path in the data map.
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

// MigrateGenesisState migrates multiple validators to the new format.
func MigrateValidators(appCodec codec.Codec, validatorsInterface interface{}) error {
	validators, ok := validatorsInterface.([]interface{})
	if !ok {
		return fmt.Errorf("failed to cast validators")
	}
	for i, validator := range validators {
		validatorMap, ok := validator.(map[string]interface{})
		if !ok {
			return fmt.Errorf("failed to cast validator data at index %d", i)
		}

		if err := MigrateValidator(appCodec, validatorMap); err != nil {
			return fmt.Errorf("failed to migrate validator at index %d: %w", i, err)
		}
	}
	return nil
}

// MigrateValidator migrates a single validator to the new format by renaming few fields and migrating the public key to proto encoding.
func MigrateValidator(appCodec codec.Codec, validator map[string]interface{}) error {
	if err := RenameProperty(validator, ".", "power", "voting_power"); err != nil {
		return fmt.Errorf("failed to rename power field: %w", err)
	}

	if err := RenameProperty(validator, ".", "accum", "proposer_priority"); err != nil {
		return fmt.Errorf("failed to rename accum field: %w", err)
	}

	if err := RenameProperty(validator, ".", "ID", "val_id"); err != nil {
		return fmt.Errorf("failed to rename ID field: %w", err)
	}

	return nil
}

// migrateGovProposalContent returns the proposal into new format with proto encoding.
func MigrateGovProposalContent(oldContent v036gov.Content) *codecTypes.Any {
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

// MigrateVoteOption migrates the vote option to the new format.
func MigrateVoteOption(oldVoteOption v034gov.VoteOption) govTypes.VoteOption {
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

// ParseUint parses the value to uint64.
func ParseUint(value interface{}) uint64 {
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

// MigrateAuthAccountsToBankBalances converts the auth accounts to bank balances.
func MigrateAuthAccountsToBankBalances(authAccounts []interface{}) ([]bankTypes.Balance, error) {

	addressCoins := map[string]types.Coins{}

	for i, account := range authAccounts {
		accountMap, ok := account.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid account format at index %d", i)
		}

		accAddress, _ := accountMap["address"].(string)
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

			if coins, ok := addressCoins[accAddress]; ok {
				addressCoins[accAddress] = append(coins, types.NewCoin(denom, amount))
			} else {
				addressCoins[accAddress] = types.NewCoins(types.NewCoin(denom, amount))
			}
		}
	}

	var balances []bankTypes.Balance

	for accAddress, coins := range addressCoins {
		balances = append(balances,
			bankTypes.Balance{
				Address: accAddress,
				Coins:   coins,
			})
	}

	return balances, nil
}

// MigrateAuthAccounts migrates the auth accounts to the new format with proto encoding.
func MigrateAuthAccounts(authData map[string]interface{}) ([]*codecTypes.Any, error) {
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
			// We skip module accounts, because heimdall v2 will initialize them from zero anyways
			continue
		}

		accAddress, _ := accountMap["address"].(string)
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

		addr, err := types.AccAddressFromHex(accAddress)
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
