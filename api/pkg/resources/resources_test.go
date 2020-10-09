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

package resources

import (
	"testing"

	"github.com/go-test/deep"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestInClusterApiserverIP(t *testing.T) {
	testCases := []struct {
		name           string
		cidr           string
		expectedResult string
	}{
		{
			name:           "Parse /24",
			cidr:           "10.10.10.0/24",
			expectedResult: "10.10.10.1",
		},
		{
			name:           "Parse /20",
			cidr:           "10.240.20.0/20",
			expectedResult: "10.240.16.1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cluster := &kubermaticv1.Cluster{}
			cluster.Spec.ClusterNetwork.Services.CIDRBlocks = []string{tc.cidr}

			result, err := InClusterApiserverIP(cluster)
			if err != nil {
				t.Fatalf("error: %v", err)
			}

			if result.String() != tc.expectedResult {
				t.Errorf("wrong result, expected: %s, result: %s", tc.expectedResult, result.String())
			}
		})
	}
}

func TestUserClusterDNSResolverIP(t *testing.T) {
	testCases := []struct {
		name           string
		cidr           string
		expectedResult string
	}{
		{
			name:           "Parse /24",
			cidr:           "10.10.10.0/24",
			expectedResult: "10.10.10.10",
		},
		{
			name:           "Parse /20",
			cidr:           "10.240.20.0/20",
			expectedResult: "10.240.16.10",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cluster := &kubermaticv1.Cluster{}
			cluster.Spec.ClusterNetwork.Services.CIDRBlocks = []string{tc.cidr}

			result, err := UserClusterDNSResolverIP(cluster)
			if err != nil {
				t.Fatalf("error: %v", err)
			}

			if result != tc.expectedResult {
				t.Errorf("wrong result, expected: %s, result: %s", tc.expectedResult, result)
			}
		})
	}
}

