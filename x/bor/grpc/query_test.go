package grpc

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	proto "github.com/0xPolygon/polyproto/bor"
	protoutil "github.com/0xPolygon/polyproto/utils"
)

// MockBorApiClient is a mock implementation of proto.BorApiClient
type MockBorApiClient struct {
	mock.Mock
}

func (m *MockBorApiClient) GetRootHash(ctx context.Context, req *proto.GetRootHashRequest, _ ...grpc.CallOption) (*proto.GetRootHashResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetRootHashResponse), args.Error(1)
}

func (m *MockBorApiClient) GetVoteOnHash(ctx context.Context, req *proto.GetVoteOnHashRequest, _ ...grpc.CallOption) (*proto.GetVoteOnHashResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetVoteOnHashResponse), args.Error(1)
}

func (m *MockBorApiClient) HeaderByNumber(ctx context.Context, req *proto.GetHeaderByNumberRequest, _ ...grpc.CallOption) (*proto.GetHeaderByNumberResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetHeaderByNumberResponse), args.Error(1)
}

func (m *MockBorApiClient) BlockByNumber(ctx context.Context, req *proto.GetBlockByNumberRequest, _ ...grpc.CallOption) (*proto.GetBlockByNumberResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetBlockByNumberResponse), args.Error(1)
}

func (m *MockBorApiClient) TransactionReceipt(ctx context.Context, req *proto.ReceiptRequest, _ ...grpc.CallOption) (*proto.ReceiptResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.ReceiptResponse), args.Error(1)
}

func (m *MockBorApiClient) BorBlockReceipt(ctx context.Context, req *proto.ReceiptRequest, _ ...grpc.CallOption) (*proto.ReceiptResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.ReceiptResponse), args.Error(1)
}

func (m *MockBorApiClient) GetStartBlockHeimdallSpanID(ctx context.Context, req *proto.GetStartBlockHeimdallSpanIDRequest, _ ...grpc.CallOption) (*proto.GetStartBlockHeimdallSpanIDResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetStartBlockHeimdallSpanIDResponse), args.Error(1)
}

func TestGetRootHash(t *testing.T) {
	t.Parallel()

	t.Run("successful root hash retrieval", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		expectedHash := "0x1234567890abcdef"
		mockClient.On("GetRootHash", mock.Anything, mock.MatchedBy(func(req *proto.GetRootHashRequest) bool {
			return req.StartBlockNumber == 100 && req.EndBlockNumber == 200
		})).Return(&proto.GetRootHashResponse{RootHash: expectedHash}, nil)

		hash, err := grpcClient.GetRootHash(context.Background(), 100, 200)
		require.NoError(t, err)
		require.Equal(t, expectedHash, hash)

		mockClient.AssertExpectations(t)
	})

	t.Run("error retrieving root hash", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		mockClient.On("GetRootHash", mock.Anything, mock.Anything).
			Return(nil, errors.New("rpc error"))

		hash, err := grpcClient.GetRootHash(context.Background(), 100, 200)
		require.Error(t, err)
		require.Empty(t, hash)
		require.Contains(t, err.Error(), "rpc error")

		mockClient.AssertExpectations(t)
	})
}

