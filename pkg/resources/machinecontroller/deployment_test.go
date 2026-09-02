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

package machinecontroller

import (
	"slices"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetFlags(t *testing.T) {
	testcases := []struct {
		name      string
		features  map[string]bool
		overrides *kubermaticv1.MachineControllerSettings
		expected  []string
	}{
		{
			name:      "no overrides, no flags added",
			features:  map[string]bool{},
			overrides: nil,
			expected: []string{
				"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				"-health-probe-address", "0.0.0.0:8085",
				"-metrics-address", "0.0.0.0:8080",
				"-ca-bundle", "/etc/kubernetes/pki/ca-bundle/ca-bundle.pem",
				"-node-csr-approver",
			},
		},
		{
			name:      "empty machineController block adds nothing",
			features:  map[string]bool{},
			overrides: &kubermaticv1.MachineControllerSettings{},
			expected: []string{
				"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				"-health-probe-address", "0.0.0.0:8085",
				"-metrics-address", "0.0.0.0:8080",
				"-ca-bundle", "/etc/kubernetes/pki/ca-bundle/ca-bundle.pem",
				"-node-csr-approver",
			},
		},
		{
			name:     "skip eviction override is passed as duration",
			features: map[string]bool{kubermaticv1.ClusterFeatureExternalCloudProvider: true},
			overrides: &kubermaticv1.MachineControllerSettings{
				SkipEvictionAfter: &metav1.Duration{Duration: 4 * time.Hour},
			},
			expected: []string{
				"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
				"-health-probe-address", "0.0.0.0:8085",
				"-metrics-address", "0.0.0.0:8080",
				"-ca-bundle", "/etc/kubernetes/pki/ca-bundle/ca-bundle.pem",
				"-node-csr-approver",
				"-node-external-cloud-provider",
				"-skip-eviction-after", "4h0m0s",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			flags := getFlags(tc.features, tc.overrides)
			if !slices.Equal(flags, tc.expected) {
				t.Fatalf("expected %v, got %v", tc.expected, flags)
			}
		})
	}
}
