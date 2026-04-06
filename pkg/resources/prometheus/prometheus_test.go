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
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// minimalTemplateData satisfies the subset of fields and methods accessed by
// the prometheusConfig Go template, without requiring a full resources.TemplateData.
type minimalTemplateData struct {
	cluster *kubermaticv1.Cluster
	seed    *kubermaticv1.Seed
	config  *kubermaticv1.KubermaticConfiguration
}

func (d *minimalTemplateData) Cluster() *kubermaticv1.Cluster { return d.cluster }
func (d *minimalTemplateData) Seed() *kubermaticv1.Seed       { return d.seed }
func (d *minimalTemplateData) KubermaticConfiguration() *kubermaticv1.KubermaticConfiguration {
	return d.config
}
func (d *minimalTemplateData) IsKonnectivityEnabled() bool { return false }

func newMinimalTemplateData() *minimalTemplateData {
	return &minimalTemplateData{
		cluster: &kubermaticv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{Name: "test-cluster"},
			Spec: kubermaticv1.ClusterSpec{
				ExposeStrategy: kubermaticv1.ExposeStrategyLoadBalancer,
			},
			Status: kubermaticv1.ClusterStatus{
				NamespaceName: "cluster-test-cluster",
				Address: kubermaticv1.ClusterAddress{
					InternalName: "apiserver.cluster-test-cluster.svc.cluster.local",
				},
			},
		},
		seed: &kubermaticv1.Seed{
			ObjectMeta: metav1.ObjectMeta{Name: "test-seed"},
		},
		config: &kubermaticv1.KubermaticConfiguration{
			Spec: kubermaticv1.KubermaticConfigurationSpec{
				UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
					Monitoring: kubermaticv1.KubermaticUserClusterMonitoringConfiguration{
						ScrapeAnnotationPrefix: "monitoring.kubermatic.io",
					},
				},
			},
		},
	}
}

// ---- StatefulSet args tests ----

func TestPrometheusAgentArgs(t *testing.T) {
	args := prometheusAgentArgs()

	requiredArgs := []string{
		"--enable-feature=agent",
		"--storage.agent.path=/var/prometheus/data",
		"--config.file=/etc/prometheus/config/prometheus.yaml",
		"--web.enable-lifecycle",
	}
	for _, required := range requiredArgs {
		t.Run("contains "+required, func(t *testing.T) {
			found := false
			for _, arg := range args {
				if arg == required {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected arg %q not found in %v", required, args)
			}
		})
	}

	forbiddenPrefixes := []string{
		"--storage.tsdb",
		"--storage.tsdb.no-lockfile",
		"--storage.tsdb.retention",
		"--storage.tsdb.min-block-duration",
		"--storage.tsdb.max-block-duration",
	}
	for _, forbidden := range forbiddenPrefixes {
		t.Run("does not contain "+forbidden, func(t *testing.T) {
			for _, arg := range args {
				if strings.HasPrefix(arg, forbidden) {
					t.Errorf("forbidden TSDB arg %q must not appear in agent mode args, got %v", forbidden, args)
				}
			}
		})
	}
}

// ---- ConfigMap template tests ----

func renderConfig(t *testing.T, td *minimalTemplateData) string {
	t.Helper()
	return renderConfigWithReplicas(t, td, 2)
}

func renderConfigWithReplicas(t *testing.T, td *minimalTemplateData, replicas int32) string {
	t.Helper()
	data := &configTemplateData{
		TemplateData:             td,
		APIServerHost:            td.cluster.Status.Address.InternalName,
		EtcdTLSConfig:            "ca_file: /etc/etcd/pki/client/ca.crt\ncert_file: /etc/etcd/pki/client/apiserver-etcd-client.crt\nkey_file: /etc/etcd/pki/client/apiserver-etcd-client.key",
		ApiserverTLSConfig:       "ca_file: /etc/kubernetes/ca.crt\ncert_file: /etc/kubernetes/prometheus-client.crt\nkey_file: /etc/kubernetes/prometheus-client.key",
		ScrapingAnnotationPrefix: "monitoring_kubermatic_io",
		RemoteWriteConfig:        buildRemoteWriteConfig(replicas),
	}
	rendered, err := renderTemplate(prometheusConfig, data)
	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}
	return rendered
}

