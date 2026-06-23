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

	"github.com/stretchr/testify/require"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticsemver "k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	require.Contains(t, config, "target_label: instance")
	require.Contains(t, config, "replacement: 'apiserver-external.cluster-test-cluster.svc.cluster.local.'")
	require.NotContains(t, envoyAgentScrapeConfig(t, config), "replacement: $1:$2\n    target_label: __address__")
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
