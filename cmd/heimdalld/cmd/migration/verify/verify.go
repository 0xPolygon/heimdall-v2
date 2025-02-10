package verify

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/cosmos-sdk/types"

	heimdallApp "github.com/0xPolygon/heimdall-v2/app"
	"github.com/0xPolygon/heimdall-v2/cmd/heimdalld/cmd/migration/utils"
	hmTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// VerifyMigration verifies the migration from Heimdall v1 to Heimdall v2 by consuming the migrated genesis file
// and verifying balances, validators, bor spans, clerk events, and checkpoints
func VerifyMigration(hv1GenesisPath, hv2GenesisPath string, logger log.Logger) error {
	logger.Info("Verifying migration")

	hv1Genesis, err := utils.LoadJSONFromFile(hv1GenesisPath)
	if err != nil {
		return err
	}

	db := dbm.NewMemDB()

	appOptions := make(simtestutil.AppOptionsMap)
	appOptions[flags.FlagHome] = heimdallApp.DefaultNodeHome

	app := heimdallApp.NewHeimdallApp(logger, db, nil, true, appOptions)

	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})

	genesisState, err := getGenesisAppState(hv2GenesisPath)
	if err != nil {
		return err
	}

	if err := verifyDataLists(hv1Genesis, hv2GenesisPath); err != nil {
		return err
	}

	delete(genesisState, "bor")
	delete(genesisState, "clerk")

	if _, err := app.ModuleManager.InitGenesis(ctx, app.AppCodec(), genesisState); err != nil {
		return err
	}

	if err := verifyBalances(ctx, app, hv1Genesis); err != nil {
		return err
	}

	if err := verifyValidators(ctx, app, hv1Genesis); err != nil {
		return err
	}

	if err := verifyCheckpoints(hv1Genesis, hv2GenesisPath); err != nil {
		return err
	}

	logger.Info("Migration verified successfully")

	return nil
}

// verifyBalances verifies the balances of all the accounts in the genesis file to the balances in the database
func verifyBalances(ctx types.Context, app *heimdallApp.HeimdallApp, hv1Genesis map[string]interface{}) error {
	authModule, ok := hv1Genesis["app_state"].(map[string]interface{})["auth"]
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

	addressCoins := map[string]types.Coins{}

	for i, account := range accounts {
		accountMap, ok := account.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid account format at index %d", i)
		}

		accAddress, _ := accountMap["address"].(string)
		coins, _ := accountMap["coins"].([]interface{})

		for _, coin := range coins {
			coinMap, ok := coin.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid coin format at index %d", i)
			}

			denom, _ := coinMap["denom"].(string)
			amountStr, _ := coinMap["amount"].(string)

			amount, ok := math.NewIntFromString(amountStr)
			if !ok {
				return fmt.Errorf("failed to parse amount at index %d: %s", i, amountStr)
			}

			if coins, ok := addressCoins[accAddress]; ok {
				addressCoins[accAddress] = append(coins, types.NewCoin(denom, amount))
			} else {
				addressCoins[accAddress] = types.NewCoins(types.NewCoin(denom, amount))
			}
		}
	}

	dbBalances := app.BankKeeper.GetAccountsBalances(ctx)
	for _, balance := range dbBalances {
		if coins, ok := addressCoins[balance.Address]; ok {
			for _, coin := range coins {
				if !balance.Coins.AmountOf(coin.Denom).Equal(coin.Amount) {
					return fmt.Errorf("mismatch in balance for address %s: expected %s, got %s", balance.Address, coin, balance.Coins.AmountOf(coin.Denom))
				}
			}
		} else {
			return fmt.Errorf("balance not found for address %s", balance.Address)
		}
	}

	return nil
}

