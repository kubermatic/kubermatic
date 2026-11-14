/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package mla

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// agentContainerName is the main container of both MLA agents.
	agentContainerName = "grafana-alloy"

	// customScrapeConfigPrefix is the default of the user-cluster-controller-manager
	// flag -monitoring-agent-scrape-config-prefix.
	customScrapeConfigPrefix = "monitoring-scraping"
	customScrapeJob          = "mla-e2e-custom"

	pathProbePodName = "mla-e2e-path-probe"
	pathProbeMetric  = "mla_e2e_path_probe_total"
	slowProbePodName = "mla-e2e-slow-probe"
	slowProbeMetric  = "mla_e2e_slow_probe_total"

	// slowScrapeTimeout covers the 5 minute interval of the scrape-slow job.
	slowScrapeTimeout = 8 * time.Minute
)

// agentState is what waitForAgents expects to find in the user cluster.
type agentState struct {
	monitoring bool
	logging    bool
	// monitoringReplicas is checked when not zero.
	monitoringReplicas int32
}

// waitForAgents waits until the MLA agents in the user cluster match want:
// enabled agents are ready, disabled ones are gone.
func waitForAgents(ctx context.Context, in Input, want agentState, timeout time.Duration) error {
	return wait.PollImmediate(ctx, 5*time.Second, timeout, func(ctx context.Context) (error, error) {
		deployment := &appsv1.Deployment{}
		err := in.UserClient.Get(ctx, types.NamespacedName{Namespace: resources.UserClusterMLANamespace, Name: resources.MLAMonitoringAgentDeploymentName}, deployment)
		switch {
		case err != nil && !apierrors.IsNotFound(err):
			return err, nil
		case err == nil && !want.monitoring:
			return errors.New("monitoring agent still exists"), nil
		case err != nil && want.monitoring:
			return errors.New("monitoring agent does not exist yet"), nil
		case err == nil:
			replicas := ptr.Deref(deployment.Spec.Replicas, 1)
			if want.monitoringReplicas != 0 && replicas != want.monitoringReplicas {
				return fmt.Errorf("monitoring agent has %d replicas, expected %d", replicas, want.monitoringReplicas), nil
			}
			if deployment.Status.ObservedGeneration < deployment.Generation || deployment.Status.ReadyReplicas != replicas || deployment.Status.UpdatedReplicas != replicas || deployment.Status.Replicas != replicas {
				return fmt.Errorf("monitoring agent rollout not complete (%d/%d ready)", deployment.Status.ReadyReplicas, replicas), nil
			}
		}

		daemonSet := &appsv1.DaemonSet{}
		err = in.UserClient.Get(ctx, types.NamespacedName{Namespace: resources.UserClusterMLANamespace, Name: resources.MLALoggingAgentDaemonSetName}, daemonSet)
		switch {
		case err != nil && !apierrors.IsNotFound(err):
			return err, nil
		case err == nil && !want.logging:
			return errors.New("logging agent still exists"), nil
		case err != nil && want.logging:
			return errors.New("logging agent does not exist yet"), nil
		case err == nil:
			status := daemonSet.Status
			if status.ObservedGeneration < daemonSet.Generation || status.DesiredNumberScheduled == 0 || status.NumberReady != status.DesiredNumberScheduled || status.UpdatedNumberScheduled != status.DesiredNumberScheduled {
				return fmt.Errorf("logging agent rollout not complete (%d/%d ready)", status.NumberReady, status.DesiredNumberScheduled), nil
			}
		}

		return nil, nil
	})
}

// patchClusterMLA changes the cluster's spec.mla and returns the revert
// function.
func patchClusterMLA(ctx context.Context, in Input, mutate func(*kubermaticv1.MLASettings)) (func(context.Context) error, error) {
	cluster := &kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: in.Cluster.Name}}

	return patchWithRevert(ctx, in.SeedClient, cluster, func() {
		if cluster.Spec.MLA == nil {
			cluster.Spec.MLA = &kubermaticv1.MLASettings{}
		}
		mutate(cluster.Spec.MLA)
	})
}

// revertAndWait runs revert and waits for both agents to be back.
func revertAndWait(ctx context.Context, in Input, revert func(context.Context) error) error {
	revertCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := revert(revertCtx); err != nil {
		return err
	}

	// the revert is applied; waiting for the rollout is pointless once the
	// check was interrupted
	if ctx.Err() != nil {
		return nil
	}
	ctx = revertCtx

	return waitForAgents(ctx, in, agentState{monitoring: true, logging: true}, 9*time.Minute)
}

