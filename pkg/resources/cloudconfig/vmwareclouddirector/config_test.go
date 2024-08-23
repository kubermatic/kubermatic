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

package vmwareclouddirector

import (
	"strings"
	"testing"
)

func TestCloudConfigToString(t *testing.T) {
	tests := []struct {
		name     string
		config   *CloudConfig
		expected string
	}{
		{
			name: "conversion-test",
			config: &CloudConfig{
				VCD: VCDConfig{
					Host:         "https://example.com",
					VDC:          "TEST_VDC",
					VApp:         "TEST_VAPP",
					Organization: "TEST_ORG",
				},
				ClusterID: "test",
			},
			expected: strings.TrimSpace(`
vcd:
    host: https://example.com
    org: TEST_ORG
    vdc: TEST_VDC
    vAppName: TEST_VAPP
clusterid: test
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
