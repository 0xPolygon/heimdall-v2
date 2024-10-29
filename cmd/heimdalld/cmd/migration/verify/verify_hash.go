package verify

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	logger "github.com/cometbft/cometbft/libs/log"
)

const (
	// REMOTE_GENESIS_HASH_URL is the URL where the precomputed genesis hash is stored in JSON format.
	REMOTE_GENESIS_HASH_URL = "http://localhost:8000/genesis-hash.json"
	// RetryDelay defines the delay between retries when fetching the remote hash fails.
	RetryDelay = 10 * time.Second
)

// VerifyMigratedGenesisHash verifies the SHA256 hash of the migrated genesis file against a remote precomputed hash.
// It computes the hash of the local file, fetches the precomputed hash from the remote JSON file, then compares the two hashes.
func VerifyMigratedGenesisHash(migratedGenesisFilePath string, logger logger.Logger) error {
	logger.Info("Generating migrated genesis hash...")

	localHash, err := computeFileHash(migratedGenesisFilePath)
	if err != nil {
		return fmt.Errorf("failed to compute local genesis hash: %w", err)
	}

	logger.Info(fmt.Sprintf("Migrated genesis hash: %s", localHash))

	for {
		remoteHash, err := fetchRemoteHash(REMOTE_GENESIS_HASH_URL)
		if err != nil {
			logger.Info("Failed to fetch remote genesis hash. Retrying...")
			logger.Info("If you wish to skip this automatic verification and verify the hash manually later, press Ctrl+C to stop the process.")
			time.Sleep(RetryDelay)
			continue
		}

		if remoteHash == localHash {
			logger.Info("Genesis hash verified successfully")
			break
		}

		return fmt.Errorf("genesis hash verification failed. Remote hash: %s, Local hash: %s", remoteHash, localHash)
	}

	return nil
}

// computeFileHash computes the SHA256 hash of the file at the given path.
// It reads the file and returns the hash as a hexadecimal string prefixed with '0x'.
func computeFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	hasher := sha256.New()

	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to hash file %s: %w", filePath, err)
	}

	checksum := fmt.Sprintf("0x%x", hasher.Sum(nil))

	return checksum, nil
}

// fetchRemoteHash fetches the precomputed genesis hash from the given URL.
// It expects the URL to return a JSON object with a "genesis_hash" field.
func fetchRemoteHash(url string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch remote hash: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
	}

	var result struct {
		GenesisHash string `json:"genesis_hash"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode JSON response: %w", err)
	}

	if result.GenesisHash == "" {
		return "", fmt.Errorf("genesis_hash not found in the response")
	}

	return result.GenesisHash, nil
}