// CheckMLAToggles turns monitoring and logging on separately and verifies that
// only the matching agent runs, then restores both.
func CheckMLAToggles(ctx context.Context, in Input) error {
	steps := []struct {
		name string
		want agentState
	}{
		{name: "monitoring only", want: agentState{monitoring: true}},
		{name: "logging only", want: agentState{logging: true}},
	}

	for _, step := range steps {
		revert, err := patchClusterMLA(ctx, in, func(s *kubermaticv1.MLASettings) {
			s.MonitoringEnabled = step.want.monitoring
			s.LoggingEnabled = step.want.logging
		})
		if err != nil {
			return err
		}

		waitErr := waitForAgents(ctx, in, step.want, 5*time.Minute)

		if err := revertAndWait(ctx, in, revert); err != nil {
			return fmt.Errorf("%s: failed to restore both agents: %w", step.name, err)
		}

		if waitErr != nil {
			return fmt.Errorf("%s: %w", step.name, waitErr)
		}
	}

	return nil
}

// CheckAgentSettings sets the agents' resources and the monitoring agent's
// replicas through the Cluster spec and verifies the agents pick them up.
func CheckAgentSettings(ctx context.Context, in Input) error {
	monitoringResources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("55m"), corev1.ResourceMemory: resource.MustParse("301Mi")},
		Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("777Mi")},
	}
	loggingResources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("33m"), corev1.ResourceMemory: resource.MustParse("99Mi")},
		Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("333Mi")},
	}

	revert, err := patchClusterMLA(ctx, in, func(s *kubermaticv1.MLASettings) {
		s.MonitoringResources = monitoringResources.DeepCopy()
		s.LoggingResources = loggingResources.DeepCopy()
		s.MonitoringReplicas = ptr.To[int32](1)
	})
	if err != nil {
		return err
	}

	checkErr := func() error {
		if err := waitForAgents(ctx, in, agentState{monitoring: true, logging: true, monitoringReplicas: 1}, 5*time.Minute); err != nil {
			return err
		}

		deployment := &appsv1.Deployment{}
		if err := in.UserClient.Get(ctx, types.NamespacedName{Namespace: resources.UserClusterMLANamespace, Name: resources.MLAMonitoringAgentDeploymentName}, deployment); err != nil {
			return err
		}
		if err := containerHasResources(deployment.Spec.Template.Spec, monitoringResources); err != nil {
			return fmt.Errorf("monitoring agent: %w", err)
		}

		daemonSet := &appsv1.DaemonSet{}
		if err := in.UserClient.Get(ctx, types.NamespacedName{Namespace: resources.UserClusterMLANamespace, Name: resources.MLALoggingAgentDaemonSetName}, daemonSet); err != nil {
			return err
		}
		if err := containerHasResources(daemonSet.Spec.Template.Spec, loggingResources); err != nil {
			return fmt.Errorf("logging agent: %w", err)
		}

		return nil
	}()

	if err := revertAndWait(ctx, in, revert); err != nil {
		return fmt.Errorf("failed to restore the agent settings: %w", err)
	}

	return checkErr
}

func containerHasResources(spec corev1.PodSpec, want corev1.ResourceRequirements) error {
	for _, container := range spec.Containers {
		if container.Name != agentContainerName {
			continue
		}

		for name, quantity := range want.Requests {
			if got := container.Resources.Requests[name]; got.Cmp(quantity) != 0 {
				return fmt.Errorf("request %s is %s, expected %s", name, got.String(), quantity.String())
			}
		}
		for name, quantity := range want.Limits {
			if got := container.Resources.Limits[name]; got.Cmp(quantity) != 0 {
				return fmt.Errorf("limit %s is %s, expected %s", name, got.String(), quantity.String())
			}
		}

		return nil
	}

	return fmt.Errorf("no container %s", agentContainerName)
}

// waitForSeries waits until query returns at least one sample in Cortex.
func waitForSeries(ctx context.Context, in Input, query string, timeout time.Duration) error {
	return wait.PollImmediate(ctx, 15*time.Second, timeout, func(ctx context.Context) (error, error) {
		samples, err := cortexQuery(ctx, in, query)
		if err != nil {
			return err, nil
		}

		if len(samples) == 0 {
			return fmt.Errorf("%s returns no data yet", query), nil
		}

		return nil, nil
	})
}

// CheckScrapeAnnotations deploys pods that use the prometheus.io/path and the
// prometheus.io/scrape-slow annotations and verifies the monitoring agent
// scrapes both.
func CheckScrapeAnnotations(ctx context.Context, in Input) error {
	pathPod := newPorterPod(pathProbePodName, pathProbeMetric, in.Cluster.Name, map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   strconv.Itoa(metricsProbePort),
		"prometheus.io/path":   "/custom/metrics",
	})
	slowPod := newPorterPod(slowProbePodName, slowProbeMetric, in.Cluster.Name, map[string]string{
		"prometheus.io/scrape-slow": "true",
		"prometheus.io/port":        strconv.Itoa(metricsProbePort),
	})

	for _, pod := range []*corev1.Pod{pathPod, slowPod} {
		if err := ensurePod(ctx, in.UserClient, in, pod); err != nil {
			return err
		}
		defer deleteUserClusterPod(in, pod)
	}

	if err := waitForSeries(ctx, in, fmt.Sprintf("%s{run=%q, kubernetes_pod_name=%q}", pathProbeMetric, in.Cluster.Name, pathProbePodName), 6*time.Minute); err != nil {
		return fmt.Errorf("prometheus.io/path: %w", err)
	}

	if err := waitForSeries(ctx, in, fmt.Sprintf("%s{run=%q, kubernetes_pod_name=%q}", slowProbeMetric, in.Cluster.Name, slowProbePodName), slowScrapeTimeout); err != nil {
		return fmt.Errorf("prometheus.io/scrape-slow: %w", err)
	}

	return nil
}