// verifyValidators verifies the validators in the genesis file to the validators in the database using basic validator info
func verifyValidators(ctx types.Context, app *heimdallApp.HeimdallApp, hv1Genesis map[string]interface{}) error {
	stakingModule, ok := hv1Genesis["app_state"].(map[string]interface{})["staking"]
	if !ok {
		return fmt.Errorf("staking module not found in app_state")
	}

	stakingModuleData, ok := stakingModule.(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to cast staking module data")
	}

	curValSetDB := app.StakeKeeper.GetCurrentValidators(ctx)

	currentValSet, ok := stakingModuleData["current_val_set"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("current_val_set not found or invalid format")
	}

	curValSetGenesis, ok := currentValSet["validators"].([]interface{})
	if !ok {
		return fmt.Errorf("current_val_set validators not found or invalid format")
	}

	if len(curValSetGenesis) > 0 {
		curValSetDbMap := map[string]hmTypes.Validator{}

		for _, valDB := range curValSetDB {
			curValSetDbMap[valDB.Signer] = valDB
		}

		for i, validator := range curValSetGenesis {
			basicValInfo, err := getValidatorBasicInfo(validator)
			if err != nil {
				return fmt.Errorf("failed to get validator basic info at index %d: %w", i, err)
			}

			validatorDB, ok := curValSetDbMap[basicValInfo.signer]
			if !ok {
				return fmt.Errorf("validator not found in current validator set database: %s", basicValInfo.signer)
			}

			if err := compareValidators(validatorDB, basicValInfo); err != nil {
				return fmt.Errorf("validator mismatch %s: %w", basicValInfo.signer, err)
			}

			delete(curValSetDbMap, basicValInfo.signer)
		}

		return nil
	}

	allValidatorsGenesis, ok := stakingModuleData["validators"].([]interface{})
	if !ok {
		return fmt.Errorf("validators not found or invalid format")
	}

	allValidatorsDB := app.StakeKeeper.GetAllValidators(ctx)

	validatorsDbMap := map[string]*hmTypes.Validator{}
	for _, valDB := range allValidatorsDB {
		validatorsDbMap[valDB.Signer] = valDB
	}

	for i, validator := range allValidatorsGenesis {
		basicValInfo, err := getValidatorBasicInfo(validator)
		if err != nil {
			return fmt.Errorf("failed to get validator basic info at index %d: %w", i, err)
		}

		validatorDB, ok := validatorsDbMap[basicValInfo.signer]
		if !ok {
			return fmt.Errorf("validator not found in database: %s", basicValInfo.signer)
		}

		if err := compareValidators(*validatorDB, basicValInfo); err != nil {
			return fmt.Errorf("validator mismatch %s: %w", basicValInfo.signer, err)
		}

		delete(validatorsDbMap, basicValInfo.signer)
	}

	return nil
}

// compareValidators compares the validator in the database to the validator in the genesis file based on basic info
func compareValidators(validatorDB hmTypes.Validator, validatorGenesis *validatorBasicInfo) error {
	if validatorDB.Signer != validatorGenesis.signer {
		return fmt.Errorf("mismatch in signer for validator %s: expected %s, got %s", validatorDB.Signer, validatorGenesis.signer, validatorDB.Signer)
	}

	if validatorDB.VotingPower != validatorGenesis.power {
		return fmt.Errorf("mismatch in power for validator %s: expected %d, got %d", validatorDB.Signer, validatorGenesis.power, validatorDB.VotingPower)
	}

	if validatorDB.Nonce != validatorGenesis.nonce {
		return fmt.Errorf("mismatch in nonce for validator %s: expected %d, got %d", validatorDB.Signer, validatorGenesis.nonce, validatorDB.Nonce)
	}

	if validatorDB.Jailed != validatorGenesis.jailed {
		return fmt.Errorf("mismatch in jailed status for validator %s: expected %t, got %t", validatorDB.Signer, validatorGenesis.jailed, validatorDB.Jailed)
	}

	return nil
}

