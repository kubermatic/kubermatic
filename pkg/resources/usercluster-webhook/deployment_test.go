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
	"encoding/json"
	"slices"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kkpresources "k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKubeVirtAcceleratorQuotaArgument(t *testing.T) {
	testCases := []struct {
		name         string
		kubeVirt     bool
		enabled      bool
		wantArgument bool
	}{
		{
			name:     "disabled",
			kubeVirt: true,
		},
		{
			name:         "enabled",
			kubeVirt:     true,
			enabled:      true,
			wantArgument: true,
		},
		{
			name:    "not rendered for another provider",
			enabled: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := newWebhookDeploymentTestData(tc.kubeVirt, tc.enabled)
			_, reconcile := DeploymentReconciler(data)()

			deployment, err := reconcile(&appsv1.Deployment{})
			if err != nil {
				t.Fatalf("failed to reconcile user-cluster-webhook deployment: %v", err)
			}

			hasArgument := deploymentContainsArgument(deployment, "-kubevirt-accelerator-quota")
			if hasArgument != tc.wantArgument {
				t.Fatalf("expected accelerator quota argument present=%t, got %t", tc.wantArgument, hasArgument)
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

func deploymentContainsArgument(deployment *appsv1.Deployment, argument string) bool {
	for _, containers := range [][]corev1.Container{
		deployment.Spec.Template.Spec.InitContainers,
		deployment.Spec.Template.Spec.Containers,
	} {
		for _, container := range containers {
			if slices.Contains(container.Args, argument) {
				return true
			}

			for i := 1; i < len(container.Args); i++ {
				if container.Args[i-1] != "-command" {
					continue
				}

				wrappedCommand := struct {
					Args []string `json:"args"`
				}{}
				if err := json.Unmarshal([]byte(container.Args[i]), &wrappedCommand); err == nil && slices.Contains(wrappedCommand.Args, argument) {
					return true
				}
			}
		}
	}

	return false
}
