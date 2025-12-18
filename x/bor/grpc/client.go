package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"cosmossdk.io/log"
	proto "github.com/0xPolygon/polyproto/bor"
	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type BorGRPCClient struct {
	conn   *grpc.ClientConn
	client proto.BorApiClient
}

func NewBorGRPCClient(address string, logger log.Logger) (*BorGRPCClient, error) {
	timeout := 5 * time.Second
	addr := address
	var dialOpts []grpc.DialOption

	logger.Info("Setting up Bor gRPC client", "address", address)

	// URL mode
	if strings.Contains(address, "://") {
		// Decide credentials and normalized address based on the provided scheme
		u, err := url.Parse(address)
		if err != nil {
			logger.Error("Invalid Bor gRPC URL", "url", address, "err", err)
			return nil, err
		}

		switch u.Scheme {
		case "https":
			// Remote secure connection
			addr = u.Host
			if addr == "" {
				err := fmt.Errorf("invalid Bor gRPC https URL %q: empty host", address)
				logger.Error("Invalid Bor gRPC https URL", "url", address, "err", err)
				return nil, err
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
				logger.Warn("Using insecure non-local Bor gRPC over http. This is discouraged", "addr", addr)
			}
			dialOpts = append(dialOpts,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)

		case "unix":
			// support unix://path for on-box Bor nodes
			path := u.Path
			if path == "" {
				err := fmt.Errorf("invalid unix Bor gRPC URL %q: empty path", address)
				logger.Error("Invalid unix Bor gRPC URL", "url", address, "err", err)
				return nil, err
			}
			addr = "unix://" + path
			dialOpts = append(dialOpts,
				grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
					//nolint:noctx // used in grpc.WithContextDialer
					return net.DialTimeout("unix", strings.TrimPrefix(addr, "unix://"), timeout)
				}),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)

		default:
			err := fmt.Errorf("unsupported Bor gRPC URL scheme %q in %q", u.Scheme, address)
			logger.Error("Unsupported Bor gRPC URL scheme", "url", address, "scheme", u.Scheme, "err", err)
			return nil, err
		}

	} else {
		// No scheme provided, treat as host:port, but only allow if local
		if !isLocalhost(addr) {
			err := fmt.Errorf("insecure non-local Bor gRPC without scheme (addr=%s); use http://localhost:port or https://host:port", addr)
			logger.Error("Refusing insecure non-local Bor gRPC without scheme", "addr", addr, "err", err)
			return nil, err
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
		logger.Error("Failed to connect to Bor gRPC", "addr", addr, "error", err)
		return nil, err
	}

	logger.Info("Connected to Bor gRPC server", "grpcAddress", address, "dialAddr", addr)

	return &BorGRPCClient{
		conn:   conn,
		client: proto.NewBorApiClient(conn),
	}, nil
}

func (c *BorGRPCClient) Close(logger log.Logger) {
	if c == nil || c.conn == nil {
		return
	}
	logger.Debug("Shutdown detected, closing Bor gRPC client")
	_ = c.conn.Close()
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
