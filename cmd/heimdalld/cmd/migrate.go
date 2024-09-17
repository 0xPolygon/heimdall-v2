package heimdalld

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

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

	if err := renameModules(genesisData); err != nil {
		return err
	}

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

func renameModules(genesisData map[string]interface{}) error {
	logger.Info("Renaming modules...")

	// The custom staking module was renamed to stake in heimdallv2
	if err := renameProperty(genesisData, "app_state", "staking", "stake"); err != nil {
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
