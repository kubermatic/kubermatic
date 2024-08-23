/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package gcp

import (
	"strings"
	"testing"
)

func TestCloudConfigAsString(t *testing.T) {
	tests := []struct {
		name     string
		config   *CloudConfig
		expected string
	}{
		{
			name: "minimum test",
			config: &CloudConfig{
				Global: GlobalOpts{
					ProjectID:      "my-project-id",
					LocalZone:      "my-zone",
					NetworkName:    "my-cool-network",
					SubnetworkName: "my-cool-subnetwork",
					TokenURL:       "nil",
					MultiZone:      true,
					Regional:       true,
					NodeTags:       []string{"tag1", "tag2"},
				},
			},
			expected: strings.TrimSpace(`[global]
project-id = "my-project-id"
local-zone = "my-zone"
network-name = "my-cool-network"
subnetwork-name = "my-cool-subnetwork"
token-url = "nil"
multizone = true
regional = true
node-tags = "tag1"
node-tags = "tag2"
`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := test.config.String()
			if err != nil {
				t.Fatalf("failed to convert to string: %v", err)
			}
			s = strings.TrimSpace(s)
			if s != test.expected {
				t.Fatalf("output is not as expected: %s", s)
			}
		})
	}
}
