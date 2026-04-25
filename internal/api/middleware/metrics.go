package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "oidc_requests_total",
		Help: "Total number of OIDC requests.",
	}, []string{"path", "method", "status"})

	signingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "signing_duration_seconds",
		Help:    "Duration of cryptographic signing operations.",
		Buckets: prometheus.DefBuckets,
	}, []string{"algorithm"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests.",
		Buckets: prometheus.DefBuckets,
	}, []string{"path"})
)

// Metrics records Prometheus metrics for each request.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Use a response wrapper to capture status code
		rw := &metricsResponseWriter{ResponseWriter: w, status: http.StatusOK}
		
		next.ServeHTTP(rw, r)
		
		duration := time.Since(start).Seconds()
		path := r.URL.Path
		status := strconv.Itoa(rw.status)
		
		httpRequestsTotal.WithLabelValues(path, r.Method, status).Inc()
		requestDuration.WithLabelValues(path).Observe(duration)
	})
}

// ObserveSigning records the duration of a signing operation.
func ObserveSigning(algo string, duration time.Duration) {
	signingDuration.WithLabelValues(algo).Observe(duration.Seconds())
}

type metricsResponseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