func TestGetVoteOnHash(t *testing.T) {
	t.Parallel()

	t.Run("successful vote retrieval - true", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		mockClient.On("GetVoteOnHash", mock.Anything, mock.MatchedBy(func(req *proto.GetVoteOnHashRequest) bool {
			return req.StartBlockNumber == 100 &&
				req.EndBlockNumber == 200 &&
				req.Hash == "0xabcd" &&
				req.MilestoneId == "milestone1"
		})).Return(&proto.GetVoteOnHashResponse{Response: true}, nil)

		vote, err := grpcClient.GetVoteOnHash(context.Background(), 100, 200, "0xabcd", "milestone1")
		require.NoError(t, err)
		require.True(t, vote)

		mockClient.AssertExpectations(t)
	})

	t.Run("successful vote retrieval - false", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		mockClient.On("GetVoteOnHash", mock.Anything, mock.Anything).
			Return(&proto.GetVoteOnHashResponse{Response: false}, nil)

		vote, err := grpcClient.GetVoteOnHash(context.Background(), 100, 200, "0xabcd", "milestone1")
		require.NoError(t, err)
		require.False(t, vote)

		mockClient.AssertExpectations(t)
	})

	t.Run("error retrieving vote", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		mockClient.On("GetVoteOnHash", mock.Anything, mock.Anything).
			Return(nil, errors.New("vote error"))

		vote, err := grpcClient.GetVoteOnHash(context.Background(), 100, 200, "0xabcd", "milestone1")
		require.Error(t, err)
		require.False(t, vote)
		require.Contains(t, err.Error(), "vote error")

		mockClient.AssertExpectations(t)
	})
}

func TestHeaderByNumber(t *testing.T) {
	t.Parallel()

	t.Run("successful header retrieval", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		parentHash := common.HexToHash("0x1234")
		mockClient.On("HeaderByNumber", mock.Anything, mock.MatchedBy(func(req *proto.GetHeaderByNumberRequest) bool {
			return req.Number == "0x64" // 100 in hex
		})).Return(&proto.GetHeaderByNumberResponse{
			Header: &proto.Header{
				Number:     100,
				ParentHash: protoutil.ConvertHashToH256(parentHash),
				Time:       1234567890,
			},
		}, nil)

		header, err := grpcClient.HeaderByNumber(context.Background(), 100)
		require.NoError(t, err)
		require.NotNil(t, header)
		require.Equal(t, big.NewInt(100), header.Number)
		require.Equal(t, parentHash, header.ParentHash)
		require.Equal(t, uint64(1234567890), header.Time)

		mockClient.AssertExpectations(t)
	})

	// Note: The "blockID too large" check in the original code is unreachable
	// since blockID is int64, so no test for it

	t.Run("error retrieving header", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		mockClient.On("HeaderByNumber", mock.Anything, mock.Anything).
			Return(nil, errors.New("header error"))

		header, err := grpcClient.HeaderByNumber(context.Background(), 100)
		require.Error(t, err)
		require.Contains(t, err.Error(), "header error")
		require.NotNil(t, header) // Returns empty header on error
		require.Nil(t, header.Number)

		mockClient.AssertExpectations(t)
	})
}

func TestBlockByNumber(t *testing.T) {
	t.Parallel()

	t.Run("successful block retrieval", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		parentHash := common.HexToHash("0xabcd")
		mockClient.On("BlockByNumber", mock.Anything, mock.MatchedBy(func(req *proto.GetBlockByNumberRequest) bool {
			return req.Number == "0xc8" // 200 in hex
		})).Return(&proto.GetBlockByNumberResponse{
			Block: &proto.Block{
				Header: &proto.Header{
					Number:     200,
					ParentHash: protoutil.ConvertHashToH256(parentHash),
					Time:       9876543210,
				},
			},
		}, nil)

		block, err := grpcClient.BlockByNumber(context.Background(), 200)
		require.NoError(t, err)
		require.NotNil(t, block)
		require.Equal(t, big.NewInt(200), block.Number())
		require.Equal(t, parentHash, block.ParentHash())
		require.Equal(t, uint64(9876543210), block.Time())

		mockClient.AssertExpectations(t)
	})

	// Note: The "blockID too large" check in the original code is unreachable
	// since blockID is int64, so no test for it

	t.Run("error retrieving block", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		mockClient.On("BlockByNumber", mock.Anything, mock.Anything).
			Return(nil, errors.New("block error"))

		block, err := grpcClient.BlockByNumber(context.Background(), 200)
		require.Error(t, err)
		require.Contains(t, err.Error(), "block error")
		require.NotNil(t, block) // Returns empty block on error

		mockClient.AssertExpectations(t)
	})
}

