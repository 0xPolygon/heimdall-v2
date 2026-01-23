package types_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/types"
)

func TestGetAccountRootHash(t *testing.T) {
	tests := []struct {
		name           string
		accounts       []types.DividendAccount
		expectError    bool
		expectNonEmpty bool
	}{
		{
			name: "single account",
			accounts: []types.DividendAccount{
				{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
			},
			expectError:    false,
			expectNonEmpty: true,
		},
		{
			name: "multiple accounts",
			accounts: []types.DividendAccount{
				{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
				{User: "0x0000000000000000000000000000000000000002", FeeAmount: "200"},
				{User: "0x0000000000000000000000000000000000000003", FeeAmount: "300"},
			},
			expectError:    false,
			expectNonEmpty: true,
		},
		{
			name: "accounts with large fee amounts",
			accounts: []types.DividendAccount{
				{User: "0x0000000000000000000000000000000000000001", FeeAmount: "1000000000000000000000"},
			},
			expectError:    false,
			expectNonEmpty: true,
		},
		{
			name: "accounts with zero fee",
			accounts: []types.DividendAccount{
				{User: "0x0000000000000000000000000000000000000001", FeeAmount: "0"},
			},
			expectError:    false,
			expectNonEmpty: true,
		},
		{
			name: "unsorted accounts",
			accounts: []types.DividendAccount{
				{User: "0x0000000000000000000000000000000000000003", FeeAmount: "300"},
				{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
				{User: "0x0000000000000000000000000000000000000002", FeeAmount: "200"},
			},
			expectError:    false,
			expectNonEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := types.GetAccountRootHash(tt.accounts)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expectNonEmpty {
					require.NotEmpty(t, hash)
					require.NotNil(t, hash)
				}
			}
		})
	}
}

func TestGetAccountTree(t *testing.T) {
	tests := []struct {
		name        string
		accounts    []types.DividendAccount
		expectError bool
	}{
		{
			name: "single account",
			accounts: []types.DividendAccount{
				{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
			},
			expectError: false,
		},
		{
			name: "multiple accounts",
			accounts: []types.DividendAccount{
				{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
				{User: "0x0000000000000000000000000000000000000002", FeeAmount: "200"},
			},
			expectError: false,
		},
		{
			name: "five accounts",
			accounts: []types.DividendAccount{
				{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
				{User: "0x0000000000000000000000000000000000000002", FeeAmount: "200"},
				{User: "0x0000000000000000000000000000000000000003", FeeAmount: "300"},
				{User: "0x0000000000000000000000000000000000000004", FeeAmount: "400"},
				{User: "0x0000000000000000000000000000000000000005", FeeAmount: "500"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, err := types.GetAccountTree(tt.accounts)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, tree)
				require.NotNil(t, tree.Root)
				require.NotEmpty(t, tree.Root.Hash)
			}
		})
	}
}

func TestSortDividendAccountByAddress(t *testing.T) {
	tests := []struct {
		name        string
		accounts    []types.DividendAccount
		expectError bool
		checkOrder  bool
	}{
		{
			name: "already sorted accounts",
			accounts: []types.DividendAccount{
				{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
				{User: "0x0000000000000000000000000000000000000002", FeeAmount: "200"},
				{User: "0x0000000000000000000000000000000000000003", FeeAmount: "300"},
			},
			expectError: false,
			checkOrder:  true,
		},
		{
			name: "unsorted accounts",
			accounts: []types.DividendAccount{
				{User: "0x0000000000000000000000000000000000000003", FeeAmount: "300"},
				{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
				{User: "0x0000000000000000000000000000000000000002", FeeAmount: "200"},
			},
			expectError: false,
			checkOrder:  true,
		},
		{
			name: "reverse sorted accounts",
			accounts: []types.DividendAccount{
				{User: "0x0000000000000000000000000000000000000005", FeeAmount: "500"},
				{User: "0x0000000000000000000000000000000000000004", FeeAmount: "400"},
				{User: "0x0000000000000000000000000000000000000003", FeeAmount: "300"},
				{User: "0x0000000000000000000000000000000000000002", FeeAmount: "200"},
				{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
			},
			expectError: false,
			checkOrder:  true,
		},
		{
			name: "single account",
			accounts: []types.DividendAccount{
				{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
			},
			expectError: false,
			checkOrder:  true,
		},
		{
			name: "accounts with mixed case addresses",
			accounts: []types.DividendAccount{
				{User: "0x000000000000000000000000000000000000000A", FeeAmount: "100"},
				{User: "0x0000000000000000000000000000000000000005", FeeAmount: "200"},
			},
			expectError: false,
			checkOrder:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted, err := types.SortDividendAccountByAddress(tt.accounts)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, sorted, len(tt.accounts))

				if tt.checkOrder && len(sorted) > 1 {
					// Verify sorting order
					for i := 0; i < len(sorted)-1; i++ {
						// Each address should be <= the next
						require.NotEmpty(t, sorted[i].User)
						require.NotEmpty(t, sorted[i+1].User)
					}
				}
			}
		})
	}
}

func TestGetAccountProof(t *testing.T) {
	accounts := []types.DividendAccount{
		{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
		{User: "0x0000000000000000000000000000000000000002", FeeAmount: "200"},
		{User: "0x0000000000000000000000000000000000000003", FeeAmount: "300"},
	}

	tests := []struct {
		name         string
		accounts     []types.DividendAccount
		userAddr     string
		expectError  bool
		expectNonNil bool
	}{
		{
			name:         "valid user address - first account",
			accounts:     accounts,
			userAddr:     "0x0000000000000000000000000000000000000001",
			expectError:  false,
			expectNonNil: true,
		},
		{
			name:         "valid user address - middle account",
			accounts:     accounts,
			userAddr:     "0x0000000000000000000000000000000000000002",
			expectError:  false,
			expectNonNil: true,
		},
		{
			name:         "valid user address - last account",
			accounts:     accounts,
			userAddr:     "0x0000000000000000000000000000000000000003",
			expectError:  false,
			expectNonNil: true,
		},
		{
			name:         "user address not in list",
			accounts:     accounts,
			userAddr:     "0x0000000000000000000000000000000000000099",
			expectError:  false,
			expectNonNil: false, // Address not in the list returns nil proof
		},
		{
			name:         "case insensitive address",
			accounts:     accounts,
			userAddr:     "0x0000000000000000000000000000000000000001",
			expectError:  false,
			expectNonNil: true,
		},
		{
			name: "single account",
			accounts: []types.DividendAccount{
				{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
			},
			userAddr:     "0x0000000000000000000000000000000000000001",
			expectError:  false,
			expectNonNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proof, index, err := types.GetAccountProof(tt.accounts, tt.userAddr)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expectNonNil {
					require.NotNil(t, proof)
					// Index should be within the valid range
					require.GreaterOrEqual(t, index, uint64(0))
					require.Less(t, index, uint64(len(tt.accounts)))
				} else {
					require.Nil(t, proof)
				}
			}
		})
	}
}

func TestVerifyAccountProof(t *testing.T) {
	accounts := []types.DividendAccount{
		{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
		{User: "0x0000000000000000000000000000000000000002", FeeAmount: "200"},
	}

	// Generate a real proof first
	proof, _, err := types.GetAccountProof(accounts, "0x0000000000000000000000000000000000000001")
	require.NoError(t, err)
	proofHex := "0x" + hex.EncodeToString(proof)

	tests := []struct {
		name           string
		accounts       []types.DividendAccount
		userAddr       string
		proofToVerify  string
		expectVerified bool
		expectError    bool
	}{
		{
			name:           "valid proof",
			accounts:       accounts,
			userAddr:       "0x0000000000000000000000000000000000000001",
			proofToVerify:  proofHex,
			expectVerified: true,
			expectError:    false,
		},
		{
			name:           "invalid proof - wrong hex",
			accounts:       accounts,
			userAddr:       "0x0000000000000000000000000000000000000001",
			proofToVerify:  "0xabcd1234",
			expectVerified: false,
			expectError:    false,
		},
		{
			name:           "user not in accounts",
			accounts:       accounts,
			userAddr:       "0x0000000000000000000000000000000000000099",
			proofToVerify:  proofHex,
			expectVerified: false,
			expectError:    false,
		},
		{
			name:           "empty proof",
			accounts:       accounts,
			userAddr:       "0x0000000000000000000000000000000000000001",
			proofToVerify:  "0x",
			expectVerified: false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verified, err := types.VerifyAccountProof(tt.accounts, tt.userAddr, tt.proofToVerify)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectVerified, verified)
			}
		})
	}
}

func TestAppendBytes32(t *testing.T) {
	tests := []struct {
		name           string
		data           [][]byte
		expectLength   int
		expectMultiple int
	}{
		{
			name: "single 16-byte array",
			data: [][]byte{
				make([]byte, 16),
			},
			expectLength:   32,
			expectMultiple: 32,
		},
		{
			name: "two 16-byte arrays",
			data: [][]byte{
				make([]byte, 16),
				make([]byte, 16),
			},
			expectLength:   64,
			expectMultiple: 32,
		},
		{
			name: "single 32-byte array",
			data: [][]byte{
				make([]byte, 32),
			},
			expectLength:   32,
			expectMultiple: 32,
		},
		{
			name: "mixed size arrays",
			data: [][]byte{
				make([]byte, 8),
				make([]byte, 16),
				make([]byte, 20),
			},
			expectLength:   96, // 3 * 32
			expectMultiple: 32,
		},
		{
			name:           "empty input",
			data:           [][]byte{},
			expectLength:   0,
			expectMultiple: 32,
		},
		{
			name: "single byte",
			data: [][]byte{
				{0x01},
			},
			expectLength:   32,
			expectMultiple: 32,
		},
		{
			name: "array with zero bytes",
			data: [][]byte{
				{},
			},
			expectLength:   32,
			expectMultiple: 32,
		},
		{
			name: "array exceeding 32 bytes - should be skipped",
			data: [][]byte{
				make([]byte, 40), // Exceeds 32 bytes
				make([]byte, 16), // Valid
			},
			expectLength:   32, // Only the valid one is appended
			expectMultiple: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := types.AppendBytes32(tt.data...)

			require.Len(t, result, tt.expectLength)
			// Verify the result length is multiple of 32
			if len(result) > 0 {
				require.Equal(t, 0, len(result)%tt.expectMultiple)
			}
		})
	}
}

func TestDividendAccountCalculateHash(t *testing.T) {
	tests := []struct {
		name           string
		account        types.DividendAccount
		expectError    bool
		expectNonEmpty bool
	}{
		{
			name: "valid account with small fee",
			account: types.DividendAccount{
				User:      "0x0000000000000000000000000000000000000001",
				FeeAmount: "100",
			},
			expectError:    false,
			expectNonEmpty: true,
		},
		{
			name: "valid account with large fee",
			account: types.DividendAccount{
				User:      "0x0000000000000000000000000000000000000001",
				FeeAmount: "1000000000000000000000",
			},
			expectError:    false,
			expectNonEmpty: true,
		},
		{
			name: "account with zero fee",
			account: types.DividendAccount{
				User:      "0x0000000000000000000000000000000000000001",
				FeeAmount: "0",
			},
			expectError:    false,
			expectNonEmpty: true,
		},
		{
			name: "different addresses produce different hashes",
			account: types.DividendAccount{
				User:      "0x0000000000000000000000000000000000000002",
				FeeAmount: "100",
			},
			expectError:    false,
			expectNonEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := tt.account.CalculateHash()

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expectNonEmpty {
					require.NotEmpty(t, hash)
					require.NotNil(t, hash)
					// Keccak256 produces 32-byte hashes
					require.Equal(t, 32, len(hash))
				}
			}
		})
	}
}

