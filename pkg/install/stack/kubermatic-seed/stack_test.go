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

func TestSeedHTTPRouteGateway(t *testing.T) {
	testCases := []struct {
		name          string
		values        string
		wantName      string
		wantNamespace string
		wantExternal  bool
	}{
		{
			name: "default Gateway values deploy bundled seed Gateway controller",
			values: `
migrateGatewayAPI: true
httpRoute:
  gatewayName: kubermatic
  gatewayNamespace: kubermatic
`,
			wantName:      "kubermatic",
			wantNamespace: "kubermatic",
			wantExternal:  false,
		},
		{
			name: "custom Gateway reference alone still deploys bundled seed Gateway controller",
			values: `
migrateGatewayAPI: true
httpRoute:
  gatewayName: seed-gateway
  gatewayNamespace: kubermatic
`,
			wantName:      "seed-gateway",
			wantNamespace: "kubermatic",
			wantExternal:  false,
		},
		{
			name: "explicit external Gateway signal skips bundled seed Gateway controller",
			values: `
migrateGatewayAPI: true
httpRoute:
  externalGateway: true
  gatewayName: kubermatic
  gatewayNamespace: kubermatic
`,
			wantName:      "kubermatic",
			wantNamespace: "kubermatic",
			wantExternal:  true,
		},
		{
			name: "explicit external custom Gateway skips bundled seed Gateway controller",
			values: `
migrateGatewayAPI: true
httpRoute:
  externalGateway: true
  gatewayName: seed-gateway
  gatewayNamespace: seed-ingress
`,
			wantName:      "seed-gateway",
			wantNamespace: "seed-ingress",
			wantExternal:  true,
		},
		{
			name: "Gateway API migration disabled ignores external Gateway signal",
			values: `
migrateGatewayAPI: false
httpRoute:
  externalGateway: true
  gatewayName: seed-gateway
  gatewayNamespace: seed-ingress
`,
			wantName:      "kubermatic",
			wantNamespace: "kubermatic",
			wantExternal:  false,
		},
		{
			name: "empty Gateway values deploy bundled seed Gateway controller",
			values: `
migrateGatewayAPI: true
`,
			wantName:      "kubermatic",
			wantNamespace: "kubermatic",
			wantExternal:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := yamled.Load(strings.NewReader(tc.values))
			if err != nil {
				t.Fatalf("failed to load Helm values: %v", err)
			}

			gateway, gotExternal := seedHTTPRouteGateway(stack.DeployOptions{HelmValues: doc})
			if gotExternal != tc.wantExternal {
				t.Fatalf("seedHTTPRouteGateway() external = %v, want %v", gotExternal, tc.wantExternal)
			}
			if gateway.Name != tc.wantName || gateway.Namespace != tc.wantNamespace {
				t.Fatalf("seedHTTPRouteGateway() Gateway = %s, want %s/%s", gateway.String(), tc.wantNamespace, tc.wantName)
			}
		})
	}
}
