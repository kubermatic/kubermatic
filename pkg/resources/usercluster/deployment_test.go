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

package usercluster

import (
	"encoding/json"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAdminDisabledGatewayAPIProtection(t *testing.T) {
	seedWith := func(disabled bool) *kubermaticv1.Seed {
		return &kubermaticv1.Seed{
			Spec: kubermaticv1.SeedSpec{
				KubeLB: &kubermaticv1.KubeLBSeedSettings{
					KubeLBSettings: kubermaticv1.KubeLBSettings{DisableGatewayAPIProtection: disabled},
				},
			},
		}
	}
	dcWith := func(disabled bool) *kubermaticv1.Datacenter {
		return &kubermaticv1.Datacenter{
			Spec: kubermaticv1.DatacenterSpec{
				KubeLB: &kubermaticv1.KubeLBDatacenterSettings{
					KubeLBSettings: kubermaticv1.KubeLBSettings{DisableGatewayAPIProtection: disabled},
				},
			},
		}
	}

	testCases := []struct {
		name     string
		seed     *kubermaticv1.Seed
		dc       *kubermaticv1.Datacenter
		expected bool
	}{
		{
			name:     "nothing configured keeps the guard",
			seed:     seedWith(false),
			dc:       dcWith(false),
			expected: false,
		},
		{
			name:     "the seed disables it for every datacenter",
			seed:     seedWith(true),
			dc:       dcWith(false),
			expected: true,
		},
		{
			name:     "a single datacenter disables it",
			seed:     seedWith(false),
			dc:       dcWith(true),
			expected: true,
		},
		{
			// Neither level can re-enable what the other turned off.
			name:     "both disabling it is still disabled",
			seed:     seedWith(true),
			dc:       dcWith(true),
			expected: true,
		},
		{
			// Seeds and datacenters without any kubeLB block are the common case.
			name:     "missing kubeLB blocks keep the guard",
			seed:     &kubermaticv1.Seed{},
			dc:       &kubermaticv1.Datacenter{},
			expected: false,
		},
		{
			name:     "nil seed and datacenter keep the guard",
			expected: false,
		},
		{
			name:     "a nil seed does not hide the datacenter setting",
			dc:       dcWith(true),
			expected: true,
		},
		{
			name:     "a nil datacenter does not hide the seed setting",
			seed:     seedWith(true),
			expected: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := adminDisabledGatewayAPIProtection(test.seed, test.dc); got != test.expected {
				t.Errorf("expected %v, got %v", test.expected, got)
			}
		})
	}
}

func TestGetLabelArgsValue(t *testing.T) {
	testCases := []struct {
		name           string
		initialLabels  map[string]string
		expectedLabels map[string]string
	}{
		{
			name:           "Labels get applied",
			initialLabels:  map[string]string{"foo": "bar"},
			expectedLabels: map[string]string{"foo": "bar"},
		},
		{
			name:           "Protected labels do not get applied",
			initialLabels:  map[string]string{"foo": "bar", "project-id": "my-project", "worker-name": "w"},
			expectedLabels: map[string]string{"foo": "bar"},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cluster := &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Labels: tc.initialLabels,
				},
			}
			result, err := getLabelsArgValue(cluster)
			if err != nil {
				t.Fatalf("error when calling getLabelsArgValue: %v", err)
			}

			actualLabels := map[string]string{}
			if err := json.Unmarshal([]byte(result), &actualLabels); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}

			if !diff.SemanticallyEqual(tc.expectedLabels, actualLabels) {
				t.Fatalf("actual labels do not match expected labels:\n%v", diff.ObjectDiff(tc.expectedLabels, actualLabels))
			}
		})
	}
}
