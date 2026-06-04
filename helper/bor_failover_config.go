package helper

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/0xPolygon/heimdall-v2/metrics"
	borgrpc "github.com/0xPolygon/heimdall-v2/x/bor/grpc"
)

// parseURLs splits a comma-separated URL string into a trimmed, non-empty,
// priority-ordered slice (first = primary). Used for the Bor RPC/gRPC failover
// endpoint lists; a single URL yields a one-element slice (no failover).
func parseURLs(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}

	return out
}

// initBorRPCClient sets the borRPCClient/borClient globals. A comma-separated
// bor_rpc_url enables HTTP failover across the priority-ordered endpoints
// (index 0 = primary); a single URL keeps the plain dial path unchanged. It
// warns (never fatals) if the primary is unreachable at startup.
func initBorRPCClient() {
	var err error

	borRPCUrls := parseURLs(conf.Custom.BorRPCUrl)
	if len(borRPCUrls) >= 2 {
		if borRPCClient, borClient, err = newBorHTTPFailoverClient(borRPCUrls, conf.Custom.BorRPCTimeout); err != nil {
			log.Fatal("unable to set up bor RPC failover client", "URLs", redactURLs(conf.Custom.BorRPCUrl), "error", err)
		}
	} else {
		if borRPCClient, err = rpc.Dial(conf.Custom.BorRPCUrl); err != nil {
			log.Fatal("unable to dial bor chain RPC client", "URL", redactURL(conf.Custom.BorRPCUrl), "error", err)
		}
		borClient = ethclient.NewClient(borRPCClient)
	}

	warnIfBorRPCInaccessible(borClient, conf.Custom.BorRPCTimeout, redactURLs(conf.Custom.BorRPCUrl))
}

// initBorGRPCClient sets the borGRPCClient global when bor gRPC is enabled,
// supporting comma-separated bor_grpc_url failover, and runs the primary's
// startup reachability and HTTP/gRPC hash-parity checks.
func initBorGRPCClient() {
	if !conf.Custom.BorGRPCFlag {
		return
	}

	grpcURLs := parseURLs(conf.Custom.BorGRPCUrl)
	if len(grpcURLs) == 0 {
		log.Fatal("bor gRPC is enabled but bor_grpc_url is empty")
	}

	primaryGRPC, grpcClient, err := buildBorGRPCClient(grpcURLs, conf.Custom.BorGRPCToken, conf.Custom.BorRPCTimeout)
	if err != nil {
		log.Fatal("unable to create bor gRPC client", "URL", redactURLs(conf.Custom.BorGRPCUrl), "error", err)
	}

	borGRPCClient = grpcClient
	warnIfBorGRPCInaccessible(primaryGRPC, conf.Custom.BorRPCTimeout, redactURL(grpcURLs[0]))
	// Fire-and-forget parity goroutine; removal is only observable in production init.
	// mutator-disable-next-line statement-deletion in production init
	verifyBorGRPCHashParity(borClient, primaryGRPC, conf.Custom.BorRPCTimeout)
}

// buildBorGRPCClient dials each priority-ordered gRPC URL and returns the
// primary concrete client (for the startup reachability and hash-parity checks)
// plus the client that serves traffic: the single client when one URL is
// configured, or a failover wrapper when several are. An invalid/undialable URL
// is skipped (mirroring the HTTP path) so a single bad fallback can't block
// startup; it errors only when no endpoint survives.
func buildBorGRPCClient(urls []string, token string, attemptTimeout time.Duration) (*borgrpc.BorGRPCClient, borgrpc.Client, error) {
	clients := make([]*borgrpc.BorGRPCClient, 0, len(urls))
	for _, u := range urls {
		c, err := borgrpc.NewBorGRPCClient(u, token, Logger)
		if err != nil {
			Logger.Warn("bor failover: skipping invalid bor gRPC URL", "url", redactURL(u), "error", err)
			continue
		}
		clients = append(clients, c)
	}

	if len(clients) == 0 {
		return nil, nil, fmt.Errorf("no valid bor gRPC endpoints among %d configured", len(urls))
	}

	if len(clients) == 1 {
		return clients[0], clients[0], nil
	}

	return clients[0], borgrpc.NewMultiBorGRPCClient(clients, Logger, metrics.BorFailover("grpc"), attemptTimeout), nil
}
