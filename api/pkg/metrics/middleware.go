package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RouteLookupFunc func(*http.Request) string

var httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "http_requests_total",
	Help: "Count of all HTTP requests",
}, []string{"code", "method"})

var httpRequestsDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "A histogram of latencies for requests.",
		Buckets: []float64{.005, .01, .025, .05, 0.1, 0.25, 0.5, 1, 1.25, 1.85, 2, 5},
	},
	[]string{"method", "route"},
)

func RegisterHttpVecs() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestsDuration)
}

// InstrumentHandler wraps the passed handler with prometheus duration and counter tracking.
func InstrumentHandler(next http.Handler, lookupRoute RouteLookupFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		promhttp.InstrumentHandlerCounter(httpRequestsTotal, next).ServeHTTP(w, r)
		httpRequestsDuration.With(prometheus.Labels{"route": lookupRoute(r), "method": r.Method}).Observe(time.Since(start).Seconds())
	}
}
