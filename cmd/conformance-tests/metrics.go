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

package main

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

	kubermaticLoginDurationMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "kubermatic_login_duration_seconds",
		Help:      "Time it took to perform the Kubermatic login, in seconds",
	}, []string{"prowjob"})

	kubermaticReconciliationDurationMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "kubermatic_reconciliation_duration_seconds",
		Help:      "Time it took for Kubermatic to fully reconcile the test cluster",
	}, []string{"prowjob", "scenario"})

	seedControlplaneDurationMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "seed_controlplane_duration_seconds",
		Help:      "Time it took the user-cluser's controlplane pods in the seed cluster to become ready",
	}, []string{"prowjob", "scenario"})

	clusterControlplaneDurationMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "cluster_controlplane_duration_seconds",
		Help:      "Time it took for all pods to be ready in a user cluster after all worker nodes have become ready",
	}, []string{"prowjob", "scenario"})

	nodeCreationDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "node_creation_duration_seconds",
		Help:      "Time it took for all nodes to spawn after the NodeDeployments were created",
	}, []string{"prowjob", "scenario"})

	nodeRadinessDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "node_readiness_duration_seconds",
		Help:      "Time it took for all nodes to become ready they appeared",
	}, []string{"prowjob", "scenario"})

	scenarioRuntimeMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "scenario_runtime_seconds",
		Help:      "Total duration of a scenario test run",
	}, []string{"prowjob", "scenario"})

	ginkgoRuntimeMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "ginkgo_runtime_seconds",
		Help:      "Number of seconds a Ginkgo run took",
	}, []string{"prowjob", "scenario", "run", "attempt"})

	ginkgoAttemptsMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "ginkgo_attempts",
		Help:      "Number of times a job has been run for a given scenario",
	}, []string{"prowjob", "scenario", "run"})

	pvctestRuntimeMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "pvctest_runtime_seconds",
		Help:      "Number of seconds a pvctest run took",
	}, []string{"prowjob", "scenario", "attempt"})

	pvctestAttemptsMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "pvctest_attempts",
		Help:      "Number of times a job has been run for a given scenario",
	}, []string{"prowjob", "scenario"})

	lbtestRuntimeMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "lbtest_runtime_seconds",
		Help:      "Number of seconds a lbtest run took",
	}, []string{"prowjob", "scenario", "attempt"})

	lbtestAttemptsMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "lbtest_attempts",
		Help:      "Number of times a job has been run for a given scenario",
	}, []string{"prowjob", "scenario"})
)

func initMetrics(endpoint string, prowjob string, instance string) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(kubermaticLoginDurationMetric)
	registry.MustRegister(kubermaticReconciliationDurationMetric)
	registry.MustRegister(seedControlplaneDurationMetric)
	registry.MustRegister(clusterControlplaneDurationMetric)
	registry.MustRegister(nodeCreationDuration)
	registry.MustRegister(nodeRadinessDuration)
	registry.MustRegister(scenarioRuntimeMetric)
	registry.MustRegister(ginkgoRuntimeMetric)
	registry.MustRegister(ginkgoAttemptsMetric)
	registry.MustRegister(pvctestRuntimeMetric)
	registry.MustRegister(pvctestAttemptsMetric)
	registry.MustRegister(lbtestRuntimeMetric)
	registry.MustRegister(lbtestAttemptsMetric)

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

	kubermaticLoginDurationMetric = kubermaticLoginDurationMetric.MustCurryWith(prowjobLabel)
	kubermaticReconciliationDurationMetric = kubermaticReconciliationDurationMetric.MustCurryWith(prowjobLabel)
	seedControlplaneDurationMetric = seedControlplaneDurationMetric.MustCurryWith(prowjobLabel)
	clusterControlplaneDurationMetric = clusterControlplaneDurationMetric.MustCurryWith(prowjobLabel)
	nodeCreationDuration = nodeCreationDuration.MustCurryWith(prowjobLabel)
	nodeRadinessDuration = nodeRadinessDuration.MustCurryWith(prowjobLabel)
	scenarioRuntimeMetric = scenarioRuntimeMetric.MustCurryWith(prowjobLabel)
	ginkgoRuntimeMetric = ginkgoRuntimeMetric.MustCurryWith(prowjobLabel)
	ginkgoAttemptsMetric = ginkgoAttemptsMetric.MustCurryWith(prowjobLabel)
	pvctestRuntimeMetric = pvctestRuntimeMetric.MustCurryWith(prowjobLabel)
	pvctestAttemptsMetric = pvctestAttemptsMetric.MustCurryWith(prowjobLabel)
	lbtestRuntimeMetric = lbtestRuntimeMetric.MustCurryWith(prowjobLabel)
	lbtestAttemptsMetric = lbtestAttemptsMetric.MustCurryWith(prowjobLabel)

	// skip setting up the metricsPusher if no endpoint is defined
	if endpoint == "" {
		return
	}

	metricsPusher = push.New(endpoint, "conformancetest")
	metricsPusher.Grouping("instance", instance)
	metricsPusher.Gatherer(registry)
}

func updateMetrics(log *zap.SugaredLogger) {
	if metricsPusher == nil {
		return
	}

	if err := metricsPusher.Push(); err != nil {
		log.Warnw("Failed to push metrics", zap.Error(err))
	}
}

//nolint:interfacer
func measureTime(metric prometheus.Gauge, log *zap.SugaredLogger, callback func() error) error {
	start := time.Now()
	err := callback()
	metric.Set(time.Since(start).Seconds())
	updateMetrics(log)

	return err
}

func timeMeasurementWrapper(metric prometheus.Gauge, log *zap.SugaredLogger, callback func() error) func() error {
	return func() error {
		return measureTime(metric, log, callback)
	}
}
