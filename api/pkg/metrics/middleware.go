package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"sync"
)

var httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "http_requests_total",
	Help: "Count of all HTTP requests",
}, []string{"code", "method"})

var durationVects = make(map[string]*prometheus.HistogramVec)
var durationVectsRWLock = &sync.RWMutex{}

var registry = initRegistry()

func initRegistry() *prometheus.Registry {
	r := prometheus.NewRegistry()
	r.MustRegister(httpRequestsTotal)

	return r
}

func getOrSetDurationVectForPath(path string) *prometheus.HistogramVec {
	// so, plan is: lets do some rw lookup whether we already have a corresponding duration registered,
	// else we'll create and register one.
	durationVectsRWLock.RLock()

	var duration *prometheus.HistogramVec

	if val, ok := durationVects[path]; ok {
		duration = val
	}

	// can't defer because we cant predict whether we will upgrade the lock
	durationVectsRWLock.RUnlock()

	if duration != nil {
		return duration
	}

	// didnt found existing -> lets create one
	durationVectsRWLock.Lock()
	defer durationVectsRWLock.Unlock()

	// since we could have a race condition between RUnlock() and Lock() we have to double check
	if val, ok := durationVects[path]; ok {
		return val
	}

	duration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "A histogram of latencies for requests.",
			Buckets:     []float64{.01, .025, .05, 0.1, 0.25, 0.5, 1, 1.25, 1.85, 2, 5},
			ConstLabels: prometheus.Labels{"path": path},
		},
		[]string{"method"},
	)

	registry.MustRegister(duration)
	durationVects[path] = duration

	return duration
}

func InstrumentHandler(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		durationVect := getOrSetDurationVectForPath(r.URL.Path)
		durationHandler := promhttp.InstrumentHandlerDuration(durationVect, next)
		promhttp.InstrumentHandlerCounter(httpRequestsTotal, durationHandler).ServeHTTP(w, r)
	}
}
