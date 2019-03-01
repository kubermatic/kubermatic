package main

import (
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac-user-cluster"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const metricNamespace = "kubermatic"

// managerMetrics contains managerMetrics that this controller will collect and expose
type managerMetrics struct {
	controllers prometheus.Gauge
}

// newMetrics creates metrics with default values initialized, so managerMetrics always show up.
func newMetrics() *managerMetrics {
	subsystem := "user_cluster_controller_manager"
	cm := &managerMetrics{
		controllers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "controllers",
			Help:      "The number of running controllers",
		}),
	}

	cm.controllers.Set(0)
	return cm
}

// registerControllers adds all Controllers to the Manager
func registerControllers(m manager.Manager, metrics *managerMetrics) error {
	addToManagerFuncs := []func(manager.Manager) (string, error){rbacusercluster.Add}
	for _, f := range addToManagerFuncs {
		name, err := f(m)
		if err != nil {
			return err
		}
		glog.Info("new controller ", name, " added")
		metrics.controllers.Inc()
	}
	return nil
}