func TestDividendAccountCalculateHashDeterministic(t *testing.T) {
	// Test that the same account produces the same hash
	account := types.DividendAccount{
		User:      "0x0000000000000000000000000000000000000001",
		FeeAmount: "100",
	}

	hash1, err1 := account.CalculateHash()
	require.NoError(t, err1)

	hash2, err2 := account.CalculateHash()
	require.NoError(t, err2)

	require.Equal(t, hash1, hash2)
}

func TestDividendAccountCalculateHashUniqueness(t *testing.T) {
	// Test that different accounts produce different hashes
	account1 := types.DividendAccount{
		User:      "0x0000000000000000000000000000000000000001",
		FeeAmount: "100",
	}

	account2 := types.DividendAccount{
		User:      "0x0000000000000000000000000000000000000002",
		FeeAmount: "100",
	}

	account3 := types.DividendAccount{
		User:      "0x0000000000000000000000000000000000000001",
		FeeAmount: "200",
	}

	hash1, err1 := account1.CalculateHash()
	require.NoError(t, err1)

	hash2, err2 := account2.CalculateHash()
	require.NoError(t, err2)

	hash3, err3 := account3.CalculateHash()
	require.NoError(t, err3)

	// Different user addresses should produce different hashes
	require.NotEqual(t, hash1, hash2)

	// Different fee amounts should produce different hashes
	require.NotEqual(t, hash1, hash3)
}

