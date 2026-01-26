package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/x/chainmanager/types"
)

func TestDefaultParams(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil default params", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()

		require.NotNil(t, params)
	})

	t.Run("default params have correct values", func(t *testing.T) {
		t.Parallel()

		params := types.DefaultParams()

		require.Equal(t, uint64(6), params.MainChainTxConfirmations)
		require.Equal(t, uint64(10), params.BorChainTxConfirmations)
		require.NotEmpty(t, params.ChainParams.StateReceiverAddress)
		require.NotEmpty(t, params.ChainParams.ValidatorSetAddress)
	})
}

func TestNewParams(t *testing.T) {
	t.Parallel()

	t.Run("creates params with all fields", func(t *testing.T) {
		t.Parallel()

		chainParams := types.ChainParams{
			BorChainId:           "137",
			HeimdallChainId:      "heimdall-137",
			StateReceiverAddress: "0x0000000000000000000000000000000000001001",
			ValidatorSetAddress:  "0x0000000000000000000000000000000000001000",
		}

		params := types.NewParams(10, 20, chainParams)

		require.Equal(t, uint64(10), params.MainChainTxConfirmations)
		require.Equal(t, uint64(20), params.BorChainTxConfirmations)
		require.Equal(t, chainParams, params.ChainParams)
	})

	t.Run("creates params with zero confirmations", func(t *testing.T) {
		t.Parallel()

		chainParams := types.ChainParams{
			StateReceiverAddress: "0x0000000000000000000000000000000000001001",
			ValidatorSetAddress:  "0x0000000000000000000000000000000000001000",
		}

		params := types.NewParams(0, 0, chainParams)

		require.Equal(t, uint64(0), params.MainChainTxConfirmations)
		require.Equal(t, uint64(0), params.BorChainTxConfirmations)
	})
}

