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

package webhook

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kkpresources "k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestUserAdmissionReadinessProbe(t *testing.T) {
	for _, tc := range []struct {
		name     string
		kubeVirt bool
		active   bool
		wantPort intstr.IntOrString
	}{
		{
			name:     "inactive KubeVirt project keeps existing metrics readiness",
			kubeVirt: true,
			wantPort: intstr.FromString("metrics"),
		},
		{
			name:     "active KubeVirt project waits for user admission",
			kubeVirt: true,
			active:   true,
			wantPort: intstr.FromInt(userWebhookListenPort),
		},
		{
			name:     "non-KubeVirt cluster remains unchanged",
			active:   true,
			wantPort: intstr.FromString("metrics"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := newWebhookDeploymentTestData(tc.kubeVirt, tc.active)
			_, reconciler := DeploymentReconciler(data)()
			deployment, err := reconciler(&appsv1.Deployment{})
			if err != nil {
				t.Fatalf("failed to reconcile Deployment: %v", err)
			}

			container := deployment.Spec.Template.Spec.Containers[0]
			if container.ReadinessProbe == nil || container.ReadinessProbe.TCPSocket == nil {
				t.Fatalf("readiness probe = %#v, want TCP probe", container.ReadinessProbe)
			}
			if got := container.ReadinessProbe.TCPSocket.Port; got != tc.wantPort {
				t.Fatalf("readiness probe port = %#v, want %#v", got, tc.wantPort)
			}
			if len(container.Ports) != 2 {
				t.Fatalf("container ports = %#v, want existing admission and metrics ports only", container.Ports)
			}
			if container.Ports[0].Name != "admission" || container.Ports[1].Name != "metrics" {
				t.Fatalf("container ports = %#v, want existing admission and metrics ports only", container.Ports)
			}
		})
	}
}

func newWebhookDeploymentTestData(kubeVirt, acceleratorQuotaEnabled bool) *kkpresources.TemplateData {
	cloud := kubermaticv1.CloudSpec{Fake: &kubermaticv1.FakeCloudSpec{}}
	datacenter := kubermaticv1.DatacenterSpec{Fake: &kubermaticv1.DatacenterSpecFake{}}
	if kubeVirt {
		cloud = kubermaticv1.CloudSpec{Kubevirt: &kubermaticv1.KubevirtCloudSpec{}}
		datacenter = kubermaticv1.DatacenterSpec{Kubevirt: &kubermaticv1.DatacenterSpecKubevirt{}}
	}

	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: "test-project",
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: cloud,
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "cluster-test",
			Address: kubermaticv1.ClusterAddress{
				InternalName: "apiserver.cluster-test.svc.cluster.local",
			},
		},
	}

	return kkpresources.NewTemplateDataBuilder().
		WithCluster(cluster).
		WithDatacenter(&kubermaticv1.Datacenter{Spec: datacenter}).
		WithSeed(&kubermaticv1.Seed{}).
		WithKubermaticImage("quay.io/kubermatic/kubermatic").
		WithVersions(kubermatic.GetFakeVersions()).
		WithKubeVirtAcceleratorQuota(acceleratorQuotaEnabled).
		Build()
}