// CheckCustomScrapeConfig adds a monitoring-scraping ConfigMap with a static
// scrape job and verifies the agent scrapes it. Then it adds an invalid one
// and verifies the agent keeps running and keeps scraping.
func CheckCustomScrapeConfig(ctx context.Context, in Input) error {
	if err := EnsureMetricsProbe(ctx, in); err != nil {
		return err
	}

	probe := &corev1.Pod{}
	if err := in.UserClient.Get(ctx, types.NamespacedName{Namespace: metav1.NamespaceDefault, Name: metricsProbePodName}, probe); err != nil {
		return err
	}
	if probe.Status.PodIP == "" {
		return errors.New("metrics probe pod has no IP")
	}

	suffix := rand.String(6)
	job := customScrapeJob + "-" + suffix

	valid := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: customScrapeConfigPrefix + "-e2e-" + suffix, Namespace: resources.UserClusterMLANamespace},
		Data: map[string]string{
			"e2e.yaml": fmt.Sprintf("- job_name: %s\n  scrape_interval: 15s\n  static_configs:\n    - targets: ['%s:%d']\n", job, probe.Status.PodIP, metricsProbePort),
		},
	}
	if err := in.UserClient.Create(ctx, valid); err != nil {
		return fmt.Errorf("failed to create scrape ConfigMap: %w", err)
	}
	defer deleteUserClusterObject(in, valid)

	upQuery := fmt.Sprintf("up{job=%q} == 1", job)
	if err := waitForSeries(ctx, in, upQuery, 6*time.Minute); err != nil {
		return fmt.Errorf("custom scrape job: %w", err)
	}

	invalid := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: customScrapeConfigPrefix + "-e2e-invalid-" + suffix, Namespace: resources.UserClusterMLANamespace},
		Data:       map[string]string{"broken.yaml": "- job_name: [this is not valid"},
	}
	if err := in.UserClient.Create(ctx, invalid); err != nil {
		return fmt.Errorf("failed to create invalid scrape ConfigMap: %w", err)
	}
	defer deleteUserClusterObject(in, invalid)

	// wait until the controller rendered the broken entry into the agent
	// config, then give the config reloader one interval to apply it
	err := wait.PollImmediate(ctx, 5*time.Second, 3*time.Minute, func(ctx context.Context) (error, error) {
		agentConfig := &corev1.ConfigMap{}
		if err := in.UserClient.Get(ctx, types.NamespacedName{Namespace: resources.UserClusterMLANamespace, Name: resources.MLAMonitoringAgentConfigMapName}, agentConfig); err != nil {
			return err, nil
		}

		for _, value := range agentConfig.Data {
			if strings.Contains(value, "this is not valid") {
				return nil, nil
			}
		}

		return errors.New("agent config does not contain the invalid scrape config yet"), nil
	})
	if err != nil {
		return fmt.Errorf("invalid scrape config was not rendered: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(45 * time.Second):
	}

	if err := waitForAgents(ctx, in, agentState{monitoring: true, logging: true}, 2*time.Minute); err != nil {
		return fmt.Errorf("agent after an invalid scrape config: %w", err)
	}

	fresh := fmt.Sprintf("time() - timestamp(up{job=%q}) < 60", job)
	if err := waitForSeries(ctx, in, fresh, 2*time.Minute); err != nil {
		return fmt.Errorf("custom scrape job stopped after an invalid scrape config: %w", err)
	}

	return nil
}

// newPorterPod returns an agnhost porter pod that serves one counter, with
// the given annotations.
func newPorterPod(name, metric, runID string, annotations map[string]string) *corev1.Pod {
	pod := newMetricsProbePod(metav1.NamespaceDefault, runID)
	pod.Name = name
	pod.Annotations = annotations
	pod.Spec.Containers[0].Env[0].Value = fmt.Sprintf("# TYPE %s counter\n%s{run=%q} 1\n", metric, metric, runID)

	return pod
}

func deleteUserClusterPod(in Input, pod *corev1.Pod) {
	deleteUserClusterObject(in, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: pod.Name, Namespace: pod.Namespace}})
}

// deleteUserClusterObject is a best-effort cleanup for objects a check created
// in the user cluster.
func deleteUserClusterObject(in Input, obj ctrlruntimeclient.Object) {
	ctx, cancel := revertContext()
	defer cancel()

	if err := in.UserClient.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		in.Log.Warnw("Failed to clean up", "object", obj.GetName(), zap.Error(err))
	}
}