func TestTransactionReceipt(t *testing.T) {
	t.Parallel()

	t.Run("successful receipt retrieval", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		txHash := common.HexToHash("0x123456")
		blockHash := common.HexToHash("0xabcdef")
		contractAddr := common.HexToAddress("0x1111111111111111111111111111111111111111")

		mockClient.On("TransactionReceipt", mock.Anything, mock.MatchedBy(func(req *proto.ReceiptRequest) bool {
			return protoutil.ConvertH256ToHash(req.Hash) == txHash
		})).Return(&proto.ReceiptResponse{
			Receipt: &proto.Receipt{
				Type:              2,
				PostState:         []byte("state"),
				Status:            1,
				CumulativeGasUsed: 21000,
				TxHash:            protoutil.ConvertHashToH256(txHash),
				ContractAddress:   protoutil.ConvertAddressToH160(contractAddr),
				GasUsed:           21000,
				EffectiveGasPrice: 1000000000,
				BlobGasUsed:       0,
				BlobGasPrice:      0,
				BlockHash:         protoutil.ConvertHashToH256(blockHash),
				BlockNumber:       100,
				TransactionIndex:  5,
			},
		}, nil)

		receipt, err := grpcClient.TransactionReceipt(context.Background(), txHash)
		require.NoError(t, err)
		require.NotNil(t, receipt)
		require.Equal(t, uint8(2), receipt.Type)
		require.Equal(t, uint64(1), receipt.Status)
		require.Equal(t, uint64(21000), receipt.CumulativeGasUsed)
		require.Equal(t, txHash, receipt.TxHash)
		require.Equal(t, contractAddr, receipt.ContractAddress)
		require.Equal(t, uint64(21000), receipt.GasUsed)
		require.Equal(t, big.NewInt(1000000000), receipt.EffectiveGasPrice)
		require.Equal(t, blockHash, receipt.BlockHash)
		require.Equal(t, big.NewInt(100), receipt.BlockNumber)
		require.Equal(t, uint(5), receipt.TransactionIndex)

		mockClient.AssertExpectations(t)
	})

	t.Run("error retrieving receipt", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		txHash := common.HexToHash("0x123456")
		mockClient.On("TransactionReceipt", mock.Anything, mock.Anything).
			Return(nil, errors.New("receipt error"))

		receipt, err := grpcClient.TransactionReceipt(context.Background(), txHash)
		require.Error(t, err)
		require.Contains(t, err.Error(), "receipt error")
		require.NotNil(t, receipt) // Returns empty receipt on error

		mockClient.AssertExpectations(t)
	})
}

func TestBorBlockReceipt(t *testing.T) {
	t.Parallel()

	t.Run("successful bor block receipt retrieval", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		txHash := common.HexToHash("0xaabbcc")
		blockHash := common.HexToHash("0xddeeff")
		mockClient.On("BorBlockReceipt", mock.Anything, mock.MatchedBy(func(req *proto.ReceiptRequest) bool {
			return protoutil.ConvertH256ToHash(req.Hash) == txHash
		})).Return(&proto.ReceiptResponse{
			Receipt: &proto.Receipt{
				Type:              0,
				PostState:         []byte{},
				Status:            1,
				CumulativeGasUsed: 50000,
				TxHash:            protoutil.ConvertHashToH256(txHash),
				ContractAddress:   protoutil.ConvertAddressToH160(common.Address{}),
				GasUsed:           50000,
				EffectiveGasPrice: 0,
				BlobGasUsed:       0,
				BlobGasPrice:      0,
				BlockHash:         protoutil.ConvertHashToH256(blockHash),
				BlockNumber:       500,
				TransactionIndex:  0,
			},
		}, nil)

		receipt, err := grpcClient.BorBlockReceipt(context.Background(), txHash)
		require.NoError(t, err)
		require.NotNil(t, receipt)
		require.Equal(t, uint8(0), receipt.Type)
		require.Equal(t, uint64(1), receipt.Status)
		require.Equal(t, uint64(50000), receipt.CumulativeGasUsed)
		require.Equal(t, big.NewInt(500), receipt.BlockNumber)

		mockClient.AssertExpectations(t)
	})

	t.Run("error retrieving bor block receipt", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		txHash := common.HexToHash("0xaabbcc")
		mockClient.On("BorBlockReceipt", mock.Anything, mock.Anything).
			Return(nil, errors.New("bor receipt error"))

		receipt, err := grpcClient.BorBlockReceipt(context.Background(), txHash)
		require.Error(t, err)
		require.Contains(t, err.Error(), "bor receipt error")
		require.NotNil(t, receipt)

		mockClient.AssertExpectations(t)
	})
}

