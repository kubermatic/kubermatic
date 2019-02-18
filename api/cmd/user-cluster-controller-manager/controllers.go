package main

import (
	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac-user-cluster/rolebinding"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac-user-cluster/role"
	"github.com/kubermatic/kubermatic/api/pkg/controller/usercluster"

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
func registerControllers(m manager.Manager) error {
	metrics := newMetrics()
	addToManagerFuncs := []func(manager.Manager) (string, error){usercluster.Add, role.Add, rolebinding.Add}
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
