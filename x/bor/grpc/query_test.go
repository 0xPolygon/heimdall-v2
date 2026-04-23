package grpc

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
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

func (m *MockBorApiClient) GetAuthor(ctx context.Context, req *proto.GetAuthorRequest, _ ...grpc.CallOption) (*proto.GetAuthorResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetAuthorResponse), args.Error(1)
}

func (m *MockBorApiClient) GetTdByHash(ctx context.Context, req *proto.GetTdByHashRequest, _ ...grpc.CallOption) (*proto.GetTdResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetTdResponse), args.Error(1)
}

func (m *MockBorApiClient) GetTdByNumber(ctx context.Context, req *proto.GetTdByNumberRequest, _ ...grpc.CallOption) (*proto.GetTdResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetTdResponse), args.Error(1)
}

func (m *MockBorApiClient) GetBlockInfoInBatch(ctx context.Context, req *proto.GetBlockInfoInBatchRequest, _ ...grpc.CallOption) (*proto.GetBlockInfoInBatchResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proto.GetBlockInfoInBatchResponse), args.Error(1)
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

func TestProtoHeaderToEthHeader_RoundTrip_Cancun(t *testing.T) {
	t.Parallel()

	src := &ethTypes.Header{
		ParentHash:       common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
		UncleHash:        ethTypes.EmptyUncleHash,
		Coinbase:         common.HexToAddress("0x0123456789abcdef0123456789abcdef01234567"),
		Root:             common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444"),
		TxHash:           common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555"),
		ReceiptHash:      common.HexToHash("0x6666666666666666666666666666666666666666666666666666666666666666"),
		Bloom:            ethTypes.Bloom{0x01, 0x02, 0x03},
		Difficulty:       big.NewInt(17),
		Number:           big.NewInt(1234567),
		GasLimit:         30_000_000,
		GasUsed:          21_000,
		Time:             1_700_000_000,
		Extra:            []byte{0xde, 0xad, 0xbe, 0xef},
		MixDigest:        common.HexToHash("0x7777777777777777777777777777777777777777777777777777777777777777"),
		Nonce:            ethTypes.BlockNonce{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8},
		BaseFee:          big.NewInt(1_000_000_000),
		WithdrawalsHash:  new(common.HexToHash("0xaabbccddeeff00112233445566778899aabbccddeeff00112233445566778899")),
		BlobGasUsed:      new(uint64(131072)),
		ExcessBlobGas:    new(uint64(262144)),
		ParentBeaconRoot: new(common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")),
		RequestsHash:     new(common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")),
	}

	pb := ethHeaderToProtoForTest(src)
	got := protoHeaderToEthHeader(pb)
	require.Equal(t, src.Hash(), got.Hash())
}

// TestProtoHeaderToEthHeader_RoundTrip_CancunZeroBlobGas proves the nil vs. zero trap on blobGasUsed/excessBlobGas is handled.
func TestProtoHeaderToEthHeader_RoundTrip_CancunZeroBlobGas(t *testing.T) {
	t.Parallel()

	zeroHash := common.Hash{}

	src := &ethTypes.Header{
		ParentHash:       common.HexToHash("0x01"),
		UncleHash:        ethTypes.EmptyUncleHash,
		Coinbase:         common.HexToAddress("0x0123456789abcdef0123456789abcdef01234567"),
		Root:             common.HexToHash("0x02"),
		TxHash:           common.HexToHash("0x03"),
		ReceiptHash:      common.HexToHash("0x04"),
		Difficulty:       big.NewInt(1),
		Number:           big.NewInt(100),
		GasLimit:         30_000_000,
		Time:             1_700_000_000,
		BaseFee:          big.NewInt(1_000_000_000),
		BlobGasUsed:      new(uint64(0)),
		ExcessBlobGas:    new(uint64(0)),
		ParentBeaconRoot: &zeroHash,
	}

	pb := ethHeaderToProtoForTest(src)
	got := protoHeaderToEthHeader(pb)
	require.Equal(t, src.Hash(), got.Hash(), "hash must match for Cancun-with-zero-blob-gas header")
	require.NotNil(t, got.BlobGasUsed, "BlobGasUsed must round-trip to &0 (not nil)")
	require.NotNil(t, got.ExcessBlobGas, "ExcessBlobGas must round-trip to &0 (not nil)")
}

func TestProtoHeaderToEthHeader_RoundTrip_PreShanghai(t *testing.T) {
	t.Parallel()

	src := &ethTypes.Header{
		ParentHash:  common.HexToHash("0x01"),
		UncleHash:   ethTypes.EmptyUncleHash,
		Coinbase:    common.HexToAddress("0x0123456789abcdef0123456789abcdef01234567"),
		Root:        common.HexToHash("0x02"),
		TxHash:      common.HexToHash("0x03"),
		ReceiptHash: common.HexToHash("0x04"),
		Difficulty:  big.NewInt(1),
		Number:      big.NewInt(100),
		GasLimit:    30_000_000,
		GasUsed:     0,
		Time:        1_600_000_000,
		Extra:       []byte{},
		MixDigest:   common.Hash{},
		Nonce:       ethTypes.BlockNonce{},
	}
	pb := ethHeaderToProtoForTest(src)
	got := protoHeaderToEthHeader(pb)
	require.Equal(t, src.Hash(), got.Hash())
}

func TestGetAuthor(t *testing.T) {
	t.Parallel()

	t.Run("successful author retrieval", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		want := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
		mockClient.On("GetAuthor", mock.Anything, mock.MatchedBy(func(req *proto.GetAuthorRequest) bool {
			return req.Number == "0x2a" // 42 in hex
		})).Return(&proto.GetAuthorResponse{
			Author: protoutil.ConvertAddressToH160(want),
		}, nil)

		got, err := grpcClient.GetAuthor(context.Background(), big.NewInt(42))
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, want, *got)

		mockClient.AssertExpectations(t)
	})

	t.Run("rpc error propagated", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		mockClient.On("GetAuthor", mock.Anything, mock.Anything).
			Return(nil, errors.New("author error"))

		got, err := grpcClient.GetAuthor(context.Background(), big.NewInt(42))
		require.Error(t, err)
		require.Nil(t, got)
		require.Contains(t, err.Error(), "author error")

		mockClient.AssertExpectations(t)
	})

	t.Run("nil author in response", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		mockClient.On("GetAuthor", mock.Anything, mock.Anything).
			Return(&proto.GetAuthorResponse{Author: nil}, nil)

		got, err := grpcClient.GetAuthor(context.Background(), big.NewInt(42))
		require.Error(t, err)
		require.Nil(t, got)

		mockClient.AssertExpectations(t)
	})

	t.Run("nil block number translates to latest", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		addr := common.HexToAddress("0x1234")
		mockClient.On("GetAuthor", mock.Anything, mock.MatchedBy(func(req *proto.GetAuthorRequest) bool {
			return req.Number == "latest"
		})).Return(&proto.GetAuthorResponse{Author: protoutil.ConvertAddressToH160(addr)}, nil)

		got, err := grpcClient.GetAuthor(context.Background(), nil)
		require.NoError(t, err)
		require.Equal(t, addr, *got)

		mockClient.AssertExpectations(t)
	})
}

