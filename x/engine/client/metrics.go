package client

import (
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "heimdall"
	subsystem = "engine"
)

var (
	// Track inbound bytes partitioned by remote host
	inboundBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "bytes_in",
			Help:      "Total bytes inbound",
		},
		[]string{"host"},
	)

	// Track outbound bytes partitioned by remote host
	outboundBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "bytes_out",
			Help:      "Total bytes outbound",
		},
		[]string{"host"},
	)

	// Track calls by RPC method
	rpcCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "calls",
			Help:      "Total number of RPC calls",
		},
		[]string{"method"},
	)

	// Track call durations by RPC method
	rpcDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "duration",
			Help:      "Histogram of RPC request durations",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	// Track errors by RPC method and error type
	rpcErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "errors",
			Help: "Total number of RPC errors encountered.",
		},
		[]string{"method", "error"},
	)
)

func init() {
	prometheus.MustRegister(inboundBytes)
	prometheus.MustRegister(outboundBytes)
	prometheus.MustRegister(rpcCalls)
	prometheus.MustRegister(rpcDuration)
	prometheus.MustRegister(rpcErrors)
}

var (
	metricsServer *http.Server
	metricsOnce   = sync.Once{}
)

// TODO: refactor this to be instantiated elsewhere & configurable.
// It should not clash with the bridge's use of same, they can share a metrics server.
func startMetricsServer() {
	metricsOnce.Do(func() {
		metricsServer = &http.Server{
			Addr:              ":2113",
			ReadTimeout:       1 * time.Second,
			ReadHeaderTimeout: 1 * time.Second,
			WriteTimeout:      1 * time.Second,
		}

		http.Handle("/metrics", promhttp.Handler())

		go func() {
			if mErr := metricsServer.ListenAndServe(); mErr != nil {
				log.Fatal("failed to start engine metrics server", "error", mErr)
			}
		}()
	})
}

type IOBytesMetrics struct {
	rc         io.ReadCloser
	onClose    func(totalBytes int64)
	totalBytes int64
}

func (m *IOBytesMetrics) Read(p []byte) (int, error) {
	n, err := m.rc.Read(p)
	m.totalBytes += int64(n)
	return n, err
}

func (m *IOBytesMetrics) Close() error {
	if m.onClose != nil {
		m.onClose(m.totalBytes)
	}
	return m.rc.Close()
}

type MetricsTransport struct {
	Transport http.RoundTripper
}

func (mt *MetricsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if mt.Transport == nil {
		mt.Transport = http.DefaultTransport
	}

	host := req.URL.Host

	if req.Body != nil {
		rc := req.Body
		req.Body = &IOBytesMetrics{
			rc: rc,
			onClose: func(totalBytes int64) {
				outboundBytes.WithLabelValues(host).Add(float64(totalBytes))
			},
		}
	}

	resp, err := mt.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if resp.Body != nil {
		rc := resp.Body
		resp.Body = &IOBytesMetrics{
			rc: rc,
			onClose: func(totalBytes int64) {
				inboundBytes.WithLabelValues(host).Add(float64(totalBytes))
			},
		}
	}

	return resp, nil
}
