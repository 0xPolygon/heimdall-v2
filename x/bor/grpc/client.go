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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type BorGRPCClient struct {
	conn   *grpc.ClientConn
	client proto.BorApiClient
}

const (
	// dialTimeout caps the per-attempt timeout for non-HTTP callers (currently just unix).
	dialTimeout = 5 * time.Second

	// MaxBlockInfoBatchSize caps GetBlockInfoInBatch inputs to match bor's
	// server-side cap. This avoids HTTP and gRPC paths to drift.
	MaxBlockInfoBatchSize = 256
)

func NewBorGRPCClient(address, token string, logger log.Logger) (*BorGRPCClient, error) {
	logger.Info("Setting up Bor gRPC client", "address", address)

	addr, transportOpts, isTLS, err := resolveTransport(address, token, logger)
	if err != nil {
		return nil, err
	}

	dialOpts := append([]grpc.DialOption(nil), transportOpts...)
	// Flipping `!= ""` to `== ""` attaches credentials with an empty token; the token+plaintext reject path is tested via resolveHTTP.
	// mutator-disable-next-line token-presence guard
	if token != "" {
		dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(bearerToken{token: token, requireSecurity: isTLS}))
	}

	conn, err := grpc.NewClient(addr, dialOpts...)
	// NewClient only errors on malformed URI, which resolveTransport has already validated.
	// mutator-disable-next-line defensive grpc.NewClient error guard
	if err != nil {
		// Operator-log line inside the unreachable error branch.
		// mutator-disable-next-line statement-deletion on log
		logger.Error("Failed to connect to Bor gRPC", "addr", addr, "error", err)
		// Return value inside the unreachable error branch; err is carried anyway.
		// mutator-disable-next-line return-value in unreachable branch
		return nil, err
	}

	// Operator-log announcing successful wiring; removal is silent but not a behavior change.
	// mutator-disable-next-line statement-deletion on log
	logger.Info("Connected to Bor gRPC server", "grpcAddress", address, "dialAddr", addr)
	return &BorGRPCClient{
		conn:   conn,
		client: proto.NewBorApiClient(conn),
	}, nil
}

// resolveTransport parses the address, picks the right credentials, and
// returns (dialAddr, transportDialOpts, isTLS, error).
func resolveTransport(address, token string, logger log.Logger) (string, []grpc.DialOption, bool, error) {
	if !strings.Contains(address, "://") {
		// Bare host:port — only allowed if localhost.
		return resolveNoScheme(address, logger)
	}
	u, err := url.Parse(address)
	if err != nil {
		// mutator-disable-next-line operator-log line for an already-error-returning branch
		logger.Error("Invalid Bor gRPC URL", "url", address, "err", err)
		return "", nil, false, err
	}
	switch u.Scheme {
	case "https":
		return resolveHTTPS(u, address, logger)
	case "http":
		return resolveHTTP(u, token, logger)
	case "unix":
		return resolveUnix(u, address, logger)
	default:
		err := fmt.Errorf("unsupported Bor gRPC URL scheme %q in %q", u.Scheme, address)
		// mutator-disable-next-line operator-log line; the err returned below carries the same message
		logger.Error("Unsupported Bor gRPC URL scheme", "url", address, "scheme", u.Scheme, "err", err)
		return "", nil, false, err
	}
}

func resolveHTTPS(u *url.URL, address string, logger log.Logger) (string, []grpc.DialOption, bool, error) {
	addr := u.Host
	if addr == "" {
		err := fmt.Errorf("invalid Bor gRPC https URL %q: empty host", address)
		// mutator-disable-next-line operator-log line; err below carries the same message
		logger.Error("Invalid Bor gRPC https URL", "url", address, "err", err)
		return "", nil, false, err
	}

	tlsCfg := &tls.Config{
		ServerName: u.Hostname(),
		MinVersion: tls.VersionTLS12,
	}
	return addr, []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))}, true, nil
}

func resolveHTTP(u *url.URL, token string, logger log.Logger) (string, []grpc.DialOption, bool, error) {
	addr := u.Host
	if addr == "" {
		err := fmt.Errorf("invalid Bor gRPC http URL %q: empty host", u.String())
		// Operator-log line; err below carries the same content.
		// mutator-disable-next-line operator-log line
		logger.Error("Invalid Bor gRPC http URL", "url", u.String(), "err", err)
		return "", nil, false, err
	}
	if !isLocalhost(addr) {
		// Bearer token + non-local plaintext would leak the token.
		if token != "" {
			err := fmt.Errorf("refusing to send bor gRPC bearer token over non-local plaintext http (addr=%s); use https:// for remote hosts", addr)
			// mutator-disable-next-line operator-log line; err below carries the same content
			logger.Error("Refusing bor gRPC bearer token over plaintext", "addr", addr, "err", err)
			return "", nil, false, err
		}
		// mutator-disable-next-line operator-log line advising against the insecure config path
		logger.Warn("Using insecure non-local Bor gRPC over http. This is discouraged", "addr", addr)
	}
	return addr, []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, false, nil
}

func resolveUnix(u *url.URL, address string, logger log.Logger) (string, []grpc.DialOption, bool, error) {
	if u.Path == "" {
		err := fmt.Errorf("invalid unix Bor gRPC URL %q: empty path", address)
		// mutator-disable-next-line operator-log line; err below carries the same content
		logger.Error("Invalid unix Bor gRPC URL", "url", address, "err", err)
		return "", nil, false, err
	}
	addr := "unix://" + u.Path
	dialer := &net.Dialer{Timeout: dialTimeout}
	opts := []grpc.DialOption{
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", strings.TrimPrefix(addr, "unix://"))
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	return addr, opts, false, nil
}

func resolveNoScheme(addr string, logger log.Logger) (string, []grpc.DialOption, bool, error) {
	if !isLocalhost(addr) {
		err := fmt.Errorf("insecure non-local Bor gRPC without scheme (addr=%s); use http://localhost:port or https://host:port", addr)
		// mutator-disable-next-line operator-log line; err below carries the same content
		logger.Error("Refusing insecure non-local Bor gRPC without scheme", "addr", addr, "err", err)
		return "", nil, false, err
	}
	return addr, []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, false, nil
}

// bearerToken implements credentials.PerRPCCredentials, attaching the
// configured bearer token to every gRPC call as the `authorization` header.
type bearerToken struct {
	token           string
	requireSecurity bool // true when the underlying transport is TLS
}

func (b bearerToken) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + b.token,
	}, nil
}

// RequireTransportSecurity returns whether per-RPC credentials require TLS.
// We only require TLS when heimdall was configured with https:// gRPC URL,
// otherwise we'd refuse to connect to localhost over plaintext.
func (b bearerToken) RequireTransportSecurity() bool {
	return b.requireSecurity
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