func TestSetResourceRequirements(t *testing.T) {
	defaultResourceRequirements := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("64Mi"),
			corev1.ResourceCPU:    resource.MustParse("20m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("1"),
		},
	}

	tests := []struct {
		name                 string
		containers           []corev1.Container
		annotations          map[string]string
		overrides            map[string]*corev1.ResourceRequirements
		defaultRequirements  map[string]*corev1.ResourceRequirements
		expectedRequirements map[string]*corev1.ResourceRequirements
	}{
		{
			name: "resource requirements set by vpa-updater",
			containers: []corev1.Container{
				{
					Name: "test",
				},
			},
			annotations: map[string]string{
				kubermaticv1.UpdatedByVPALabelKey: `[{"name":"test","requires":{"limits":{"cpu":"1","memory":"512Mi"},"requests":{"cpu":"20m","memory":"64Mi"}}}]`,
			},
			defaultRequirements: map[string]*corev1.ResourceRequirements{
				"test": defaultResourceRequirements.DeepCopy(),
			},
			expectedRequirements: map[string]*corev1.ResourceRequirements{
				"test": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("64Mi"),
						corev1.ResourceCPU:    resource.MustParse("20m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
						corev1.ResourceCPU:    resource.MustParse("1"),
					},
				},
			},
		},
		{
			name: "resource requirements set by vpa-updater (multiple containers)",
			containers: []corev1.Container{
				{
					Name: "test-1",
				},
				{
					Name: "test-2",
				},
				{
					Name: "test-3",
				},
			},
			annotations: map[string]string{
				kubermaticv1.UpdatedByVPALabelKey: `[{"name":"test-1","requires":{"limits":{"cpu":"100m","memory":"32Mi"},"requests":{"cpu":"10m","memory":"16Mi"}}},{"name":"test-2","requires":{"limits":{"cpu":"2","memory":"256Mi"},"requests":{"cpu":"20m","memory":"64Mi"}}},{"name":"test-3","requires":{"limits":{"cpu":"1","memory":"2Gi"},"requests":{"cpu":"500m","memory":"1Gi"}}}]`,
			},
			defaultRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": defaultResourceRequirements.DeepCopy(),
				"test-2": defaultResourceRequirements.DeepCopy(),
				"test-3": defaultResourceRequirements.DeepCopy(),
			},
			expectedRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("16Mi"),
						corev1.ResourceCPU:    resource.MustParse("10m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("32Mi"),
						corev1.ResourceCPU:    resource.MustParse("100m"),
					},
				},
				"test-2": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("64Mi"),
						corev1.ResourceCPU:    resource.MustParse("20m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("256Mi"),
						corev1.ResourceCPU:    resource.MustParse("2"),
					},
				},
				"test-3": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						corev1.ResourceCPU:    resource.MustParse("500m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("2Gi"),
						corev1.ResourceCPU:    resource.MustParse("1"),
					},
				},
			},
		},
		{
			name: "resource requirements set by vpa-updater (multiple containers, one not managed by vpa)",
			containers: []corev1.Container{
				{
					Name: "test-1",
				},
				{
					Name: "test-2",
				},
				{
					Name: "test-3",
				},
			},
			annotations: map[string]string{
				kubermaticv1.UpdatedByVPALabelKey: `[{"name":"test-1","requires":{"limits":{"cpu":"100m","memory":"32Mi"},"requests":{"cpu":"10m","memory":"16Mi"}}},{"name":"test-3","requires":{"limits":{"cpu":"1","memory":"2Gi"},"requests":{"cpu":"500m","memory":"1Gi"}}}]`,
			},
			defaultRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": defaultResourceRequirements.DeepCopy(),
				"test-2": defaultResourceRequirements.DeepCopy(),
				"test-3": defaultResourceRequirements.DeepCopy(),
			},
			expectedRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("16Mi"),
						corev1.ResourceCPU:    resource.MustParse("10m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("32Mi"),
						corev1.ResourceCPU:    resource.MustParse("100m"),
					},
				},
				"test-2": defaultResourceRequirements.DeepCopy(),
				"test-3": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						corev1.ResourceCPU:    resource.MustParse("500m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("2Gi"),
						corev1.ResourceCPU:    resource.MustParse("1"),
					},
				},
			},
		},
		{
			name: "resource requirements set by vpa-updater (multiple containers, one using overrides)",
			containers: []corev1.Container{
				{
					Name: "test-1",
				},
				{
					Name: "test-2",
				},
				{
					Name: "test-3",
				},
			},
			annotations: map[string]string{
				kubermaticv1.UpdatedByVPALabelKey: `[{"name":"test-1","requires":{"limits":{"cpu":"100m","memory":"32Mi"},"requests":{"cpu":"10m","memory":"16Mi"}}},{"name":"test-3","requires":{"limits":{"cpu":"1","memory":"2Gi"},"requests":{"cpu":"500m","memory":"1Gi"}}}]`,
			},
			overrides: map[string]*corev1.ResourceRequirements{
				"test-2": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
						corev1.ResourceCPU:    resource.MustParse("200m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("2Gi"),
						corev1.ResourceCPU:    resource.MustParse("3"),
					},
				},
			},
			defaultRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": defaultResourceRequirements.DeepCopy(),
				"test-2": defaultResourceRequirements.DeepCopy(),
				"test-3": defaultResourceRequirements.DeepCopy(),
			},
			expectedRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("16Mi"),
						corev1.ResourceCPU:    resource.MustParse("10m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("32Mi"),
						corev1.ResourceCPU:    resource.MustParse("100m"),
					},
				},
				"test-2": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
						corev1.ResourceCPU:    resource.MustParse("200m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("2Gi"),
						corev1.ResourceCPU:    resource.MustParse("3"),
					},
				},
				"test-3": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						corev1.ResourceCPU:    resource.MustParse("500m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("2Gi"),
						corev1.ResourceCPU:    resource.MustParse("1"),
					},
				},
			},
		},
		{
			name: "resource requirements set by vpa-updater (multiple containers, one using overrides, one defaults)",
			containers: []corev1.Container{
				{
					Name: "test-1",
				},
				{
					Name: "test-2",
				},
				{
					Name: "test-3",
				},
			},
			annotations: map[string]string{
				kubermaticv1.UpdatedByVPALabelKey: `[{"name":"test-3","requires":{"limits":{"cpu":"1","memory":"2Gi"},"requests":{"cpu":"500m","memory":"1Gi"}}}]`,
			},
			overrides: map[string]*corev1.ResourceRequirements{
				"test-2": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
						corev1.ResourceCPU:    resource.MustParse("200m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("2Gi"),
						corev1.ResourceCPU:    resource.MustParse("3"),
					},
				},
			},
			defaultRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": defaultResourceRequirements.DeepCopy(),
				"test-2": defaultResourceRequirements.DeepCopy(),
				"test-3": defaultResourceRequirements.DeepCopy(),
			},
			expectedRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": defaultResourceRequirements.DeepCopy(),
				"test-2": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
						corev1.ResourceCPU:    resource.MustParse("200m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("2Gi"),
						corev1.ResourceCPU:    resource.MustParse("3"),
					},
				},
				"test-3": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						corev1.ResourceCPU:    resource.MustParse("500m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("2Gi"),
						corev1.ResourceCPU:    resource.MustParse("1"),
					},
				},
			},
		},
		{
			name: "empty vpa label (expected to take defaults)",
			containers: []corev1.Container{
				{
					Name: "test-1",
				},
			},
			annotations: map[string]string{
				kubermaticv1.UpdatedByVPALabelKey: "",
			},
			defaultRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": defaultResourceRequirements.DeepCopy(),
			},
			expectedRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": defaultResourceRequirements.DeepCopy(),
			},
		},
		{
			name: "no vpa label (expected to take defaults)",
			containers: []corev1.Container{
				{
					Name: "test-1",
				},
			},
			annotations: map[string]string{},
			defaultRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": defaultResourceRequirements.DeepCopy(),
			},
			expectedRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": defaultResourceRequirements.DeepCopy(),
			},
		},
		{
			name: "two containers, no vpa label, both containers with overrides",
			containers: []corev1.Container{
				{
					Name: "test-1",
				},
				{
					Name: "test-2",
				},
			},
			annotations: map[string]string{},
			overrides: map[string]*corev1.ResourceRequirements{
				"test-1": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("16Mi"),
						corev1.ResourceCPU:    resource.MustParse("10m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("32Mi"),
						corev1.ResourceCPU:    resource.MustParse("100m"),
					},
				},
				"test-2": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						corev1.ResourceCPU:    resource.MustParse("500m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("2Gi"),
						corev1.ResourceCPU:    resource.MustParse("1"),
					},
				},
			},
			defaultRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": defaultResourceRequirements.DeepCopy(),
				"test-2": defaultResourceRequirements.DeepCopy(),
			},
			expectedRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("16Mi"),
						corev1.ResourceCPU:    resource.MustParse("10m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("32Mi"),
						corev1.ResourceCPU:    resource.MustParse("100m"),
					},
				},
				"test-2": {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						corev1.ResourceCPU:    resource.MustParse("500m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("2Gi"),
						corev1.ResourceCPU:    resource.MustParse("1"),
					},
				},
			},
		},
		{
			name: "one container, no vpa label, only cpu requirements set",
			containers: []corev1.Container{
				{
					Name: "test-1",
				},
			},
			annotations: map[string]string{},
			defaultRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				},
			},
			expectedRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				},
			},
		},
		{
			name: "one container, no vpa label, no resource requirements",
			containers: []corev1.Container{
				{
					Name: "test-1",
				},
			},
			annotations: map[string]string{},
			defaultRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": {},
			},
			expectedRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": {},
			},
		},
		{
			name: "default requirements containing more containers then expected",
			containers: []corev1.Container{
				{
					Name: "test-2",
				},
			},
			annotations: map[string]string{},
			defaultRequirements: map[string]*corev1.ResourceRequirements{
				"test-1": defaultResourceRequirements.DeepCopy(),
				"test-2": defaultResourceRequirements.DeepCopy(),
			},
			expectedRequirements: map[string]*corev1.ResourceRequirements{
				"test-2": defaultResourceRequirements.DeepCopy(),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := SetResourceRequirements(tc.containers, tc.defaultRequirements, tc.overrides, tc.annotations)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			for _, container := range tc.containers {
				if tc.expectedRequirements[container.Name].Requests.Cpu() != nil && !container.Resources.Requests.Cpu().Equal(*tc.expectedRequirements[container.Name].Requests.Cpu()) {
					t.Errorf("invalid resource requirements: expected requested cpu %v, but got %v", container.Resources.Requests.Cpu(), tc.expectedRequirements[container.Name].Requests.Cpu())
				}
				if tc.expectedRequirements[container.Name].Requests.Memory() != nil && !container.Resources.Requests.Memory().Equal(*tc.expectedRequirements[container.Name].Requests.Memory()) {
					t.Errorf("invalid resource requirements: expected requested memory %v, but got %v", container.Resources.Requests.Memory(), tc.expectedRequirements[container.Name].Requests.Memory())
				}
				if tc.expectedRequirements[container.Name].Limits.Cpu() != nil && !container.Resources.Limits.Cpu().Equal(*tc.expectedRequirements[container.Name].Limits.Cpu()) {
					t.Errorf("invalid resource requirements: expected cpu limit %v, but got %v", container.Resources.Requests.Cpu(), tc.expectedRequirements[container.Name].Requests.Cpu())
				}
				if tc.expectedRequirements[container.Name].Limits.Memory() != nil && !container.Resources.Limits.Memory().Equal(*tc.expectedRequirements[container.Name].Limits.Memory()) {
					t.Errorf("invalid resource requirements: expected memory limit %v, but got %v", container.Resources.Limits.Memory(), tc.expectedRequirements[container.Name].Limits.Memory())
				}
			}
		})
	}
}