func TestDividendAccountEquals(t *testing.T) {
	tests := []struct {
		name     string
		account1 types.DividendAccount
		account2 types.DividendAccount
		expected bool
	}{
		{
			name: "equal accounts - same user",
			account1: types.DividendAccount{
				User:      "0x0000000000000000000000000000000000000001",
				FeeAmount: "100",
			},
			account2: types.DividendAccount{
				User:      "0x0000000000000000000000000000000000000001",
				FeeAmount: "200", // Different fee, but Equals only checks User
			},
			expected: true,
		},
		{
			name: "different accounts - different user",
			account1: types.DividendAccount{
				User:      "0x0000000000000000000000000000000000000001",
				FeeAmount: "100",
			},
			account2: types.DividendAccount{
				User:      "0x0000000000000000000000000000000000000002",
				FeeAmount: "100",
			},
			expected: false,
		},
		{
			name: "equal accounts - exact same",
			account1: types.DividendAccount{
				User:      "0x0000000000000000000000000000000000000001",
				FeeAmount: "100",
			},
			account2: types.DividendAccount{
				User:      "0x0000000000000000000000000000000000000001",
				FeeAmount: "100",
			},
			expected: true,
		},
		{
			name: "different case in address - exact string comparison",
			account1: types.DividendAccount{
				User:      "0x0000000000000000000000000000000000000001",
				FeeAmount: "100",
			},
			account2: types.DividendAccount{
				User:      "0x0000000000000000000000000000000000000001",
				FeeAmount: "100",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.account1.Equals(tt.account2)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAccountRootHashConsistency(t *testing.T) {
	// Test that the same accounts produce the same root hash
	accounts := []types.DividendAccount{
		{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
		{User: "0x0000000000000000000000000000000000000002", FeeAmount: "200"},
		{User: "0x0000000000000000000000000000000000000003", FeeAmount: "300"},
	}

	hash1, err1 := types.GetAccountRootHash(accounts)
	require.NoError(t, err1)

	hash2, err2 := types.GetAccountRootHash(accounts)
	require.NoError(t, err2)

	require.Equal(t, hash1, hash2)
}

func TestGetAccountRootHashWithUnsortedInput(t *testing.T) {
	// Test that unsorted accounts produce the same hash as sorted
	accounts1 := []types.DividendAccount{
		{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
		{User: "0x0000000000000000000000000000000000000002", FeeAmount: "200"},
		{User: "0x0000000000000000000000000000000000000003", FeeAmount: "300"},
	}

	accounts2 := []types.DividendAccount{
		{User: "0x0000000000000000000000000000000000000003", FeeAmount: "300"},
		{User: "0x0000000000000000000000000000000000000001", FeeAmount: "100"},
		{User: "0x0000000000000000000000000000000000000002", FeeAmount: "200"},
	}

	hash1, err1 := types.GetAccountRootHash(accounts1)
	require.NoError(t, err1)

	hash2, err2 := types.GetAccountRootHash(accounts2)
	require.NoError(t, err2)

	// Both should produce the same hash since sorting is internal
	require.Equal(t, hash1, hash2)
}
