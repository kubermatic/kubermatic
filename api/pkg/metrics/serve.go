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
		log.Logger.Infow("the Prometheus exposed metrics", "listenAddress", addr, "path", path)

		http.Handle(path, promhttp.Handler())
		log.Logger.Fatal(http.ListenAndServe(addr, nil))
	})
}