func TestGetTdByHash(t *testing.T) {
	t.Parallel()

	t.Run("successful td retrieval", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		wantHash := common.HexToHash("0x01")
		mockClient.On("GetTdByHash", mock.Anything, mock.MatchedBy(func(req *proto.GetTdByHashRequest) bool {
			got := protoutil.ConvertH256ToHash(req.Hash)
			return common.BytesToHash(got[:]) == wantHash
		})).Return(&proto.GetTdResponse{TotalDifficulty: 12345}, nil)

		got, err := grpcClient.GetTdByHash(context.Background(), wantHash)
		require.NoError(t, err)
		require.Equal(t, uint64(12345), got)

		mockClient.AssertExpectations(t)
	})

	t.Run("rpc error propagated", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		mockClient.On("GetTdByHash", mock.Anything, mock.Anything).
			Return(nil, errors.New("td error"))

		got, err := grpcClient.GetTdByHash(context.Background(), common.Hash{})
		require.Error(t, err)
		require.Equal(t, uint64(0), got)
		require.Contains(t, err.Error(), "td error")

		mockClient.AssertExpectations(t)
	})
}

func TestGetTdByNumber(t *testing.T) {
	t.Parallel()

	t.Run("successful td retrieval", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		mockClient.On("GetTdByNumber", mock.Anything, mock.MatchedBy(func(req *proto.GetTdByNumberRequest) bool {
			return req.Number == "0x63" // 99 in hex
		})).Return(&proto.GetTdResponse{TotalDifficulty: 54321}, nil)

		got, err := grpcClient.GetTdByNumber(context.Background(), big.NewInt(99))
		require.NoError(t, err)
		require.Equal(t, uint64(54321), got)

		mockClient.AssertExpectations(t)
	})

	t.Run("rpc error propagated", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		mockClient.On("GetTdByNumber", mock.Anything, mock.Anything).
			Return(nil, errors.New("td number error"))

		got, err := grpcClient.GetTdByNumber(context.Background(), big.NewInt(99))
		require.Error(t, err)
		require.Equal(t, uint64(0), got)

		mockClient.AssertExpectations(t)
	})
}

