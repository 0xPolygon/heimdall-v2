package engine

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// For the HTTP transport
	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests",
			Help: "Total number of HTTP requests made.",
		},
		[]string{"method", "status_code"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_ms",
			Help:    "Histogram of HTTP request durations.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status_code"},
	)

	// For specific RPC calls
	rpcCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_calls",
			Help: "Total number of RPC calls made.",
		},
		[]string{"rpc"},
	)

	rpcCallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rpc_call_duration_ms",
			Help:    "Histogram of HTTP request durations.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)
	rpcErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_errors",
			Help: "Total number of RPC errors encountered.",
		},
		[]string{"rpc", "error"},
	)
)

func init() {
	prometheus.MustRegister(httpRequests)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(rpcCalls)
	prometheus.MustRegister(rpcCallDuration)
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

type MetricsTransport struct {
	Transport http.RoundTripper
}

func (mt *MetricsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := mt.Transport.RoundTrip(req)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		return nil, err
	}

	httpRequests.WithLabelValues(req.Method, fmt.Sprintf("%d", resp.StatusCode)).Inc()
	httpRequestDuration.WithLabelValues(req.Method, fmt.Sprintf("%d", resp.StatusCode)).Observe(float64(duration))

	return resp, nil
}
