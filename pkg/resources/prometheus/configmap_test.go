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

package prometheus

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticsemver "k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestTunnelingEnvoyAgentScrapeUsesAPIServerProxy(t *testing.T) {
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
		Spec: kubermaticv1.ClusterSpec{
			ExposeStrategy: kubermaticv1.ExposeStrategyTunneling,
			Version:        *kubermaticsemver.NewSemverOrDie("1.34.1"),
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "cluster-test-cluster",
			Address: kubermaticv1.ClusterAddress{
				InternalName: "apiserver-external.cluster-test-cluster.svc.cluster.local.",
			},
		},
	}

	data := resources.NewTemplateDataBuilder().
		WithCluster(cluster).
		WithSeed(&kubermaticv1.Seed{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kubermatic",
			},
		}).
		WithKubermaticConfiguration(&kubermaticv1.KubermaticConfiguration{}).
		Build()

	_, reconciler := ConfigMapReconciler(data)()
	cm, err := reconciler(&corev1.ConfigMap{})
	require.NoError(t, err)

	config := cm.Data["prometheus.yaml"]
	require.Contains(t, config, "- job_name: 'envoy-agent'")
	require.Contains(t, config, "scheme: https")
	require.Contains(t, config, "replacement: /api/v1/namespaces/${1}/pods/${2}:${3}/proxy${4}")
	require.Contains(t, config, "replacement: 'apiserver-external.cluster-test-cluster.svc.cluster.local.'")
	require.NotContains(t, envoyAgentScrapeConfig(t, config), "replacement: $1:$2")
}

func envoyAgentScrapeConfig(t *testing.T, config string) string {
	t.Helper()

	start := strings.Index(config, "- job_name: 'envoy-agent'")
	require.NotEqual(t, -1, start)

	next := strings.Index(config[start+1:], "\n- job_name:")
	if next == -1 {
		return config[start:]
	}

	return config[start : start+1+next]
}

func TestKonnectivityAgentAlertsRequireSchedulableNodes(t *testing.T) {
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: *kubermaticsemver.NewSemverOrDie("1.34.1"),
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "cluster-test-cluster",
		},
	}

	data := resources.NewTemplateDataBuilder().
		WithCluster(cluster).
		WithSeed(&kubermaticv1.Seed{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kubermatic",
			},
		}).
		WithKubermaticConfiguration(&kubermaticv1.KubermaticConfiguration{}).
		WithKonnectivityEnabled(true).
		Build()

	_, reconciler := ConfigMapReconciler(data)()
	cm, err := reconciler(&corev1.ConfigMap{})
	require.NoError(t, err)

	var rules struct {
		Groups []struct {
			Rules []struct {
				Alert string `json:"alert"`
				Expr  string `json:"expr"`
				For   string `json:"for"`
			} `json:"rules"`
		} `json:"groups"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(cm.Data["rules.yaml"]), &rules))

	expected := map[string]string{
		"KonnectivityNoAgents":        "5m",
		"KonnectivityAgentsUnhealthy": "15m",
	}

	var config struct {
		ScrapeConfigs []struct {
			JobName              string `json:"job_name"`
			MetricRelabelConfigs []struct {
				Action string `json:"action"`
				Regex  string `json:"regex"`
			} `json:"metric_relabel_configs"`
		} `json:"scrape_configs"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(cm.Data["prometheus.yaml"]), &config))

	var ksmKeep *regexp.Regexp
	for _, sc := range config.ScrapeConfigs {
		if sc.JobName != "kube-state-metrics" {
			continue
		}
		for _, mrc := range sc.MetricRelabelConfigs {
			if mrc.Action == "keep" {
				ksmKeep = regexp.MustCompile(mrc.Regex)
			}
		}
	}
	require.NotNil(t, ksmKeep, "kube-state-metrics keep filter not found")

	found := 0
	for _, group := range rules.Groups {
		for _, rule := range group.Rules {
			wantFor, ok := expected[rule.Alert]
			if !ok {
				continue
			}
			found++

			require.Equal(t, wantFor, rule.For, rule.Alert)
			require.Contains(t, rule.Expr, "and on(namespace)", rule.Alert)
			require.Contains(t, rule.Expr, `kube_node_status_condition{condition="Ready",status="true"} == 1`, rule.Alert)
			require.Contains(t, rule.Expr, "unless on(namespace, node) kube_node_spec_unschedulable == 1", rule.Alert)

			// the gate silently opens if the scrape drops a metric it reads
			for _, metric := range regexp.MustCompile(`kube_[a-z_]+`).FindAllString(rule.Expr, -1) {
				require.True(t, ksmKeep.MatchString(metric), "%s: %s is dropped by the kube-state-metrics scrape", rule.Alert, metric)
			}
		}
	}
	require.Equal(t, len(expected), found)
}