func TestGetBlockInfoInBatch(t *testing.T) {
	t.Parallel()

	t.Run("successful batch retrieval", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		hdr100 := makeTestHeader(t, 100)
		hdr101 := makeTestHeader(t, 101)
		author100 := common.HexToAddress("0x1111111111111111111111111111111111111111")
		author101 := common.HexToAddress("0x2222222222222222222222222222222222222222")

		mockClient.On("GetBlockInfoInBatch", mock.Anything, mock.MatchedBy(func(req *proto.GetBlockInfoInBatchRequest) bool {
			return req.StartBlockNumber == 100 && req.EndBlockNumber == 101
		})).Return(&proto.GetBlockInfoInBatchResponse{
			Blocks: []*proto.BlockInfo{
				{Header: ethHeaderToProtoForTest(hdr100), TotalDifficulty: 500, Author: protoutil.ConvertAddressToH160(author100)},
				{Header: ethHeaderToProtoForTest(hdr101), TotalDifficulty: 600, Author: protoutil.ConvertAddressToH160(author101)},
			},
		}, nil)

		headers, tds, authors, err := grpcClient.GetBlockInfoInBatch(context.Background(), 100, 101)
		require.NoError(t, err)
		require.Len(t, headers, 2)
		require.Len(t, tds, 2)
		require.Len(t, authors, 2)
		require.Equal(t, hdr100.Hash(), headers[0].Hash())
		require.Equal(t, hdr101.Hash(), headers[1].Hash())
		require.Equal(t, uint64(500), tds[0])
		require.Equal(t, uint64(600), tds[1])
		require.Equal(t, author100, authors[0])
		require.Equal(t, author101, authors[1])

		mockClient.AssertExpectations(t)
	})

	t.Run("empty response", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		mockClient.On("GetBlockInfoInBatch", mock.Anything, mock.Anything).
			Return(&proto.GetBlockInfoInBatchResponse{Blocks: nil}, nil)

		headers, tds, authors, err := grpcClient.GetBlockInfoInBatch(context.Background(), 100, 105)
		require.NoError(t, err)
		require.Empty(t, headers)
		require.Empty(t, tds)
		require.Empty(t, authors)
	})

	t.Run("rpc error propagated", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		mockClient.On("GetBlockInfoInBatch", mock.Anything, mock.Anything).
			Return(nil, errors.New("batch error"))

		_, _, _, err := grpcClient.GetBlockInfoInBatch(context.Background(), 100, 101)
		require.Error(t, err)
		require.Contains(t, err.Error(), "batch error")
	})

	t.Run("invalid range rejected locally", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		_, _, _, err := grpcClient.GetBlockInfoInBatch(context.Background(), -1, 5)
		require.Error(t, err)

		_, _, _, err = grpcClient.GetBlockInfoInBatch(context.Background(), 10, 5)
		require.Error(t, err)

		mockClient.AssertNotCalled(t, "GetBlockInfoInBatch", mock.Anything, mock.Anything)
	})
}

