package grpc

import (
	"context"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"cosmossdk.io/log"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/0xPolygon/heimdall-v2/x/bor/failover"
)

const (
	primaryGRPCEndpoint = 0
	genesisBlock        = int64(0)

	// primaryAnchorFailureThreshold mirrors the HTTP transport: a fallback may
	// provisionally establish the expected genesis only after the primary has
	// failed this many consecutive probes. The primary stays authoritative and
	// reclaims the expectation once reachable (see checkGenesis), so this only
	// lets failover engage during a window where the primary is unreachable.
	primaryAnchorFailureThreshold = 2
)

// Client is the behavior shared by the single-endpoint *BorGRPCClient and the
// failover *MultiBorGRPCClient, so helper code can hold either transparently.
type Client interface {
	GetRootHash(ctx context.Context, startBlock, endBlock uint64) (string, error)
	GetVoteOnHash(ctx context.Context, startBlock, endBlock uint64, rootHash, milestoneId string) (bool, error)
	HeaderByNumber(ctx context.Context, blockID int64) (*ethTypes.Header, error)
	BlockByNumber(ctx context.Context, blockID int64) (*ethTypes.Block, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*ethTypes.Receipt, error)
	BorBlockReceipt(ctx context.Context, txHash common.Hash) (*ethTypes.Receipt, error)
	GetAuthor(ctx context.Context, blockNum *big.Int) (*common.Address, error)
	GetTdByHash(ctx context.Context, hash common.Hash) (uint64, error)
	GetTdByNumber(ctx context.Context, blockNum *big.Int) (uint64, error)
	GetBlockInfoInBatch(ctx context.Context, start, end int64) ([]*ethTypes.Header, []uint64, []common.Address, error)
	Close(logger log.Logger)
}

// EndpointHeaderFetcher is the per-endpoint header API used by validation
// hooks. *BorGRPCClient satisfies it; tests can inject small stubs.
type EndpointHeaderFetcher interface {
	HeaderByNumber(ctx context.Context, blockID int64) (*ethTypes.Header, error)
}

// EndpointValidator is an optional probe-time validation hook. Returning an
// error keeps the endpoint out of the healthy candidate set.
type EndpointValidator func(context.Context, int, EndpointHeaderFetcher) error

var (
	_ Client = (*BorGRPCClient)(nil)
	_ Client = (*MultiBorGRPCClient)(nil)
)

// MultiBorGRPCClient wraps N priority-ordered Bor gRPC clients (index 0 =
// primary) and fails over to the next validated one on a transport error or a
// per-attempt timeout. A background prober reverts to a higher-priority
// endpoint once it recovers. It shares the failover core with the HTTP path.
type MultiBorGRPCClient struct {
	clients              []*BorGRPCClient
	health               *failover.Health
	attemptTimeout       time.Duration
	expectedGenesis      atomic.Pointer[common.Hash]
	expectedByPrimary    atomic.Bool
	primaryProbeFailures atomic.Int32
	validators           []EndpointValidator
}

// NewMultiBorGRPCClient wraps already-dialed, priority-ordered clients with
// failover and starts the background prober.
func NewMultiBorGRPCClient(clients []*BorGRPCClient, logger log.Logger, m failover.Metrics, attemptTimeout time.Duration, validators ...EndpointValidator) *MultiBorGRPCClient {
	if len(clients) < 1 {
		panic("bor failover: endpoint count must be positive")
	}
	mc := &MultiBorGRPCClient{clients: clients, attemptTimeout: attemptTimeout, validators: validators}
	mc.captureExpectedGenesis()
	mc.health = failover.New(len(clients), mc.probe, m, logger)
	mc.health.Start()
	return mc
}

