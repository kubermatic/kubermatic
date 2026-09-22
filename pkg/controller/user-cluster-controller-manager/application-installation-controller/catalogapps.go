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

package applicationinstallationcontroller

import (
	"slices"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling/modifier"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// tolerationKey is one place in a chart's values where a workload's tolerations are configured.
type tolerationKey struct {
	// path to the tolerations inside the values, e.g. {"controller", "tolerations"}.
	path []string

	// chartDefaults are the tolerations the chart ships at this path. Helm replaces lists instead of
	// merging them, so they have to be repeated or setting the key would take them away.
	chartDefaults []corev1.Toleration
}

var (
	tolerateControlPlane = corev1.Toleration{Key: "node-role.kubernetes.io/control-plane", Effect: corev1.TaintEffectNoSchedule}
	tolerateMaster       = corev1.Toleration{Key: "node-role.kubernetes.io/master", Effect: corev1.TaintEffectNoSchedule}
	tolerateNvidiaGPU    = corev1.Toleration{Key: "nvidia.com/gpu", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule}
)

// applicationTolerationKeys maps an ApplicationDefinition to the values keys that carry the
// tolerations of its workloads. Verified against the newest chart version each definition in
// pkg/ee/default-application-catalog/applicationdefinitions pins; a chart that renames or restructures
// one of these keys silently stops being covered, because Helm ignores unknown values.
//
// Three shipped applications cannot be covered this way, as their charts expose no toleration key at
// all: aikit (only its post-install hook job has one), k8sgpt-operator and kubevirt. The nodeSelector
// that aikit and k8sgpt-operator do offer is no substitute, as it picks a node without granting the
// Pod permission to tolerate that node's taints. They therefore keep running on the worker nodes
// that carry no taint, and a Pod of theirs that is Pending schedules onto the next untainted node to
// join. Only a cluster whose worker nodes are all tainted leaves them Pending, and with the NoSchedule
// effect that does not show up when the taint is added, as running Pods are left alone, but at their
// next restart or rollout.
// Covering them needs a Helm post-renderer, which matches on the kind of the rendered workload
// instead of on a chart's values.
var applicationTolerationKeys = map[string][]tolerationKey{
	"argocd": {
		{path: []string{"global", "tolerations"}},
	},
	"cert-manager": {
		{path: []string{"tolerations"}},
		{path: []string{"webhook", "tolerations"}},
		{path: []string{"cainjector", "tolerations"}},
		{path: []string{"startupapicheck", "tolerations"}},
	},
	"envoy-gateway": {
		{path: []string{"deployment", "pod", "tolerations"}},
		{path: []string{"certgen", "job", "tolerations"}},
	},
	"falco": {
		{path: []string{"tolerations"}, chartDefaults: []corev1.Toleration{tolerateMaster, tolerateControlPlane}},
	},
	"flux2": {
		{path: []string{"cli", "tolerations"}},
		{path: []string{"helmController", "tolerations"}},
		{path: []string{"imageAutomationController", "tolerations"}},
		{path: []string{"imageReflectionController", "tolerations"}},
		{path: []string{"kustomizeController", "tolerations"}},
		{path: []string{"notificationController", "tolerations"}},
		{path: []string{"sourceController", "tolerations"}},
	},
	"kube-state-metrics": {
		{path: []string{"tolerations"}},
	},
	"kube-vip": {
		{path: []string{"tolerations"}, chartDefaults: []corev1.Toleration{{
			Key: "node-role.kubernetes.io/control-plane", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule,
		}}},
	},
	"kubernetes-mcp-server": {
		{path: []string{"tolerations"}},
	},
	"kueue": {
		{path: []string{"controllerManager", "tolerations"}},
		{path: []string{"kueueViz", "backend", "tolerations"}},
		{path: []string{"kueueViz", "frontend", "tolerations"}},
	},
	"local-ai": {
		{path: []string{"tolerations"}},
	},
	"metallb": {
		{path: []string{"controller", "tolerations"}},
		{path: []string{"speaker", "tolerations"}},
	},
	"nginx": {
		{path: []string{"controller", "tolerations"}},
		{path: []string{"controller", "admissionWebhooks", "patch", "tolerations"}},
		{path: []string{"defaultBackend", "tolerations"}},
	},
	"node-exporter": {
		{path: []string{"tolerations"}, chartDefaults: []corev1.Toleration{{
			Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule,
		}}},
	},
	"nvidia-gpu-operator": {
		{path: []string{"daemonsets", "tolerations"}, chartDefaults: []corev1.Toleration{tolerateNvidiaGPU}},
		{path: []string{"operator", "tolerations"}, chartDefaults: []corev1.Toleration{{
			Key: "node-role.kubernetes.io/control-plane", Operator: corev1.TolerationOpEqual, Effect: corev1.TaintEffectNoSchedule,
		}}},
		{path: []string{"node-feature-discovery", "worker", "tolerations"}, chartDefaults: []corev1.Toleration{
			{Key: "node-role.kubernetes.io/control-plane", Operator: corev1.TolerationOpEqual, Effect: corev1.TaintEffectNoSchedule},
			tolerateNvidiaGPU,
		}},
	},
	"trivy": {
		{path: []string{"tolerations"}},
	},
	"trivy-operator": {
		{path: []string{"tolerations"}},
		{path: []string{"nodeCollector", "tolerations"}},
	},
}

// generateApplicationTolerationValues returns the Helm values that put the given tolerations on the
// workloads of the named application, or nil when the application is not covered. These values are
// merged with override, so every list has to carry the entries that are already configured.
func generateApplicationTolerationValues(appName string, workloadTolerations []corev1.Toleration, currentValues map[string]any) map[string]any {
	keys, covered := applicationTolerationKeys[appName]
	if !covered || len(workloadTolerations) == 0 {
		return nil
	}

	values := map[string]any{}
	for _, key := range keys {
		// The chart defaults come first, then whatever the ApplicationDefinition or the user has
		// configured at this key, so that neither is lost when the list is replaced.
		tolerations, _ := modifier.AppendTolerations(slices.Clone(key.chartDefaults), currentTolerations(currentValues, key.path))
		tolerations, _ = modifier.AppendTolerations(tolerations, workloadTolerations)

		setNestedValue(values, key.path, resources.TolerationsToHelmValues(tolerations))
	}

	return values
}

// currentTolerations reads the tolerations already configured at the given path. Values that do not
// describe tolerations are ignored; the chart, not KKP, is the place where those are reported.
func currentTolerations(values map[string]any, path []string) []corev1.Toleration {
	raw, found, err := unstructured.NestedSlice(values, path...)
	if err != nil || !found {
		return nil
	}

	tolerations := make([]corev1.Toleration, 0, len(raw))
	for _, item := range raw {
		rawToleration, ok := item.(map[string]any)
		if !ok {
			return nil
		}

		toleration := corev1.Toleration{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(rawToleration, &toleration); err != nil {
			return nil
		}
		tolerations = append(tolerations, toleration)
	}

	return tolerations
}

// setNestedValue stores a value in a tree of maps, creating the intermediate levels as needed.
func setNestedValue(values map[string]any, path []string, value any) {
	for _, field := range path[:len(path)-1] {
		child, ok := values[field].(map[string]any)
		if !ok {
			child = map[string]any{}
			values[field] = child
		}
		values = child
	}

	values[path[len(path)-1]] = value
}
