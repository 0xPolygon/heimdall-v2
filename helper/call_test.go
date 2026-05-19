package helper

import (
	"encoding/hex"
	"errors"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/contracts/erc20"
	"github.com/0xPolygon/heimdall-v2/contracts/rootchain"
	"github.com/0xPolygon/heimdall-v2/contracts/slashmanager"
	"github.com/0xPolygon/heimdall-v2/contracts/stakemanager"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/contracts/statereceiver"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
)

// TestCheckpointSigs tests signature recovery with checkpoint data
func TestCheckpointSigs(t *testing.T) {
	t.Parallel()

	// Create test data with multiple signers
	numSigners := 3
	privKeys := make([]*secp256k1.PrivKey, numSigners)
	expectedAddresses := make([]string, numSigners)

	// Generate private keys and expected addresses
	for i := 0; i < numSigners; i++ {
		privKeys[i] = secp256k1.GenPrivKey()
		expectedAddresses[i] = hex.EncodeToString(privKeys[i].PubKey().Address().Bytes())
	}

	// Message that all signers will sign (checkpoint data)
	checkpointData := []byte("test checkpoint data for signing")

	// Collect signatures
	var allSigs []byte
	for i := 0; i < numSigners; i++ {
		sig, err := privKeys[i].Sign(checkpointData)
		require.NoError(t, err, "Error signing checkpoint data")
		allSigs = append(allSigs, sig...)
	}

	t.Log("Checkpoint data:", hex.EncodeToString(checkpointData))
	t.Log("Combined signatures:", hex.EncodeToString(allSigs))
	t.Log("Signatures count:", len(allSigs)/65)

	// Test FetchSigners function
	signerList, err := FetchSigners(checkpointData, allSigs)
	require.NoError(t, err, "Error fetching signer list")
	require.Len(t, signerList, numSigners, "Incorrect number of signers recovered")

	// Verify each recovered signer matches the expected address
	for i := 0; i < numSigners; i++ {
		t.Logf("Signer %d - Expected: %s, Recovered: %s", i, expectedAddresses[i], signerList[i])
		require.Equal(t, expectedAddresses[i], signerList[i], "Signer address mismatch at index %d", i)
	}

	t.Log("All signers successfully verified")
}

// TestCheckpointSigsWithInvalidData tests error handling
func TestCheckpointSigsWithInvalidData(t *testing.T) {
	t.Parallel()

	checkpointData := []byte("test data")

	// Test with empty signatures
	_, err := FetchSigners(checkpointData, []byte{})
	require.NoError(t, err, "Should handle empty signatures")

	// Test with incomplete signature (less than 65 bytes)
	incompleteSig := make([]byte, 32)
	_, err = FetchSigners(checkpointData, incompleteSig)
	require.Error(t, err, "Should error on incomplete signature")
}

// FetchSigners fetches the signers' list
func FetchSigners(voteBytes []byte, sigInput []byte) ([]string, error) {
	const sigLength = 65

	if len(sigInput)%sigLength != 0 {
		return nil, errors.New("invalid signature length")
	}

	numSigners := len(sigInput) / sigLength
	signersList := make([]string, numSigners)

	// Recover public key and address for each signature
	for i := 0; i < numSigners; i++ {
		sigStart := i * sigLength
		sigEnd := sigStart + sigLength
		signature := sigInput[sigStart:sigEnd]

		pKey, err := signing.RecoverPubKey(voteBytes, signature)
		if err != nil {
			return nil, err
		}

		pk := secp256k1.PubKey{Key: pKey}
		signersList[i] = hex.EncodeToString(pk.Address().Bytes())
	}
	return signersList, nil
}

// TestPopulateABIs tests that package level ABIs cache works as expected
// by not invoking JSON methods after contracts ABIs' init
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

	assert.NotNil(t, &contractCallerObjFirst)
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
	assert.Equalf(t, ContractsABIsMap[erc20.Erc20MetaData.ABI], &contractCallerObjFirst.PolTokenABI,
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
	assert.NotNil(t, &contractCallerObjSecond)

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
	assert.Equalf(t, ContractsABIsMap[erc20.Erc20MetaData.ABI], &contractCallerObjSecond.PolTokenABI,
		"values for %s not equals", erc20.Erc20MetaData.ABI)
}
