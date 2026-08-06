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

package resources

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

func TestGatewayAPIProtectionDisabled(t *testing.T) {
	clusterWith := func(disabled bool) *kubermaticv1.Cluster {
		return &kubermaticv1.Cluster{
			Spec: kubermaticv1.ClusterSpec{
				KubeLB: &kubermaticv1.KubeLB{DisableGatewayAPIProtection: disabled},
			},
		}
	}

	testCases := []struct {
		name string
		// datacenterFlag is what the seed-controller-manager passed on the command line.
		datacenterFlag bool
		cluster        *kubermaticv1.Cluster
		expected       bool
	}{
		{
			name:     "nothing configured keeps the guard",
			cluster:  clusterWith(false),
			expected: false,
		},
		{
			name:           "the datacenter disables it for every cluster",
			datacenterFlag: true,
			cluster:        clusterWith(false),
			expected:       true,
		},
		{
			// The case that separates these semantics from precedence: a cluster cannot bring the guard
			// back once an admin has turned it off for the whole datacenter.
			name:           "a cluster cannot re-enable what the datacenter disabled",
			datacenterFlag: true,
			cluster:        clusterWith(false),
			expected:       true,
		},
		{
			name:     "a cluster can opt out on its own",
			cluster:  clusterWith(true),
			expected: true,
		},
		{
			// Clusters without any kubeLB block are the common case and must not panic.
			name:     "a cluster without kubeLB settings keeps the guard",
			cluster:  &kubermaticv1.Cluster{},
			expected: false,
		},
		{
			name:     "a nil cluster keeps the guard",
			cluster:  nil,
			expected: false,
		},
		{
			name:           "a nil cluster still honours the datacenter",
			datacenterFlag: true,
			cluster:        nil,
			expected:       true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			r := &reconciler{kubeLBDisableGatewayAPIProtection: test.datacenterFlag}

			if got := r.gatewayAPIProtectionDisabled(test.cluster); got != test.expected {
				t.Errorf("expected %v, got %v", test.expected, got)
			}
		})
	}
}