func TestBuildRemoteWriteConfig(t *testing.T) {
	tests := []struct {
		replicas        int32
		expectedURLs    []string
		notExpectedURLs []string
	}{
		{
			replicas: 1,
			expectedURLs: []string{
				"prometheus-0.prometheus.monitoring.svc.cluster.local:9090/api/v1/write",
				"seed-prometheus-0",
			},
			notExpectedURLs: []string{
				"prometheus-1",
			},
		},
		{
			replicas: 2,
			expectedURLs: []string{
				"prometheus-0.prometheus.monitoring.svc.cluster.local:9090/api/v1/write",
				"prometheus-1.prometheus.monitoring.svc.cluster.local:9090/api/v1/write",
				"seed-prometheus-0",
				"seed-prometheus-1",
			},
		},
		{
			replicas: 3,
			expectedURLs: []string{
				"prometheus-0.prometheus.monitoring.svc.cluster.local:9090/api/v1/write",
				"prometheus-1.prometheus.monitoring.svc.cluster.local:9090/api/v1/write",
				"prometheus-2.prometheus.monitoring.svc.cluster.local:9090/api/v1/write",
				"seed-prometheus-2",
			},
		},
		{
			// 0 replicas (monitoring not installed) → empty output, no WAL buildup.
			replicas:        0,
			expectedURLs:    []string{},
			notExpectedURLs: []string{"remote_write:", "prometheus-0", "prometheus-1"},
		},
	}
	for _, tc := range tests {
		t.Run(strings.Repeat("replica,", int(tc.replicas)), func(t *testing.T) {
			result := buildRemoteWriteConfig(tc.replicas)
			for _, s := range tc.expectedURLs {
				if !strings.Contains(result, s) {
					t.Errorf("expected %q in output, not found:\n%s", s, result)
				}
			}
			for _, s := range tc.notExpectedURLs {
				if strings.Contains(result, s) {
					t.Errorf("unexpected %q found in output:\n%s", s, result)
				}
			}
			// Every non-empty config must include the write_relabel_configs keep regex.
			if result != "" && !strings.Contains(result, writeRelabelRegex) {
				t.Errorf("write_relabel_configs regex missing from output:\n%s", result)
			}
		})
	}
}

func TestPrometheusConfigRemoteWrite(t *testing.T) {
	config := renderConfig(t, newMinimalTemplateData())

	requiredStrings := []string{
		"remote_write:",
		"prometheus-0.prometheus.monitoring.svc.cluster.local:9090/api/v1/write",
		"prometheus-1.prometheus.monitoring.svc.cluster.local:9090/api/v1/write",
		"seed-prometheus-0",
		"seed-prometheus-1",
		"write_relabel_configs",
	}
	for _, s := range requiredStrings {
		if !strings.Contains(config, s) {
			t.Errorf("expected %q in rendered config, but not found.\nRendered:\n%s", s, config)
		}
	}

	// Verify replica count propagates — 3 replicas should produce a third target.
	config3 := renderConfigWithReplicas(t, newMinimalTemplateData(), 3)
	if !strings.Contains(config3, "prometheus-2.prometheus.monitoring.svc.cluster.local") {
		t.Errorf("expected prometheus-2 target for 3-replica config.\nRendered:\n%s", config3)
	}
}

func TestPrometheusConfigWriteRelabelRegex(t *testing.T) {
	config := renderConfig(t, newMinimalTemplateData())

	// Verify the keep regex covers all metric families referenced by
	// charts/monitoring/prometheus/rules/src/kubermatic-seed/usercluster-monitoring.yaml.
	requiredMetricPrefixes := []string{
		"up|",
		"etcd_",
		"machine_controller_",
		"machine_deployment_",
		"konnectivity_network_proxy_server_",
		"envoy_cluster_upstream_",
		"kube_node_status_condition",
		"kube_node_info",
		"process_open_fds",
		"process_max_fds",
		"process_resident_memory_bytes",
		"grpc_server_handling_seconds_bucket",
	}
	for _, prefix := range requiredMetricPrefixes {
		if !strings.Contains(config, prefix) {
			t.Errorf("write_relabel_configs regex missing metric family %q.\nRendered:\n%s", prefix, config)
		}
	}
}

