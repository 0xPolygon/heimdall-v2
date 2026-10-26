package helper

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/contracts/erc20"
	"github.com/0xPolygon/heimdall-v2/contracts/rootchain"
	"github.com/0xPolygon/heimdall-v2/contracts/slashmanager"
	"github.com/0xPolygon/heimdall-v2/contracts/stakemanager"
	"github.com/0xPolygon/heimdall-v2/contracts/stakinginfo"
	"github.com/0xPolygon/heimdall-v2/contracts/statereceiver"
	"github.com/0xPolygon/heimdall-v2/contracts/statesender"
	"github.com/0xPolygon/heimdall-v2/contracts/validatorset"
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

// TestPopulateABIs tests that every caller gets the ABI parsed by its generated binding
func TestPopulateABIs(t *testing.T) {
	t.Parallel()

	caller, err := NewContractCaller()
	require.NoError(t, err)

	tests := []struct {
		got      abi.ABI
		metaData *bind.MetaData
	}{
		{caller.RootChainABI, rootchain.RootchainMetaData},
		{caller.StakingInfoABI, stakinginfo.StakinginfoMetaData},
		{caller.ValidatorSetABI, validatorset.ValidatorsetMetaData},
		{caller.StateReceiverABI, statereceiver.StatereceiverMetaData},
		{caller.StateSenderABI, statesender.StatesenderMetaData},
		{caller.StakeManagerABI, stakemanager.StakemanagerMetaData},
		{caller.SlashManagerABI, slashmanager.SlashmanagerMetaData},
		{caller.PolTokenABI, erc20.Erc20MetaData},
	}
	for _, tt := range tests {
		want, err := tt.metaData.GetAbi()
		require.NoError(t, err)
		require.NotEmpty(t, tt.got.Methods)
		require.Equal(t, *want, tt.got)
	}
}

// TestContractInstanceCacheConcurrent guards against data races on
// ContractCaller.contractInstanceCache.
// Run with the race detector to assert no race:
//
//	go test -race -run TestContractInstanceCacheConcurrent ./helper/...
func TestContractInstanceCacheConcurrent(t *testing.T) {
	caller, err := NewContractCaller()
	if err != nil {
		t.Fatalf("NewContractCaller: %v", err)
	}

	addresses := []string{
		"0x0000000000000000000000000000000000000001",
		"0x0000000000000000000000000000000000000002",
		"0x0000000000000000000000000000000000000003",
		"0x0000000000000000000000000000000000000004",
		"0x0000000000000000000000000000000000000005",
		"0x0000000000000000000000000000000000000006",
		"0x0000000000000000000000000000000000000007",
		"0x0000000000000000000000000000000000000008",
	}

	const goroutines = 64
	const itersPerGoroutine = 250

	var wg sync.WaitGroup
	wg.Add(goroutines)
	start := make(chan struct{})
	for g := 0; g < goroutines; g++ {
		gID := g
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < itersPerGoroutine; i++ {
				addr := addresses[(gID+i)%len(addresses)]
				switch (gID + i) % 8 {
				case 0:
					_, _ = caller.GetStakingInfoInstance(addr)
				case 1:
					_, _ = caller.GetStateSenderInstance(addr)
				case 2:
					_, _ = caller.GetStateReceiverInstance(addr)
				case 3:
					_, _ = caller.GetRootChainInstance(addr)
				case 4:
					_, _ = caller.GetStakeManagerInstance(addr)
				case 5:
					_, _ = caller.GetSlashManagerInstance(addr)
				case 6:
					_, _ = caller.GetTokenInstance(addr)
				case 7:
					_, _ = caller.GetValidatorSetInstance(addr)
				}
			}
		}()
	}
	close(start)
	wg.Wait()
}

// TestContractInstanceCacheSameAddressDifferentBindings guards against two
// binding types configured at the same address colliding in the cache.
func TestContractInstanceCacheSameAddressDifferentBindings(t *testing.T) {
	t.Parallel()

	caller, err := NewContractCaller()
	require.NoError(t, err)

	const address = "0x0000000000000000000000000000000000000009"

	rootChain, err := caller.GetRootChainInstance(address)
	require.NoError(t, err)
	_, err = caller.GetStakingInfoInstance(address)
	require.NoError(t, err)

	cachedRootChain, err := caller.GetRootChainInstance(address)
	require.NoError(t, err)
	require.Same(t, rootChain, cachedRootChain)
}

// TestDecodeLog checks that a log is decoded only when both its index and its
// emitting contract match, and only as the event its topic identifies.
func TestDecodeLog(t *testing.T) {
	t.Parallel()

	caller, err := NewContractCaller()
	require.NoError(t, err)

	rootChain := common.HexToAddress("0x00000000000000000000000000000000000000aa")
	event := caller.RootChainABI.Events[NewHeaderBlockEvent]
	data, err := event.Inputs.NonIndexed().Pack(big.NewInt(1), big.NewInt(256), [32]byte{1})
	require.NoError(t, err)

	headerBlockLog := &ethTypes.Log{
		Address: rootChain,
		Topics: []common.Hash{
			event.ID,
			common.BytesToHash(common.HexToAddress("0x00000000000000000000000000000000000000bb").Bytes()),
			common.BigToHash(big.NewInt(10000)),
			common.BigToHash(big.NewInt(0)),
		},
		Data:  data,
		Index: 3,
	}
	otherContractLog := *headerBlockLog
	otherContractLog.Address = common.HexToAddress("0x00000000000000000000000000000000000000cc")
	otherContractLog.Index = 4
	receipt := &ethTypes.Receipt{Logs: []*ethTypes.Log{headerBlockLog, &otherContractLog}}

	decoded, err := caller.DecodeNewHeaderBlockEvent(rootChain.Hex(), receipt, 3)
	require.NoError(t, err)
	require.Equal(t, int64(256), decoded.End.Int64())
	require.Equal(t, int64(10000), decoded.HeaderBlockId.Int64())

	_, err = caller.DecodeNewHeaderBlockEvent(rootChain.Hex(), receipt, 4)
	require.ErrorContains(t, err, errEventNotFound, "log at the index from another contract must not match")

	_, err = caller.DecodeNewHeaderBlockEvent(rootChain.Hex(), receipt, 5)
	require.ErrorContains(t, err, errEventNotFound)

	_, err = caller.DecodeValidatorTopupFeesEvent(rootChain.Hex(), receipt, 3)
	require.ErrorContains(t, err, "topic event mismatch")
}

// hangingBackend never answers a contract call until its context is done.
type hangingBackend struct {
	bind.ContractBackend
}

func (hangingBackend) CallContract(ctx context.Context, _ ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// TestMainChainViewCallHonorsTimeout guards against a hung L1 endpoint blocking
// a contract view call indefinitely.
func TestMainChainViewCallHonorsTimeout(t *testing.T) {
	t.Parallel()

	rootChain, err := rootchain.NewRootchain(common.Address{}, hangingBackend{})
	require.NoError(t, err)
	caller := &ContractCaller{MainChainTimeout: 50 * time.Millisecond}

	done := make(chan error, 1)
	go func() {
		_, err := caller.GetLastChildBlock(rootChain)
		done <- err
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(5 * time.Second):
		t.Fatal("GetLastChildBlock did not return after MainChainTimeout")
	}
}
