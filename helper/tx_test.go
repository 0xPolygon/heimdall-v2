package helper

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEthClient implements EthClient interface for testing.
type mockEthClient struct {
	blockByNumberFn    func(ctx context.Context, number *big.Int) (*types.Block, error)
	suggestGasTipCapFn func(ctx context.Context) (*big.Int, error)
	pendingNonceAtFn   func(ctx context.Context, account common.Address) (uint64, error)
	estimateGasFn      func(ctx context.Context, call ethereum.CallMsg) (uint64, error)
	chainIDFn          func(ctx context.Context) (*big.Int, error)
}

func (m *mockEthClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	if m.blockByNumberFn != nil {
		return m.blockByNumberFn(ctx, number)
	}
	return nil, errors.New("not implemented")
}

func (m *mockEthClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	if m.suggestGasTipCapFn != nil {
		return m.suggestGasTipCapFn(ctx)
	}
	return nil, errors.New("not implemented")
}

func (m *mockEthClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	if m.pendingNonceAtFn != nil {
		return m.pendingNonceAtFn(ctx, account)
	}
	return 0, errors.New("not implemented")
}

func (m *mockEthClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
	if m.estimateGasFn != nil {
		return m.estimateGasFn(ctx, call)
	}
	return 0, errors.New("not implemented")
}

func (m *mockEthClient) ChainID(ctx context.Context) (*big.Int, error) {
	if m.chainIDFn != nil {
		return m.chainIDFn(ctx)
	}
	return nil, errors.New("not implemented")
}

// mockClientParams holds parameters for creating a mock client.
type mockClientParams struct {
	blockNumber  int64
	baseFee      *big.Int
	suggestedTip *big.Int
	nonce        uint64
	gasLimit     uint64
	chainID      int64
}

// setupMockClient creates a mock client with the specified parameters.
func setupMockClient(params mockClientParams) *mockEthClient {
	return &mockEthClient{
		blockByNumberFn: func(ctx context.Context, number *big.Int) (*types.Block, error) {
			return types.NewBlockWithHeader(
				&types.Header{
					Number:  big.NewInt(params.blockNumber),
					BaseFee: params.baseFee,
				},
			), nil
		},
		suggestGasTipCapFn: func(ctx context.Context) (*big.Int, error) {
			return params.suggestedTip, nil
		},
		pendingNonceAtFn: func(ctx context.Context, account common.Address) (uint64, error) {
			return params.nonce, nil
		},
		estimateGasFn: func(ctx context.Context, call ethereum.CallMsg) (uint64, error) {
			return params.gasLimit, nil
		},
		chainIDFn: func(ctx context.Context) (*big.Int, error) {
			return big.NewInt(params.chainID), nil
		},
	}
}

func TestGenerateAuthObj_NormalEIP1559Flow(t *testing.T) {
	t.Parallel()

	InitTestHeimdallConfig("")

	baseFee := big.NewInt(20000000000)     // 20 Gwei
	suggestedTip := big.NewInt(1000000000) // 1 Gwei

	mockClient := setupMockClient(mockClientParams{
		blockNumber:  1000,
		baseFee:      baseFee,
		suggestedTip: suggestedTip,
		nonce:        5,
		gasLimit:     21000,
		chainID:      1,
	})

	address := common.HexToAddress("0x0000000000000000000000000000000000000000")
	data := []byte("test data")

	auth, err := generateAuthObjWithClient(mockClient, address, data)
	require.NoError(t, err)
	require.NotNil(t, auth)

	// Verify EIP-1559 fields are set.
	assert.NotNil(t, auth.GasFeeCap, "GasFeeCap should be set")
	assert.NotNil(t, auth.GasTipCap, "GasTipCap should be set")
	assert.Nil(t, auth.GasPrice, "GasPrice should be nil for EIP-1559")

	// Verify gas fee cap calculation: (baseFee * 2) + tipCap = (20 * 2) + 1 = 41 Gwei.
	expectedFeeCap := new(big.Int).Mul(baseFee, big.NewInt(2))
	expectedFeeCap.Add(expectedFeeCap, suggestedTip)
	assert.Equal(t, expectedFeeCap, auth.GasFeeCap, "GasFeeCap should be (baseFee * 2) + tipCap")

	// Verify tip cap is the suggested tip (since it's below config max).
	assert.Equal(t, suggestedTip, auth.GasTipCap, "GasTipCap should be the suggested tip")

	// Verify other fields.
	assert.Equal(t, uint64(21000), auth.GasLimit, "GasLimit should match estimated gas")
	assert.Equal(t, big.NewInt(5), auth.Nonce, "Nonce should match pending nonce")
}

