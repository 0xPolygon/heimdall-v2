package heimdalld

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/cometbft/cometbft/crypto"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	cosmossecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cmdhelper "github.com/0xPolygon/heimdall-v2/cmd"
	"github.com/0xPolygon/heimdall-v2/helper"
	hmTypes "github.com/0xPolygon/heimdall-v2/types"
	stakingcli "github.com/0xPolygon/heimdall-v2/x/stake/client/cli"
	stakeTypes "github.com/0xPolygon/heimdall-v2/x/stake/types"
)

// TODO HV2 - this function was heavily modified, review carefully
// testnetCmd initialises files required to start heimdall testnet
func testnetCmd(_ *server.Context, cdc *codec.LegacyAmino, mbm module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-testnet",
		Short: "Initialize files for a Heimdall testnet",
		Long: `testnet will create "v" + "n" number of directories and populate each with
necessary files (private validator, genesis, config, etc.).

Note, strict routability for addresses is turned off in the config file.
Optionally, it will fill in persistent_peers list in config file using either hostnames or IPs.

Example:
testnet --v 4 --n 8 --output-dir ./output --starting-ip-address 192.168.10.2
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			cliCdc := clientCtx.Codec
			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config
			outDir := viper.GetString(flagOutputDir)

			// create chain id
			chainID := viper.GetString(flags.FlagChainID)
			if chainID == "" {
				suffix, err := cmdhelper.GenerateRandomString(6)
				if err != nil {
					return err
				}
				chainID = fmt.Sprintf("heimdall-%v", suffix)
			}

			// num of validators = validators in genesis files
			numValidators := viper.GetInt(flagNumValidators)

			// get total number of validators to be generated
			totalValidators := getTotalNumberOfNodes()

			// first validators start ID
			// there is no validator with id = 0
			startID := viper.GetInt64(stakingcli.FlagValidatorID)
			if startID == 0 {
				startID = 1
			}

			// signers data to dump in the signer-dump file
			signers := make([]ValidatorAccountFormatter, totalValidators)

			// Initialise variables for all validators
			nodeIDs := make([]string, totalValidators)
			valPubKeys := make([]crypto.PubKey, totalValidators)
			privKeys := make([]crypto.PrivKey, totalValidators)
			validators := make([]*stakeTypes.Validator, numValidators)
			dividendAccounts := make([]hmTypes.DividendAccount, numValidators)
			genFiles := make([]string, totalValidators)
			var err error

			nodeDaemonHomeName := viper.GetString(flagNodeDaemonHome)
			nodeCliHomeName := viper.GetString(flagNodeCliHome)

			for i := 0; i < totalValidators; i++ {
				// get node dir name = PREFIX+INDEX
				nodeDirName := fmt.Sprintf("%s%d", viper.GetString(flagNodeDirPrefix), i)

				// generate node and client dir
				nodeDir := filepath.Join(outDir, nodeDirName, nodeDaemonHomeName)
				clientDir := filepath.Join(outDir, nodeDirName, nodeCliHomeName)

				// set root in config
				config.SetRoot(nodeDir)

				// create config folder
				err := os.MkdirAll(filepath.Join(nodeDir, ""), nodeDirPerm)
				if err != nil {
					_ = os.RemoveAll(outDir)
					return err
				}

				err = os.MkdirAll(clientDir, nodeDirPerm)
				if err != nil {
					_ = os.RemoveAll(outDir)
					return err
				}

				nodeIDs[i], valPubKeys[i], privKeys[i], err = InitializeNodeValidatorFiles(config)
				if err != nil {
					return err
				}

				genFiles[i] = config.GenesisFile()

				cosmosPrivKey := &cosmossecp256k1.PrivKey{Key: privKeys[i].Bytes()}

				if i < numValidators {
					// create validator account
					validators[i], err = stakeTypes.NewValidator(
						uint64(startID+int64(i)),
						0,
						0,
						1,
						10000,
						cosmosPrivKey.PubKey(),
						valPubKeys[i].Address().String(),
					)
					if err != nil {
						return err
					}

					// create dividend account for validator
					dividendAccounts[i] = hmTypes.DividendAccount{
						User:      validators[i].Signer,
						FeeAmount: big.NewInt(0).String(),
					}
				}

				signers[i] = GetSignerInfo(valPubKeys[i], privKeys[i].Bytes(), cdc)

				WriteDefaultHeimdallConfig(filepath.Join(config.RootDir, "heimdall-config.toml"), helper.GetDefaultHeimdallConfig())
			}

			// other data
			for i := 0; i < totalValidators; i++ {
				populatePersistentPeersInConfigAndWriteIt(config)
			}

			appGenState := mbm.DefaultGenesis(cliCdc)

			appState, err := json.MarshalIndent(appGenState, "", " ")
			if err != nil {
				return err
			}

			// app state json
			appStateJSON, err := json.Marshal(appState)
			if err != nil {
				return err
			}

			for i := 0; i < totalValidators; i++ {
				if err = genutil.ExportGenesisFileWithTime(config.GenesisFile(), chainID, nil, appStateJSON, cmttime.Now()); err != nil {
					return err
				}
			}

			// TODO HV2 - do we need this?
			/*
				// dump signer information in a json file
				// TODO move to const string flag
				dump := viper.GetBool("signer-dump")
				if dump {
					signerJSON, err := json.MarshalIndent(signers, "", "  ")
					if err != nil {
						return err
					}

					if err := common.WriteFileAtomic(filepath.Join(outDir, "signer-dump.json"), signerJSON, 0600); err != nil {
						fmt.Println("Error writing signer-dump", err)
						return err
					}
				}
			*/

			fmt.Printf("Successfully initialized %d node directories\n", totalValidators)
			return nil
		},
	}

	cmd.Flags().Int(flagNumValidators, 4,
		"Number of validators to initialize the testnet with",
	)

	cmd.Flags().Int(flagNumNonValidators, 8,
		"Number of non-validators to initialize the testnet with",
	)

	cmd.Flags().StringP(flagOutputDir, "o", "./mytestnet",
		"Directory to store initialization data for the testnet",
	)

	cmd.Flags().String(flagNodeDirPrefix, "node",
		"Prefix the directory name for each node with (node results in node0, node1, ...)",
	)

	cmd.Flags().String(flagNodeDaemonHome, "heimdalld",
		"Home directory of the node's daemon configuration",
	)

	cmd.Flags().String(flagNodeHostPrefix, "node",
		"Hostname prefix (node results in persistent peers list ID0@node0:26656, ID1@node1:26656, ...)")

	cmd.Flags().String(flags.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")
	cmd.Flags().Bool("signer-dump", true, "dumps all signer information in a json file")

	return cmd
}