// TestGetBlockInfoInBatch_RangeBoundary verifies the conditional boundary on the
// range validation, ensuring each boundary is covered.
func TestGetBlockInfoInBatch_RangeBoundary(t *testing.T) {
	t.Parallel()

	// return a mock that responds to any GetBlockInfoInBatch call.
	makeClientWithEmptyResponse := func() (*MockBorApiClient, *BorGRPCClient) {
		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}
		mockClient.On("GetBlockInfoInBatch", mock.Anything, mock.Anything).
			Return(&proto.GetBlockInfoInBatchResponse{Blocks: nil}, nil)
		return mockClient, grpcClient
	}

	// end <= start would reject start==end=50.
	t.Run("start_equals_end_allowed single block range", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}
		hdr := makeTestHeader(t, 50)

		mockClient.On("GetBlockInfoInBatch", mock.Anything, mock.MatchedBy(func(req *proto.GetBlockInfoInBatchRequest) bool {
			return req.StartBlockNumber == 50 && req.EndBlockNumber == 50
		})).Return(&proto.GetBlockInfoInBatchResponse{
			Blocks: []*proto.BlockInfo{
				{Header: ethHeaderToProtoForTest(hdr), TotalDifficulty: 100, Author: nil},
			},
		}, nil)

		headers, tds, authors, err := grpcClient.GetBlockInfoInBatch(context.Background(), 50, 50)
		require.NoError(t, err, "start==end is a valid single-block range and must not be rejected")
		require.Len(t, headers, 1)
		require.Len(t, tds, 1)
		require.Len(t, authors, 1)
		require.Equal(t, hdr.Hash(), headers[0].Hash())
		require.Equal(t, uint64(100), tds[0])
		mockClient.AssertExpectations(t)
	})

	// start <= 0 would reject start=0 (genesis block range).
	t.Run("zero_start_allowed genesis range", func(t *testing.T) {
		t.Parallel()

		_, grpcClient := makeClientWithEmptyResponse()
		_, _, _, err := grpcClient.GetBlockInfoInBatch(context.Background(), 0, 5)
		require.NoError(t, err, "start=0 is valid (genesis block) and must not be rejected")
	})

	// end <= 0 would reject end=0.
	t.Run("zero_end_allowed single genesis block", func(t *testing.T) {
		t.Parallel()

		_, grpcClient := makeClientWithEmptyResponse()
		_, _, _, err := grpcClient.GetBlockInfoInBatch(context.Background(), 0, 0)
		require.NoError(t, err, "end=0 with start=0 is valid (single genesis block) and must not be rejected")
	})

	t.Run("negative_start_rejected", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		_, _, _, err := grpcClient.GetBlockInfoInBatch(context.Background(), -1, 10)
		require.Error(t, err, "negative start must be rejected")
		mockClient.AssertNotCalled(t, "GetBlockInfoInBatch")
	})

	t.Run("negative_end_rejected", func(t *testing.T) {
		t.Parallel()

		mockClient := new(MockBorApiClient)
		grpcClient := &BorGRPCClient{client: mockClient}

		_, _, _, err := grpcClient.GetBlockInfoInBatch(context.Background(), 0, -1)
		require.Error(t, err, "negative end must be rejected")
		mockClient.AssertNotCalled(t, "GetBlockInfoInBatch")
	})
}

