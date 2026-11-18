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

package cloudcontroller

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

func TestHetznerLoadBalancerLocation(t *testing.T) {
	testcases := []struct {
		name     string
		dc       *kubermaticv1.DatacenterSpecHetzner
		expected string
	}{
		{
			name:     "no datacenter spec",
			expected: "",
		},
		{
			name:     "location is used as is",
			dc:       &kubermaticv1.DatacenterSpecHetzner{Location: "fsn1"},
			expected: "fsn1",
		},
		{
			name:     "location wins over datacenter",
			dc:       &kubermaticv1.DatacenterSpecHetzner{Location: "hel1", Datacenter: "fsn1-dc8"},
			expected: "hel1",
		},
		{
			name:     "location is derived from datacenter name",
			dc:       &kubermaticv1.DatacenterSpecHetzner{Datacenter: "fsn1-dc8"},
			expected: "fsn1",
		},
		{
			name:     "numeric datacenter ID is skipped",
			dc:       &kubermaticv1.DatacenterSpecHetzner{Datacenter: "4"},
			expected: "",
		},
		{
			name:     "neither location nor datacenter",
			dc:       &kubermaticv1.DatacenterSpecHetzner{Network: "net"},
			expected: "",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hetznerLoadBalancerLocation(tc.dc); got != tc.expected {
				t.Errorf("expected location %q, got %q", tc.expected, got)
			}
		})
	}
}
