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
	"reflect"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/diff"

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

	if !diff.SemanticallyEqual(defaults, backup) {
		t.Fatalf("The defaults have changed:\n%v", diff.ObjectDiff(defaults, backup))
	}
}

func TestGetNodePortsAllowedIPRanges(t *testing.T) {
	testCases := []struct {
		name                string
		cluster             *kubermaticv1.Cluster
		allowedIPRanges     kubermaticv1.NetworkRanges
		allowedIPRange      string
		seedAllowedIPRanges kubermaticv1.NetworkRanges
		expectedResult      kubermaticv1.NetworkRanges
	}{
		{
			name: "Duplicate entry in allowedIPRanges",
			allowedIPRanges: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{"10.10.10.0/24", "::/0"},
			},
			allowedIPRange: "10.10.10.0/24",
			expectedResult: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{"10.10.10.0/24", "::/0"},
			},
		},
		{
			name: "Unique entries in allowedIPRanges",
			allowedIPRanges: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{"20.20.20.0/24", "::/0"},
			},
			allowedIPRange: "10.10.10.0/24",
			expectedResult: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{"20.20.20.0/24", "::/0", "10.10.10.0/24"},
			},
		},
		{
			name: "Empty entry in allowedIPRange",
			allowedIPRanges: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{"20.20.20.0/24", "::/0"},
			},
			allowedIPRange: "",
			expectedResult: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{"20.20.20.0/24", "::/0"},
			},
		},
		{
			name: "No specified IP ranges, IPv4 only cluster",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{
							CIDRBlocks: []string{"192.168.0.0/16"},
						},
					},
				},
			},
			expectedResult: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{IPv4MatchAnyCIDR},
			},
		},
		{
			name: "No specified IP ranges, IPv6 only cluster",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{
							CIDRBlocks: []string{"fd00::/8"},
						},
					},
				},
			},
			expectedResult: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{IPv6MatchAnyCIDR},
			},
		},
		{
			name: "No specified IP ranges, dual-stack cluster",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						Pods: kubermaticv1.NetworkRanges{
							CIDRBlocks: []string{"192.168.0.0/16", "fd00::/8"},
						},
					},
				},
			},
			expectedResult: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{IPv4MatchAnyCIDR, IPv6MatchAnyCIDR},
			},
		},
		{
			name: "Specified seed allowed IP ranges",
			seedAllowedIPRanges: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{"10.10.10.0/24", "::/0"},
			},
			expectedResult: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{"10.10.10.0/24", "::/0"},
			},
		},
		{
			name: "Specified seed allowed IP ranges but there are set cluster allowed IP ranges",
			allowedIPRanges: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{"20.20.20.0/24", "::/0"},
			},
			seedAllowedIPRanges: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{"10.10.10.0/24", "::/0"},
			},
			expectedResult: kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{"20.20.20.0/24", "::/0"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetNodePortsAllowedIPRanges(tc.cluster, &tc.allowedIPRanges, tc.allowedIPRange, &tc.seedAllowedIPRanges)
			if !reflect.DeepEqual(result, tc.expectedResult) {
				t.Errorf("wrong result, expected: %s, result: %s", tc.expectedResult, result)
			}
		})
	}
}
