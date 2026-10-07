package keeper

import (
	"context"
	"errors"
	"math/big"
	"testing"

	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper/mocks"
)

// TestFetchBorHeader covers fetchBorHeader directly (unexported, so this file
// lives in package keeper rather than keeper_test): the success path, plain
// error propagation, and normalizing a nil header or a nil Number into an
// error so no caller has to guard either directly.
func TestFetchBorHeader(t *testing.T) {
	ctx := context.Background()

	t.Run("success returns the header unchanged", func(t *testing.T) {
		mc := &mocks.IContractCaller{}
		want := &ethTypes.Header{Number: big.NewInt(42)}
		mc.On("GetBorChainBlock", mock.Anything, mock.Anything).Return(want, nil)

		got, err := fetchBorHeader(ctx, mc, nil)
		require.NoError(t, err)
		require.Same(t, want, got)
	})

	t.Run("a plain error from the client propagates", func(t *testing.T) {
		mc := &mocks.IContractCaller{}
		fetchErr := errors.New("rpc unreachable")
		mc.On("GetBorChainBlock", mock.Anything, mock.Anything).Return((*ethTypes.Header)(nil), fetchErr)

		got, err := fetchBorHeader(ctx, mc, nil)
		require.ErrorIs(t, err, fetchErr)
		require.Nil(t, got)
	})

	t.Run("a nil header with a nil error is normalized into an error", func(t *testing.T) {
		mc := &mocks.IContractCaller{}
		mc.On("GetBorChainBlock", mock.Anything, mock.Anything).Return((*ethTypes.Header)(nil), nil)

		got, err := fetchBorHeader(ctx, mc, nil)
		require.Error(t, err)
		require.Nil(t, got)
		require.Contains(t, err.Error(), "nil header returned")
	})

	t.Run("a non-nil header with a nil Number is normalized into an error", func(t *testing.T) {
		mc := &mocks.IContractCaller{}
		mc.On("GetBorChainBlock", mock.Anything, mock.Anything).Return(&ethTypes.Header{Number: nil}, nil)

		got, err := fetchBorHeader(ctx, mc, nil)
		require.Error(t, err)
		require.Nil(t, got)
		require.Contains(t, err.Error(), "nil Number")
	})
}
