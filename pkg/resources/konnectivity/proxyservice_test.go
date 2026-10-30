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

package konnectivity

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestServiceReconcilerPublishesNotReadyAddresses(t *testing.T) {
	strategies := []struct {
		strategy    kubermaticv1.ExposeStrategy
		serviceType corev1.ServiceType
		exposeKey   string
		exposeValue string
	}{
		{kubermaticv1.ExposeStrategyNodePort, corev1.ServiceTypeNodePort, nodeportproxy.DefaultExposeAnnotationKey, "NodePort"},
		{kubermaticv1.ExposeStrategyLoadBalancer, corev1.ServiceTypeNodePort, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey, "true"},
		{kubermaticv1.ExposeStrategyTunneling, corev1.ServiceTypeClusterIP, nodeportproxy.DefaultExposeAnnotationKey, "SNI,Tunneling"},
	}
	for _, strategy := range strategies {
		for _, state := range []string{"new", "existing", "already published"} {
			t.Run(string(strategy.strategy)+"/"+state, func(t *testing.T) {
				service := &corev1.Service{}
				expectedNodePort := int32(0)
				if state != "new" {
					service.Spec.Ports = []corev1.ServicePort{{NodePort: 32017}}
					service.Spec.ClusterIP = "10.0.0.17"
					service.Annotations = map[string]string{"example.com/custom": "preserved"}
					if strategy.strategy != kubermaticv1.ExposeStrategyTunneling {
						expectedNodePort = 32017
					}
				}
				service.Spec.PublishNotReadyAddresses = state == "already published"
				name, reconcile := ServiceReconciler(strategy.strategy, "api.example.test")()
				if name != "konnectivity-server" {
					t.Fatalf("unexpected Service name %q", name)
				}
				got, err := reconcile(service)
				if err != nil {
					t.Fatal(err)
				}
				if !got.Spec.PublishNotReadyAddresses {
					t.Error("agents cannot reach the proxy sidecar before the API server Pod is ready")
				}
				if got.Spec.Type != strategy.serviceType {
					t.Errorf("Service type = %q, want %q", got.Spec.Type, strategy.serviceType)
				}
				if diff := cmp.Diff(map[string]string{"app": "apiserver"}, got.Spec.Selector); diff != "" {
					t.Errorf("unexpected selector (-want +got):\n%s", diff)
				}
				wantPort := corev1.ServicePort{Name: "secure", Port: 443, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromInt(8132), NodePort: expectedNodePort}
				if diff := cmp.Diff([]corev1.ServicePort{wantPort}, got.Spec.Ports); diff != "" {
					t.Errorf("unexpected ports (-want +got):\n%s", diff)
				}
				if got.Annotations[strategy.exposeKey] != strategy.exposeValue {
					t.Errorf("expose annotation = %q, want %q", got.Annotations[strategy.exposeKey], strategy.exposeValue)
				}
				if state != "new" && (got.Spec.ClusterIP != "10.0.0.17" || got.Annotations["example.com/custom"] != "preserved") {
					t.Error("reconciliation changed an existing Service's ClusterIP or custom annotation")
				}
				first := got.DeepCopy()
				again, err := reconcile(got)
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(first, again); diff != "" {
					t.Errorf("reconciliation is not idempotent (-first +second):\n%s", diff)
				}
			})
		}
	}
}