// TestGetBlockInfoInBatch_NilHeaderBreak verifies that when the server returns a
// BlockInfo with nil Header inside the list, iteration stops at that point.
func TestGetBlockInfoInBatch_NilHeaderBreak(t *testing.T) {
	t.Parallel()

	mockClient := new(MockBorApiClient)
	grpcClient := &BorGRPCClient{client: mockClient}
	hdr100 := makeTestHeader(t, 100)

	// Server returns block 100 with a valid header, then block 101 with a nil Header.
	// Only block 100 must be returned.
	mockClient.On("GetBlockInfoInBatch", mock.Anything, mock.Anything).
		Return(&proto.GetBlockInfoInBatchResponse{
			Blocks: []*proto.BlockInfo{
				{Header: ethHeaderToProtoForTest(hdr100), TotalDifficulty: 500, Author: nil},
				{Header: nil, TotalDifficulty: 600, Author: nil}, // nil Header — must trigger break
			},
		}, nil)

	headers, tds, authors, err := grpcClient.GetBlockInfoInBatch(context.Background(), 100, 101)
	require.NoError(t, err)
	require.Len(t, headers, 1, "iteration must stop at the nil-header block")
	require.Len(t, tds, 1)
	require.Len(t, authors, 1)
	require.Equal(t, hdr100.Hash(), headers[0].Hash())
}

// TestProtoHeaderToEthHeader_NilInput verifies that protoHeaderToEthHeader(nil) returns nil.
func TestProtoHeaderToEthHeader_NilInput(t *testing.T) {
	t.Parallel()

	result := protoHeaderToEthHeader(nil)
	require.Nil(t, result, "protoHeaderToEthHeader(nil) must return nil")
}

// makeTestHeader builds a header for batch tests.
func makeTestHeader(t *testing.T, num uint64) *ethTypes.Header {
	t.Helper()
	return &ethTypes.Header{
		ParentHash:  common.BigToHash(big.NewInt(int64(num - 1))),
		UncleHash:   ethTypes.EmptyUncleHash,
		Root:        common.BigToHash(big.NewInt(int64(num + 1000))),
		TxHash:      common.BigToHash(big.NewInt(int64(num + 2000))),
		ReceiptHash: common.BigToHash(big.NewInt(int64(num + 3000))),
		Difficulty:  big.NewInt(1),
		Number:      new(big.Int).SetUint64(num),
		GasLimit:    30_000_000,
		Time:        1_700_000_000 + num,
		Extra:       []byte("test"),
	}
}

func ethHeaderToProtoForTest(h *ethTypes.Header) *proto.Header {
	out := &proto.Header{
		Number:      h.Number.Uint64(),
		ParentHash:  protoutil.ConvertHashToH256(h.ParentHash),
		Time:        h.Time,
		UncleHash:   protoutil.ConvertHashToH256(h.UncleHash),
		Coinbase:    protoutil.ConvertAddressToH160(h.Coinbase),
		StateRoot:   protoutil.ConvertHashToH256(h.Root),
		TxRoot:      protoutil.ConvertHashToH256(h.TxHash),
		ReceiptRoot: protoutil.ConvertHashToH256(h.ReceiptHash),
		Bloom:       h.Bloom.Bytes(),
		GasLimit:    h.GasLimit,
		GasUsed:     h.GasUsed,
		ExtraData:   append([]byte(nil), h.Extra...),
		MixDigest:   protoutil.ConvertHashToH256(h.MixDigest),
		Nonce:       h.Nonce[:],
	}
	if h.Difficulty != nil {
		out.Difficulty = h.Difficulty.Bytes()
	}
	if h.BaseFee != nil {
		out.BaseFee = h.BaseFee.Bytes()
	}
	if h.WithdrawalsHash != nil {
		out.WithdrawalsHash = protoutil.ConvertHashToH256(*h.WithdrawalsHash)
	}
	// BlobGasUsed / ExcessBlobGas are proto3 optional
	// We use *uint64 as the pointer copy to preserve nil vs. zero.
	out.BlobGasUsed = h.BlobGasUsed
	out.ExcessBlobGas = h.ExcessBlobGas
	if h.ParentBeaconRoot != nil {
		out.ParentBeaconBlockRoot = protoutil.ConvertHashToH256(*h.ParentBeaconRoot)
	}
	if h.RequestsHash != nil {
		out.RequestsHash = protoutil.ConvertHashToH256(*h.RequestsHash)
	}
	return out
}
