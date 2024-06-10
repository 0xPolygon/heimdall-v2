package helper

import (
	"testing"

	//authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	//"github.com/ethereum/go-ethereum/common"

	"github.com/0xPolygon/heimdall-v2/contracts/erc20"
	"github.com/0xPolygon/heimdall-v2/contracts/rootchain"
	"github.com/0xPolygon/heimdall-v2/contracts/slashmanager"
	"github.com/0xPolygon/heimdall-v2/contracts/stakemanager"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/contracts/statereceiver"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"

	//"github.com/0xPolygon/heimdall-v2/x/types"
	//authTypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/stretchr/testify/assert"
)

const (
	testCometBFTNode = "tcp://localhost:26657"
)

// TODO HV2 Please write this test once authtypes is finalised and RecoverPubKey is implemented inside
// TestCheckpointSigs decodes signers from checkpoint sigs data
// func TestCheckpointSigs(t *testing.T) {
// 	t.Parallel()

// 	viper.Set(CometBFTNodeFlag, testCometBFTNode)
// 	viper.Set("log_level", "info")
// 	InitHeimdallConfig(os.ExpandEnv("$HOME/.heimdalld"))

// 	contractCallerObj, err := NewContractCaller()
// 	if err != nil {
// 		t.Error("Error creating contract caller")
// 	}

// 	txHashStr := "0x9c2a9e20e1fecdae538f72b01dd0fd5008cc90176fd603b92b59274d754cbbd8"
// 	txHash := common.HexToHash(txHashStr)

// 	voteSignBytes, sigs, txData, err := contractCallerObj.GetCheckpointSign(txHash)
// 	if err != nil {
// 		t.Error("Error fetching checkpoint tx input args")
// 	}

// 	t.Log("checkpoint args", "vote", hex.EncodeToString(voteSignBytes), "sigs", hex.EncodeToString(sigs), "txData", hex.EncodeToString(txData))

// 	signerList, err := FetchSigners(voteSignBytes, sigs)
// 	if err != nil {
// 		t.Error("Error fetching signer list from tx input args")
// 	}

// 	t.Log("signers list", signerList)
// }

// // FetchSigners fetches the signers' list
// func FetchSigners(voteBytes []byte, sigInput []byte) ([]string, error) {
// 	const sigLength = 65

// 	signersList := make([]string, len(sigInput))

// 	// Calculate total stake Power of all Signers.
// 	for i := 0; i < len(sigInput); i += sigLength {
// 		signature := sigInput[i : i+sigLength]

// 		pKey, err := authTypes.RecoverPubkey(voteBytes, signature)
// 		if err != nil {
// 			return nil, err
// 		}

// 		signersList[i] = types.NewPubKey(pKey).Address().String()
// 	}

// 	return signersList, nil
// }