func TestParams_ValidateBasic(t *testing.T) {
	t.Parallel()

	t.Run("validates params with valid addresses", func(t *testing.T) {
		t.Parallel()

		params := types.NewParams(6, 10, types.ChainParams{
			PolTokenAddress:       "0x1234567890123456789012345678901234567890",
			StakingManagerAddress: "0x1234567890123456789012345678901234567891",
			SlashManagerAddress:   "0x1234567890123456789012345678901234567892",
			RootChainAddress:      "0x1234567890123456789012345678901234567893",
			StakingInfoAddress:    "0x1234567890123456789012345678901234567894",
			StateSenderAddress:    "0x1234567890123456789012345678901234567895",
			StateReceiverAddress:  "0x1234567890123456789012345678901234567896",
			ValidatorSetAddress:   "0x1234567890123456789012345678901234567897",
		})

		err := params.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("rejects invalid pol token address", func(t *testing.T) {
		t.Parallel()

		validAddress := "0x1234567890123456789012345678901234567890"
		params := types.NewParams(6, 10, types.ChainParams{
			PolTokenAddress:       "invalid",
			StakingManagerAddress: validAddress,
			SlashManagerAddress:   validAddress,
			RootChainAddress:      validAddress,
			StakingInfoAddress:    validAddress,
			StateSenderAddress:    validAddress,
			StateReceiverAddress:  validAddress,
			ValidatorSetAddress:   validAddress,
		})

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "pol_token_address")
	})

	t.Run("rejects invalid staking manager address", func(t *testing.T) {
		t.Parallel()

		validAddress := "0x1234567890123456789012345678901234567890"
		params := types.NewParams(6, 10, types.ChainParams{
			PolTokenAddress:       validAddress,
			StakingManagerAddress: "not-an-address",
			SlashManagerAddress:   validAddress,
			RootChainAddress:      validAddress,
			StakingInfoAddress:    validAddress,
			StateSenderAddress:    validAddress,
			StateReceiverAddress:  validAddress,
			ValidatorSetAddress:   validAddress,
		})

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "staking_manager_address")
	})

	t.Run("rejects invalid slash manager address", func(t *testing.T) {
		t.Parallel()

		validAddress := "0x1234567890123456789012345678901234567890"
		params := types.NewParams(6, 10, types.ChainParams{
			PolTokenAddress:       validAddress,
			StakingManagerAddress: validAddress,
			SlashManagerAddress:   "0xinvalid",
			RootChainAddress:      validAddress,
			StakingInfoAddress:    validAddress,
			StateSenderAddress:    validAddress,
			StateReceiverAddress:  validAddress,
			ValidatorSetAddress:   validAddress,
		})

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "slash_manager_address")
	})

	t.Run("rejects invalid root chain address", func(t *testing.T) {
		t.Parallel()

		validAddress := "0x1234567890123456789012345678901234567890"
		params := types.NewParams(6, 10, types.ChainParams{
			PolTokenAddress:       validAddress,
			StakingManagerAddress: validAddress,
			SlashManagerAddress:   validAddress,
			RootChainAddress:      "bad_address",
			StakingInfoAddress:    validAddress,
			StateSenderAddress:    validAddress,
			StateReceiverAddress:  validAddress,
			ValidatorSetAddress:   validAddress,
		})

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "root_chain_address")
	})

	t.Run("rejects invalid staking info address", func(t *testing.T) {
		t.Parallel()

		validAddress := "0x1234567890123456789012345678901234567890"
		params := types.NewParams(6, 10, types.ChainParams{
			PolTokenAddress:       validAddress,
			StakingManagerAddress: validAddress,
			SlashManagerAddress:   validAddress,
			RootChainAddress:      validAddress,
			StakingInfoAddress:    "123",
			StateSenderAddress:    validAddress,
			StateReceiverAddress:  validAddress,
			ValidatorSetAddress:   validAddress,
		})

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "staking_info_address")
	})

	t.Run("rejects invalid state sender address", func(t *testing.T) {
		t.Parallel()

		validAddress := "0x1234567890123456789012345678901234567890"
		params := types.NewParams(6, 10, types.ChainParams{
			PolTokenAddress:       validAddress,
			StakingManagerAddress: validAddress,
			SlashManagerAddress:   validAddress,
			RootChainAddress:      validAddress,
			StakingInfoAddress:    validAddress,
			StateSenderAddress:    "not_hex",
			StateReceiverAddress:  validAddress,
			ValidatorSetAddress:   validAddress,
		})

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "state_sender_address")
	})

	t.Run("rejects invalid state receiver address", func(t *testing.T) {
		t.Parallel()

		validAddress := "0x1234567890123456789012345678901234567890"
		params := types.NewParams(6, 10, types.ChainParams{
			PolTokenAddress:       validAddress,
			StakingManagerAddress: validAddress,
			SlashManagerAddress:   validAddress,
			RootChainAddress:      validAddress,
			StakingInfoAddress:    validAddress,
			StateSenderAddress:    validAddress,
			StateReceiverAddress:  "",
			ValidatorSetAddress:   validAddress,
		})

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "state_receiver_address")
	})

	t.Run("rejects invalid validator set address", func(t *testing.T) {
		t.Parallel()

		validAddress := "0x1234567890123456789012345678901234567890"
		params := types.NewParams(6, 10, types.ChainParams{
			PolTokenAddress:       validAddress,
			StakingManagerAddress: validAddress,
			SlashManagerAddress:   validAddress,
			RootChainAddress:      validAddress,
			StakingInfoAddress:    validAddress,
			StateSenderAddress:    validAddress,
			StateReceiverAddress:  validAddress,
			ValidatorSetAddress:   "0x",
		})

		err := params.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator_set_address")
	})

	t.Run("all addresses must be valid if provided", func(t *testing.T) {
		t.Parallel()

		validAddress := "0x1234567890123456789012345678901234567890"
		params := types.NewParams(6, 10, types.ChainParams{
			PolTokenAddress:       validAddress,
			StakingManagerAddress: validAddress,
			SlashManagerAddress:   validAddress,
			RootChainAddress:      validAddress,
			StakingInfoAddress:    validAddress,
			StateSenderAddress:    validAddress,
			StateReceiverAddress:  "0x0000000000000000000000000000000000001001",
			ValidatorSetAddress:   "0x0000000000000000000000000000000000001000",
		})

		err := params.ValidateBasic()
		require.NoError(t, err)
	})
}

func TestDefaultConstants(t *testing.T) {
	t.Parallel()

	t.Run("default constants have correct values", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, uint64(6), types.DefaultMainChainTxConfirmations)
		require.Equal(t, uint64(10), types.DefaultBorChainTxConfirmations)
		require.Equal(t, "0x0000000000000000000000000000000000001001", types.DefaultStateReceiverAddress)
		require.Equal(t, "0x0000000000000000000000000000000000001000", types.DefaultValidatorSetAddress)
	})
}
