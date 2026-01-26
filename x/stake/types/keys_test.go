package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/stake/types"
)

func TestModuleName(t *testing.T) {
	t.Parallel()

	t.Run("module name is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "stake", types.ModuleName)
	})
}

func TestStoreKey(t *testing.T) {
	t.Parallel()

	t.Run("store key matches module name", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, types.ModuleName, types.StoreKey)
	})
}

func TestRouterKey(t *testing.T) {
	t.Parallel()

	t.Run("router key matches module name", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, types.ModuleName, types.RouterKey)
	})
}

func TestDefaultLogIndexUnit(t *testing.T) {
	t.Parallel()

	t.Run("default log index unit is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, 100000, types.DefaultLogIndexUnit)
	})

	t.Run("default log index unit is greater than zero", func(t *testing.T) {
		t.Parallel()

		require.Greater(t, types.DefaultLogIndexUnit, 0)
	})
}

func TestValidatorsKey(t *testing.T) {
	t.Parallel()

	t.Run("validators key prefix is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.ValidatorsKey)
		require.NotEmpty(t, types.ValidatorsKey)
	})
}

func TestValidatorSetKey(t *testing.T) {
	t.Parallel()

	t.Run("validator set key prefix is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.ValidatorSetKey)
		require.NotEmpty(t, types.ValidatorSetKey)
	})
}

func TestCurrentValidatorSetKey(t *testing.T) {
	t.Parallel()

	t.Run("current validator set key is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.CurrentValidatorSetKey)
		require.NotEmpty(t, types.CurrentValidatorSetKey)
	})
}

func TestStakeSequenceKey(t *testing.T) {
	t.Parallel()

	t.Run("stake sequence key prefix is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.StakeSequenceKey)
		require.NotEmpty(t, types.StakeSequenceKey)
	})
}

func TestSignerKey(t *testing.T) {
	t.Parallel()

	t.Run("signer key prefix is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.SignerKey)
		require.NotEmpty(t, types.SignerKey)
	})
}

func TestLastBlockTxsKey(t *testing.T) {
	t.Parallel()

	t.Run("last block txs key is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.LastBlockTxsKey)
		require.NotEmpty(t, types.LastBlockTxsKey)
	})
}

func TestPreviousBlockValidatorSetKey(t *testing.T) {
	t.Parallel()

	t.Run("previous block validator set key is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.PreviousBlockValidatorSetKey)
		require.NotEmpty(t, types.PreviousBlockValidatorSetKey)
	})
}

func TestPenultimateBlockValidatorSetKey(t *testing.T) {
	t.Parallel()

	t.Run("penultimate block validator set key is defined", func(t *testing.T) {
		t.Parallel()

		require.NotNil(t, types.PenultimateBlockValidatorSetKey)
		require.NotEmpty(t, types.PenultimateBlockValidatorSetKey)
	})
}

func TestKeyUniqueness(t *testing.T) {
	t.Parallel()

	t.Run("all key prefixes are unique", func(t *testing.T) {
		t.Parallel()

		keys := [][]byte{
			types.ValidatorsKey,
			types.ValidatorSetKey,
			types.CurrentValidatorSetKey,
			types.StakeSequenceKey,
			types.SignerKey,
			types.LastBlockTxsKey,
			types.PreviousBlockValidatorSetKey,
			types.PenultimateBlockValidatorSetKey,
		}

		seen := make(map[byte]bool)
		for _, key := range keys {
			require.NotEmpty(t, key, "key should not be empty")
			prefix := key[0]
			require.False(t, seen[prefix], "duplicate key prefix: 0x%x", prefix)
			seen[prefix] = true
		}
	})
}

func TestEmptyPubKey(t *testing.T) {
	t.Parallel()

	t.Run("empty pub key is defined", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, types.PubKey{}, types.EmptyPubKey)
	})

	t.Run("empty pub key has all zero bytes", func(t *testing.T) {
		t.Parallel()

		for i := range types.EmptyPubKey {
			require.Equal(t, byte(0), types.EmptyPubKey[i])
		}
	})
}

func TestPubKeySize(t *testing.T) {
	t.Parallel()

	t.Run("pub key is 65 bytes", func(t *testing.T) {
		t.Parallel()

		var pk types.PubKey
		require.Equal(t, 65, len(pk))
	})
}

func TestSecp256k1Type(t *testing.T) {
	t.Parallel()

	t.Run("secp256k1 type is defined", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, types.Secp256k1Type)
	})
}

func TestModuleConstants(t *testing.T) {
	t.Parallel()

	t.Run("all module constants are non-empty", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, types.ModuleName)
		require.NotEmpty(t, types.StoreKey)
		require.NotEmpty(t, types.RouterKey)
	})

	t.Run("module constants are consistent", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, types.ModuleName, types.StoreKey)
		require.Equal(t, types.ModuleName, types.RouterKey)
	})
}