func TestReceiptResponseToTypesReceipt(t *testing.T) {
	t.Parallel()

	t.Run("converts receipt with all fields", func(t *testing.T) {
		t.Parallel()

		txHash := common.HexToHash("0x111")
		blockHash := common.HexToHash("0x222")
		contractAddr := common.HexToAddress("0x3333333333333333333333333333333333333333")

		protoReceipt := &proto.Receipt{
			Type:              2,
			PostState:         []byte("state_data"),
			Status:            1,
			CumulativeGasUsed: 30000,
			TxHash:            protoutil.ConvertHashToH256(txHash),
			ContractAddress:   protoutil.ConvertAddressToH160(contractAddr),
			GasUsed:           15000,
			EffectiveGasPrice: 2000000000,
			BlobGasUsed:       1000,
			BlobGasPrice:      500,
			BlockHash:         protoutil.ConvertHashToH256(blockHash),
			BlockNumber:       1234,
			TransactionIndex:  10,
		}

		result := receiptResponseToTypesReceipt(protoReceipt)

		require.NotNil(t, result)
		require.Equal(t, uint8(2), result.Type)
		require.Equal(t, []byte("state_data"), result.PostState)
		require.Equal(t, uint64(1), result.Status)
		require.Equal(t, uint64(30000), result.CumulativeGasUsed)
		require.Equal(t, txHash, result.TxHash)
		require.Equal(t, contractAddr, result.ContractAddress)
		require.Equal(t, uint64(15000), result.GasUsed)
		require.Equal(t, big.NewInt(2000000000), result.EffectiveGasPrice)
		require.Equal(t, uint64(1000), result.BlobGasUsed)
		require.Equal(t, big.NewInt(500), result.BlobGasPrice)
		require.Equal(t, blockHash, result.BlockHash)
		require.Equal(t, big.NewInt(1234), result.BlockNumber)
		require.Equal(t, uint(10), result.TransactionIndex)
	})

	t.Run("converts receipt with zero values", func(t *testing.T) {
		t.Parallel()

		protoReceipt := &proto.Receipt{
			Type:              0,
			PostState:         nil,
			Status:            0,
			CumulativeGasUsed: 0,
			TxHash:            protoutil.ConvertHashToH256(common.Hash{}),
			ContractAddress:   protoutil.ConvertAddressToH160(common.Address{}),
			GasUsed:           0,
			EffectiveGasPrice: 0,
			BlobGasUsed:       0,
			BlobGasPrice:      0,
			BlockHash:         protoutil.ConvertHashToH256(common.Hash{}),
			BlockNumber:       0,
			TransactionIndex:  0,
		}

		result := receiptResponseToTypesReceipt(protoReceipt)

		require.NotNil(t, result)
		require.Equal(t, uint8(0), result.Type)
		require.Equal(t, uint64(0), result.Status)
		require.Equal(t, uint64(0), result.CumulativeGasUsed)
		require.Equal(t, big.NewInt(0), result.EffectiveGasPrice)
		require.Equal(t, big.NewInt(0), result.BlobGasPrice)
		require.Equal(t, big.NewInt(0), result.BlockNumber)
		require.Equal(t, uint(0), result.TransactionIndex)
	})
}
