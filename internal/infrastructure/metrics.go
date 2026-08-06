// Package infrastructure provides shared foundational components for all
// business modules: configuration loading, database connectivity, unit-of-work
// transactions, structured logging, Prometheus metrics, and HTTP routing.
package infrastructure

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// rate(http_requests_total{job="zhisuo-server"}[5m])
// histogram_quantile(0.99, rate(http_request_duration_seconds_bucket{job="zhisuo-server"}[5m]))
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests by method, path, and response status.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds by method, path, and response status.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
)

var reg = prometheus.NewRegistry()

// InitMetrics registers the Prometheus HTTP metrics (counters and histograms) and Go runtime collectors.
func InitMetrics() {
	reg.MustRegister(
		httpRequestsTotal,
		httpRequestDurationSeconds,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
}

func metricsHandler() gin.HandlerFunc {
	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// MetricsMiddleware returns a Gin handler that records request count and duration, attaching exemplars when a trace_id is present.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" || strings.HasPrefix(c.Request.URL.Path, "/swagger/") {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		method := c.Request.Method
		labels := prometheus.Labels{
			"method": method,
			"path":   path,
			"status": status,
		}

		httpRequestsTotal.With(labels).Inc()

		obs := httpRequestDurationSeconds.With(labels)
		if eo, ok := obs.(prometheus.ExemplarObserver); ok {
			if id, ok := c.Request.Context().Value(reqIDKey).(string); ok && id != "" {
				eo.ObserveWithExemplar(time.Since(start).Seconds(), prometheus.Labels{"trace_id": id})
				return
			}
		}
		obs.Observe(time.Since(start).Seconds())
	}
}
