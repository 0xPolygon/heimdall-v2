package heimdalld

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/state"
	blockidxkv "github.com/cometbft/cometbft/state/indexer/block/kv"
	"github.com/cometbft/cometbft/state/txindex/kv"
	"github.com/cometbft/cometbft/store"
	"github.com/spf13/cobra"

	"github.com/0xPolygon/heimdall-v2/helper"
)

func PruneCmd() *cobra.Command {
	var (
		height        int64
		dataDir       string
		dbBackend     string
		pruneIndexers bool
	)

	cmd := &cobra.Command{
		Use:   "prune-comet",
		Short: "Prune CometBFT block store up to a specified height",
		Long:  `Prunes the CometBFT block store up to the given height.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if height <= 0 {
				return fmt.Errorf("invalid --height %d (must be > 0)", height)
			}

			// Apply default data dir if empty: $HOME/.cometbft/data
			if dataDir == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to resolve home directory: %w", err)
				}
				dataDir = filepath.Join(homeDir, ".cometbft", "data")
			}
			logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

			// Report parsed values (placeholder; no pruning logic)
			logger.Info("Parsed prune-blocks parameters",
				"height", height,
				"data_dir", dataDir,
				"db_backend", dbBackend,
				"prune_indexers", pruneIndexers,
			)

			// Check if databases exist
			blockstoreDBPath := filepath.Join(dataDir, "blockstore.db")
			stateDBPath := filepath.Join(dataDir, "state.db")

			if _, err := os.Stat(blockstoreDBPath); os.IsNotExist(err) {
				_, _ = fmt.Fprintf(os.Stderr, "Error: blockstore.db not found in %s\n", dataDir)
				os.Exit(1)
			}

			if _, err := os.Stat(stateDBPath); os.IsNotExist(err) {
				_, _ = fmt.Fprintf(os.Stderr, "Error: state.db not found in %s\n", dataDir)
				os.Exit(1)
			}

			// Open blockstore database
			logger.Info("Opening blockstore database", "path", blockstoreDBPath)
			dbType := dbm.BackendType(dbBackend)
			blockStoreDB, err := dbm.NewDB("blockstore", dbType, dataDir)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: failed to open blockstore database: %v\n", err)
				os.Exit(1)
			}
			defer func() {
				if err := blockStoreDB.Close(); err != nil {
					logger.Error("Failed to close blockstore database", "err", err)
				} else {
					logger.Info("blockstore database closed")
				}
			}()

			blockStore := store.NewBlockStore(blockStoreDB)
			// Open state database
			logger.Info("Opening state database", "path", stateDBPath)
			stateDB, err := dbm.NewDB("state", dbType, dataDir)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: failed to open state database: %v\n", err)
				os.Exit(1)
			}
			defer func() {
				if err := stateDB.Close(); err != nil {
					logger.Error("Failed to close state database", "err", err)
				} else {
					logger.Info("State database closed")
				}
			}()

			stateStore := state.NewStore(stateDB, state.StoreOptions{})

			var txIndexDB dbm.DB
			var txIndexer *kv.TxIndex
			var blockIndexer *blockidxkv.BlockerIndexer
			var wg sync.WaitGroup

			// Only open the tx_index database if pruning indexers
			if pruneIndexers {
				logger.Info("Opening tx_index database")
				txIndexDB, err = dbm.NewDB("tx_index", dbType, dataDir)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "Error: failed to open tx_index database: %v\n", err)
					os.Exit(1)
				}
				defer func() {
					if err := txIndexDB.Close(); err != nil {
						logger.Error("Failed to close tx_index database", "err", err)
					} else {
						logger.Info("tx_index database closed")
					}
				}()

				// Create indexers
				txIndexer = kv.NewTxIndex(txIndexDB)
				blockIndexer = blockidxkv.New(dbm.NewPrefixDB(txIndexDB, []byte("block_events")))
			}

			// Get current block store information
			base := blockStore.Base()
			currentHeight := blockStore.Height()

			logger.Info("Block store information",
				"base", base,
				"height", currentHeight,
				"requestedRetainHeight", height,
			)

			// Validate height
			if height <= base {
				_, _ = fmt.Fprintf(os.Stderr, "Error: requested height %d is less than or equal to current base height %d\n", height, base)
				os.Exit(1)
			}

			if height > currentHeight {
				_, _ = fmt.Fprintf(os.Stderr, "Error: requested height %d is greater than current height %d\n", height, currentHeight)
				os.Exit(1)
			}

			if (currentHeight - height) < helper.EnforcedMinRetainBlocks {
				_, _ = fmt.Fprintf(os.Stderr, "Error: requested distance between current height %d and height %d is too short (%d). It must be at least %d\n", currentHeight, height, currentHeight-height, helper.EnforcedMinRetainBlocks)
				os.Exit(1)
			}

			// Create a Pruner instance
			var pruner *state.Pruner
			if pruneIndexers {
				pruner = state.NewPruner(
					stateStore,
					blockStore,
					blockIndexer,
					txIndexer,
					logger,
					state.WithIndexerPruning(true),
				)
			} else {
				pruner = state.NewPruner(
					stateStore,
					blockStore,
					nil,
					nil,
					logger,
				)
			}

			// Get the current application retain height
			currentRetainHeight, err := pruner.GetApplicationRetainHeight()
			if err != nil {
				logger.Info("No existing application retain height found", "err", err)
				currentRetainHeight = 0
			}
			logger.Info("Current application retain height", "height", currentRetainHeight)

			// Set the application retain height to the requested height
			logger.Info("Setting application retain height", "height", height)
			if err := pruner.SetApplicationBlockRetainHeight(height); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: failed to set application retain height: %v\n", err)
				os.Exit(1)
			}

			// Load current state
			logger.Info("Loading current state")
			st, err := stateStore.Load()
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: failed to load state: %v\n", err)
				os.Exit(1)
			}

			// Perform pruning using the blockStore directly
			// (pruneBlocksToRetainHeight is private, so we call the underlying method)
			logger.Info("Starting block pruning", "pruneToHeight", height)
			pruned, evidenceRetainHeight, err := blockStore.PruneBlocks(height, st)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error: failed to prune blocks: %v\n", err)
				os.Exit(1)
			}

			logger.Info("Block pruning completed",
				"pruned", pruned,
				"evidenceRetainHeight", evidenceRetainHeight,
			)

			// Prune states
			if pruned > 0 {
				logger.Info("Pruning states", "from", base, "to", height, "evidenceRetainHeight", evidenceRetainHeight)
				prunedStates, err := stateStore.PruneStates(base, height, evidenceRetainHeight)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "Error: failed to prune states: %v\n", err)
					os.Exit(1)
				}
				logger.Info("State pruning completed", "prunedStates", prunedStates)

				compactDBAsync(&wg, logger, "pruned blockstore", blockStoreDB)
				compactDBAsync(&wg, logger, "pruned statestore", stateDB)
			}

			// Get new base height
			newBase := blockStore.Base()
			logger.Info("Pruning successful",
				"blocksRemoved", pruned,
				"oldBase", base,
				"newBase", newBase,
				"currentHeight", currentHeight,
			)

			// Prune indexers if requested
			var txIndexerPruned, blockIndexerPruned int64

			if pruneIndexers {
				// Set tx indexer retain height
				logger.Info("Setting tx indexer retain height", "height", height)
				if err := pruner.SetTxIndexerRetainHeight(height); err != nil {
					logger.Error("Failed to set tx indexer retain height", "err", err)
				} else {
					// Prune tx indexer
					logger.Info("Pruning tx indexer", "targetHeight", height)
					currentTxRetainHeight, err := pruner.GetTxIndexerRetainHeight()
					if err != nil {
						logger.Error("Failed to get tx indexer retain height", "err", err)
					} else {
						txIndexerPruned, _, err = txIndexer.Prune(currentTxRetainHeight)
						if err != nil {
							logger.Error("Failed to prune tx indexer", "err", err)
						} else {
							logger.Info("Tx indexer pruning completed", "pruned", txIndexerPruned)
						}
					}
				}

				// Set block indexer retain height
				logger.Info("Setting block indexer retain height", "height", height)
				if err := pruner.SetBlockIndexerRetainHeight(height); err != nil {
					logger.Error("Failed to set block indexer retain height", "err", err)
				} else {
					// Prune block indexer
					logger.Info("Pruning block indexer", "targetHeight", height)
					currentBlockRetainHeight, err := pruner.GetBlockIndexerRetainHeight()
					if err != nil {
						logger.Error("Failed to get block indexer retain height", "err", err)
					} else {
						blockIndexerPruned, _, err = blockIndexer.Prune(currentBlockRetainHeight)
						if err != nil {
							logger.Error("Failed to prune block indexer", "err", err)
						} else {
							logger.Info("Block indexer pruning completed", "pruned", blockIndexerPruned)

							compactDBAsync(&wg, logger, "block indexer and txindexer", txIndexDB)
						}
					}
				}
			}

			wg.Wait()
			logger.Info("All compactions done")

			fmt.Printf("\n✓ Successfully pruned data:\n")
			fmt.Printf("  Blocks pruned: %d\n", pruned)
			if pruneIndexers {
				fmt.Printf("  Tx index entries pruned: %d\n", txIndexerPruned)
				fmt.Printf("  Block index entries pruned: %d\n", blockIndexerPruned)
			}
			fmt.Printf("  Old base height: %d\n", base)
			fmt.Printf("  New base height: %d\n", newBase)
			fmt.Printf("  Current height: %d\n", currentHeight)
			fmt.Printf("  Evidence retain height: %d\n", evidenceRetainHeight)
			fmt.Printf("  Application retain height set to: %d\n", height)
			return nil
		},
	}

	cmd.Flags().Int64Var(&height, "height", 0, "Height to prune up to (not including this height)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "Path to the data directory (default: $HOME/.cometbft/data)")
	cmd.Flags().StringVar(&dbBackend, "db-backend", "goleveldb", "Database backend type (goleveldb, cleveldb, boltdb, rocksdb, badgerdb)")
	cmd.Flags().BoolVar(&pruneIndexers, "prune-indexers", false, "Also prune tx and block indexers")

	return cmd
}

func compactDBAsync(wg *sync.WaitGroup, logger log.Logger, name string, db dbm.DB) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		logger.Info(fmt.Sprintf("Starting compaction of %s", name))
		start := time.Now()

		if err := db.Compact(nil, nil); err != nil {
			logger.Error(fmt.Sprintf("Compaction of %s failed", name),
				"error", err,
				"duration", time.Since(start),
			)
			return
		}

		logger.Info(fmt.Sprintf("Compaction of %s completed", name),
			"duration", time.Since(start),
		)
	}()
}
