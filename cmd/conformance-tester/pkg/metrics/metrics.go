/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/zap"
)

const (
	metricNamespace = "conformancetest"
)

var (
	metricsPusher *push.Pusher

	KubermaticLoginDurationMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "kubermatic_login_duration_seconds",
		Help:      "Time it took to perform the Kubermatic login, in seconds",
	}, []string{"prowjob"})

	KubermaticReconciliationDurationMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "kubermatic_reconciliation_duration_seconds",
		Help:      "Time it took for Kubermatic to fully reconcile the test cluster",
	}, []string{"prowjob", "scenario"})

	SeedControlplaneDurationMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "seed_controlplane_duration_seconds",
		Help:      "Time it took the user-cluster's controlplane pods in the seed cluster to become ready",
	}, []string{"prowjob", "scenario"})

	ClusterControlplaneDurationMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "cluster_controlplane_duration_seconds",
		Help:      "Time it took for all pods to be ready in a user cluster after all worker nodes have become ready",
	}, []string{"prowjob", "scenario"})

	NodeCreationDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "node_creation_duration_seconds",
		Help:      "Time it took for all nodes to spawn after the NodeDeployments were created",
	}, []string{"prowjob", "scenario"})

	NodeRadinessDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "node_readiness_duration_seconds",
		Help:      "Time it took for all nodes to become ready they appeared",
	}, []string{"prowjob", "scenario"})

	ScenarioRuntimeMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "scenario_runtime_seconds",
		Help:      "Total duration of a scenario test run",
	}, []string{"prowjob", "scenario"})

	GinkgoRuntimeMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "ginkgo_runtime_seconds",
		Help:      "Number of seconds a Ginkgo run took",
	}, []string{"prowjob", "scenario", "run", "attempt"})

	GinkgoAttemptsMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "ginkgo_attempts",
		Help:      "Number of times a job has been run for a given scenario",
	}, []string{"prowjob", "scenario", "run"})

	PVCTestRuntimeMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "pvctest_runtime_seconds",
		Help:      "Number of seconds a pvctest run took",
	}, []string{"prowjob", "scenario", "attempt"})

	PVCTestAttemptsMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "pvctest_attempts",
		Help:      "Number of times a job has been run for a given scenario",
	}, []string{"prowjob", "scenario"})

	LBTestRuntimeMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "lbtest_runtime_seconds",
		Help:      "Number of seconds a lbtest run took",
	}, []string{"prowjob", "scenario", "attempt"})

	LBTestAttemptsMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "lbtest_attempts",
		Help:      "Number of times a job has been run for a given scenario",
	}, []string{"prowjob", "scenario"})

	NetworkPolicyTestRuntimeMetric = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "network_policy_test_runtime_seconds",
			Help: "Runtime of network policy tests.",
		}, []string{"scenario"})

	NetworkPolicyTestAttemptsMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "network_policy_test_attempts_total",
			Help: "Number of attempts for network policy tests.",
		}, []string{"scenario"})
)

func Setup(endpoint string, prowjob string, instance string) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(KubermaticLoginDurationMetric)
	registry.MustRegister(KubermaticReconciliationDurationMetric)
	registry.MustRegister(SeedControlplaneDurationMetric)
	registry.MustRegister(ClusterControlplaneDurationMetric)
	registry.MustRegister(NodeCreationDuration)
	registry.MustRegister(NodeRadinessDuration)
	registry.MustRegister(ScenarioRuntimeMetric)
	registry.MustRegister(GinkgoRuntimeMetric)
	registry.MustRegister(GinkgoAttemptsMetric)
	registry.MustRegister(PVCTestRuntimeMetric)
	registry.MustRegister(PVCTestAttemptsMetric)
	registry.MustRegister(LBTestRuntimeMetric)
	registry.MustRegister(LBTestAttemptsMetric)

	// make sure prowjob and instance are always defined, so that the
	// metrics are properly defined; otherwise all the surrounding code
	// would need to carefully check the metrics to avoid calling
	// WithLabelValues(), which will panic if the label values are empty.
	if prowjob == "" {
		prowjob = "local"
	}

	prowjobLabel := prometheus.Labels{
		"prowjob": prowjob,
	}

	KubermaticLoginDurationMetric = KubermaticLoginDurationMetric.MustCurryWith(prowjobLabel)
	KubermaticReconciliationDurationMetric = KubermaticReconciliationDurationMetric.MustCurryWith(prowjobLabel)
	SeedControlplaneDurationMetric = SeedControlplaneDurationMetric.MustCurryWith(prowjobLabel)
	ClusterControlplaneDurationMetric = ClusterControlplaneDurationMetric.MustCurryWith(prowjobLabel)
	NodeCreationDuration = NodeCreationDuration.MustCurryWith(prowjobLabel)
	NodeRadinessDuration = NodeRadinessDuration.MustCurryWith(prowjobLabel)
	ScenarioRuntimeMetric = ScenarioRuntimeMetric.MustCurryWith(prowjobLabel)
	GinkgoRuntimeMetric = GinkgoRuntimeMetric.MustCurryWith(prowjobLabel)
	GinkgoAttemptsMetric = GinkgoAttemptsMetric.MustCurryWith(prowjobLabel)
	PVCTestRuntimeMetric = PVCTestRuntimeMetric.MustCurryWith(prowjobLabel)
	PVCTestAttemptsMetric = PVCTestAttemptsMetric.MustCurryWith(prowjobLabel)
	LBTestRuntimeMetric = LBTestRuntimeMetric.MustCurryWith(prowjobLabel)
	LBTestAttemptsMetric = LBTestAttemptsMetric.MustCurryWith(prowjobLabel)

	// skip setting up the metricsPusher if no endpoint is defined
	if endpoint == "" {
		return
	}

	metricsPusher = push.New(endpoint, "conformancetest")
	metricsPusher.Grouping("instance", instance)
	metricsPusher.Gatherer(registry)
}

func UpdateMetrics(log *zap.SugaredLogger) {
	if metricsPusher == nil {
		return
	}

	if err := metricsPusher.Push(); err != nil {
		log.Warnw("Failed to push metrics", zap.Error(err))
	}
}

//nolint:interfacer
func MeasureTime(metric prometheus.Gauge, log *zap.SugaredLogger, callback func() error) error {
	start := time.Now()
	err := callback()
	metric.Set(time.Since(start).Seconds())
	UpdateMetrics(log)

	return err
}

func TimeMeasurementWrapper(metric prometheus.Gauge, log *zap.SugaredLogger, callback func() error) func() error {
	return func() error {
		return MeasureTime(metric, log, callback)
	}
}