func TestSetResourceRequirementsDoesNotChangeDefaults(t *testing.T) {
	defaults := map[string]*corev1.ResourceRequirements{
		"test": {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("100Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
	}

	backup := map[string]*corev1.ResourceRequirements{}
	for k, v := range defaults {
		backup[k] = v.DeepCopy()
	}

	overrides := map[string]*corev1.ResourceRequirements{
		"test": {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("200Mi"),
				corev1.ResourceCPU:    resource.MustParse("200m"),
			},
		},
	}

	err := SetResourceRequirements(nil, defaults, overrides, nil)
	if err != nil {
		t.Fatalf("function should not have returned an error, but got: %v", err)
	}

	if diff := deep.Equal(backup, defaults); diff != nil {
		t.Fatalf("The defaults have changed: %v\n", diff)
	}
}

func TestKubernetesComponentImage(t *testing.T) {
	tests := []struct {
		name         string
		kc           KubernetesComponent
		cg           ImageContextGetter
		currentImage string
		wantErr      bool
		want         string
	}{
		{
			name: "Hyperkube image when nil cluster is returned by ImageContextGetter",
			kc:   Hyperkube,
			cg: &fakeImageContextGetter{
				cluster:  nil,
				registry: "my-registry.org",
			},
			currentImage: "",
			wantErr:      true,
		},
		{
			name: "Hyperkube image when current image is empty",
			kc:   Hyperkube,
			cg: &fakeImageContextGetter{
				cluster: &kubermaticv1.Cluster{
					Spec: kubermaticv1.ClusterSpec{
						Version: *semver.NewSemverOrDie("1.18.5"),
					},
				},
				registry: "my-registry.org",
			},
			currentImage: "",
			want:         "my-registry.org/hyperkube-amd64:v1.18.5",
		},
		{
			name: "Hyperkube image when current image is using legacy repository",
			kc:   Hyperkube,
			cg: &fakeImageContextGetter{
				cluster: &kubermaticv1.Cluster{
					Spec: kubermaticv1.ClusterSpec{
						Version: *semver.NewSemverOrDie("1.18.5"),
					},
				},
				registry: "my-registry.org",
			},
			currentImage: "my-registry.org/google_containers/hyperkube-amd64:v1.18.5",
			want:         "my-registry.org/google_containers/hyperkube-amd64:v1.18.5",
		},
		{
			name: "Hyperkube when current image is using legacy repository, but version does not match",
			kc:   Hyperkube,
			cg: &fakeImageContextGetter{
				cluster: &kubermaticv1.Cluster{
					Spec: kubermaticv1.ClusterSpec{
						Version: *semver.NewSemverOrDie("1.18.6"),
					},
				},
				registry: "my-registry.org",
			},
			currentImage: "my-registry.org/google_containers/hyperkube-amd64:v1.18.5",
			want:         "my-registry.org/hyperkube-amd64:v1.18.6",
		},
		{
			name: "CoreDNS image when current image is empty",
			kc:   CoreDNS,
			cg: &fakeImageContextGetter{
				cluster:  &kubermaticv1.Cluster{},
				registry: "my-registry.org",
			},
			currentImage: "",
			want:         "my-registry.org/coredns:1.3.1",
		},
		{
			name: "CoreDNS image when current image is using legacy repository",
			kc:   CoreDNS,
			cg: &fakeImageContextGetter{
				cluster:  &kubermaticv1.Cluster{},
				registry: "my-registry.org",
			},
			currentImage: "my-registry.org/google_containers/coredns:1.3.1",
			want:         "my-registry.org/google_containers/coredns:1.3.1",
		},
		{
			name: "MetricsServer image when current image is empty",
			kc:   MetricsServer,
			cg: &fakeImageContextGetter{
				cluster:  &kubermaticv1.Cluster{},
				registry: "my-registry.org",
			},
			currentImage: "",
			want:         "my-registry.org/metrics-server-amd64:v0.3.6",
		},
		{
			name: "MetricsServer image when current image is using legacy repository",
			kc:   MetricsServer,
			cg: &fakeImageContextGetter{
				cluster:  &kubermaticv1.Cluster{},
				registry: "my-registry.org",
			},
			currentImage: "my-registry.org/google_containers/metrics-server-amd64:v0.3.6",
			want:         "my-registry.org/google_containers/metrics-server-amd64:v0.3.6",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.kc.Image(tt.cg, tt.currentImage)
			if (err != nil) != tt.wantErr {
				t.Errorf("Error expected = %v, but got: %v", tt.wantErr, err)
			}
			if got != tt.want {
				t.Errorf("Expected %q, but got %q", tt.want, got)
			}
		})
	}
}

type fakeImageContextGetter struct {
	cluster  *kubermaticv1.Cluster
	registry string
}

func (f *fakeImageContextGetter) ImageRegistry(string) string {
	return f.registry
}

func (f *fakeImageContextGetter) Cluster() *kubermaticv1.Cluster {
	return f.cluster
}
