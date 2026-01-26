package helper

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/contracts/rootchain"
)

func TestUnpackSigAndVotes(t *testing.T) {
	t.Parallel()

	// Get the RootChain ABI
	abi, err := getABI(rootchain.RootchainMetaData.ABI)
	require.NoError(t, err, "Error while getting RootChainABI")

	// Test data for the current ABI: submitHeaderBlock(bytes data, bytes sigs)
	testData := []byte("test checkpoint data")
	testSigs := []byte("test signatures data")

	// Pack the data using the current ABI
	method := abi.Methods["submitHeaderBlock"]
	packed, err := method.Inputs.Pack(testData, testSigs)
	require.NoError(t, err, "Error packing test data")

	// Prepend method selector
	methodID := method.ID
	payload := append(methodID, packed...)

	// Test unpacking
	unpackedData, unpackedSigs, reserved, err := UnpackSigAndVotes(payload, abi)
	require.NoError(t, err, "Error while unpacking payload")
	require.Equal(t, testData, unpackedData, "Data doesn't match")
	require.Equal(t, testSigs, unpackedSigs, "Sigs don't match")
	require.Empty(t, reserved, "Reserved field should be empty")

	t.Log("Successfully unpacked data:", hex.EncodeToString(unpackedData))
	t.Log("Successfully unpacked sigs:", hex.EncodeToString(unpackedSigs))
}

// TestUnpackSigAndVotesWithRealSignatures tests unpacking with actual ECDSA signatures
func TestUnpackSigAndVotesWithRealSignatures(t *testing.T) {
	t.Parallel()

	// Create a private key for testing
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()

	// Message to sign (checkpoint data)
	message := []byte("checkpoint data to sign")

	// Sign the message
	signature, err := privKey.Sign(message)
	require.NoError(t, err, "Error signing message")

	// Get the RootChain ABI
	abi, err := getABI(rootchain.RootchainMetaData.ABI)
	require.NoError(t, err, "Error while getting RootChainABI")

	// Pack using current ABI
	method := abi.Methods["submitHeaderBlock"]
	packed, err := method.Inputs.Pack(message, signature)
	require.NoError(t, err, "Error packing data")

	// Prepend method selector
	payload := append(method.ID, packed...)

	// Unpack
	unpackedData, unpackedSigs, _, err := UnpackSigAndVotes(payload, abi)
	require.NoError(t, err, "Error unpacking")
	require.Equal(t, message, unpackedData, "Data mismatch")
	require.Equal(t, signature, unpackedSigs, "Signature mismatch")

	// Verify signature recovery works
	recoveredPubKey, err := signing.RecoverPubKey(unpackedData, unpackedSigs)
	require.NoError(t, err, "Error recovering public key")

	recovered := secp256k1.PubKey{Key: recoveredPubKey}
	require.Equal(t, pubKey.Address(), recovered.Address(), "Recovered address doesn't match original")

	t.Log("Successfully recovered signer address:", hex.EncodeToString(recovered.Address().Bytes()))
}

func TestGetPowerFromAmount(t *testing.T) {
	t.Parallel()

	scenarios := map[string]string{
		"48000000000000000000000": "48000",
		"10000000000000000000000": "10000",
		"1000000000000000000000":  "1000",
		"4800000000000000000000":  "4800",
		"480000000000000000000":   "480",
		"20000000000000000000":    "20",
		"10000000000000000000":    "10",
		"1000000000000000000":     "1",
	}

	for k, v := range scenarios {
		bv, _ := big.NewInt(0).SetString(k, 10)
		p, err := GetPowerFromAmount(bv)
		require.Nil(t, err, "Error must be nil", "input:", k, "output", v)
		require.Equal(t, p.String(), v, "Power must match")
	}
}
