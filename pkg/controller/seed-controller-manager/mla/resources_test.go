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
	"path/filepath"
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGatewayConfigMapResolver(t *testing.T) {
	cluster := &kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	_, reconcile := GatewayConfigMapReconciler(cluster, "mla", nil)()
	cm, err := reconcile(&corev1.ConfigMap{})
	if err != nil {
		t.Fatalf("failed to reconcile ConfigMap: %v", err)
	}

	config := cm.Data["nginx.conf"]

	if strings.Contains(config, "kube-dns.kube-system") {
		t.Error("nginx config must not hardcode the kube-dns Service as resolver")
	}

	if !strings.Contains(nginxScript, "> "+resolverConfigFile) {
		t.Errorf("nginx startup script does not write the resolver config to %q", resolverConfigFile)
	}

	// the resolver config written by the startup script must be included by the nginx config
	for _, line := range strings.Split(config, "\n") {
		pattern, ok := strings.CutPrefix(strings.TrimSpace(line), "include ")
		if !ok {
			continue
		}

		matched, err := filepath.Match(strings.TrimSuffix(pattern, ";"), resolverConfigFile)
		if err != nil {
			t.Fatalf("invalid include pattern %q: %v", pattern, err)
		}
		if matched {
			return
		}
	}

	t.Errorf("nginx config does not include the resolver config %q", resolverConfigFile)
}
