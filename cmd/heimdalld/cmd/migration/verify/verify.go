// Package verify provides tools to validate that a Heimdall v1 genesis file has been properly
// migrated to Heimdall v2, by verifying balances, validators, bor spans, clerk events, and checkpoints.
package verify

import (
	"encoding/json"
	"fmt"
	"os"
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
	hmTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// RunMigrationVerification verifies a Heimdall v2 genesis file migrated from v1.
// It checks consistency of balances, validators, bor spans, clerk events, and checkpoints.
func RunMigrationVerification(hv1GenesisPath, hv2GenesisPath string, logger log.Logger) error {
	logger.Info("Verifying migration")

	db := dbm.NewMemDB()
	appOptions := make(simtestutil.AppOptionsMap)
	appOptions[flags.FlagHome] = heimdallApp.DefaultNodeHome

	app := heimdallApp.NewHeimdallApp(logger, db, nil, true, appOptions)
	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})

	logger.Info("Verifying data lists for bor and clerk")
	if err := verifyDataLists(hv1GenesisPath, hv2GenesisPath, logger); err != nil {
		return err
	}

	logger.Info("Verifying balances")
	if err := verifyBalances(ctx, app, hv1GenesisPath, logger); err != nil {
		return err
	}

	logger.Info("Verifying validators")
	if err := verifyValidators(ctx, app, hv1GenesisPath, logger); err != nil {
		return err
	}

	logger.Info("Verifying checkpoints")
	if err := verifyCheckpoints(hv1GenesisPath, hv2GenesisPath, logger); err != nil {
		return err
	}

	logger.Info("Verifying topup")
	if err := verifyTopup(hv1GenesisPath, hv2GenesisPath, logger); err != nil {
		return err
	}

	logger.Info("Verifying governance proposals")
	if err := verifyGov(hv1GenesisPath, hv2GenesisPath, logger); err != nil {
		return err
	}

	genesisState, err := getGenesisAppState(hv2GenesisPath)
	if err != nil {
		return err
	}

	logger.Info("Initializing genesis state")
	if _, err := app.ModuleManager.InitGenesis(ctx, app.AppCodec(), genesisState); err != nil {
		return err
	}

	logger.Info("Migration verified successfully")
	return nil
}

// streamToAppState streams a large genesis JSON file and extracts only the app_state section.
func streamToAppState(filePath string) (map[string]json.RawMessage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("failed to close file: %v\n", err)
		}
	}(file)

	decoder := json.NewDecoder(file)
	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("token stream error: %w", err)
		}
		if key, ok := tok.(string); ok && key == "app_state" {
			break
		}
	}

	var appState map[string]json.RawMessage
	if err := decoder.Decode(&appState); err != nil {
		return nil, fmt.Errorf("failed to decode app_state: %w", err)
	}
	return appState, nil
}

// getGenesisAppState loads app_state from a full genesis file using standard Cosmos SDK GenesisDoc.
func getGenesisAppState(path string) (heimdallApp.GenesisState, error) {
	doc, err := cmttypes.GenesisDocFromFile(path)
	if err != nil {
		return nil, err
	}
	var state heimdallApp.GenesisState
	if err := json.Unmarshal(doc.AppState, &state); err != nil {
		return nil, err
	}
	return state, nil
}

// validatorBasicInfo represents simplified validator state used for comparison.
type validatorBasicInfo struct {
	signer string
	power  int64
	nonce  uint64
	jailed bool
}

