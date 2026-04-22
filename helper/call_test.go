package helper

import (
	"encoding/hex"
	"errors"
	"math/big"
	"sync"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
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

func makeHeader(blockNum uint64) *ethTypes.Header {
	return &ethTypes.Header{Number: new(big.Int).SetUint64(blockNum)}
}

func TestGetCachedFinalizedBlockNumber_EmptyCache(t *testing.T) {
	t.Parallel()

	cc := &ContractCaller{finalizedBlockCache: &finalizedBlockCache{}}
	assert.Equal(t, uint64(0), cc.getCachedFinalizedBlockNumber())
}

func TestGetCachedFinalizedBlockNumber_NilCachePointer(t *testing.T) {
	t.Parallel()

	cc := &ContractCaller{finalizedBlockCache: nil}
	assert.Equal(t, uint64(0), cc.getCachedFinalizedBlockNumber())
}

func TestUpdateFinalizedBlockCache_SetsValue(t *testing.T) {
	t.Parallel()

	cc := &ContractCaller{finalizedBlockCache: &finalizedBlockCache{}}

	cc.updateFinalizedBlockCache(makeHeader(1000))
	assert.Equal(t, uint64(1000), cc.getCachedFinalizedBlockNumber())
}

func TestUpdateFinalizedBlockCache_MonotonicIncrease(t *testing.T) {
	t.Parallel()

	cc := &ContractCaller{finalizedBlockCache: &finalizedBlockCache{}}

	cc.updateFinalizedBlockCache(makeHeader(1000))
	assert.Equal(t, uint64(1000), cc.getCachedFinalizedBlockNumber())

	// Higher value should update
	cc.updateFinalizedBlockCache(makeHeader(2000))
	assert.Equal(t, uint64(2000), cc.getCachedFinalizedBlockNumber())

	// Lower value must NOT update (monotonic guarantee)
	cc.updateFinalizedBlockCache(makeHeader(1500))
	assert.Equal(t, uint64(2000), cc.getCachedFinalizedBlockNumber())

	// Equal value should not change anything
	cc.updateFinalizedBlockCache(makeHeader(2000))
	assert.Equal(t, uint64(2000), cc.getCachedFinalizedBlockNumber())
}

func TestUpdateFinalizedBlockCache_NilInputIgnored(t *testing.T) {
	t.Parallel()

	cc := &ContractCaller{finalizedBlockCache: &finalizedBlockCache{}}

	cc.updateFinalizedBlockCache(makeHeader(1000))

	// nil header should be ignored
	cc.updateFinalizedBlockCache(nil)
	assert.Equal(t, uint64(1000), cc.getCachedFinalizedBlockNumber())

	// header with nil Number should be ignored
	cc.updateFinalizedBlockCache(&ethTypes.Header{Number: nil})
	assert.Equal(t, uint64(1000), cc.getCachedFinalizedBlockNumber())
}

func TestUpdateFinalizedBlockCache_NilCachePointerNoOp(t *testing.T) {
	t.Parallel()

	cc := &ContractCaller{finalizedBlockCache: nil}

	// Should not panic when finalizedBlockCache pointer is nil
	cc.updateFinalizedBlockCache(makeHeader(1000))
	assert.Equal(t, uint64(0), cc.getCachedFinalizedBlockNumber())
}

func TestFinalizedBlockCache_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	cc := &ContractCaller{finalizedBlockCache: &finalizedBlockCache{}}

	var wg sync.WaitGroup

	// Simulate concurrent reads and writes (race detector will catch issues)
	for i := 0; i < 100; i++ {
		wg.Add(2)

		blockNum := uint64(i)

		go func() {
			defer wg.Done()
			cc.updateFinalizedBlockCache(makeHeader(blockNum))
		}()

		go func() {
			defer wg.Done()
			_ = cc.getCachedFinalizedBlockNumber()
		}()
	}

	wg.Wait()

	// Cache should hold the highest value written
	assert.Equal(t, uint64(99), cc.getCachedFinalizedBlockNumber())
}