// TestPopulateABIs tests that package level ABIs cache works as expected
// by not invoking json methods after contracts ABIs' init
func TestPopulateABIs(t *testing.T) {

	t.Log("ABIs map should be empty and all ABIs not found")
	assert.True(t, len(ContractsABIsMap) == 0)
	_, found := ContractsABIsMap[rootchain.RootchainMetaData.ABI]
	assert.False(t, found)
	_, found = ContractsABIsMap[stakinginfo.StakinginfoMetaData.ABI]
	assert.False(t, found)
	_, found = ContractsABIsMap[statereceiver.StatereceiverMetaData.ABI]
	assert.False(t, found)
	_, found = ContractsABIsMap[statesender.StatesenderMetaData.ABI]
	assert.False(t, found)
	_, found = ContractsABIsMap[stakemanager.StakemanagerMetaData.ABI]
	assert.False(t, found)
	_, found = ContractsABIsMap[slashmanager.SlashmanagerMetaData.ABI]
	assert.False(t, found)
	_, found = ContractsABIsMap[erc20.Erc20MetaData.ABI]
	assert.False(t, found)

	t.Log("Should create a new contract caller and populate its ABIs by decoding json")

	contractCallerObjFirst, err := NewContractCaller()
	if err != nil {
		t.Error("Error creating contract caller")
	}

	assert.Equalf(t, ContractsABIsMap[rootchain.RootchainMetaData.ABI], &contractCallerObjFirst.RootChainABI,
		"values for %s not equals", rootchain.RootchainMetaData.ABI)
	assert.Equalf(t, ContractsABIsMap[stakinginfo.StakinginfoMetaData.ABI], &contractCallerObjFirst.StakingInfoABI,
		"values for %s not equals", stakinginfo.StakinginfoMetaData.ABI)
	assert.Equalf(t, ContractsABIsMap[statereceiver.StatereceiverMetaData.ABI], &contractCallerObjFirst.StateReceiverABI,
		"values for %s not equals", statereceiver.StatereceiverMetaData.ABI)
	assert.Equalf(t, ContractsABIsMap[statesender.StatesenderMetaData.ABI], &contractCallerObjFirst.StateSenderABI,
		"values for %s not equals", statesender.StatesenderMetaData.ABI)
	assert.Equalf(t, ContractsABIsMap[stakemanager.StakemanagerMetaData.ABI], &contractCallerObjFirst.StakeManagerABI,
		"values for %s not equals", stakemanager.StakemanagerMetaData.ABI)
	assert.Equalf(t, ContractsABIsMap[slashmanager.SlashmanagerMetaData.ABI], &contractCallerObjFirst.SlashManagerABI,
		"values for %s not equals", slashmanager.SlashmanagerMetaData.ABI)
	assert.Equalf(t, ContractsABIsMap[erc20.Erc20MetaData.ABI], &contractCallerObjFirst.MaticTokenABI,
		"values for %s not equals", erc20.Erc20MetaData.ABI)

	t.Log("ABIs map should not be empty and all ABIs found")
	assert.True(t, len(ContractsABIsMap) == 8)
	_, found = ContractsABIsMap[rootchain.RootchainMetaData.ABI]
	assert.True(t, found)
	_, found = ContractsABIsMap[stakinginfo.StakinginfoMetaData.ABI]
	assert.True(t, found)
	_, found = ContractsABIsMap[statereceiver.StatereceiverMetaData.ABI]
	assert.True(t, found)
	_, found = ContractsABIsMap[statesender.StatesenderMetaData.ABI]
	assert.True(t, found)
	_, found = ContractsABIsMap[stakemanager.StakemanagerMetaData.ABI]
	assert.True(t, found)
	_, found = ContractsABIsMap[slashmanager.SlashmanagerMetaData.ABI]
	assert.True(t, found)
	_, found = ContractsABIsMap[erc20.Erc20MetaData.ABI]
	assert.True(t, found)

	t.Log("Should create a new contract caller and populate its ABIs by using cached map")

	contractCallerObjSecond, err := NewContractCaller()
	if err != nil {
		t.Log("Error creating contract caller")
	}

	assert.Equalf(t, ContractsABIsMap[rootchain.RootchainMetaData.ABI], &contractCallerObjSecond.RootChainABI,
		"values for %s not equals", rootchain.RootchainMetaData.ABI)
	assert.Equalf(t, ContractsABIsMap[stakinginfo.StakinginfoMetaData.ABI], &contractCallerObjSecond.StakingInfoABI,
		"values for %s not equals", stakinginfo.StakinginfoMetaData.ABI)
	assert.Equalf(t, ContractsABIsMap[statereceiver.StatereceiverMetaData.ABI], &contractCallerObjSecond.StateReceiverABI,
		"values for %s not equals", statereceiver.StatereceiverMetaData.ABI)
	assert.Equalf(t, ContractsABIsMap[statesender.StatesenderMetaData.ABI], &contractCallerObjSecond.StateSenderABI,
		"values for %s not equals", statesender.StatesenderMetaData.ABI)
	assert.Equalf(t, ContractsABIsMap[stakemanager.StakemanagerMetaData.ABI], &contractCallerObjSecond.StakeManagerABI,
		"values for %s not equals", stakemanager.StakemanagerMetaData.ABI)
	assert.Equalf(t, ContractsABIsMap[slashmanager.SlashmanagerMetaData.ABI], &contractCallerObjSecond.SlashManagerABI,
		"values for %s not equals", slashmanager.SlashmanagerMetaData.ABI)
	assert.Equalf(t, ContractsABIsMap[erc20.Erc20MetaData.ABI], &contractCallerObjSecond.MaticTokenABI,
		"values for %s not equals", erc20.Erc20MetaData.ABI)
}