// verifyBalances streams and validates that all account balances in the genesis match what's in the DB.
func verifyBalances(ctx types.Context, app *heimdallApp.HeimdallApp, hv1GenesisPath string, logger log.Logger) error {
	appState, err := streamToAppState(hv1GenesisPath)
	if err != nil {
		return fmt.Errorf("failed to stream hv1 genesis: %w", err)
	}
	authRaw, ok := appState["auth"]
	if !ok {
		return fmt.Errorf("auth module not found in app_state")
	}

	type baseAccount struct {
		Address string `json:"address"`
		Coins   []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"coins"`
	}

	var accounts []json.RawMessage
	if err := json.Unmarshal(authRaw, &map[string]interface{}{"accounts": &accounts}); err != nil {
		return fmt.Errorf("failed to extract accounts: %w", err)
	}

	addressCoins := make(map[string]types.Coins)
	for i, raw := range accounts {
		var acc baseAccount
		if err := json.Unmarshal(raw, &acc); err != nil {
			return fmt.Errorf("failed to unmarshal account at %d: %w", i, err)
		}

		for _, coin := range acc.Coins {
			amount, ok := math.NewIntFromString(coin.Amount)
			if !ok {
				return fmt.Errorf("invalid amount '%s' for address %s", coin.Amount, acc.Address)
			}
			addressCoins[acc.Address] = append(addressCoins[acc.Address], types.NewCoin(coin.Denom, amount))
		}
	}

	for _, balance := range app.BankKeeper.GetAccountsBalances(ctx) {
		expected, ok := addressCoins[balance.Address]
		if !ok {
			logger.Error("missing balance", "address", balance.Address)
			return fmt.Errorf("balance not found for address %s", balance.Address)
		}
		for _, coin := range expected {
			actual := balance.Coins.AmountOf(coin.Denom)
			if !actual.Equal(coin.Amount) {
				logger.Error("balance mismatch", "address", balance.Address, "denom", coin.Denom, "expected", coin.Amount.String(), "got", actual.String())
				return fmt.Errorf("balance mismatch for %s in %s", balance.Address, coin.Denom)
			}
		}
	}
	logger.Info("All balances match")
	return nil
}

// verifyValidators streams and validates validator records in the genesis against DB values.
func verifyValidators(ctx types.Context, app *heimdallApp.HeimdallApp, hv1GenesisPath string, logger log.Logger) error {
	appState, err := streamToAppState(hv1GenesisPath)
	if err != nil {
		return err
	}
	stakingRaw, ok := appState["staking"]
	if !ok {
		return fmt.Errorf("staking module not found")
	}
	var staking map[string]json.RawMessage
	if err := json.Unmarshal(stakingRaw, &staking); err != nil {
		return fmt.Errorf("failed to parse staking module: %w", err)
	}

	// Try current_val_set first
	var currentValSet struct {
		Validators []json.RawMessage `json:"validators"`
	}
	if raw, ok := staking["current_val_set"]; ok {
		if err := json.Unmarshal(raw, &currentValSet); err != nil {
			return fmt.Errorf("invalid current_val_set: %w", err)
		}
		if len(currentValSet.Validators) > 0 {
			return compareValidatorsSet(currentValSet.Validators, app.StakeKeeper.GetCurrentValidators(ctx), logger)
		}
	}

	// Else fall back to validators
	var all []json.RawMessage
	if raw, ok := staking["validators"]; ok {
		if err := json.Unmarshal(raw, &all); err != nil {
			return fmt.Errorf("invalid validators array: %w", err)
		}
		ptrs := make([]hmTypes.Validator, 0, len(app.StakeKeeper.GetAllValidators(ctx)))
		for _, v := range app.StakeKeeper.GetAllValidators(ctx) {
			if v != nil {
				ptrs = append(ptrs, *v)
			}
		}
		return compareValidatorsSet(all, ptrs, logger)
	}
	return fmt.Errorf("no validators found")
}

// compareValidatorsSet matches a list of genesis validators with the DB validator set.
func compareValidatorsSet(genesisVals []json.RawMessage, dbVals []hmTypes.Validator, logger log.Logger) error {
	dbMap := make(map[string]hmTypes.Validator)
	for _, val := range dbVals {
		dbMap[val.Signer] = val
	}

	for i, raw := range genesisVals {
		info, err := getValidatorBasicInfoFromRaw(raw)
		if err != nil {
			return fmt.Errorf("validator %d parse error: %w", i, err)
		}
		dbVal, ok := dbMap[info.signer]
		if !ok {
			logger.Error("validator not found in DB", "signer", info.signer)
			return fmt.Errorf("validator %s not found in DB", info.signer)
		}
		if err := compareValidators(dbVal, info); err != nil {
			logger.Error("validator mismatch", "signer", info.signer, "error", err.Error())
			return err
		}
	}
	logger.Info("All validators match")
	return nil
}

// getValidatorBasicInfoFromRaw extracts validator info from a RawMessage.
func getValidatorBasicInfoFromRaw(raw json.RawMessage) (*validatorBasicInfo, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	signer, _ := m["signer"].(string)
	powerStr, _ := m["power"].(string)
	nonceStr, _ := m["nonce"].(string)
	jailed, _ := m["jailed"].(bool)

	power, err := strconv.ParseInt(powerStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid power: %w", err)
	}
	nonce, err := strconv.ParseUint(nonceStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid nonce: %w", err)
	}

	return &validatorBasicInfo{signer, power, nonce, jailed}, nil
}

// verifyCheckpoints streams and validates the checkpoint list, count, and order between v1 and v2 genesis.
func verifyCheckpoints(hv1Path, hv2Path string, logger log.Logger) error {
	hv1App, err := streamToAppState(hv1Path)
	if err != nil {
		return err
	}
	hv2App, err := streamToAppState(hv2Path)
	if err != nil {
		return err
	}

	hv1CP, hv2CP := hv1App["checkpoint"], hv2App["checkpoint"]
	var m1, m2 map[string]interface{}
	if err := json.Unmarshal(hv1CP, &m1); err != nil {
		return err
	}
	if err := json.Unmarshal(hv2CP, &m2); err != nil {
		return err
	}

	hv1List := m1["checkpoints"].([]interface{})
	hv2List := m2["checkpoints"].([]interface{})

	if len(hv1List) != len(hv2List) {
		logger.Error("checkpoint count mismatch", "v1", len(hv1List), "v2", len(hv2List))
		return fmt.Errorf("checkpoint count mismatch")
	}

	hv1Ack, _ := strconv.Atoi(m1["ack_count"].(string))
	hv2Ack, _ := strconv.Atoi(m2["ack_count"].(string))
	if hv1Ack != hv2Ack {
		logger.Error("ack_count mismatch", "v1", hv1Ack, "v2", hv2Ack)
		return fmt.Errorf("ack_count mismatch")
	}

	for i := 1; i < len(hv1List); i++ {
		prev := hv1List[i-1].(map[string]interface{})
		curr := hv1List[i].(map[string]interface{})
		pSB, _ := strconv.Atoi(prev["start_block"].(string))
		cSB, _ := strconv.Atoi(curr["start_block"].(string))
		if pSB >= cSB {
			logger.Error("v1 checkpoint start_block disorder", "index", i)
			return fmt.Errorf("v1 checkpoint start_block disorder at %d", i)
		}
	}

	for i, cp := range hv2List {
		entry := cp.(map[string]interface{})
		id, _ := strconv.Atoi(entry["id"].(string))
		if id != i+1 {
			logger.Error("v2 checkpoint ID not sequential", "index", i, "id", id)
			return fmt.Errorf("v2 checkpoint ID not sequential at %d", i)
		}
		if i > 0 {
			prev := hv2List[i-1].(map[string]interface{})
			pSB, _ := strconv.Atoi(prev["start_block"].(string))
			cSB, _ := strconv.Atoi(entry["start_block"].(string))
			if pSB >= cSB {
				logger.Error("v2 checkpoint start_block disorder", "index", i)
				return fmt.Errorf("v2 checkpoint start_block disorder at %d", i)
			}
		}
	}

	logger.Info("All checkpoints verified")
	return nil
}

// verifyDataLists checks count equality of bor spans and clerk event records between v1 and v2 genesis.
func verifyDataLists(hv1Path, hv2Path string, logger log.Logger) error {
	hv1, err := streamToAppState(hv1Path)
	if err != nil {
		return err
	}
	hv2, err := streamToAppState(hv2Path)
	if err != nil {
		return err
	}

	if err := compareListLength(hv1, hv2, "bor", "spans", logger); err != nil {
		return err
	}
	if err := compareListLength(hv1, hv2, "clerk", "event_records", logger); err != nil {
		return err
	}
	logger.Info("All bor spans and clerk events verified")
	return nil
}

// compareListLength verifies equal array length for a given module and key between two app states.
func compareListLength(hv1, hv2 map[string]json.RawMessage, module, key string, logger log.Logger) error {
	getCount := func(app map[string]json.RawMessage) (int, error) {
		modRaw, ok := app[module]
		if !ok {
			return 0, fmt.Errorf("module %s missing", module)
		}
		var mod map[string]json.RawMessage
		if err := json.Unmarshal(modRaw, &mod); err != nil {
			return 0, err
		}
		listRaw, ok := mod[key]
		if !ok {
			return 0, fmt.Errorf("key %s missing in %s", key, module)
		}
		var list []json.RawMessage
		if err := json.Unmarshal(listRaw, &list); err != nil {
			return 0, err
		}
		return len(list), nil
	}

	n1, err := getCount(hv1)
	if err != nil {
		return err
	}
	n2, err := getCount(hv2)
	if err != nil {
		return err
	}
	if n1 != n2 {
		logger.Error("mismatched count in module", "module", module, "key", key, "v1", n1, "v2", n2)
		return fmt.Errorf("%s.%s count mismatch", module, key)
	}
	return nil
}

// compareValidators checks if the fields of a validator in the DB match those in the genesis.
func compareValidators(db hmTypes.Validator, g *validatorBasicInfo) error {
	if db.Signer != g.signer || db.VotingPower != g.power || db.Nonce != g.nonce || db.Jailed != g.jailed {
		return fmt.Errorf("validator fields mismatch")
	}
	return nil
}

// verifyTopup compares the topup module between two genesis files,
// checking that topup_sequences (v2) and tx_sequences (v1) have same count,
// and that all dividend_accounts match by user and feeAmount.
func verifyTopup(hv1GenesisPath, hv2GenesisPath string, logger log.Logger) error {
	// Load app_state
	appState1, err := streamToAppState(hv1GenesisPath)
	if err != nil {
		return fmt.Errorf("failed to stream hv1 genesis: %w", err)
	}
	appState2, err := streamToAppState(hv2GenesisPath)
	if err != nil {
		return fmt.Errorf("failed to stream hv2 genesis: %w", err)
	}

	topupV1Raw, ok1 := appState1["topup"]
	topupV2Raw, ok2 := appState2["topup"]
	if !ok1 || !ok2 {
		return fmt.Errorf("topup module not found in one of the genesis files")
	}

	var topupV1, topupV2 struct {
		TxSequences      []string `json:"tx_sequences"`    // v1
		TopupSequences   []string `json:"topup_sequences"` // v2
		DividendAccounts []struct {
			User      string `json:"user"`
			FeeAmount string `json:"feeAmount"`
		} `json:"dividend_accounts"`
	}

	if err := json.Unmarshal(topupV1Raw, &topupV1); err != nil {
		return fmt.Errorf("failed to parse topup module in v1: %w", err)
	}
	if err := json.Unmarshal(topupV2Raw, &topupV2); err != nil {
		return fmt.Errorf("failed to parse topup module in v2: %w", err)
	}

	// Check tx_sequences vs topup_sequences count
	if len(topupV1.TxSequences) != len(topupV2.TopupSequences) {
		logger.Error("Mismatch in topup/tx sequence count",
			"v1", len(topupV1.TxSequences),
			"v2", len(topupV2.TopupSequences),
		)
		return fmt.Errorf("topup sequence count mismatch: v1 has %d, v2 has %d",
			len(topupV1.TxSequences), len(topupV2.TopupSequences))
	}

	// Check dividend_accounts count
	if len(topupV1.DividendAccounts) != len(topupV2.DividendAccounts) {
		logger.Error("Mismatch in dividend accounts",
			"v1", len(topupV1.DividendAccounts),
			"v2", len(topupV2.DividendAccounts),
		)
		return fmt.Errorf("dividend account count mismatch: v1 has %d, v2 has %d",
			len(topupV1.DividendAccounts), len(topupV2.DividendAccounts))
	}

	// Check individual dividend accounts by index
	for i, a1 := range topupV1.DividendAccounts {
		a2 := topupV2.DividendAccounts[i]
		if a1.User != a2.User || a1.FeeAmount != a2.FeeAmount {
			logger.Error("Mismatch in dividend account",
				"index", i,
				"user_v1", a1.User, "user_v2", a2.User,
				"fee_v1", a1.FeeAmount, "fee_v2", a2.FeeAmount,
			)
			return fmt.Errorf("dividend account mismatch at index %d", i)
		}
	}

	logger.Info("Topup module verified successfully")
	return nil
}

// verifyGov ensures that the number of governance proposals matches between v1 and v2 genesis files.
func verifyGov(hv1GenesisPath, hv2GenesisPath string, logger log.Logger) error {
	appState1, err := streamToAppState(hv1GenesisPath)
	if err != nil {
		return fmt.Errorf("failed to stream hv1 genesis: %w", err)
	}
	appState2, err := streamToAppState(hv2GenesisPath)
	if err != nil {
		return fmt.Errorf("failed to stream hv2 genesis: %w", err)
	}

	govV1Raw, ok1 := appState1["gov"]
	govV2Raw, ok2 := appState2["gov"]
	if !ok1 || !ok2 {
		return fmt.Errorf("gov module not found in one of the genesis files")
	}

	var govV1, govV2 struct {
		Proposals []json.RawMessage `json:"proposals"`
	}

	if err := json.Unmarshal(govV1Raw, &govV1); err != nil {
		return fmt.Errorf("failed to parse gov module in v1: %w", err)
	}
	if err := json.Unmarshal(govV2Raw, &govV2); err != nil {
		return fmt.Errorf("failed to parse gov module in v2: %w", err)
	}

	if len(govV1.Proposals) != len(govV2.Proposals) {
		logger.Error("Mismatch in number of governance proposals",
			"v1", len(govV1.Proposals), "v2", len(govV2.Proposals),
		)
		return fmt.Errorf("governance proposal count mismatch: v1 has %d, v2 has %d",
			len(govV1.Proposals), len(govV2.Proposals))
	}

	logger.Info("Governance proposal count verified successfully")
	return nil
}
