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

package kubermaticseed

import (
	"strings"
	"testing"

	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/util/yamled"
)

func TestSeedExternalGatewayFromHTTPRouteValues(t *testing.T) {
	testCases := []struct {
		name   string
		values string
		want   bool
	}{
		{
			name: "default Gateway values deploy bundled seed Gateway controller",
			values: `
migrateGatewayAPI: true
httpRoute:
  gatewayName: kubermatic
  gatewayNamespace: kubermatic
`,
			want: false,
		},
		{
			name: "custom Gateway name uses external seed Gateway",
			values: `
migrateGatewayAPI: true
httpRoute:
  gatewayName: seed-gateway
  gatewayNamespace: kubermatic
`,
			want: true,
		},
		{
			name: "custom Gateway namespace uses external seed Gateway",
			values: `
migrateGatewayAPI: true
httpRoute:
  gatewayName: kubermatic
  gatewayNamespace: seed-ingress
`,
			want: true,
		},
		{
			name: "Gateway API migration disabled",
			values: `
migrateGatewayAPI: false
httpRoute:
  gatewayName: seed-gateway
  gatewayNamespace: seed-ingress
`,
			want: false,
		},
		{
			name: "empty Gateway values deploy bundled seed Gateway controller",
			values: `
migrateGatewayAPI: true
`,
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := yamled.Load(strings.NewReader(tc.values))
			if err != nil {
				t.Fatalf("failed to load Helm values: %v", err)
			}

			_, got := seedExternalGatewayFromHTTPRouteValues(stack.DeployOptions{HelmValues: doc})
			if got != tc.want {
				t.Fatalf("seedExternalGatewayFromHTTPRouteValues() = %v, want %v", got, tc.want)
			}
		})
	}
}
