//go:build bench
// +build bench

// Benchmark for HTTP vs. gRPC comparison against a running kurtosis devnet.
// Env vars:
//   BOR_RPC_URL (required) — bor HTTP RPC URL (e.g., http://127.0.0.1:XXXXX)
//   BOR_GRPC_URL (required) — bor gRPC URL (e.g., http://127.0.0.1:XXXXX)
//   BOR_GRPC_TOKEN (optional) — bearer token when gRPC auth is enabled
//   BENCH_BLOCK_RANGE (optional, default 10:100) — "start:end" for batch tests
//
// Usage:
//   BOR_RPC_URL=... BOR_GRPC_URL=... go test -tags bench -run=^$ -bench=. -benchmem -benchtime=5s ./helper/

package helper

import (
	"context"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	borgrpc "github.com/0xPolygon/heimdall-v2/x/bor/grpc"
)

var (
	sinkBytes   []byte
	sinkBool    bool
	sinkHeader  *ethTypes.Header
	sinkBlock   *ethTypes.Block
	sinkAddress *common.Address
	sinkUint64  uint64
	sinkErr     error
)

func newBenchCaller(b *testing.B) (httpCC, grpcCC *ContractCaller) {
	b.Helper()
	borRPC := os.Getenv("BOR_RPC_URL")
	borGRPC := os.Getenv("BOR_GRPC_URL")
	if borRPC == "" || borGRPC == "" {
		b.Skip("BOR_RPC_URL and BOR_GRPC_URL must both be set")
	}
	token := os.Getenv("BOR_GRPC_TOKEN")

	rpcClient, err := rpc.Dial(borRPC)
	if err != nil {
		b.Fatalf("dial bor HTTP RPC: %v", err)
	}
	httpClient := ethclient.NewClient(rpcClient)

	gcl, err := borgrpc.NewBorGRPCClient(borGRPC, token, Logger)
	if err != nil {
		b.Fatalf("dial bor gRPC: %v", err)
	}

	httpCC = &ContractCaller{
		BorChainClient:    httpClient,
		BorChainRPCClient: rpcClient,
		BorChainTimeout:   10 * time.Second,
		BorChainGrpcFlag:  false,
	}
	grpcCC = &ContractCaller{
		BorChainClient:     httpClient,
		BorChainRPCClient:  rpcClient,
		BorChainTimeout:    10 * time.Second,
		BorChainGrpcFlag:   true,
		BorChainGrpcClient: gcl,
	}
	return httpCC, grpcCC
}

func parseBenchRange(b *testing.B) (uint64, uint64) {
	b.Helper()
	raw := os.Getenv("BENCH_BLOCK_RANGE")
	if raw == "" {
		return 10, 100
	}
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		b.Fatalf("BENCH_BLOCK_RANGE must be in the form start:end, got %q", raw)
	}
	startN, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		b.Fatalf("BENCH_BLOCK_RANGE start %q: %v", parts[0], err)
	}
	endN, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		b.Fatalf("BENCH_BLOCK_RANGE end %q: %v", parts[1], err)
	}
	if endN < startN {
		b.Fatalf("BENCH_BLOCK_RANGE end (%d) must be >= start (%d)", endN, startN)
	}
	return startN, endN
}

func latestHash(b *testing.B, cc *ContractCaller) common.Hash {
	b.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h, err := cc.BorChainClient.HeaderByNumber(ctx, nil)
	if err != nil {
		b.Fatalf("latest header: %v", err)
	}
	return h.Hash()
}

func BenchmarkM_GetBorChainBlock(b *testing.B) {
	httpCC, grpcCC := newBenchCaller(b)
	b.Run("http", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkHeader, sinkErr = httpCC.GetBorChainBlock(context.Background(), nil)
		}
	})
	b.Run("grpc", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkHeader, sinkErr = grpcCC.GetBorChainBlock(context.Background(), nil)
		}
	})
}

func BenchmarkM_GetBorChainBlockAuthor(b *testing.B) {
	httpCC, grpcCC := newBenchCaller(b)
	blockNum := big.NewInt(1)
	b.Run("http", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkAddress, sinkErr = httpCC.GetBorChainBlockAuthor(context.Background(), blockNum)
		}
	})
	b.Run("grpc", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkAddress, sinkErr = grpcCC.GetBorChainBlockAuthor(context.Background(), blockNum)
		}
	})
}

func BenchmarkM_GetBorChainBlockTd(b *testing.B) {
	httpCC, grpcCC := newBenchCaller(b)
	hash := latestHash(b, httpCC)
	b.Run("http", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkUint64, sinkErr = httpCC.GetBorChainBlockTd(context.Background(), hash)
		}
	})
	b.Run("grpc", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkUint64, sinkErr = grpcCC.GetBorChainBlockTd(context.Background(), hash)
		}
	})
}

func BenchmarkM_GetBorChainBlockInfoInBatch(b *testing.B) {
	httpCC, grpcCC := newBenchCaller(b)
	startU, endU := parseBenchRange(b)
	start, end := int64(startU), int64(endU)
	b.Run("http", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _, sinkErr = httpCC.GetBorChainBlockInfoInBatch(context.Background(), start, end)
		}
	})
	b.Run("grpc", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _, sinkErr = grpcCC.GetBorChainBlockInfoInBatch(context.Background(), start, end)
		}
	})
}

func BenchmarkM_GetRootHash(b *testing.B) {
	httpCC, grpcCC := newBenchCaller(b)
	start, end := parseBenchRange(b)
	const checkpointLen = uint64(256)
	b.Run("http", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkBytes, sinkErr = httpCC.GetRootHash(start, end, checkpointLen)
		}
	})
	b.Run("grpc", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkBytes, sinkErr = grpcCC.GetRootHash(start, end, checkpointLen)
		}
	})
}

func BenchmarkM_CheckIfBlocksExist(b *testing.B) {
	httpCC, grpcCC := newBenchCaller(b)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h, err := httpCC.BorChainClient.HeaderByNumber(ctx, nil)
	if err != nil {
		b.Fatalf("latest header: %v", err)
	}
	num := h.Number.Uint64()
	b.Run("http", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkBool, sinkErr = httpCC.CheckIfBlocksExist(num)
		}
	})
	b.Run("grpc", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkBool, sinkErr = grpcCC.CheckIfBlocksExist(num)
		}
	})
}

func BenchmarkM_GetBlockByNumber(b *testing.B) {
	httpCC, grpcCC := newBenchCaller(b)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h, err := httpCC.BorChainClient.HeaderByNumber(ctx, nil)
	if err != nil {
		b.Fatalf("latest header: %v", err)
	}
	num := h.Number.Uint64()
	b.Run("http", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkBlock, sinkErr = httpCC.GetBlockByNumber(context.Background(), num)
		}
	})
	b.Run("grpc", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sinkBlock, sinkErr = grpcCC.GetBlockByNumber(context.Background(), num)
		}
	})
}
