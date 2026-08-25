package helper

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	logger "cosmossdk.io/log"
	"github.com/quic-go/quic-go/http3"
)

type quicSidecarConfig struct {
	listenAddr string
	upstream   *url.URL
	tlsConfig  *tls.Config
}

func StartQUICSidecar(ctx context.Context, log logger.Logger) error {
	cfg, err := buildQUICSidecarConfig(conf)
	if err != nil {
		return err
	}
	if cfg == nil {
		return nil
	}

	proxy := httputil.NewSingleHostReverseProxy(cfg.upstream)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("QUIC sidecar upstream request failed", "path", r.URL.Path, "error", err)
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}

	server := &http3.Server{
		Addr:      cfg.listenAddr,
		TLSConfig: cfg.tlsConfig,
		Handler:   proxy,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Info("Starting Heimdall QUIC sidecar", "listenAddr", cfg.listenAddr, "upstream", cfg.upstream.String())
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) && ctx.Err() != nil {
		return nil
	}

	return err
}

func buildQUICSidecarConfig(cfg CustomAppConfig) (*quicSidecarConfig, error) {
	if !cfg.Custom.EnableQUICSidecar {
		return nil, nil
	}

	listenAddr, err := deriveQUICSidecarListenAddr(cfg.API.Address, cfg.Custom.QUICSidecarListenAddr)
	if err != nil {
		return nil, err
	}

	upstream, err := resolveQUICSidecarUpstream(cfg.API.Address, cfg.Custom.QUICSidecarUpstreamURL)
	if err != nil {
		return nil, err
	}

	tlsConfig, err := resolveQUICSidecarTLSConfig(
		listenAddr,
		cfg.Custom.QUICSidecarCertFile,
		cfg.Custom.QUICSidecarKeyFile,
	)
	if err != nil {
		return nil, err
	}

	return &quicSidecarConfig{
		listenAddr: listenAddr,
		upstream:   upstream,
		tlsConfig:  tlsConfig,
	}, nil
}

func deriveQUICSidecarListenAddr(apiAddr, override string) (string, error) {
	if override != "" {
		return override, nil
	}

	u, err := normalizeHeimdallHTTPURL(apiAddr)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		return "", fmt.Errorf("unable to derive QUIC sidecar listen address from %q", apiAddr)
	}

	return u.Host, nil
}

func resolveQUICSidecarUpstream(apiAddr, override string) (*url.URL, error) {
	target := override
	if target == "" {
		target = apiAddr
	}

	return normalizeHeimdallHTTPURL(target)
}

func normalizeHeimdallHTTPURL(raw string) (*url.URL, error) {
	switch {
	case strings.HasPrefix(raw, "tcp://"):
		raw = "http://" + strings.TrimPrefix(raw, "tcp://")
	case !strings.Contains(raw, "://"):
		raw = "http://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported QUIC sidecar upstream scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("missing host in QUIC sidecar URL %q", raw)
	}

	return u, nil
}

func resolveQUICSidecarTLSConfig(listenAddr, certFile, keyFile string) (*tls.Config, error) {
	switch {
	case certFile == "" && keyFile == "":
		if !isLoopbackListenAddr(listenAddr) {
			return nil, fmt.Errorf("quic sidecar requires cert and key files for non-loopback listener %q", listenAddr)
		}
		cert, err := generateQUICSidecarCertificate(listenAddr)
		if err != nil {
			return nil, err
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS13,
			NextProtos:   []string{http3.NextProtoH3},
		}, nil
	case certFile == "" || keyFile == "":
		return nil, errors.New("quic sidecar cert and key files must be configured together")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		NextProtos:   []string{http3.NextProtoH3},
	}, nil
}

func isLoopbackListenAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	switch host {
	case "", "localhost":
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	return ip.IsLoopback()
}

func generateQUICSidecarCertificate(listenAddr string) (tls.Certificate, error) {
	host, _, err := net.SplitHostPort(listenAddr)
	if err != nil {
		host = listenAddr
	}
	if host == "" {
		host = "localhost"
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "heimdall-quic-sidecar",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if host == "localhost" {
		template.DNSNames = []string{"localhost"}
		template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
	} else if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  priv,
	}, nil
}