// getValidatorBasicInfo extracts the basic info of a validator from the genesis file
func getValidatorBasicInfo(validator interface{}) (*validatorBasicInfo, error) {
	validatorMap, ok := validator.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid validator format")
	}

	jailed, ok := validatorMap["jailed"].(bool)
	if !ok {
		return nil, errors.New("jailed not found or invalid format")
	}

	signer, ok := validatorMap["signer"].(string)
	if !ok {
		return nil, errors.New("signer not found or invalid format")
	}

	powerStr, ok := validatorMap["power"].(string)
	if !ok {
		return nil, errors.New("power not found or invalid format")
	}

	nonceStr, ok := validatorMap["nonce"].(string)
	if !ok {
		return nil, errors.New("nonce not found or invalid format")
	}

	power, err := strconv.ParseInt(powerStr, 10, 64)
	if err != nil {
		return nil, err
	}

	nonce, err := strconv.ParseUint(nonceStr, 10, 64)
	if err != nil {
		return nil, err
	}

	return &validatorBasicInfo{
		power:  power,
		signer: signer,
		nonce:  nonce,
		jailed: jailed,
	}, nil
}

// verifyDataLists verifies the count the bor spans, clerk events, and checkpoints in the genesis file
func verifyDataLists(hv1Genesis map[string]interface{}, hv2GenesisPath string) error {
	hv2Genesis, err := utils.LoadJSONFromFile(hv2GenesisPath)
	if err != nil {
		return err
	}

	hv1BorSpans, ok := hv1Genesis["app_state"].(map[string]interface{})["bor"].(map[string]interface{})["spans"].([]interface{})
	if !ok {
		return errors.New("bor spans not found or invalid format")
	}

	hv2BorSpans, ok := hv2Genesis["app_state"].(map[string]interface{})["bor"].(map[string]interface{})["spans"].([]interface{})
	if !ok {
		return errors.New("bor spans not found or invalid format")
	}

	if len(hv1BorSpans) != len(hv2BorSpans) {
		return fmt.Errorf("mismatch in bor spans count: expected %d, got %d", len(hv1BorSpans), len(hv2BorSpans))
	}

	hv1ClerkEvents, ok := hv1Genesis["app_state"].(map[string]interface{})["clerk"].(map[string]interface{})["event_records"].([]interface{})
	if !ok {
		return errors.New("clerk events not found or invalid format")
	}

	hv2ClerkEvents, ok := hv2Genesis["app_state"].(map[string]interface{})["clerk"].(map[string]interface{})["event_records"].([]interface{})
	if !ok {
		return errors.New("clerk events not found or invalid format")
	}

	if len(hv1ClerkEvents) != len(hv2ClerkEvents) {
		return fmt.Errorf("mismatch in clerk events count: expected %d, got %d", len(hv1ClerkEvents), len(hv2ClerkEvents))
	}

	hv1Checkpoints, ok := hv1Genesis["app_state"].(map[string]interface{})["checkpoint"].(map[string]interface{})["checkpoints"].([]interface{})
	if !ok {
		return errors.New("checkpoints not found or invalid format")
	}

	hv2Checkpoints, ok := hv2Genesis["app_state"].(map[string]interface{})["checkpoint"].(map[string]interface{})["checkpoints"].([]interface{})
	if !ok {
		return errors.New("checkpoints not found or invalid format")
	}

	if len(hv1Checkpoints) != len(hv2Checkpoints) {
		return fmt.Errorf("mismatch in checkpoints count: expected %d, got %d", len(hv1Checkpoints), len(hv2Checkpoints))
	}

	return nil
}