// probe validates endpoint i for the background prober: it must return the
// genesis header whose hash matches the expected one before it is treated as a
// healthy fallback. The gRPC API has no chain-id call, so the genesis hash is
// the chain-identity anchor.
func (mc *MultiBorGRPCClient) probe(i int) error {
	ctx, cancel := context.WithTimeout(context.Background(), mc.health.ProbeTimeout())
	defer cancel()

	h, err := mc.clients[i].HeaderByNumber(ctx, genesisBlock)
	if err == nil && h == nil {
		err = fmt.Errorf("bor gRPC endpoint %d returned nil genesis header", i)
	}
	if err != nil {
		mc.recordProbeFailure(i)
		return err
	}

	reclaiming := i == primaryGRPCEndpoint && !mc.expectedByPrimary.Load()
	if err := mc.checkGenesis(i, h.Hash()); err != nil {
		return err
	}
	if err := mc.validateEndpoint(ctx, i); err != nil {
		mc.recordProbeFailure(i)
		return err
	}
	mc.recordProbeSuccess(i)
	if reclaiming {
		// The authoritative primary just (re)established the identity; if a stale
		// fallback was the active endpoint during the outage it may be serving
		// wrong-network data, so move traffic back to the primary at once instead
		// of waiting out the promotion threshold. Reclaim also demotes the other
		// endpoints so a fallback validated against the provisional identity can't
		// remain an in-call candidate until re-validated.
		mc.health.Reclaim(primaryGRPCEndpoint)
	}

	return nil
}

func (mc *MultiBorGRPCClient) recordProbeFailure(i int) {
	if i == primaryGRPCEndpoint {
		mc.primaryProbeFailures.Add(1)
	}
}

func (mc *MultiBorGRPCClient) recordProbeSuccess(i int) {
	if i == primaryGRPCEndpoint {
		mc.primaryProbeFailures.Store(0)
	}
}

func (mc *MultiBorGRPCClient) validateEndpoint(ctx context.Context, i int) error {
	for _, validate := range mc.validators {
		if validate == nil {
			continue
		}
		if err := validate(ctx, i, mc.clients[i]); err != nil {
			return err
		}
	}
	return nil
}

// checkGenesis compares endpoint i's genesis hash against the expected one. The
// primary is authoritative: whenever it answers it (re)establishes the
// expectation, reclaiming a provisional one a fallback set while the primary was
// unreachable — so the real primary is never rejected. A fallback may
// provisionally anchor only after the primary has been unreachable for
// primaryAnchorFailureThreshold probes (see canAnchor); every endpoint that does
// not anchor must match the current expectation.
func (mc *MultiBorGRPCClient) checkGenesis(i int, got common.Hash) error {
	if i == primaryGRPCEndpoint && !mc.expectedByPrimary.Load() {
		mc.expectedGenesis.Store(&got)
		mc.expectedByPrimary.Store(true)
		return nil
	}

	expected := mc.expectedGenesis.Load()
	if expected == nil {
		if !mc.canAnchor(i) {
			return fmt.Errorf("bor gRPC endpoint %d: expected genesis not yet known", i)
		}
		if mc.expectedGenesis.CompareAndSwap(nil, &got) {
			return nil
		}
		expected = mc.expectedGenesis.Load() // lost the race; compare against the winner
	}

	if *expected != got {
		return fmt.Errorf("bor gRPC endpoint %d genesis %s != expected %s", i, got.Hex(), expected.Hex())
	}

	return nil
}

// canAnchor reports whether a fallback may provisionally establish the expected
// genesis: only after the primary has failed primaryAnchorFailureThreshold
// consecutive probes, so failover still engages when the primary is never
// reachable at boot. The primary itself anchors via checkGenesis's reclaim path.
func (mc *MultiBorGRPCClient) canAnchor(i int) bool {
	return i == primaryGRPCEndpoint || mc.primaryProbeFailures.Load() >= primaryAnchorFailureThreshold
}

// captureExpectedGenesis best-effort records the primary's genesis hash at
// startup so fallbacks can be validated before the first request.
func (mc *MultiBorGRPCClient) captureExpectedGenesis() {
	ctx, cancel := context.WithTimeout(context.Background(), mc.attemptTimeout)
	defer cancel()

	h, err := mc.clients[primaryGRPCEndpoint].HeaderByNumber(ctx, genesisBlock)
	if err == nil && h != nil {
		got := h.Hash()
		if mc.expectedGenesis.CompareAndSwap(nil, &got) {
			mc.expectedByPrimary.Store(true)
		}
	}
}

