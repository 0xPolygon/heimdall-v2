package engine

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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
			Name:    "http_request_duration_seconds",
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
			Name:    "rpc_call_duration_seconds",
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

type MetricsTransport struct {
	Transport http.RoundTripper
}

func (mt *MetricsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := mt.Transport.RoundTrip(req)
	duration := time.Since(start).Seconds()

	if err != nil {
		return nil, err
	}

	httpRequests.WithLabelValues(req.Method, fmt.Sprintf("%d", resp.StatusCode)).Inc()
	httpRequestDuration.WithLabelValues(req.Method, fmt.Sprintf("%d", resp.StatusCode)).Observe(duration)

	return resp, nil
}