// verifyCheckpoints verifies the checkpoints in the genesis files by comparing the data in both versions
func verifyCheckpoints(hv1Genesis map[string]interface{}, hv2GenesisPath string) error {
	hv2Genesis, err := utils.LoadJSONFromFile(hv2GenesisPath)
	if err != nil {
		return err
	}

	// ensure checkpoints data exist in both versions
	hv1CheckpointData, ok := hv1Genesis["app_state"].(map[string]interface{})["checkpoint"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("checkpoint module not found in v1 app_state")
	}

	hv2CheckpointData, ok := hv2Genesis["app_state"].(map[string]interface{})["checkpoint"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("checkpoint module not found in v2 app_state")
	}

	// ensure checkpoints data is in the correct format for both versions
	hv1Checkpoints, ok := hv1CheckpointData["checkpoints"].([]interface{})
	if !ok {
		return fmt.Errorf("checkpoints not found or invalid format in v1")
	}

	hv2Checkpoints, ok := hv2CheckpointData["checkpoints"].([]interface{})
	if !ok {
		return fmt.Errorf("checkpoints not found or invalid format in v2")
	}

	// ensure checkpoints count is the same in both versions
	if len(hv1Checkpoints) != len(hv2Checkpoints) {
		return fmt.Errorf("mismatch in checkpoints count: v1 has %d, v2 has %d", len(hv1Checkpoints), len(hv2Checkpoints))
	}

	// ensure ack_count is present in both versions
	hv1AckCountStr, ok := hv1CheckpointData["ack_count"].(string)
	if !ok {
		return fmt.Errorf("ack_count not found or invalid in v1")
	}

	hv2AckCountStr, ok := hv2CheckpointData["ack_count"].(string)
	if !ok {
		return fmt.Errorf("ack_count not found or invalid in v2")
	}

	// ensure ack_count is the same in both versions
	hv1AckCount, err := strconv.Atoi(hv1AckCountStr)
	if err != nil {
		return fmt.Errorf("failed to convert v1 ack_count to integer: %w", err)
	}

	hv2AckCount, err := strconv.Atoi(hv2AckCountStr)
	if err != nil {
		return fmt.Errorf("failed to convert v2 ack_count to integer: %w", err)
	}

	if hv1AckCount != hv2AckCount {
		return fmt.Errorf("mismatch in ack_count: v1 has %d, v2 has %d", hv1AckCount, hv2AckCount)
	}

	// ensure checkpoints are ordered by growing start_block in both versions
	for i := 1; i < len(hv1Checkpoints); i++ {
		hv1StartBlockPrev, _ := strconv.Atoi(hv1Checkpoints[i-1].(map[string]interface{})["start_block"].(string))
		hv1StartBlockCurr, _ := strconv.Atoi(hv1Checkpoints[i].(map[string]interface{})["start_block"].(string))
		if hv1StartBlockPrev >= hv1StartBlockCurr {
			return fmt.Errorf("checkpoints in v1 are not ordered by growing start_block at index %d", i)
		}
	}

	for i := 1; i < len(hv2Checkpoints); i++ {
		hv2StartBlockPrev, _ := strconv.Atoi(hv2Checkpoints[i-1].(map[string]interface{})["start_block"].(string))
		hv2StartBlockCurr, _ := strconv.Atoi(hv2Checkpoints[i].(map[string]interface{})["start_block"].(string))
		if hv2StartBlockPrev >= hv2StartBlockCurr {
			return fmt.Errorf("checkpoints in v2 are not ordered by growing start_block at index %d", i)
		}
	}

	// ensure IDs are sequential in v2 checkpoints
	for i := 0; i < len(hv2Checkpoints); i++ {
		id, _ := strconv.Atoi(hv2Checkpoints[i].(map[string]interface{})["id"].(string))
		if id != i+1 {
			return fmt.Errorf("checkpoints in v2 have non-sequential IDs at index %d", i)
		}
	}

	return nil
}

// getGenesisAppState reads the genesis file and returns the app state
func getGenesisAppState(hv2GenesisPath string) (heimdallApp.GenesisState, error) {
	genDoc, err := cmttypes.GenesisDocFromFile(hv2GenesisPath)
	if err != nil {
		return nil, err
	}

	var genesisState heimdallApp.GenesisState
	if err := json.Unmarshal(genDoc.AppState, &genesisState); err != nil {
		return nil, err
	}

	return genesisState, nil
}

// validatorBasicInfo contains the basic info of a validator
type validatorBasicInfo struct {
	power  int64
	signer string
	nonce  uint64
	jailed bool
}
