/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8c.io/kubermatic/v2/pkg/log"
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
