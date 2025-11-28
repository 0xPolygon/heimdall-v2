package grpc

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	proto "github.com/0xPolygon/polyproto/bor"
)

type BorGRPCClient struct {
	conn   *grpc.ClientConn
	client proto.BorApiClient
}

func NewBorGRPCClient(address string) *BorGRPCClient {
	timeout := 30 * time.Second
	addr := address
	var dialOpts []grpc.DialOption
	log.Info("Setting up Bor gRPC client", "address", address)

	// URL mode
	if strings.Contains(address, "://") {
		// Decide credentials and normalized address based on the provided scheme
		u, err := url.Parse(address)
		if err != nil {
			log.Crit("Invalid Bor gRPC URL", "url", address, "err", err)
		}

		switch u.Scheme {
		case "https":
			// Remote secure connection
			addr = u.Host
			if addr == "" {
				log.Crit("Invalid Bor gRPC https URL", "url", address)
			}

			tlsCfg := &tls.Config{
				ServerName: strings.Split(addr, ":")[0],
				MinVersion: tls.VersionTLS12,
			}
			dialOpts = append(dialOpts,
				grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)),
			)

		case "http":
			// plaintext only allowed for local host
			addr = u.Host
			if !isLocalhost(addr) {
				log.Crit("Refusing insecure non-local Bor gRPC over http; use https or localhost only",
					"addr", addr)
			}
			dialOpts = append(dialOpts,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)

		case "unix":
			// support unix://path for on-box Bor nodes
			path := u.Path
			if path == "" {
				log.Crit("Invalid unix Bor gRPC URL", "url", address)
			}
			addr = "unix://" + path
			dialOpts = append(dialOpts,
				grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
					return net.DialTimeout("unix", strings.TrimPrefix(addr, "unix://"), timeout)
				}),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)

		default:
			log.Crit("Unsupported Bor gRPC URL scheme", "url", address, "scheme", u.Scheme)
		}

	} else {
		// No scheme provided, treat as host:port, but only allow if local
		if !isLocalhost(addr) {
			log.Crit("Refusing insecure non-local Bor gRPC without scheme; use https://host:port",
				"addr", addr)
		}
		dialOpts = append(dialOpts,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}

	// Retry options
	retryOpts := []grpcRetry.CallOption{
		grpcRetry.WithMax(10000),
		grpcRetry.WithBackoff(grpcRetry.BackoffLinear(5 * time.Second)),
		grpcRetry.WithCodes(codes.Internal, codes.Unavailable, codes.Aborted, codes.NotFound),
	}

	dialOpts = append(dialOpts,
		grpc.WithStreamInterceptor(grpcRetry.StreamClientInterceptor(retryOpts...)),
		grpc.WithUnaryInterceptor(grpcRetry.UnaryClientInterceptor(retryOpts...)),
	)

	// dial using address and dialOpts
	conn, err := grpc.NewClient(addr, dialOpts...)
	if err != nil {
		log.Crit("Failed to connect to Bor gRPC", "addr", addr, "error", err)
	}

	log.Info("Connected to Bor gRPC server", "grpcAddress", address, "dialAddr", addr)

	return &BorGRPCClient{
		conn:   conn,
		client: proto.NewBorApiClient(conn),
	}
}

func (h *BorGRPCClient) Close() {
	log.Debug("Shutdown detected, Closing Bor gRPC client")
	err := h.conn.Close()
	if err != nil {
		return
	}
}

// isLocalhost returns true if host/port refers to localhost/loopback.
func isLocalhost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