// retriable reports whether err warrants failover: transport-level gRPC codes
// (including a per-attempt deadline) only. Logical conditions surface as
// ethereum.NotFound or validation errors (code Unknown) and are not retried.
func retriable(err error) bool {
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

func (mc *MultiBorGRPCClient) GetRootHash(ctx context.Context, startBlock, endBlock uint64) (string, error) {
	return failover.Call(mc.health, ctx, mc.attemptTimeout,
		func(ctx context.Context, i int) (string, error) {
			return mc.clients[i].GetRootHash(ctx, startBlock, endBlock)
		}, retriable)
}

func (mc *MultiBorGRPCClient) GetVoteOnHash(ctx context.Context, startBlock, endBlock uint64, rootHash, milestoneId string) (bool, error) {
	return failover.Call(mc.health, ctx, mc.attemptTimeout,
		func(ctx context.Context, i int) (bool, error) {
			return mc.clients[i].GetVoteOnHash(ctx, startBlock, endBlock, rootHash, milestoneId)
		}, retriable)
}

func (mc *MultiBorGRPCClient) HeaderByNumber(ctx context.Context, blockID int64) (*ethTypes.Header, error) {
	return failover.Call(mc.health, ctx, mc.attemptTimeout,
		func(ctx context.Context, i int) (*ethTypes.Header, error) {
			return mc.clients[i].HeaderByNumber(ctx, blockID)
		}, retriable)
}

func (mc *MultiBorGRPCClient) BlockByNumber(ctx context.Context, blockID int64) (*ethTypes.Block, error) {
	return failover.Call(mc.health, ctx, mc.attemptTimeout,
		func(ctx context.Context, i int) (*ethTypes.Block, error) {
			return mc.clients[i].BlockByNumber(ctx, blockID)
		}, retriable)
}

func (mc *MultiBorGRPCClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*ethTypes.Receipt, error) {
	return failover.Call(mc.health, ctx, mc.attemptTimeout,
		func(ctx context.Context, i int) (*ethTypes.Receipt, error) {
			return mc.clients[i].TransactionReceipt(ctx, txHash)
		}, retriable)
}

func (mc *MultiBorGRPCClient) BorBlockReceipt(ctx context.Context, txHash common.Hash) (*ethTypes.Receipt, error) {
	return failover.Call(mc.health, ctx, mc.attemptTimeout,
		func(ctx context.Context, i int) (*ethTypes.Receipt, error) {
			return mc.clients[i].BorBlockReceipt(ctx, txHash)
		}, retriable)
}

func (mc *MultiBorGRPCClient) GetAuthor(ctx context.Context, blockNum *big.Int) (*common.Address, error) {
	return failover.Call(mc.health, ctx, mc.attemptTimeout,
		func(ctx context.Context, i int) (*common.Address, error) {
			return mc.clients[i].GetAuthor(ctx, blockNum)
		}, retriable)
}

func (mc *MultiBorGRPCClient) GetTdByHash(ctx context.Context, hash common.Hash) (uint64, error) {
	return failover.Call(mc.health, ctx, mc.attemptTimeout,
		func(ctx context.Context, i int) (uint64, error) {
			return mc.clients[i].GetTdByHash(ctx, hash)
		}, retriable)
}

func (mc *MultiBorGRPCClient) GetTdByNumber(ctx context.Context, blockNum *big.Int) (uint64, error) {
	return failover.Call(mc.health, ctx, mc.attemptTimeout,
		func(ctx context.Context, i int) (uint64, error) {
			return mc.clients[i].GetTdByNumber(ctx, blockNum)
		}, retriable)
}

// batchResult bundles GetBlockInfoInBatch's three result slices so the generic
// failover.Call (single value + error) can carry them.
type batchResult struct {
	headers []*ethTypes.Header
	tds     []uint64
	authors []common.Address
}

func (mc *MultiBorGRPCClient) GetBlockInfoInBatch(ctx context.Context, start, end int64) ([]*ethTypes.Header, []uint64, []common.Address, error) {
	res, err := failover.Call(mc.health, ctx, mc.attemptTimeout,
		func(ctx context.Context, i int) (batchResult, error) {
			h, td, a, e := mc.clients[i].GetBlockInfoInBatch(ctx, start, end)
			return batchResult{headers: h, tds: td, authors: a}, e
		}, retriable)

	return res.headers, res.tds, res.authors, err
}

// Close stops the prober and closes every underlying client.
func (mc *MultiBorGRPCClient) Close(logger log.Logger) {
	mc.health.Stop()
	for _, c := range mc.clients {
		c.Close(logger)
	}
}