func TestGenerateAuthObj_BaseFeeNil(t *testing.T) {
	t.Parallel()

	InitTestHeimdallConfig("")

	mockClient := &mockEthClient{
		blockByNumberFn: func(ctx context.Context, number *big.Int) (*types.Block, error) {
			// Return a block with nil baseFee (pre-EIP-1559 chain).
			return types.NewBlockWithHeader(
				&types.Header{
					Number:  big.NewInt(1000),
					BaseFee: nil,
				},
			), nil
		},
	}

	address := common.HexToAddress("0x0000000000000000000000000000000000000000")
	data := []byte("test data")

	auth, err := generateAuthObjWithClient(mockClient, address, data)

	assert.Error(t, err, "Should return error when baseFee is nil")
	assert.Nil(t, auth, "Auth should be nil when baseFee is nil")
	assert.Contains(t, err.Error(), "baseFee is nil", "Error message should mention baseFee")
}

func TestGenerateAuthObj_FeeCapExceedsConfiguredMaximum(t *testing.T) {
	t.Parallel()

	InitTestHeimdallConfig("")

	// Set a very high base fee that will cause calculated fee cap to exceed config.
	baseFee := big.NewInt(300000000000)    // 300 Gwei - this will result in 600+ Gwei fee cap
	suggestedTip := big.NewInt(1000000000) // 1 Gwei

	mockClient := setupMockClient(mockClientParams{
		blockNumber:  1000,
		baseFee:      baseFee,
		suggestedTip: suggestedTip,
		nonce:        5,
		gasLimit:     21000,
		chainID:      1,
	})

	address := common.HexToAddress("0x0000000000000000000000000000000000000000")
	data := []byte("test data")

	auth, err := generateAuthObjWithClient(mockClient, address, data)
	require.NoError(t, err)
	require.NotNil(t, auth)

	// The calculated fee cap would be (300 * 2) + 1 = 601 Gwei.
	// But it should be capped to configured maximum (500 Gwei = DefaultMainChainGasFeeCap).
	configuredMax := big.NewInt(DefaultMainChainGasFeeCap)
	assert.Equal(t, configuredMax, auth.GasFeeCap, "GasFeeCap should be capped to configured maximum")
}

func TestGenerateAuthObj_TipCapExceedsConfiguredMaximum(t *testing.T) {
	t.Parallel()

	InitTestHeimdallConfig("")

	baseFee := big.NewInt(20000000000)      // 20 Gwei
	suggestedTip := big.NewInt(50000000000) // 50 Gwei - exceeds configured max (10 Gwei)

	mockClient := setupMockClient(mockClientParams{
		blockNumber:  1000,
		baseFee:      baseFee,
		suggestedTip: suggestedTip,
		nonce:        5,
		gasLimit:     21000,
		chainID:      1,
	})

	address := common.HexToAddress("0x0000000000000000000000000000000000000000")
	data := []byte("test data")

	auth, err := generateAuthObjWithClient(mockClient, address, data)
	require.NoError(t, err)
	require.NotNil(t, auth)

	// Tip cap should be capped to configured maximum (10 Gwei = DefaultMainChainGasTipCap).
	configuredTipMax := big.NewInt(DefaultMainChainGasTipCap)
	assert.Equal(t, configuredTipMax, auth.GasTipCap, "GasTipCap should be capped to configured maximum")

	// Fee cap should be (baseFee * 2) + configuredTipMax = (20 * 2) + 10 = 50 Gwei.
	expectedFeeCap := new(big.Int).Mul(baseFee, big.NewInt(2))
	expectedFeeCap.Add(expectedFeeCap, configuredTipMax)
	assert.Equal(t, expectedFeeCap, auth.GasFeeCap, "GasFeeCap should use capped tip")
}

