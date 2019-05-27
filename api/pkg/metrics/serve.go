package metrics

import (
	"net/http"
	"sync"

	"github.com/kubermatic/kubermatic/api/pkg/log"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var once = &sync.Once{}

// ServeForever the prometheus metrics endpoint
func ServeForever(addr, path string) {
	once.Do(func() {
		log.GetLogger().Infof("Prometheus metrics exposed on: %s%s", addr, path)

		http.Handle(path, promhttp.Handler())
		log.GetLogger().Fatal(http.ListenAndServe(addr, nil))
	})
}
