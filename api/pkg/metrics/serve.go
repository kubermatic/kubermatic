package metrics

import (
	"log"
	"net/http"
	"sync"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var once = &sync.Once{}

// ServeForever the prometheus metrics endpoint
func ServeForever(addr, path string) {
	once.Do(func() {
		glog.Infof("Prometheus metrics exposed on: %s%s", addr, path)

		http.Handle(path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
		log.Fatal(http.ListenAndServe(addr, nil))
	})
}