func TestGenerateAuthObj_TipCapExceedsFeeCap(t *testing.T) {
	t.Parallel()

	InitTestHeimdallConfig("")

	// Edge case: very high base fee causes fee cap to be clamped, but tip cap is still below fee cap before clamping.
	// but tip cap is still below fee cap before clamping.
	// After fee cap clamping, tip might exceed fee cap.
	baseFee := big.NewInt(250000000000)    // 250 Gwei
	suggestedTip := big.NewInt(5000000000) // 5 Gwei (below config max of 10 Gwei)

	mockClient := setupMockClient(mockClientParams{
		blockNumber:  1000,
		baseFee:      baseFee,
		suggestedTip: suggestedTip,
		nonce:        5,
		gasLimit:     21000,
		chainID:      1,
	})

	address := common.HexToAddress("0x0000000000000000000000000000000000000000")
	data := []byte("test data")

	auth, err := generateAuthObjWithClient(mockClient, address, data)
	require.NoError(t, err)
	require.NotNil(t, auth)

	// Calculated fee cap would be (250 * 2) + 5 = 505 Gwei, capped to 500 Gwei.
	// Tip cap (5 Gwei) should not exceed fee cap (500 Gwei) - this case is fine

	// Tip cap should never exceed fee cap.
	assert.True(t, auth.GasTipCap.Cmp(auth.GasFeeCap) <= 0,
		"GasTipCap (%s) should not exceed GasFeeCap (%s)", auth.GasTipCap, auth.GasFeeCap)
}

func TestGenerateAuthObj_BlockByNumberError(t *testing.T) {
	t.Parallel()

	InitTestHeimdallConfig("")

	mockClient := &mockEthClient{
		blockByNumberFn: func(ctx context.Context, number *big.Int) (*types.Block, error) {
			return nil, errors.New("RPC error: failed to fetch block")
		},
	}

	address := common.HexToAddress("0x0000000000000000000000000000000000000000")
	data := []byte("test data")

	auth, err := generateAuthObjWithClient(mockClient, address, data)

	assert.Error(t, err, "Should return error when BlockByNumber fails")
	assert.Nil(t, auth, "Auth should be nil when BlockByNumber fails")
}

func TestGenerateAuthObj_SuggestGasTipCapError(t *testing.T) {
	t.Parallel()

	InitTestHeimdallConfig("")

	baseFee := big.NewInt(20000000000) // 20 Gwei

	mockClient := &mockEthClient{
		blockByNumberFn: func(ctx context.Context, number *big.Int) (*types.Block, error) {
			return types.NewBlockWithHeader(
				&types.Header{
					Number:  big.NewInt(1000),
					BaseFee: baseFee,
				},
			), nil
		},
		suggestGasTipCapFn: func(ctx context.Context) (*big.Int, error) {
			return nil, errors.New("RPC error: failed to suggest gas tip cap")
		},
	}

	address := common.HexToAddress("0x0000000000000000000000000000000000000000")
	data := []byte("test data")

	auth, err := generateAuthObjWithClient(mockClient, address, data)

	assert.Error(t, err, "Should return error when SuggestGasTipCap fails")
	assert.Nil(t, auth, "Auth should be nil when SuggestGasTipCap fails")
}
