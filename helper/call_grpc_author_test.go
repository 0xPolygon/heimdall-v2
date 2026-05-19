package helper

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// fakeBorGRPCClient is an in-memory BorGRPCClienter that returns whatever the
// test configured. Only the methods the tests exercise are implemented with
// real behavior
type fakeBorGRPCClient struct {
	authorAddr *common.Address
	authorErr  error
	calls      int
}

func (f *fakeBorGRPCClient) GetAuthor(_ context.Context, _ *big.Int) (*common.Address, error) {
	f.calls++
	return f.authorAddr, f.authorErr
}

func (f *fakeBorGRPCClient) HeaderByNumber(_ context.Context, _ int64) (*ethTypes.Header, error) {
	panic("fakeBorGRPCClient.HeaderByNumber: unexpected call")
}
func (f *fakeBorGRPCClient) BlockByNumber(_ context.Context, _ int64) (*ethTypes.Block, error) {
	panic("fakeBorGRPCClient.BlockByNumber: unexpected call")
}
func (f *fakeBorGRPCClient) GetRootHash(_ context.Context, _ uint64, _ uint64) (string, error) {
	panic("fakeBorGRPCClient.GetRootHash: unexpected call")
}
func (f *fakeBorGRPCClient) GetVoteOnHash(_ context.Context, _ uint64, _ uint64, _ string, _ string) (bool, error) {
	panic("fakeBorGRPCClient.GetVoteOnHash: unexpected call")
}
func (f *fakeBorGRPCClient) GetTdByHash(_ context.Context, _ common.Hash) (uint64, error) {
	panic("fakeBorGRPCClient.GetTdByHash: unexpected call")
}
func (f *fakeBorGRPCClient) GetTdByNumber(_ context.Context, _ *big.Int) (uint64, error) {
	panic("fakeBorGRPCClient.GetTdByNumber: unexpected call")
}
func (f *fakeBorGRPCClient) GetBlockInfoInBatch(_ context.Context, _, _ int64) ([]*ethTypes.Header, []uint64, []common.Address, error) {
	panic("fakeBorGRPCClient.GetBlockInfoInBatch: unexpected call")
}
func (f *fakeBorGRPCClient) TransactionReceipt(_ context.Context, _ common.Hash) (*ethTypes.Receipt, error) {
	panic("fakeBorGRPCClient.TransactionReceipt: unexpected call")
}
func (f *fakeBorGRPCClient) BorBlockReceipt(_ context.Context, _ common.Hash) (*ethTypes.Receipt, error) {
	panic("fakeBorGRPCClient.BorBlockReceipt: unexpected call")
}

// TestGetBorChainBlockAuthor_GRPC_HappyPath ensures that when the gRPC client
// returns a non-nil address and no error, the caller returns that exact
// pointer (kills the happy-path return_value mutant that would replace it with
// nil) and the err is nil.
func TestGetBorChainBlockAuthor_GRPC_HappyPath(t *testing.T) {
	t.Parallel()

	want := common.HexToAddress("0x000000000000000000000000000000000000abcd")
	fake := &fakeBorGRPCClient{authorAddr: &want, authorErr: nil}
	c := ContractCaller{
		BorChainGrpcFlag:   true,
		BorChainGrpcClient: fake,
		BorChainTimeout:    time.Second,
	}

	got, err := c.GetBorChainBlockAuthor(context.Background(), big.NewInt(42))
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, want, *got, "returned pointer must reference the fake's configured address")
	require.Equal(t, 1, fake.calls)
}

// TestGetBorChainBlockAuthor_GRPC_ErrorPropagates ensures that when the gRPC
// client returns a non-nil error, the caller returns that exact error (kills
// the `if err != nil` branch_removal / negate_conditional and the
// `return nil, err` return_value→zero mutants).
func TestGetBorChainBlockAuthor_GRPC_ErrorPropagates(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("grpc GetAuthor transport failure")
	fake := &fakeBorGRPCClient{authorAddr: nil, authorErr: sentinel}
	c := ContractCaller{
		BorChainGrpcFlag:   true,
		BorChainGrpcClient: fake,
		BorChainTimeout:    time.Second,
	}

	got, err := c.GetBorChainBlockAuthor(context.Background(), big.NewInt(42))
	require.Nil(t, got)
	require.ErrorIs(t, err, sentinel, "error from gRPC GetAuthor must propagate unchanged")
}

// TestGetBorChainBlockAuthor_GRPC_NilAuthorIsNotFound ensures that when the
// gRPC client returns (nil, nil), and the caller maps it to ethereum.NotFound
// (kills the `if author == nil` branch_removal / negate_conditional).
func TestGetBorChainBlockAuthor_GRPC_NilAuthorIsNotFound(t *testing.T) {
	t.Parallel()

	fake := &fakeBorGRPCClient{authorAddr: nil, authorErr: nil}
	c := ContractCaller{
		BorChainGrpcFlag:   true,
		BorChainGrpcClient: fake,
		BorChainTimeout:    time.Second,
	}

	got, err := c.GetBorChainBlockAuthor(context.Background(), big.NewInt(42))
	require.Nil(t, got)
	require.ErrorIs(t, err, ethereum.NotFound, "nil author must map to ethereum.NotFound sentinel")
}