func TestPrometheusConfigNoRuleFilesOrAlerting(t *testing.T) {
	config := renderConfig(t, newMinimalTemplateData())

	forbiddenStrings := []string{
		"rule_files:",
		"alerting:",
		"alertmanagers:",
		"alertmanager.monitoring.svc.cluster.local",
	}
	for _, s := range forbiddenStrings {
		if strings.Contains(config, s) {
			t.Errorf("agent mode config must not contain %q (rules and alerting are evaluated at seed level).\nRendered:\n%s", s, config)
		}
	}
}

func TestPrometheusConfigExternalLabels(t *testing.T) {
	config := renderConfig(t, newMinimalTemplateData())

	if !strings.Contains(config, `cluster: "test-cluster"`) {
		t.Errorf("expected external_label cluster=test-cluster in config.\nRendered:\n%s", config)
	}
	if !strings.Contains(config, `seed_cluster: "test-seed"`) {
		t.Errorf("expected external_label seed_cluster=test-seed in config.\nRendered:\n%s", config)
	}
}

func TestPrometheusConfigKonnectivityScrapeJob(t *testing.T) {
	td := newMinimalTemplateData()

	// Without Konnectivity — job must be absent.
	config := renderConfig(t, td)
	if strings.Contains(config, "konnectivity-server") {
		t.Errorf("konnectivity-server scrape job must not appear when Konnectivity is disabled")
	}

	// With Konnectivity enabled — job must be present.
	tdKonnectivity := newMinimalTemplateData()
	// Override IsKonnectivityEnabled by embedding a wrapper.
	configData := &configTemplateData{
		TemplateData:             &konnectivityTemplateData{minimalTemplateData: tdKonnectivity},
		APIServerHost:            tdKonnectivity.cluster.Status.Address.InternalName,
		EtcdTLSConfig:            "ca_file: /etc/etcd/pki/client/ca.crt\ncert_file: /etc/etcd/pki/client/apiserver-etcd-client.crt\nkey_file: /etc/etcd/pki/client/apiserver-etcd-client.key",
		ApiserverTLSConfig:       "ca_file: /etc/kubernetes/ca.crt\ncert_file: /etc/kubernetes/prometheus-client.crt\nkey_file: /etc/kubernetes/prometheus-client.key",
		ScrapingAnnotationPrefix: "monitoring_kubermatic_io",
		RemoteWriteConfig:        buildRemoteWriteConfig(2),
	}
	rendered, err := renderTemplate(prometheusConfig, configData)
	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}
	if !strings.Contains(rendered, "konnectivity-server") {
		t.Errorf("konnectivity-server scrape job must appear when Konnectivity is enabled.\nRendered:\n%s", rendered)
	}
}

func TestPrometheusConfigTunnelingScrapeJob(t *testing.T) {
	td := newMinimalTemplateData()

	// Without Tunneling — envoy-agent job must be absent.
	config := renderConfig(t, td)
	if strings.Contains(config, "envoy-agent") {
		t.Errorf("envoy-agent scrape job must not appear for non-Tunneling expose strategy")
	}

	// With Tunneling — envoy-agent job must be present.
	tdTunnel := newMinimalTemplateData()
	tdTunnel.cluster.Spec.ExposeStrategy = kubermaticv1.ExposeStrategyTunneling
	configTunnel := renderConfig(t, tdTunnel)
	if !strings.Contains(configTunnel, "envoy-agent") {
		t.Errorf("envoy-agent scrape job must appear for Tunneling expose strategy.\nRendered:\n%s", configTunnel)
	}
}

// konnectivityTemplateData wraps minimalTemplateData with Konnectivity enabled.
type konnectivityTemplateData struct {
	*minimalTemplateData
}

func (d *konnectivityTemplateData) IsKonnectivityEnabled() bool { return true }
