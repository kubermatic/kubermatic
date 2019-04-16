package clusterdeletion

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	subsystem = "kubermatic_cluster"
)

var (
	registerMetrics sync.Once
	staleLBs        = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: subsystem,
			Name:      "stale_lbs",
			Help:      "The number of cloud load balancers that couldn't be cleaned up within the 2h grace period",
		},
		[]string{"cluster"},
	)
)

func init() {
	registerMetrics.Do(func() {
		prometheus.MustRegister(staleLBs)
	})
}
